package handlers

import (
	"net/http"

	"github.com/Grimid86/cgrates-ui/backend/pkg/branding"
	"github.com/Grimid86/cgrates-ui/backend/pkg/db"
	"github.com/Grimid86/cgrates-ui/backend/pkg/i18n"
	"github.com/Grimid86/cgrates-ui/backend/pkg/middleware"
	"github.com/Grimid86/cgrates-ui/backend/pkg/pulsar"
	"github.com/Grimid86/cgrates-ui/backend/pkg/redis"
	"github.com/labstack/echo/v4"
)

// Dependencies holds all service dependencies
type Dependencies struct {
	DB        *db.Pool
	Redis     *redis.Client
	Pulsar    *pulsar.Client
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
		"portal":  "admin",
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

// Auth
func (h *Handler) Login(c echo.Context) error {
	var req struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required"`
		MFACode  string `json:"mfa_code"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	// TODO: Authenticate, verify mandatory MFA, issue short-lived JWT
	return c.JSON(http.StatusOK, map[string]interface{}{
		"access_token":  "eyJ...",
		"refresh_token": "eyJ...",
		"expires_in":    900,
		"mfa_required":  true,
		"user": map[string]interface{}{
			"id":    1,
			"email": req.Email,
			"role":  "root",
		},
	})
}

func (h *Handler) RefreshToken(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{"access_token": "eyJ...", "expires_in": 900})
}

func (h *Handler) Logout(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) SetupMFA(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"secret":       "JBSWY3DPEHPK3PXP",
		"qr_code_url":  "otpauth://totp/UI-Bill:root@system.local?secret=JBSWY3DPEHPK3PXP&issuer=UI-Bill",
		"backup_codes": []string{"12345678", "87654321"},
	})
}

func (h *Handler) VerifyMFA(c echo.Context) error {
	var req struct {
		Code string `json:"code" validate:"required"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	// TODO: Verify TOTP code
	return c.JSON(http.StatusOK, map[string]interface{}{"verified": true})
}

func (h *Handler) DisableMFA(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

// Tenants
func (h *Handler) ListTenants(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": []map[string]interface{}{
			{"id": 1, "name": "System", "code": "system", "is_active": true},
		},
	})
}

func (h *Handler) GetTenant(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":     c.Param("id"),
		"name":   "Example Tenant",
		"limits": map[string]interface{}{"max_subscribers": 100000},
	})
}

func (h *Handler) CreateTenant(c echo.Context) error {
	var req struct {
		Name   string                 `json:"name" validate:"required"`
		Code   string                 `json:"code" validate:"required"`
		Limits map[string]interface{} `json:"limits"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}
	// TODO: Create tenant, publish config.changes event
	return c.JSON(http.StatusCreated, map[string]interface{}{"id": 99, "name": req.Name})
}

func (h *Handler) UpdateTenant(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) SuspendTenant(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) DeleteTenant(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

// Users
func (h *Handler) ListUsers(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{"data": []map[string]interface{}{}})
}

func (h *Handler) GetUser(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{"id": c.Param("id")})
}

func (h *Handler) CreateUser(c echo.Context) error {
	return c.JSON(http.StatusCreated, map[string]interface{}{"id": "new-uuid"})
}

func (h *Handler) UpdateUser(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) ResetUserMFA(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) ResetUserPassword(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) DeleteUser(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

// RBAC
func (h *Handler) ListRoles(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"roles": []map[string]interface{}{
			{"code": "root", "name": "System Root", "permissions": []string{"*"}},
			{"code": "admin", "name": "Tenant Admin", "permissions": []string{"subscriber:read", "subscriber:write"}},
		},
	})
}

func (h *Handler) GetRole(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{"id": c.Param("id")})
}

func (h *Handler) CreateRole(c echo.Context) error {
	return c.JSON(http.StatusCreated, map[string]interface{}{"id": "new-role"})
}

func (h *Handler) UpdateRolePermissions(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) ListPermissions(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"permissions": []map[string]interface{}{
			{"code": "subscriber:read", "resource": "subscriber", "action": "read"},
			{"code": "subscriber:write", "resource": "subscriber", "action": "write"},
		},
	})
}

func (h *Handler) GetRBACMatrix(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"matrix": map[string]interface{}{
			"root":  map[string]bool{"subscriber:read": true, "subscriber:write": true},
			"admin": map[string]bool{"subscriber:read": true, "subscriber:write": true},
		},
	})
}

// Branding
func (h *Handler) UpdateBranding(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) UploadLogo(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{"logo_url": "https://cdn.example.com/logo.png"})
}

func (h *Handler) ListEmailTemplates(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{"templates": []map[string]interface{}{}})
}

func (h *Handler) UpdateEmailTemplate(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) PreviewEmailTemplate(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"subject": "Welcome!",
		"body":    "<html><body><h1>Welcome to Orange Billing</h1></body></html>",
	})
}

// Localization
func (h *Handler) ImportTranslations(c echo.Context) error {
	return c.JSON(http.StatusAccepted, map[string]interface{}{"import_id": "uuid", "status": "processing"})
}

// Monitoring
func (h *Handler) Metrics(c echo.Context) error {
	return c.String(http.StatusOK, "# HELP http_requests_total Total HTTP requests\n")
}

func (h *Handler) GetAuditLog(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"data":       []map[string]interface{}{},
		"pagination": map[string]interface{}{"page": 1, "per_page": 20, "total": 0},
	})
}

func (h *Handler) GetPulsarTopics(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"topics": []string{
			"persistent://billing/events/charges",
			"persistent://billing/audit/events",
			"persistent://billing/notifications/email",
		},
	})
}

func (h *Handler) GetPulsarLag(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"lag": map[string]int64{
			"billing.events.charges": 0,
			"billing.audit.events":   0,
		},
	})
}

func (h *Handler) GetDatabaseStatus(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"primary":       map[string]interface{}{"status": "up", "lag_seconds": 0},
		"replicas":      []map[string]interface{}{},
		"partitions":    []string{"audit_log_y2026m01", "audit_log_y2026m02"},
		"vacuum_status": "healthy",
	})
}

// API Keys
func (h *Handler) ListAPIKeys(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{"keys": []map[string]interface{}{}})
}

func (h *Handler) CreateAPIKey(c echo.Context) error {
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"id":  "new-key",
		"key": "sc_live_xxxxxxxxxxxxxxxx",
	})
}

func (h *Handler) DeleteAPIKey(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}
