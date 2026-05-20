// Package config provides centralized configuration management for all gateways.
// Each gateway loads its own config based on GATEWAY_TYPE environment variable.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// GatewayType identifies which portal this instance serves
type GatewayType string

const (
	GatewaySelfCare GatewayType = "selfcare"
	GatewayOperator GatewayType = "operator"
	GatewayAdmin    GatewayType = "admin"
)

// Config holds all configuration for a gateway instance
type Config struct {
	// Gateway identification
	GatewayType GatewayType `validate:"required,oneof=selfcare operator admin"`
	Port        string      `validate:"required,numeric"`
	Environment string      `validate:"oneof=development staging production"`

	// Database
	DB struct {
		Host     string `validate:"required"`
		Port     int    `validate:"required,numeric"`
		Database string `validate:"required"`
		User     string `validate:"required"`
		Password string `validate:"required"`
		SSLMode  string `validate:"oneof=disable require verify-full"`
		MaxConns int    `validate:"gt=0"`
	}

	// Redis
	Redis struct {
		Host     string `validate:"required"`
		Port     int    `validate:"required,numeric"`
		Password string
		DB       int
	}

	// Apache Pulsar
	Pulsar struct {
		URL        string `validate:"required,uri"`
		AdminURL   string `validate:"omitempty,uri"`
		Tenant     string `validate:"required"`
		Token      string
	}

	// CGRateS RPC
	CGRateS struct {
		Host     string `validate:"required"`
		Port     int    `validate:"required,numeric"`
		Protocol string `validate:"oneof=jsonrpc gob"`
		Timeout  time.Duration
	}

	// Security
	Security struct {
		JWTSecret        string        `validate:"required,min=32"`
		JWTAccessTTL     time.Duration `validate:"required"`
		JWTRefreshTTL    time.Duration `validate:"required"`
		CSRFSecret       string        `validate:"required,min=16"`
		BcryptCost       int           `validate:"gte=10,lte=14"`
		MFAEnabled       bool
		MFAIssuer        string
		RateLimitRPS     int           `validate:"gt=0"`
		RateLimitRPM     int           `validate:"gt=0"`
		IdempotencyTTL   time.Duration `validate:"required"`
		MaxFailedLoginAttempts int     `validate:"gt=0"`
		LockoutDuration  time.Duration `validate:"required"`
	}

	// Branding / CDN
	Branding struct {
		CDNBaseURL     string `validate:"omitempty,uri"`
		MaxLogoSizeMB  int    `validate:"gte=1,lte=10"`
	}

	// MinIO / S3 Storage
	MinIO struct {
		Endpoint  string `validate:"required,uri"`
		AccessKey string `validate:"required"`
		SecretKey string `validate:"required"`
		Bucket    string `validate:"required"`
		UseSSL    bool
	}

	// CORS
	CORS struct {
		AllowedOrigins []string `validate:"required,min=1"`
	}

	// Audit
	Audit struct {
		AsyncToPulsar bool
		SyncForAuth   bool
	}
}

// DSN returns PostgreSQL connection string
func (c *Config) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.DB.User, c.DB.Password, c.DB.Host, c.DB.Port, c.DB.Database, c.DB.SSLMode)
}

// RedisAddr returns Redis connection address
func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}

