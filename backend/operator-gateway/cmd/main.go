// Operator Gateway — BSS API server for operator staff
// Port: 8082 (default)
// Responsibilities: subscriber CRUD, balance management, tariffs, CDR, bulk operations
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Grimid86/cgrates-ui/backend/pkg/auth"
	"github.com/Grimid86/cgrates-ui/backend/pkg/branding"
	"github.com/Grimid86/cgrates-ui/backend/pkg/cgrates"
	"github.com/Grimid86/cgrates-ui/backend/pkg/config"
	"github.com/Grimid86/cgrates-ui/backend/pkg/db"
	"github.com/Grimid86/cgrates-ui/backend/pkg/i18n"
	"github.com/Grimid86/cgrates-ui/backend/pkg/middleware"
	"github.com/Grimid86/cgrates-ui/backend/pkg/pulsar"
	"github.com/Grimid86/cgrates-ui/backend/pkg/redis"
	"github.com/Grimid86/cgrates-ui/backend/pkg/storage"
	"github.com/Grimid86/cgrates-ui/backend/operator-gateway/handlers"
	"github.com/labstack/echo/v4"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if cfg.GatewayType != config.GatewayOperator {
		log.Fatalf("Invalid gateway type: expected operator, got %s", cfg.GatewayType)
	}

	dbPool, err := db.New(db.Config{
		DSN:         cfg.DSN(),
		MaxConns:    int32(cfg.DB.MaxConns),
		MinConns:    5,
		MaxLifetime: 30 * time.Minute,
		MaxIdleTime: 10 * time.Minute,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	redisClient, err := redis.New(redis.Config{
		Addr:     cfg.RedisAddr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	pulsarClient, err := pulsar.New(pulsar.Config{
		URL:    cfg.Pulsar.URL,
		Token:  cfg.Pulsar.Token,
		Tenant: cfg.Pulsar.Tenant,
		Topic:  "persistent://billing/commands/balance.adjust",
	})
	if err != nil {
		log.Printf("Warning: Pulsar not available: %v", err)
	} else {
		defer pulsarClient.Close()
	}

	cgratesClient := cgrates.NewClient(cfg.CGRateS.Host, cfg.CGRateS.Port, cfg.CGRateS.Timeout)
	i18nSvc := i18n.New(dbPool, redisClient)
	brandingSvc := branding.New(dbPool, redisClient)

	minioClient, err := storage.New(storage.Config{
		Endpoint:  cfg.MinIO.Endpoint,
		AccessKey: cfg.MinIO.AccessKey,
		SecretKey: cfg.MinIO.SecretKey,
		Bucket:    cfg.MinIO.Bucket,
		UseSSL:    cfg.MinIO.UseSSL,
	})
	if err != nil {
		log.Printf("Warning: MinIO not available: %v", err)
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	jwtCfg := middleware.NewJWTConfig(cfg.Security.JWTSecret, cfg.Security.JWTAccessTTL, cfg.Security.JWTRefreshTTL)

	e.Use(middleware.CORSMiddleware(middleware.DefaultCORSConfig(cfg.CORS.AllowedOrigins)))
	e.Use(middleware.SanitizeMiddleware(middleware.DefaultSanitizeConfig()))
	e.Use(middleware.AuditMiddleware(middleware.AuditConfig{
		Pulsar:     pulsarClient,
		PortalType: "operator",
		Async:      true,
	}))

	staffAuth := auth.NewStaffAuth(dbPool, jwtCfg, cfg)

	h := handlers.New(handlers.Dependencies{
		DB:        dbPool,
		Redis:     redisClient,
		Pulsar:    pulsarClient,
		CGRateS:   cgratesClient,
		I18n:      i18nSvc,
		Branding:  brandingSvc,
		Storage:   minioClient,
		JWTConfig: jwtCfg,
		StaffAuth: staffAuth,
		Config:    cfg,
	})

	// Public
	e.GET("/health", h.Health)
	e.GET("/api/v1/branding", h.GetBranding)
	e.GET("/api/v1/locales", h.GetLocales)
	e.GET("/api/v1/translations/:locale", h.GetTranslations)

	// Auth
	e.POST("/api/v1/auth/login", h.Login, middleware.EndpointRateLimiter(redisClient, 10, time.Minute, middleware.LoginKeyExtractor))
	e.POST("/api/v1/auth/refresh", h.RefreshToken)
	e.POST("/api/v1/auth/logout", h.Logout, middleware.JWTMiddleware(jwtCfg))

	// MFA (JWT required)
	mfa := e.Group("/api/v1/auth/mfa")
	mfa.Use(middleware.JWTMiddleware(jwtCfg))
	mfa.POST("/setup", h.SetupMFA)
	mfa.POST("/verify", h.VerifyMFA)
	mfa.POST("/disable", h.DisableMFA)

	// Protected
	api := e.Group("/api/v1")
	api.Use(middleware.JWTMiddleware(jwtCfg))
	api.Use(middleware.RateLimiter(middleware.RateLimiterConfig{
		Redis:        redisClient,
		LimitRPS:     cfg.Security.RateLimitRPS,
		Window:       time.Second,
		KeyPrefix:    "rl:op:",
		KeyExtractor: middleware.DefaultKeyExtractor,
	}))
	api.Use(middleware.IdempotencyMiddleware(dbPool, redisClient, cfg))
	api.Use(middleware.CSRFMiddleware(middleware.DefaultCSRFConfig(cfg.Security.CSRFSecret)))
	api.Use(middleware.TokenBlacklistMiddleware(redisClient))

	// Subscribers
	api.GET("/subscribers", h.ListSubscribers, middleware.RequirePermission("subscriber:read"))
	api.GET("/subscribers/:id", h.GetSubscriber, middleware.RequirePermission("subscriber:read"))
	api.POST("/subscribers", h.CreateSubscriber, middleware.RequirePermission("subscriber:write"))
	api.PUT("/subscribers/:id", h.UpdateSubscriber, middleware.RequirePermission("subscriber:write"))
	api.POST("/subscribers/:id/block", h.BlockSubscriber, middleware.RequirePermission("subscriber:write"))
	api.POST("/subscribers/:id/unblock", h.UnblockSubscriber, middleware.RequirePermission("subscriber:write"))
	api.POST("/subscribers/:id/migrate-tariff", h.MigrateTariff, middleware.RequirePermission("subscriber:write"))

	// Balance
	api.POST("/subscribers/:id/balance/adjust", h.AdjustBalance, middleware.RequirePermission("balance:write"))
	api.POST("/subscribers/:id/balance/freeze", h.FreezeBalance, middleware.RequirePermission("balance:write"))
	api.POST("/subscribers/:id/balance/unfreeze", h.UnfreezeBalance, middleware.RequirePermission("balance:write"))

	// Tariffs
	api.GET("/tariffs", h.ListTariffs, middleware.RequirePermission("tariff:read"))
	api.GET("/tariffs/:id", h.GetTariff, middleware.RequirePermission("tariff:read"))
	api.POST("/tariffs", h.CreateTariff, middleware.RequirePermission("tariff:write"))
	api.PUT("/tariffs/:id", h.UpdateTariff, middleware.RequirePermission("tariff:write"))
	api.DELETE("/tariffs/:id", h.DeleteTariff, middleware.RequirePermission("tariff:delete"))
	api.POST("/tariffs/:id/activate", h.ActivateTariff, middleware.RequirePermission("tariff:write"))

	// CDR
	api.GET("/cdr", h.ListCDR, middleware.RequirePermission("cdr:read"))
	api.GET("/cdr/export", h.ExportCDR, middleware.RequirePermission("cdr:export"))

	// Sessions
	api.GET("/sessions/active", h.GetActiveSessions, middleware.RequirePermission("session:read"))

	// Bulk
	api.POST("/bulk/tariff-change", h.BulkTariffChange, middleware.RequirePermission("tariff:write"))
	api.POST("/bulk/bonus", h.BulkBonus, middleware.RequirePermission("balance:write"))

	// Reports
	api.GET("/reports/usage", h.UsageReport, middleware.RequirePermission("report:read"))
	api.GET("/reports/revenue", h.RevenueReport, middleware.RequirePermission("report:read"))

	go func() {
		log.Printf("Operator Gateway starting on port %s", cfg.Port)
		if err := e.Start(":" + cfg.Port); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Operator Gateway...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}
}
