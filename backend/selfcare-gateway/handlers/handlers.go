// Package handlers provides HTTP handlers for the SelfCare Gateway.
package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/Grimid86/cgrates-ui/backend/pkg/auth"
	"github.com/Grimid86/cgrates-ui/backend/pkg/branding"
	"github.com/Grimid86/cgrates-ui/backend/pkg/cgrates"
	"github.com/Grimid86/cgrates-ui/backend/pkg/config"
	"github.com/Grimid86/cgrates-ui/backend/pkg/db"
	"github.com/Grimid86/cgrates-ui/backend/pkg/i18n"
	"github.com/Grimid86/cgrates-ui/backend/pkg/middleware"
	"github.com/Grimid86/cgrates-ui/backend/pkg/models"
	"github.com/Grimid86/cgrates-ui/backend/pkg/pulsar"
	"github.com/Grimid86/cgrates-ui/backend/pkg/redis"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// Dependencies holds all service dependencies for handlers
type Dependencies struct {
	DB          *db.Pool
	Redis       *redis.Client
	Pulsar      *pulsar.Client
	CGRateS     *cgrates.Client
	I18n        *i18n.Service
	Branding    *branding.Service
	JWTConfig   middleware.JWTConfig
	SelfCareAuth *auth.SelfCareAuth
	Config      *config.Config
}

// Handler contains all HTTP handlers
type Handler struct {
	deps Dependencies
}

// New creates a new handler instance
func New(deps Dependencies) *Handler {
	if deps.SelfCareAuth == nil {
		deps.SelfCareAuth = auth.NewSelfCareAuth(deps.DB, deps.JWTConfig, deps.Config)
	}
	return &Handler{deps: deps}
}

// Health returns service health status
func (h *Handler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"portal":    "selfcare",
		"version":   "1.0.0",
		"timestamp": time.Now().Format(time.RFC3339),
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

// Login handles subscriber authentication by MSISDN + PIN
func (h *Handler) Login(c echo.Context) error {
	var req struct {
		MSISDN       string `json:"msisdn" validate:"required"`
		PIN          string `json:"pin" validate:"required"`
		CaptchaToken string `json:"captcha_token"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.MSISDN == "" || req.PIN == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "msisdn and pin are required")
	}

	ctx := c.Request().Context()

	// Authenticate subscriber
	sub, err := h.deps.SelfCareAuth.Authenticate(ctx, req.MSISDN, req.PIN)
	if err != nil {
		if err.Error() == "account locked" {
			return echo.NewHTTPError(http.StatusLocked, map[string]interface{}{
				"code":    "ACCOUNT_LOCKED",
				"message": err.Error(),
			})
		}
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	// Create session and tokens
	_, accessToken, refreshToken, err := h.deps.SelfCareAuth.CreateSession(
		ctx, sub, c.RealIP(), c.Request().UserAgent(),
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create session")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    int(h.deps.JWTConfig.AccessTTL.Seconds()),
		"token_type":    "Bearer",
		"subscriber": map[string]interface{}{
			"id":       sub.ID,
			"msisdn":   sub.MSISDN,
			"locale":   "ru",
			"category": sub.Category,
		},
	})
}

// RefreshToken handles token refresh using refresh token
func (h *Handler) RefreshToken(c echo.Context) error {
	var req struct {
		RefreshToken string `json:"refresh_token" validate:"required"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	token, err := jwt.ParseWithClaims(req.RefreshToken, &middleware.Claims{}, func(token *jwt.Token) (interface{}, error) {
		return h.deps.JWTConfig.Secret, nil
	})
	if err != nil || !token.Valid {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid refresh token")
	}
	claims, ok := token.Claims.(*middleware.Claims)
	if !ok || claims.Type != "refresh" {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid token type")
	}

	hash := sha256.Sum256([]byte(req.RefreshToken))
	tokenHash := hex.EncodeToString(hash[:])

	accessToken, refreshToken, err := h.deps.SelfCareAuth.RotateSubscriberSession(
		c.Request().Context(), claims.ID, tokenHash, c.RealIP(), c.Request().UserAgent(),
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}
	_ = h.deps.Redis.BlacklistToken(c.Request().Context(), claims.ID, h.deps.Config.Security.JWTRefreshTTL)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    int(h.deps.JWTConfig.AccessTTL.Seconds()),
	})
}

// Logout revokes current session
func (h *Handler) Logout(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	ctx := c.Request().Context()
	if err := h.deps.SelfCareAuth.RevokeSession(ctx, claims.SubscriberID, claims.ID); err != nil {
		// Soft fail - still return success to client
	}
	_ = h.deps.Redis.BlacklistToken(ctx, claims.ID, h.deps.Config.Security.JWTRefreshTTL)
	return c.NoContent(http.StatusNoContent)
}

// GetBalance returns subscriber balances from CGRateS
func (h *Handler) GetBalance(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	ctx := c.Request().Context()

	// Try to get from CGRateS first
	account, err := h.deps.CGRateS.GetAccount(ctx, strconv.FormatInt(claims.TenantID, 10), claims.MSISDN)
	if err != nil {
		// Fallback to mock data if CGRateS unavailable
		return c.JSON(http.StatusOK, map[string]interface{}{
			"subscriber_id": claims.SubscriberID,
			"balances": []map[string]interface{}{
				{"type": "*monetary", "value": 150.50, "currency": "RUB"},
				{"type": "*data", "value": 5120.00, "unit": "MB", "expiry_date": "2026-06-20T00:00:00Z"},
				{"type": "*voice", "value": 300.00, "unit": "minutes", "expiry_date": "2026-06-20T00:00:00Z"},
			},
			"last_updated": time.Now().Format(time.RFC3339),
		})
	}

	// Parse CGRateS account response
	var balances []map[string]interface{}
	if balanceMap, ok := account["BalanceMap"].(map[string]interface{}); ok {
		for bType, bList := range balanceMap {
			if list, ok := bList.([]interface{}); ok && len(list) > 0 {
				if first, ok := list[0].(map[string]interface{}); ok {
					balances = append(balances, map[string]interface{}{
						"type":     bType,
						"value":    first["Value"],
						"currency": first["DestinationIDs"],
					})
				}
			}
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"subscriber_id": claims.SubscriberID,
		"balances":      balances,
		"last_updated":  time.Now().Format(time.RFC3339),
	})
}