// CGRateSAddr returns CGRateS RPC address
func (c *Config) CGRateSAddr() string {
	return fmt.Sprintf("%s:%d", c.CGRateS.Host, c.CGRateS.Port)
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{}

	// Gateway type
	gt := os.Getenv("GATEWAY_TYPE")
	switch gt {
	case "selfcare":
		cfg.GatewayType = GatewaySelfCare
	case "operator":
		cfg.GatewayType = GatewayOperator
	case "admin":
		cfg.GatewayType = GatewayAdmin
	default:
		return nil, fmt.Errorf("invalid GATEWAY_TYPE: %s (must be selfcare, operator, or admin)", gt)
	}

	// Server
	cfg.Port = getEnv("PORT", "8080")
	cfg.Environment = getEnv("ENVIRONMENT", "development")

	// Database
	cfg.DB.Host = getEnv("POSTGRES_HOST", "localhost")
	cfg.DB.Port = getIntEnv("POSTGRES_PORT", 5432)
	cfg.DB.Database = getEnv("POSTGRES_DB", "uibill")
	cfg.DB.User = getEnv("POSTGRES_USER", "postgres")
	cfg.DB.Password = getEnv("POSTGRES_PASSWORD", "postgres")
	cfg.DB.SSLMode = getEnv("POSTGRES_SSLMODE", "disable")
	cfg.DB.MaxConns = getIntEnv("DB_MAX_CONNS", 25)

	// Redis
	cfg.Redis.Host = getEnv("REDIS_HOST", "localhost")
	cfg.Redis.Port = getIntEnv("REDIS_PORT", 6379)
	cfg.Redis.Password = getEnv("REDIS_PASSWORD", "")
	cfg.Redis.DB = getIntEnv("REDIS_DB", 0)

	// Pulsar
	cfg.Pulsar.URL = getEnv("PULSAR_URL", "pulsar://localhost:6650")
	cfg.Pulsar.AdminURL = getEnv("PULSAR_ADMIN_URL", "")
	cfg.Pulsar.Tenant = getEnv("PULSAR_TENANT", "billing")
	cfg.Pulsar.Token = getEnv("PULSAR_TOKEN", "")

	// CGRateS
	cfg.CGRateS.Host = getEnv("CGRATES_HOST", "localhost")
	cfg.CGRateS.Port = getIntEnv("CGRATES_PORT", 2012)
	cfg.CGRateS.Protocol = getEnv("CGRATES_PROTOCOL", "jsonrpc")
	cfg.CGRateS.Timeout = getDurationEnv("CGRATES_TIMEOUT", 10*time.Second)

	// Security
	switch cfg.GatewayType {
	case GatewaySelfCare:
		cfg.Security.JWTSecret = getEnv("JWT_SECRET_SELFCARE", "")
		cfg.Security.RateLimitRPM = getIntEnv("RATE_LIMIT_SELFCARE_RPM", 30)
		cfg.Security.RateLimitRPS = cfg.Security.RateLimitRPM / 60
	case GatewayOperator:
		cfg.Security.JWTSecret = getEnv("JWT_SECRET_OPERATOR", "")
		cfg.Security.RateLimitRPS = getIntEnv("RATE_LIMIT_OPERATOR_RPS", 100)
	case GatewayAdmin:
		cfg.Security.JWTSecret = getEnv("JWT_SECRET_ADMIN", "")
		cfg.Security.RateLimitRPS = getIntEnv("RATE_LIMIT_ADMIN_RPS", 50)
	}
	cfg.Security.JWTAccessTTL = getDurationEnv("JWT_TTL_ACCESS", 15*time.Minute)
	cfg.Security.JWTRefreshTTL = getDurationEnv("JWT_TTL_REFRESH", 7*24*time.Hour)
	cfg.Security.CSRFSecret = getEnv("CSRF_SECRET", "change-me-csrf-secret")
	cfg.Security.BcryptCost = getIntEnv("BCRYPT_COST", 12)
	cfg.Security.MFAEnabled = getBoolEnv("MFA_ENABLED", false)
	cfg.Security.MFAIssuer = getEnv("MFA_ISSUER", "UI-Bill")
	cfg.Security.IdempotencyTTL = getDurationEnv("IDEMPOTENCY_TTL", 24*time.Hour)
	cfg.Security.MaxFailedLoginAttempts = getIntEnv("MAX_FAILED_LOGIN_ATTEMPTS", 5)
	cfg.Security.LockoutDuration = getDurationEnv("LOCKOUT_DURATION", 15*time.Minute)

	// Branding
	cfg.Branding.CDNBaseURL = getEnv("CDN_BASE_URL", "")
	cfg.Branding.MaxLogoSizeMB = getIntEnv("MAX_LOGO_SIZE_MB", 2)

	// MinIO
	cfg.MinIO.Endpoint = getEnv("MINIO_ENDPOINT", "http://localhost:9000")
	cfg.MinIO.AccessKey = getEnv("MINIO_ACCESS_KEY", "minioadmin")
	cfg.MinIO.SecretKey = getEnv("MINIO_SECRET_KEY", "minioadmin")
	cfg.MinIO.Bucket = getEnv("MINIO_BUCKET", "uibill")
	cfg.MinIO.UseSSL = getEnv("MINIO_USE_SSL", "false") == "true"

	// CORS origins per gateway
	cfg.CORS.AllowedOrigins = getCORSOrigins(cfg.GatewayType)

	// Audit
	cfg.Audit.AsyncToPulsar = getBoolEnv("AUDIT_ASYNC_PULSAR", true)
	cfg.Audit.SyncForAuth = getBoolEnv("AUDIT_SYNC_AUTH", true)

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func getCORSOrigins(gt GatewayType) []string {
	var envKey string
	switch gt {
	case GatewaySelfCare:
		envKey = "CORS_ORIGINS_SELFCARE"
	case GatewayOperator:
		envKey = "CORS_ORIGINS_OPERATOR"
	case GatewayAdmin:
		envKey = "CORS_ORIGINS_ADMIN"
	}
	if v := os.Getenv(envKey); v != "" {
		return []string{v}
	}
	// Development defaults
	return []string{"*"}
}
