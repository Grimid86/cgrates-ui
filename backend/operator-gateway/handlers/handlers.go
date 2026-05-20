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

func New(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}

func (h *Handler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "healthy",
		"portal":  "operator",
		"version": "1.0.0",
	})
}

func (h *Handler) GetBranding(c echo.Context) error {
	domain := c.QueryParam("domain")
	if domain == "" {
		domain = c.Request().Host
	}
	cfg, err := h.deps.Branding.GetByDomain(c.Request().Context(), domain)
	if err != nil {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"product_name": "CGRateS Billing",
			"colors":       map[string]string{"primary": "#007bff"},
		})
	}
	return c.JSON(http.StatusOK, cfg)
}

func (h *Handler) GetLocales(c echo.Context) error {
	locales, err := h.deps.I18n.GetLocales(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load locales")
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"locales": locales})
}

func (h *Handler) GetTranslations(c echo.Context) error {
	locale := c.Param("locale")
	if locale == "" {
		locale = "en"
	}
	translations, err := h.deps.I18n.GetTranslations(c.Request().Context(), locale)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load translations")
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"locale": locale, "translations": translations})
}

func (h *Handler) Login(c echo.Context) error {
	var req struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required"`
		MFACode  string `json:"mfa_code"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	// TODO: Authenticate, verify MFA for Admin/Reseller roles, issue JWT
	return c.JSON(http.StatusOK, map[string]interface{}{
		"access_token":  "eyJ...",
		"refresh_token": "eyJ...",
		"expires_in":    900,
		"user": map[string]interface{}{
			"id":          100,
			"email":       req.Email,
			"role":        "admin",
			"permissions": []string{"subscriber:read", "subscriber:write"},
		},
	})
}

func (h *Handler) RefreshToken(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{"access_token": "eyJ...", "expires_in": 900})
}

func (h *Handler) Logout(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

// Subscribers
func (h *Handler) ListSubscribers(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": []map[string]interface{}{
			{"id": "uuid", "msisdn": "79001234567", "status": "active"},
		},
		"pagination": map[string]interface{}{"page": 1, "per_page": 20, "total": 1},
	})
}

func (h *Handler) GetSubscriber(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{"id": c.Param("id"), "msisdn": "79001234567"})
}

func (h *Handler) CreateSubscriber(c echo.Context) error {
	return c.JSON(http.StatusCreated, map[string]interface{}{"id": "new-uuid"})
}

func (h *Handler) UpdateSubscriber(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) BlockSubscriber(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) UnblockSubscriber(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) MigrateTariff(c echo.Context) error {
	return c.JSON(http.StatusAccepted, map[string]interface{}{"batch_id": "uuid", "status": "queued"})
}

// Balance
func (h *Handler) AdjustBalance(c echo.Context) error {
	var req struct {
		BalanceType      string  `json:"balance_type" validate:"required"`
		Amount           float64 `json:"amount" validate:"required"`
		Operation        string  `json:"operation" validate:"required,oneof=credit debit"`
		Reason           string  `json:"reason"`
		NotifySubscriber bool    `json:"notify_subscriber"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"transaction_id": "uuid",
		"status":         "processing",
	})
}

func (h *Handler) FreezeBalance(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) UnfreezeBalance(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

// Tariffs
func (h *Handler) ListTariffs(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{"data": []map[string]interface{}{}})
}

func (h *Handler) GetTariff(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{"id": c.Param("id")})
}

func (h *Handler) CreateTariff(c echo.Context) error {
	return c.JSON(http.StatusCreated, map[string]interface{}{"id": "new-uuid"})
}

func (h *Handler) UpdateTariff(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) DeleteTariff(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) ActivateTariff(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

// CDR
func (h *Handler) ListCDR(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{"data": []map[string]interface{}{}, "pagination": map[string]interface{}{}})
}

func (h *Handler) ExportCDR(c echo.Context) error {
	return c.JSON(http.StatusAccepted, map[string]interface{}{"export_id": "uuid", "status": "processing"})
}

// Sessions
func (h *Handler) GetActiveSessions(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"data":         []map[string]interface{}{},
		"total_active": 0,
	})
}

// Bulk
func (h *Handler) BulkTariffChange(c echo.Context) error {
	return c.JSON(http.StatusAccepted, map[string]interface{}{"batch_id": "uuid", "status": "queued"})
}

func (h *Handler) BulkBonus(c echo.Context) error {
	return c.JSON(http.StatusAccepted, map[string]interface{}{"batch_id": "uuid", "status": "queued"})
}

// Reports
func (h *Handler) UsageReport(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{})
}

func (h *Handler) RevenueReport(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{})
}
