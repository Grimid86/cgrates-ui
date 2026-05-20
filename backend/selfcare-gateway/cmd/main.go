// SelfCare Gateway — Subscriber-facing API server
// Port: 8081 (default)
// Responsibilities: balance read, CDR history, top-up, profile, notifications
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
	"github.com/Grimid86/cgrates-ui/backend/selfcare-gateway/handlers"
	"github.com/labstack/echo/v4"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if cfg.GatewayType != config.GatewaySelfCare {
		log.Fatalf("Invalid gateway type: expected selfcare, got %s", cfg.GatewayType)
	}

	// Initialize database
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

	// Initialize Redis
	redisClient, err := redis.New(redis.Config{
		Addr:     cfg.RedisAddr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize Pulsar producer
	pulsarClient, err := pulsar.New(pulsar.Config{
		URL:    cfg.Pulsar.URL,
		Token:  cfg.Pulsar.Token,
		Tenant: cfg.Pulsar.Tenant,
		Topic:  "persistent://billing/events/topups",
	})
	if err != nil {
		log.Printf("Warning: Pulsar not available: %v", err)
	} else {
		defer pulsarClient.Close()
	}

	// Initialize CGRateS client
	cgratesClient := cgrates.NewClient(cfg.CGRateS.Host, cfg.CGRateS.Port, cfg.CGRateS.Timeout)

	// Initialize services
	i18nSvc := i18n.New(dbPool, redisClient)
	brandingSvc := branding.New(dbPool, redisClient)

	// Initialize Echo
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Middleware chain (SelfCare specific)
	jwtCfg := middleware.NewJWTConfig(cfg.Security.JWTSecret, cfg.Security.JWTAccessTTL, cfg.Security.JWTRefreshTTL)

	// Global middleware
	e.Use(middleware.CORSMiddleware(middleware.DefaultCORSConfig(cfg.CORS.AllowedOrigins)))
	e.Use(middleware.SanitizeMiddleware(middleware.DefaultSanitizeConfig()))
	e.Use(middleware.AuditMiddleware(middleware.AuditConfig{
		Pulsar:     pulsarClient,
		PortalType: "selfcare",
		Async:      true,
	}))

	// Initialize auth service
	selfcareAuth := auth.NewSelfCareAuth(dbPool, jwtCfg, cfg)

	// Initialize handlers
	h := handlers.New(handlers.Dependencies{
		DB:           dbPool,
		Redis:        redisClient,
		Pulsar:       pulsarClient,
		CGRateS:      cgratesClient,
		I18n:         i18nSvc,
		Branding:     brandingSvc,
		JWTConfig:    jwtCfg,
		SelfCareAuth: selfcareAuth,
		Config:       cfg,
	})

	// Public routes (no auth)
	e.GET("/health", h.Health)
	e.GET("/api/v1/branding", h.GetBranding)
	e.GET("/api/v1/locales", h.GetLocales)
	e.GET("/api/v1/translations/:locale", h.GetTranslations)

	// Auth routes
	e.POST("/api/v1/auth/login", h.Login, middleware.EndpointRateLimiter(redisClient, 5, time.Minute, middleware.LoginKeyExtractor))
	e.POST("/api/v1/auth/refresh", h.RefreshToken)
	e.POST("/api/v1/auth/logout", h.Logout, middleware.JWTMiddleware(jwtCfg))

	// Protected routes (JWT required)
	api := e.Group("/api/v1")
	api.Use(middleware.JWTMiddleware(jwtCfg))
	api.Use(middleware.RateLimiter(middleware.RateLimiterConfig{
		Redis:        redisClient,
		LimitRPM:     cfg.Security.RateLimitRPM,
		Window:       time.Minute,
		KeyPrefix:    "rl:sc:",
		KeyExtractor: middleware.DefaultKeyExtractor,
	}))
	api.Use(middleware.IdempotencyMiddleware(dbPool, redisClient, cfg))
	api.Use(middleware.CSRFMiddleware(middleware.DefaultCSRFConfig(cfg.Security.CSRFSecret)))
	api.Use(middleware.TokenBlacklistMiddleware(redisClient))

	// Balance
	api.GET("/balance", h.GetBalance)

	// CDR
	api.GET("/cdr", h.GetCDRHistory)

	// Top-up
	api.POST("/topup", h.TopUp)

	// Profile
	api.GET("/profile", h.GetProfile)
	api.PUT("/profile", h.UpdateProfile)
	api.PUT("/profile/change-pin", h.ChangePIN)
	api.GET("/profile/sessions", h.GetSessions)
	api.DELETE("/profile/sessions/:id", h.RevokeSession)

	// Start server
	go func() {
		log.Printf("SelfCare Gateway starting on port %s", cfg.Port)
		if err := e.Start(":" + cfg.Port); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down SelfCare Gateway...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}
}
