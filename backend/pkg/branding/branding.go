// Package branding provides white-label configuration retrieval with caching.
package branding

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Grimid86/cgrates-ui/backend/pkg/db"
	"github.com/Grimid86/cgrates-ui/backend/pkg/redis"
)

// Service handles branding configuration
type Service struct {
	db    *db.Pool
	redis *redis.Client
}

// New creates a new branding service
func New(dbPool *db.Pool, redisClient *redis.Client) *Service {
	return &Service{db: dbPool, redis: redisClient}
}

// Config represents tenant branding configuration
type Config struct {
	TenantID           int64             `json:"tenant_id"`
	ProductName        string            `json:"product_name"`
	PrimaryColor       string            `json:"primary_color"`
	SecondaryColor     string            `json:"secondary_color"`
	AccentColor        string            `json:"accent_color"`
	DangerColor        string            `json:"danger_color"`
	LogoURL            string            `json:"logo_url"`
	FaviconURL         string            `json:"favicon_url"`
	LoginBackgroundURL string            `json:"login_background_url"`
	SupportEmail       string            `json:"support_email"`
	SupportPhone       string            `json:"support_phone"`
	SupportTelegram    string            `json:"support_telegram"`
	Timezone           string            `json:"timezone"`
	DateFormat         string            `json:"date_format"`
	FirstDayOfWeek     int               `json:"first_day_of_week"`
	CurrencySymbol     string            `json:"currency_symbol"`
	CSSVariables       map[string]string `json:"css_variables"`
}

// GetByTenantID returns branding for a tenant
func (s *Service) GetByTenantID(ctx context.Context, tenantID int64) (*Config, error) {
	cacheKey := fmt.Sprintf("branding:%d", tenantID)

	// Try cache
	if s.redis != nil {
		cached, err := s.redis.Get(ctx, cacheKey)
		if err == nil && cached != "" {
			var cfg Config
			if err := json.Unmarshal([]byte(cached), &cfg); err == nil {
				return &cfg, nil
			}
		}
	}

	// Load from DB
	query := `
		SELECT tenant_id, product_name, primary_color, secondary_color,
		       accent_color, danger_color, logo_url, favicon_url,
		       login_background_url, support_email, support_phone,
		       support_telegram, timezone, date_format, first_day_of_week, currency_symbol
		FROM branding_config
		WHERE tenant_id = $1
	`
	var cfg Config
	err := s.db.QueryRow(ctx, query, tenantID).Scan(
		&cfg.TenantID, &cfg.ProductName, &cfg.PrimaryColor, &cfg.SecondaryColor,
		&cfg.AccentColor, &cfg.DangerColor, &cfg.LogoURL, &cfg.FaviconURL,
		&cfg.LoginBackgroundURL, &cfg.SupportEmail, &cfg.SupportPhone,
		&cfg.SupportTelegram, &cfg.Timezone, &cfg.DateFormat, &cfg.FirstDayOfWeek, &cfg.CurrencySymbol,
	)
	if err != nil {
		return nil, fmt.Errorf("load branding: %w", err)
	}

	// Generate CSS variables
	cfg.CSSVariables = map[string]string{
		"--brand-primary":    cfg.PrimaryColor,
		"--brand-secondary":  cfg.SecondaryColor,
		"--brand-accent":     cfg.AccentColor,
		"--brand-danger":     cfg.DangerColor,
		"--brand-logo-url":   fmt.Sprintf("url('%s')", cfg.LogoURL),
		"--brand-product-name": cfg.ProductName,
	}

	// Cache
	if s.redis != nil {
		data, _ := json.Marshal(cfg)
		s.redis.Set(ctx, cacheKey, data, 5*time.Minute)
	}

	return &cfg, nil
}

// GetByDomain returns branding for a domain
func (s *Service) GetByDomain(ctx context.Context, domain string) (*Config, error) {
	var tenantID int64
	query := `SELECT tenant_id FROM domain_tenant_mapping WHERE domain = $1 AND is_active = true`
	err := s.db.QueryRow(ctx, query, domain).Scan(&tenantID)
	if err != nil {
		// Return default system branding
		tenantID = 1
	}
	return s.GetByTenantID(ctx, tenantID)
}

// InvalidateCache clears branding cache for a tenant
func (s *Service) InvalidateCache(ctx context.Context, tenantID int64) error {
	if s.redis == nil {
		return nil
	}
	return s.redis.Delete(ctx, fmt.Sprintf("branding:%d", tenantID))
}
