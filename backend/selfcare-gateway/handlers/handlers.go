// Package handlers provides HTTP handlers for the SelfCare Gateway.
package handlers

import (
	"net/http"

	"github.com/Grimid86/cgrates-ui/backend/pkg/branding"
	"github.com/Grimid86/cgrates-ui/backend/pkg/cgrates"
	"github.com/Grimid86/cgrates-ui/backend/pkg/db"
	"github.com/Grimid86/cgrates-ui/backend/pkg/i18n"
	"github.com/Grimid86/cgrates-ui/backend/pkg/middleware"
	"github.com/Grimid86/cgrates-ui/backend/pkg/pulsar"
	"github.com/Grimid86/cgrates-ui/backend/pkg/redis"
	"github.com/labstack/echo/v4"
)

// Dependencies holds all service dependencies for handlers
type Dependencies struct {
	DB        *db.Pool
	Redis     *redis.Client
	Pulsar    *pulsar.Client
	CGRateS   *cgrates.Client
	I18n      *i18n.Service
	Branding  *branding.Service
	JWTConfig middleware.JWTConfig
}

// Handler contains all HTTP handlers
type Handler struct {
	deps Dependencies
}

// New creates a new handler instance
func New(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}

// Health returns service health status
func (h *Handler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"portal":    "selfcare",
		"version":   "1.0.0",
		"timestamp": "2026-05-20T09:20:27Z",
	})
}

// GetBranding returns branding configuration for current tenant (public)
func (h *Handler) GetBranding(c echo.Context) error {
	domain := c.QueryParam("domain")
	if domain == "" {
		domain = c.Request().Host
	}

	cfg, err := h.deps.Branding.GetByDomain(c.Request().Context(), domain)
	if err != nil {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"product_name": "CGRateS Billing",
			"colors": map[string]string{
				"primary": "#007bff",
			},
		})
	}

	return c.JSON(http.StatusOK, cfg)
}

// GetLocales returns available locales (public)
func (h *Handler) GetLocales(c echo.Context) error {
	locales, err := h.deps.I18n.GetLocales(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load locales")
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"locales": locales})
}

// GetTranslations returns translations for a locale (public)
func (h *Handler) GetTranslations(c echo.Context) error {
	locale := c.Param("locale")
	if locale == "" {
		locale = "en"
	}

	translations, err := h.deps.I18n.GetTranslations(c.Request().Context(), locale)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load translations")
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"locale":       locale,
		"translations": translations,
	})
}

// Login handles subscriber authentication
func (h *Handler) Login(c echo.Context) error {
	var req struct {
		MSISDN       string `json:"msisdn" validate:"required"`
		PIN          string `json:"pin" validate:"required"`
		CaptchaToken string `json:"captcha_token"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	// TODO: Validate MSISDN + PIN against subscriber_credentials table
	// TODO: Generate JWT tokens
	// TODO: Return subscriber info

	return c.JSON(http.StatusOK, map[string]interface{}{
		"access_token":  "eyJhbGciOiJIUzI1NiIs...",
		"refresh_token": "eyJhbGciOiJIUzI1NiIs...",
		"expires_in":    900,
		"token_type":    "Bearer",
	})
}

// RefreshToken handles token refresh
func (h *Handler) RefreshToken(c echo.Context) error {
	var req struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	// TODO: Validate refresh token, issue new access token
	return c.JSON(http.StatusOK, map[string]interface{}{
		"access_token": "eyJhbGciOiJIUzI1NiIs...",
		"expires_in":   900,
	})
}

// Logout revokes current session
func (h *Handler) Logout(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}
	// TODO: Revoke session in database
	return c.NoContent(http.StatusNoContent)
}

// GetBalance returns subscriber balances
func (h *Handler) GetBalance(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	// TODO: Fetch from CGRateS via ApierV1.GetAccount
	return c.JSON(http.StatusOK, map[string]interface{}{
		"subscriber_id": claims.SubscriberID,
		"balances": []map[string]interface{}{
			{"type": "*monetary", "value": 150.50, "currency": "RUB"},
			{"type": "*data", "value": 5120.00, "unit": "MB"},
			{"type": "*voice", "value": 300.00, "unit": "minutes"},
		},
	})
}

// GetCDRHistory returns call detail records
func (h *Handler) GetCDRHistory(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	// TODO: Fetch from CGRateS via ApierV1.GetCDRs with MSISDN filter
	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": []map[string]interface{}{
			{
				"id":               "uuid",
				"type":             "voice",
				"destination":      "79009876543",
				"duration_seconds": 125,
				"cost":             12.50,
				"currency":         "RUB",
				"started_at":       "2026-05-19T14:30:00Z",
			},
		},
		"pagination": map[string]interface{}{
			"page":        1,
			"per_page":    20,
			"total":       145,
			"total_pages": 8,
		},
	})
}

// TopUp initiates a top-up operation
func (h *Handler) TopUp(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	var req struct {
		Amount        float64 `json:"amount" validate:"required,gt=0"`
		Currency      string  `json:"currency" validate:"required"`
		PaymentMethod string  `json:"payment_method" validate:"required"`
		ReturnURL     string  `json:"return_url" validate:"required,url"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	// TODO: Validate idempotency key
	// TODO: Publish to Pulsar topic billing.events.topups
	// TODO: Return payment provider URL

	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"status":         "pending",
		"payment_url":    "https://payment.provider.com/session/abc123",
		"transaction_id": "uuid",
		"expires_at":     "2026-05-20T09:30:00Z",
	})
}

// GetProfile returns subscriber profile
func (h *Handler) GetProfile(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"msisdn": claims.MSISDN,
		"locale": claims.Locale,
		"tariff_plan": map[string]interface{}{
			"id":   "uuid",
			"name": "Unlimited Plus",
		},
	})
}

// UpdateProfile updates subscriber profile
func (h *Handler) UpdateProfile(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}
	// TODO: Update subscriber_credentials
	return c.NoContent(http.StatusNoContent)
}

// ChangePIN changes subscriber PIN
func (h *Handler) ChangePIN(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	var req struct {
		OldPIN string `json:"old_pin" validate:"required"`
		NewPIN string `json:"new_pin" validate:"required,min=4,max=6"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	// TODO: Verify old PIN, update hash
	return c.NoContent(http.StatusNoContent)
}

// GetSessions returns active subscriber sessions
func (h *Handler) GetSessions(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}
	// TODO: Return active sessions from subscriber_sessions table
	return c.JSON(http.StatusOK, map[string]interface{}{
		"sessions": []map[string]interface{}{},
	})
}

// RevokeSession revokes a specific session
func (h *Handler) RevokeSession(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}
	// TODO: Revoke session by ID
	return c.NoContent(http.StatusNoContent)
}
