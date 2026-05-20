// Package i18n provides localization support with Redis caching.
package i18n

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Grimid86/cgrates-ui/backend/pkg/db"
	"github.com/Grimid86/cgrates-ui/backend/pkg/redis"
)

// Service handles translation loading and caching
type Service struct {
	db    *db.Pool
	redis *redis.Client
}

// New creates a new i18n service
func New(dbPool *db.Pool, redisClient *redis.Client) *Service {
	return &Service{db: dbPool, redis: redisClient}
}

// Translation represents a single translation
type Translation struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Category string `json:"category"`
}

// GetTranslations loads all translations for a locale, using cache if available
func (s *Service) GetTranslations(ctx context.Context, locale string) (map[string]map[string]string, error) {
	cacheKey := fmt.Sprintf("i18n:%s", locale)

	// Try cache first
	if s.redis != nil {
		cached, err := s.redis.Get(ctx, cacheKey)
		if err == nil && cached != "" {
			var result map[string]map[string]string
			if err := json.Unmarshal([]byte(cached), &result); err == nil {
				return result, nil
			}
		}
	}

	// Load from database
	query := `
		SELECT t.key, t.value, t.category
		FROM translation t
		JOIN language l ON t.language_id = l.id
		WHERE l.code = $1 AND l.is_active = true
	`
	rows, err := s.db.Query(ctx, query, locale)
	if err != nil {
		return nil, fmt.Errorf("load translations: %w", err)
	}
	defer rows.Close()

	result := make(map[string]map[string]string)
	for rows.Next() {
		var key, value, category string
		if err := rows.Scan(&key, &value, &category); err != nil {
			continue
		}
		if result[category] == nil {
			result[category] = make(map[string]string)
		}
		result[category][key] = value
	}

	// Cache result
	if s.redis != nil {
		data, _ := json.Marshal(result)
		s.redis.Set(ctx, cacheKey, data, time.Hour)
	}

	return result, nil
}

// GetTranslation gets a single translation with fallback
func (s *Service) GetTranslation(ctx context.Context, locale, key, category string) string {
	translations, err := s.GetTranslations(ctx, locale)
	if err != nil {
		return key // Fallback to key itself
	}

	if cat, ok := translations[category]; ok {
		if val, ok := cat[key]; ok {
			return val
		}
	}

	// Try English fallback
	if locale != "en" {
		return s.GetTranslation(ctx, "en", key, category)
	}

	return key
}

// InvalidateCache clears translation cache for a locale
func (s *Service) InvalidateCache(ctx context.Context, locale string) error {
	if s.redis == nil {
		return nil
	}
	return s.redis.Delete(ctx, fmt.Sprintf("i18n:%s", locale))
}

// GetLocales returns available active locales
func (s *Service) GetLocales(ctx context.Context) ([]map[string]interface{}, error) {
	query := `SELECT code, name, default_rtl FROM language WHERE is_active = true ORDER BY name`
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("load locales: %w", err)
	}
	defer rows.Close()

	var locales []map[string]interface{}
	for rows.Next() {
		var code, name string
		var rtl bool
		if err := rows.Scan(&code, &name, &rtl); err != nil {
			continue
		}
		locales = append(locales, map[string]interface{}{
			"code": code,
			"name": name,
			"rtl":  rtl,
		})
	}

	return locales, nil
}