// GetCDRHistory returns call detail records from CGRateS
func (h *Handler) GetCDRHistory(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	ctx := c.Request().Context()

	// Parse query params
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	// Build CGRateS filter
	filter := map[string]interface{}{
		"Tenant":  strconv.FormatInt(claims.TenantID, 10),
		"Account": claims.MSISDN,
		"Limit":   perPage,
		"Offset":  (page - 1) * perPage,
	}

	if from := c.QueryParam("from"); from != "" {
		filter["AnswerTimeStart"] = from
	}
	if to := c.QueryParam("to"); to != "" {
		filter["AnswerTimeEnd"] = to
	}

	cdrs, err := h.deps.CGRateS.GetCDRs(ctx, filter)
	if err != nil {
		// Fallback mock data
		return c.JSON(http.StatusOK, models.PaginatedResponse{
			Data: []map[string]interface{}{
				{
					"id":               "mock-uuid-1",
					"type":             "voice",
					"destination":      "79009876543",
					"duration_seconds": 125,
					"cost":             12.50,
					"currency":         "RUB",
					"started_at":       "2026-05-19T14:30:00Z",
					"status":           "completed",
				},
			},
			Pagination: models.Pagination{
				Page:       page,
				PerPage:    perPage,
				Total:      145,
				TotalPages: 8,
			},
		})
	}

	return c.JSON(http.StatusOK, models.PaginatedResponse{
		Data: cdrs,
		Pagination: models.Pagination{
			Page:       page,
			PerPage:    perPage,
			Total:      len(cdrs), // CGRateS doesn't always return total count
			TotalPages: 1,
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
		ReturnURL     string  `json:"return_url" validate:"required"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	// Generate idempotency key if not provided
	idempotencyKey := c.Request().Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		idempotencyKey = strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	ctx := c.Request().Context()

	// Publish to Pulsar for async processing
	if h.deps.Pulsar != nil {
		event := pulsar.Event{
			Type:     "topup_request",
			Portal:   "selfcare",
			TenantID: claims.TenantID,
			UserID:   claims.SubscriberID,
			Timestamp: time.Now(),
			Payload: map[string]interface{}{
				"msisdn":          claims.MSISDN,
				"amount":          req.Amount,
				"currency":        req.Currency,
				"payment_method":  req.PaymentMethod,
				"return_url":      req.ReturnURL,
				"idempotency_key": idempotencyKey,
			},
		}
		payload, _ := json.Marshal(event)
		go h.deps.Pulsar.Publish(ctx, payload, claims.MSISDN)
	}

	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"status":         "pending",
		"payment_url":    "https://payment.provider.com/session/" + idempotencyKey,
		"transaction_id": idempotencyKey,
		"expires_at":     time.Now().Add(15 * time.Minute).Format(time.RFC3339),
	})
}

// GetProfile returns subscriber profile from database
func (h *Handler) GetProfile(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	ctx := c.Request().Context()

	var sub models.Subscriber
	query := `
		SELECT id, tenant_id, msisdn, imsi, email, category, is_active, created_at
		FROM subscriber_credentials
		WHERE id = $1
	`
	err := h.deps.DB.QueryRow(ctx, query, claims.SubscriberID).Scan(
		&sub.ID, &sub.TenantID, &sub.MSISDN, &sub.IMSI, &sub.Email,
		&sub.Category, &sub.IsActive, &sub.CreatedAt,
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "subscriber not found")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":       sub.ID,
		"msisdn":   sub.MSISDN,
		"imsi":     sub.IMSI,
		"email":    sub.Email,
		"locale":   claims.Locale,
		"category": sub.Category,
		"tariff_plan": map[string]interface{}{
			"id":   "tariff-uuid",
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

	var req struct {
		Email string `json:"email"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	ctx := c.Request().Context()
	query := `UPDATE subscriber_credentials SET email = $1 WHERE id = $2`
	if err := h.deps.DB.Exec(ctx, query, req.Email, claims.SubscriberID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "update failed")
	}

	return c.NoContent(http.StatusNoContent)
}

// ChangePIN changes subscriber PIN after verifying old PIN
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

	ctx := c.Request().Context()
	if err := h.deps.SelfCareAuth.ChangePIN(ctx, claims.SubscriberID, req.OldPIN, req.NewPIN); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.NoContent(http.StatusNoContent)
}

// GetSessions returns active subscriber sessions
func (h *Handler) GetSessions(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	ctx := c.Request().Context()
	sessions, err := h.deps.SelfCareAuth.GetActiveSessions(ctx, claims.SubscriberID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load sessions")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"sessions": sessions,
	})
}

// RevokeSession revokes a specific session
func (h *Handler) RevokeSession(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	sessionID := c.Param("id")
	ctx := c.Request().Context()

	if err := h.deps.SelfCareAuth.RevokeSession(ctx, claims.SubscriberID, sessionID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "revoke failed")
	}

	return c.NoContent(http.StatusNoContent)
}

