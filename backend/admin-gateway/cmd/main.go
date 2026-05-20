// Admin Gateway — OSS API server for system administrators
// Port: 8083 (default)
// Responsibilities: tenant management, RBAC, branding, system monitoring, audit log
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Grimid86/cgrates-ui/backend/pkg/branding"
	"github.com/Grimid86/cgrates-ui/backend/pkg/config"
	"github.com/Grimid86/cgrates-ui/backend/pkg/db"
	"github.com/Grimid86/cgrates-ui/backend/pkg/i18n"
	"github.com/Grimid86/cgrates-ui/backend/pkg/middleware"
	"github.com/Grimid86/cgrates-ui/backend/pkg/pulsar"
	"github.com/Grimid86/cgrates-ui/backend/pkg/redis"
	"github.com/Grimid86/cgrates-ui/backend/admin-gateway/handlers"
	"github.com/labstack/echo/v4"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if cfg.GatewayType != config.GatewayAdmin {
		log.Fatalf("Invalid gateway type: expected admin, got %s", cfg.GatewayType)
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
		Topic:  "persistent://billing/config/config.changes",
	})
	if err != nil {
		log.Printf("Warning: Pulsar not available: %v", err)
	} else {
		defer pulsarClient.Close()
	}

	i18nSvc := i18n.New(dbPool, redisClient)
	brandingSvc := branding.New(dbPool, redisClient)

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	jwtCfg := middleware.NewJWTConfig(cfg.Security.JWTSecret, cfg.Security.JWTAccessTTL, cfg.Security.JWTRefreshTTL)

	e.Use(middleware.CORSMiddleware(middleware.DefaultCORSConfig(cfg.CORS.AllowedOrigins)))
	e.Use(middleware.SanitizeMiddleware(middleware.DefaultSanitizeConfig()))
	e.Use(middleware.AuditMiddleware(middleware.AuditConfig{
		Pulsar:      pulsarClient,
		PortalType:  "admin",
		Async:       true,
		SyncForAuth: true,
	}))

	h := handlers.New(handlers.Dependencies{
		DB:        dbPool,
		Redis:     redisClient,
		Pulsar:    pulsarClient,
		I18n:      i18nSvc,
		Branding:  brandingSvc,
		JWTConfig: jwtCfg,
	})

	// Public
	e.GET("/health", h.Health)
	e.GET("/api/v1/branding", h.GetBranding)
	e.GET("/api/v1/locales", h.GetLocales)
	e.GET("/api/v1/translations/:locale", h.GetTranslations)

	// Auth
	e.POST("/api/v1/auth/login", h.Login, middleware.EndpointRateLimiter(redisClient, 5, time.Minute, middleware.LoginKeyExtractor))
	e.POST("/api/v1/auth/refresh", h.RefreshToken)
	e.POST("/api/v1/auth/logout", h.Logout, middleware.JWTMiddleware(jwtCfg))
	e.POST("/api/v1/auth/mfa/setup", h.SetupMFA, middleware.JWTMiddleware(jwtCfg))
	e.POST("/api/v1/auth/mfa/verify", h.VerifyMFA)
	e.POST("/api/v1/auth/mfa/disable", h.DisableMFA, middleware.JWTMiddleware(jwtCfg))

	// Protected
	api := e.Group("/api/v1")
	api.Use(middleware.JWTMiddleware(jwtCfg))
	api.Use(middleware.RateLimiter(middleware.RateLimiterConfig{
		Redis:        redisClient,
		LimitRPS:     cfg.Security.RateLimitRPS,
		Window:       time.Second,
		KeyPrefix:    "rl:ad:",
		KeyExtractor: middleware.DefaultKeyExtractor,
	}))
	api.Use(middleware.CSRFMiddleware(middleware.DefaultCSRFConfig(cfg.Security.CSRFSecret)))
	api.Use(middleware.RequireMFA()) // MFA mandatory for admin

	// Tenants
	api.GET("/tenants", h.ListTenants, middleware.RequirePermission("tenant:read"))
	api.GET("/tenants/:id", h.GetTenant, middleware.RequirePermission("tenant:read"))
	api.POST("/tenants", h.CreateTenant, middleware.RequirePermission("tenant:create"))
	api.PUT("/tenants/:id", h.UpdateTenant, middleware.RequirePermission("tenant:update"))
	api.POST("/tenants/:id/suspend", h.SuspendTenant, middleware.RequirePermission("tenant:update"))
	api.DELETE("/tenants/:id", h.DeleteTenant, middleware.RequirePermission("tenant:delete"))

	// Staff Users
	api.GET("/users", h.ListUsers, middleware.RequirePermission("user:read"))
	api.GET("/users/:id", h.GetUser, middleware.RequirePermission("user:read"))
	api.POST("/users", h.CreateUser, middleware.RequirePermission("user:create"))
	api.PUT("/users/:id", h.UpdateUser, middleware.RequirePermission("user:update"))
	api.POST("/users/:id/reset-mfa", h.ResetUserMFA, middleware.RequirePermission("user:reset_mfa"))
	api.POST("/users/:id/reset-password", h.ResetUserPassword, middleware.RequirePermission("user:update"))
	api.DELETE("/users/:id", h.DeleteUser, middleware.RequirePermission("user:delete"))

	// RBAC
	api.GET("/rbac/roles", h.ListRoles, middleware.RequirePermission("user:read"))
	api.GET("/rbac/roles/:id", h.GetRole, middleware.RequirePermission("user:read"))
	api.POST("/rbac/roles", h.CreateRole, middleware.RequirePermission("user:create"))
	api.PUT("/rbac/roles/:id/permissions", h.UpdateRolePermissions, middleware.RequirePermission("user:update"))
	api.GET("/rbac/permissions", h.ListPermissions, middleware.RequirePermission("user:read"))
	api.GET("/rbac/matrix", h.GetRBACMatrix, middleware.RequirePermission("user:read"))

	// Branding / White Label
	api.GET("/branding", h.GetBranding) // Public, but admin can also access
	api.PUT("/branding", h.UpdateBranding, middleware.RequirePermission("branding:manage"))
	api.POST("/branding/logo", h.UploadLogo, middleware.RequirePermission("branding:manage"))
	api.GET("/branding/email-templates", h.ListEmailTemplates, middleware.RequirePermission("branding:read"))
	api.PUT("/branding/email-templates/:id", h.UpdateEmailTemplate, middleware.RequirePermission("branding:manage"))
	api.POST("/branding/email-templates/:id/preview", h.PreviewEmailTemplate, middleware.RequirePermission("branding:manage"))

	// Localization
	api.GET("/locales", h.GetLocales)
	api.GET("/translations/:locale", h.GetTranslations)
	api.POST("/translations", h.ImportTranslations, middleware.RequirePermission("translation:manage"))

	// System Monitoring
	api.GET("/metrics", h.Metrics) // Prometheus format
	api.GET("/audit-log", h.GetAuditLog, middleware.RequirePermission("system:monitor"))
	api.GET("/pulsar/topics", h.GetPulsarTopics, middleware.RequirePermission("system:monitor"))
	api.GET("/pulsar/lag", h.GetPulsarLag, middleware.RequirePermission("system:monitor"))
	api.GET("/database/status", h.GetDatabaseStatus, middleware.RequirePermission("system:monitor"))

	// API Keys
	api.GET("/api-keys", h.ListAPIKeys, middleware.RequirePermission("api_key:read"))
	api.POST("/api-keys", h.CreateAPIKey, middleware.RequirePermission("api_key:manage"))
	api.DELETE("/api-keys/:id", h.DeleteAPIKey, middleware.RequirePermission("api_key:manage"))

	go func() {
		log.Printf("Admin Gateway starting on port %s", cfg.Port)
		if err := e.Start(":" + cfg.Port); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Admin Gateway...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}
}
