package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Grimid86/cgrates-ui/backend/pkg/auth"
	"github.com/Grimid86/cgrates-ui/backend/pkg/branding"
	"github.com/Grimid86/cgrates-ui/backend/pkg/config"
	"github.com/Grimid86/cgrates-ui/backend/pkg/cgrates"
	"github.com/Grimid86/cgrates-ui/backend/pkg/db"
	"github.com/Grimid86/cgrates-ui/backend/pkg/i18n"
	"github.com/Grimid86/cgrates-ui/backend/pkg/middleware"
	"github.com/Grimid86/cgrates-ui/backend/pkg/models"
	"github.com/Grimid86/cgrates-ui/backend/pkg/pulsar"
	"github.com/Grimid86/cgrates-ui/backend/pkg/redis"
	"github.com/Grimid86/cgrates-ui/backend/pkg/security"
	"github.com/Grimid86/cgrates-ui/backend/pkg/storage"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

// Dependencies holds all service dependencies for handlers
type Dependencies struct {
	DB         *db.Pool
	Redis      *redis.Client
	Pulsar     *pulsar.Client
	CGRateS    *cgrates.Client
	I18n       *i18n.Service
	Branding   *branding.Service
	Storage    *storage.Client
	JWTConfig  middleware.JWTConfig
	StaffAuth  *auth.StaffAuth
	Config     *config.Config
}

// Handler contains all HTTP handlers
type Handler struct {
	deps Dependencies
}

func New(deps Dependencies) *Handler {
	if deps.StaffAuth == nil {
		deps.StaffAuth = auth.NewStaffAuth(deps.DB, deps.JWTConfig, deps.Config)
	}
	return &Handler{deps: deps}
}

func (h *Handler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"portal":    "operator",
		"version":   "1.0.0",
		"timestamp": time.Now().Format(time.RFC3339),
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

// Login handles operator authentication
func (h *Handler) Login(c echo.Context) error {
	var req struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required"`
		MFACode  string `json:"mfa_code"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	ctx := c.Request().Context()

	// Authenticate user
	user, err := h.deps.StaffAuth.Authenticate(ctx, req.Email, req.Password, "operator")
	if err != nil {
		if err.Error() == "account locked" {
			return echo.NewHTTPError(http.StatusLocked, map[string]interface{}{
				"code":    "ACCOUNT_LOCKED",
				"message": err.Error(),
			})
		}
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	// Check MFA for admin/reseller roles
	mfaVerified := false
	if user.MFAEnabled && (user.RoleCode == "admin" || user.RoleCode == "reseller") {
		if req.MFACode == "" {
			return echo.NewHTTPError(http.StatusForbidden, "mfa code required")
		}

		if user.MFASecret == nil || *user.MFASecret == "" {
			return echo.NewHTTPError(http.StatusInternalServerError, "mfa secret not configured")
		}
		secret, err := security.DecryptAES(*user.MFASecret, h.deps.Config.Security.CSRFSecret)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to decrypt mfa secret")
		}

		if totp.Validate(req.MFACode, secret) {
			mfaVerified = true
		}

		// Try backup codes
		if !mfaVerified && len(user.MFABackupCodes) > 0 {
			var remainingCodes []string
			for _, hash := range user.MFABackupCodes {
				if security.VerifyPassword(req.MFACode, hash) {
					mfaVerified = true
					continue
				}
				remainingCodes = append(remainingCodes, hash)
			}
			if mfaVerified {
				codesJSON, _ := json.Marshal(remainingCodes)
				_ = h.deps.DB.Exec(ctx, `UPDATE users SET mfa_backup_codes = $1 WHERE id = $2`, codesJSON, user.ID)
			}
		}

		if !mfaVerified {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid mfa code")
		}
	}

	// Load permissions
	perms, _ := h.deps.StaffAuth.GetUserPermissions(ctx, user.RoleID)

	// Create session
	_, accessToken, refreshToken, err := h.deps.StaffAuth.CreatePortalSession(
		ctx, user, "operator", c.RealIP(), mfaVerified,
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create session")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    int(h.deps.JWTConfig.AccessTTL.Seconds()),
		"user": map[string]interface{}{
			"id":          user.ID,
			"email":       user.Email,
			"role":        user.RoleCode,
			"tenant_id":   user.TenantID,
			"permissions": perms,
			"mfa_required": user.MFAEnabled && (user.RoleCode == "admin" || user.RoleCode == "reseller"),
		},
	})
}

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

	accessToken, refreshToken, err := h.deps.StaffAuth.RotatePortalSession(c.Request().Context(), claims.ID, tokenHash, c.RealIP())
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

func (h *Handler) Logout(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}
	_ = h.deps.StaffAuth.RevokeSession(c.Request().Context(), claims.ID)
	_ = h.deps.Redis.BlacklistToken(c.Request().Context(), claims.ID, h.deps.Config.Security.JWTRefreshTTL)
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) SetupMFA(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	ctx := c.Request().Context()

	var userEmail string
	err := h.deps.DB.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, claims.UserID).Scan(&userEmail)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load user")
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "UI-Bill",
		AccountName: userEmail,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate mfa secret")
	}
	secret := key.Secret()

	encryptedSecret, err := security.EncryptAES(secret, h.deps.Config.Security.CSRFSecret)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to encrypt mfa secret")
	}

	var backupCodesPlain []string
	var backupCodesHash []string
	for i := 0; i < 8; i++ {
		code, err := security.GenerateRandomToken(4)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate backup codes")
		}
		backupCodesPlain = append(backupCodesPlain, code)
		hash, err := security.HashPassword(code, bcrypt.DefaultCost)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to hash backup code")
		}
		backupCodesHash = append(backupCodesHash, hash)
	}
	backupCodesJSON, _ := json.Marshal(backupCodesHash)

	_ = h.deps.DB.Exec(ctx,
		`UPDATE users SET mfa_secret = $1, mfa_backup_codes = $2, mfa_enabled = false WHERE id = $3`,
		encryptedSecret, backupCodesJSON, claims.UserID)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"secret":       secret,
		"qr_code_url":  key.URL(),
		"backup_codes": backupCodesPlain,
	})
}

func (h *Handler) VerifyMFA(c echo.Context) error {
	var req struct {
		Code string `json:"code" validate:"required"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	ctx := c.Request().Context()
	var encryptedSecret string
	err := h.deps.DB.QueryRow(ctx, `SELECT mfa_secret FROM users WHERE id = $1`, claims.UserID).Scan(&encryptedSecret)
	if err != nil || encryptedSecret == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "mfa not set up")
	}

	secret, err := security.DecryptAES(encryptedSecret, h.deps.Config.Security.CSRFSecret)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to decrypt mfa secret")
	}

	valid := totp.Validate(req.Code, secret)
	if !valid {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid mfa code")
	}

	_ = h.deps.DB.Exec(ctx, `UPDATE users SET mfa_enabled = true WHERE id = $1`, claims.UserID)

	return c.JSON(http.StatusOK, map[string]interface{}{"verified": true})
}

func (h *Handler) DisableMFA(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}
	ctx := c.Request().Context()
	_ = h.deps.DB.Exec(ctx,
		`UPDATE users SET mfa_enabled = false, mfa_secret = NULL, mfa_backup_codes = '[]' WHERE id = $1`,
		claims.UserID)
	return c.NoContent(http.StatusNoContent)
}

// Subscribers
func (h *Handler) ListSubscribers(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	ctx := c.Request().Context()
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	search := c.QueryParam("search")
	status := c.QueryParam("status")

	// Build query with tenant isolation
	var args []interface{}
	args = append(args, claims.TenantID)

	query := `
		SELECT id, tenant_id, msisdn, imsi, email, category, is_active, created_at, last_login_at
		FROM subscriber_credentials
		WHERE tenant_id = $1
	`
	argCount := 1

	if search != "" {
		argCount++
		query += fmt.Sprintf(` AND (msisdn ILIKE $%d OR imsi ILIKE $%d OR email ILIKE $%d)`, argCount, argCount, argCount)
		args = append(args, "%"+search+"%")
	}
	if status != "" {
		argCount++
		query += fmt.Sprintf(` AND is_active = $%d`, argCount)
		args = append(args, status == "active")
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM (" + query + ") t"
	var total int
	_ = h.deps.DB.QueryRow(ctx, countQuery, args...).Scan(&total)

	// Add pagination
	argCount++
	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, argCount, argCount+1)
	args = append(args, perPage, (page-1)*perPage)

	rows, err := h.deps.DB.Query(ctx, query, args...)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "query failed")
	}
	defer rows.Close()

	var subscribers []map[string]interface{}
	for rows.Next() {
		var sub models.Subscriber
		var lastLogin *time.Time
		err := rows.Scan(&sub.ID, &sub.TenantID, &sub.MSISDN, &sub.IMSI, &sub.Email,
			&sub.Category, &sub.IsActive, &sub.CreatedAt, &lastLogin)
		if err != nil {
			continue
		}
		subscribers = append(subscribers, map[string]interface{}{
			"id":            sub.ID,
			"msisdn":        sub.MSISDN,
			"imsi":          sub.IMSI,
			"email":         sub.Email,
			"category":      sub.Category,
			"status":        map[bool]string{true: "active", false: "inactive"}[sub.IsActive],
			"created_at":    sub.CreatedAt,
			"last_activity": lastLogin,
		})
	}

	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}

	return c.JSON(http.StatusOK, models.PaginatedResponse{
		Data: subscribers,
		Pagination: models.Pagination{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

func (h *Handler) GetSubscriber(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	ctx := c.Request().Context()
	subID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid subscriber id")
	}

	var sub models.Subscriber
	query := `
		SELECT id, tenant_id, msisdn, imsi, email, category, is_active, created_at, last_login_at
		FROM subscriber_credentials
		WHERE id = $1 AND tenant_id = $2
	`
	var lastLogin *time.Time
	err = h.deps.DB.QueryRow(ctx, query, subID, claims.TenantID).Scan(
		&sub.ID, &sub.TenantID, &sub.MSISDN, &sub.IMSI, &sub.Email,
		&sub.Category, &sub.IsActive, &sub.CreatedAt, &lastLogin,
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "subscriber not found")
	}

	// Get balance from CGRateS
	balance, _ := h.deps.CGRateS.GetAccount(ctx, strconv.FormatInt(claims.TenantID, 10), sub.MSISDN)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":         sub.ID,
		"msisdn":     sub.MSISDN,
		"imsi":       sub.IMSI,
		"email":      sub.Email,
		"category":   sub.Category,
		"status":     map[bool]string{true: "active", false: "inactive"}[sub.IsActive],
		"created_at": sub.CreatedAt,
		"last_activity": lastLogin,
		"balance":    balance,
	})
}

func (h *Handler) CreateSubscriber(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	var req struct {
		MSISDN   string  `json:"msisdn" validate:"required"`
		IMSI     *string `json:"imsi"`
		Email    *string `json:"email"`
		Category string  `json:"category" default:"prepaid"`
		PIN      string  `json:"pin" validate:"required,min=4,max=6"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	if !security.ValidateMSISDN(req.MSISDN) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid msisdn")
	}

	ctx := c.Request().Context()

	// Hash PIN
	pinHash, err := security.HashPassword(req.PIN, 12)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to hash pin")
	}

	query := `
		INSERT INTO subscriber_credentials (tenant_id, msisdn, imsi, email, category, pin_hash)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	var id int64
	err = h.deps.DB.QueryRow(ctx, query, claims.TenantID, req.MSISDN, req.IMSI, req.Email, req.Category, pinHash).Scan(&id)
	if err != nil {
		return echo.NewHTTPError(http.StatusConflict, "subscriber already exists")
	}

	// Create in CGRateS
	_ = h.deps.CGRateS.Call(ctx, "ApierV1.SetAccount", map[string]interface{}{
		"Tenant":  strconv.FormatInt(claims.TenantID, 10),
		"Account": req.MSISDN,
	}, nil)

	return c.JSON(http.StatusCreated, map[string]interface{}{"id": id, "msisdn": req.MSISDN})
}

func (h *Handler) UpdateSubscriber(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	subID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid subscriber id")
	}

	var req struct {
		Email    *string `json:"email"`
		Category *string `json:"category"`
		IMSI     *string `json:"imsi"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	ctx := c.Request().Context()
	query := `UPDATE subscriber_credentials SET email = COALESCE($1, email), category = COALESCE($2, category), imsi = COALESCE($3, imsi) WHERE id = $4 AND tenant_id = $5`
	if err := h.deps.DB.Exec(ctx, query, req.Email, req.Category, req.IMSI, subID, claims.TenantID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "update failed")
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) BlockSubscriber(c echo.Context) error {
	return h.setSubscriberStatus(c, false)
}

func (h *Handler) UnblockSubscriber(c echo.Context) error {
	return h.setSubscriberStatus(c, true)
}

func (h *Handler) setSubscriberStatus(c echo.Context, active bool) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	subID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid subscriber id")
	}

	ctx := c.Request().Context()
	query := `UPDATE subscriber_credentials SET is_active = $1 WHERE id = $2 AND tenant_id = $3`
	if err := h.deps.DB.Exec(ctx, query, active, subID, claims.TenantID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "update failed")
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) MigrateTariff(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	var req struct {
		NewTariffID string `json:"new_tariff_id" validate:"required"`
		EffectiveDate string `json:"effective_date"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	// TODO: Implement actual tariff migration via CGRateS
	return c.JSON(http.StatusAccepted, map[string]interface{}{"batch_id": "migration-" + strconv.FormatInt(time.Now().Unix(), 10), "status": "queued"})
}

// Balance
func (h *Handler) AdjustBalance(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	subID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid subscriber id")
	}

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

	if !middleware.ValidateBalanceType(req.BalanceType) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid balance type")
	}

	ctx := c.Request().Context()

	// Get subscriber MSISDN
	var msisdn string
	if err := h.deps.DB.QueryRow(ctx, `SELECT msisdn FROM subscriber_credentials WHERE id = $1 AND tenant_id = $2`, subID, claims.TenantID).Scan(&msisdn); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "subscriber not found")
	}

	// Perform CGRateS RPC
	tenantStr := strconv.FormatInt(claims.TenantID, 10)
	var cgratesErr error
	if req.Operation == "credit" {
		cgratesErr = h.deps.CGRateS.AddBalance(ctx, tenantStr, msisdn, req.BalanceType, req.Amount, "*out")
	} else {
		cgratesErr = h.deps.CGRateS.Call(ctx, "ApierV1.DebitBalance", map[string]interface{}{
			"Tenant":      tenantStr,
			"Account":     msisdn,
			"BalanceType": req.BalanceType,
			"Value":       req.Amount,
			"Directions":  "*out",
		}, nil)
	}

	if cgratesErr != nil {
		// Async fallback via Pulsar
		if h.deps.Pulsar != nil {
			event := map[string]interface{}{
				"type":         "balance_adjust",
				"subscriber_id": subID,
				"msisdn":       msisdn,
				"amount":       req.Amount,
				"operation":    req.Operation,
				"reason":       req.Reason,
			}
			payload, _ := json.Marshal(event)
			go h.deps.Pulsar.PublishTo(context.Background(), "persistent://billing/commands/commands.balance.adjust", payload, msisdn)
		}
	}

	// Insert balance history
	historyQuery := `
		INSERT INTO balance_history (tenant_id, subscriber_id, balance_type, amount_after, operation, extra_data)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	extraData := map[string]interface{}{"reason": req.Reason, "operator_id": claims.UserID}
	extraJSON, _ := json.Marshal(extraData)
	_ = h.deps.DB.Exec(ctx, historyQuery, claims.TenantID, subID, req.BalanceType, req.Amount, req.Operation, extraJSON)

	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"transaction_id": strconv.FormatInt(time.Now().UnixNano(), 10),
		"status":         "processing",
	})
}

func (h *Handler) FreezeBalance(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	subID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid subscriber id")
	}

	ctx := c.Request().Context()

	// Update subscriber balance_frozen_at
	query := `UPDATE subscriber_credentials SET balance_frozen_at = NOW() WHERE id = $1 AND tenant_id = $2`
	if err := h.deps.DB.Exec(ctx, query, subID, claims.TenantID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "freeze failed")
	}

	// Log to balance_history
	_ = h.deps.DB.Exec(ctx, `
		INSERT INTO balance_history (tenant_id, subscriber_id, balance_type, amount_before, amount_after, operation, extra_data)
		VALUES ($1, $2, '*monetary', NULL, NULL, 'freeze', '{"reason":"manual"}')
	`, claims.TenantID, subID)

	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) UnfreezeBalance(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	subID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid subscriber id")
	}

	ctx := c.Request().Context()

	// Clear subscriber balance_frozen_at
	query := `UPDATE subscriber_credentials SET balance_frozen_at = NULL WHERE id = $1 AND tenant_id = $2`
	if err := h.deps.DB.Exec(ctx, query, subID, claims.TenantID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "unfreeze failed")
	}

	// Log to balance_history
	_ = h.deps.DB.Exec(ctx, `
		INSERT INTO balance_history (tenant_id, subscriber_id, balance_type, amount_before, amount_after, operation, extra_data)
		VALUES ($1, $2, '*monetary', NULL, NULL, 'unfreeze', '{"reason":"manual"}')
	`, claims.TenantID, subID)

	return c.NoContent(http.StatusNoContent)
}

// Tariffs
func (h *Handler) ListTariffs(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	ctx := c.Request().Context()
	rows, err := h.deps.DB.Query(ctx, `
		SELECT id, name, cgrates_tp_id, status, created_at
		FROM tariff_sandbox
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`, claims.TenantID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "query failed")
	}
	defer rows.Close()

	var tariffs []map[string]interface{}
	for rows.Next() {
		var id int64
		var name, status string
		var cgratesTPID *string
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &cgratesTPID, &status, &createdAt); err != nil {
			continue
		}
		tariffs = append(tariffs, map[string]interface{}{
			"id":            id,
			"name":          name,
			"cgrates_tp_id": cgratesTPID,
			"status":        status,
			"created_at":    createdAt,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"data": tariffs})
}

func (h *Handler) GetTariff(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	tariffID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid tariff id")
	}

	ctx := c.Request().Context()
	var t struct {
		ID          int64                  `json:"id"`
		Name        string                 `json:"name"`
		CGRatesTPID string                 `json:"cgrates_tp_id"`
		Config      map[string]interface{} `json:"config"`
		Status      string                 `json:"status"`
		CreatedAt   time.Time              `json:"created_at"`
		UpdatedAt   time.Time              `json:"updated_at"`
	}

	var configJSON []byte
	query := `
		SELECT id, name, cgrates_tp_id, config_json, status, created_at, updated_at
		FROM tariff_sandbox
		WHERE id = $1 AND tenant_id = $2
	`
	if err := h.deps.DB.QueryRow(ctx, query, tariffID, claims.TenantID).Scan(
		&t.ID, &t.Name, &t.CGRatesTPID, &configJSON, &t.Status, &t.CreatedAt, &t.UpdatedAt,
	); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "tariff not found")
	}
	_ = json.Unmarshal(configJSON, &t.Config)

	return c.JSON(http.StatusOK, t)
}

func (h *Handler) CreateTariff(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	var req struct {
		Name      string                 `json:"name" validate:"required"`
		Config    map[string]interface{} `json:"config"`
		TPID      string                 `json:"cgrates_tp_id"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	ctx := c.Request().Context()
	configJSON, _ := json.Marshal(req.Config)

	query := `
		INSERT INTO tariff_sandbox (tenant_id, name, cgrates_tp_id, config_json, status, created_by)
		VALUES ($1, $2, $3, $4, 'draft', $5)
		RETURNING id
	`
	var id int64
	if err := h.deps.DB.QueryRow(ctx, query, claims.TenantID, req.Name, req.TPID, configJSON, claims.UserID).Scan(&id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "create failed")
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{"id": id})
}

func (h *Handler) UpdateTariff(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	tariffID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid tariff id")
	}

	var req struct {
		Name   string                 `json:"name" validate:"required"`
		Config map[string]interface{} `json:"config"`
		TPID   string                 `json:"cgrates_tp_id"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	ctx := c.Request().Context()
	configJSON, _ := json.Marshal(req.Config)

	query := `
		UPDATE tariff_sandbox
		SET name = $1, config_json = $2, cgrates_tp_id = $3, updated_at = NOW()
		WHERE id = $4 AND tenant_id = $5 AND status != 'active'
	`
	if err := h.deps.DB.Exec(ctx, query, req.Name, configJSON, req.TPID, tariffID, claims.TenantID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "update failed")
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) DeleteTariff(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	tariffID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid tariff id")
	}

	ctx := c.Request().Context()

	// Soft delete: archive active/testing tariffs, hard delete drafts
	query := `
		UPDATE tariff_sandbox
		SET status = CASE WHEN status = 'draft' THEN 'archived' ELSE 'archived' END,
		    updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2
	`
	if err := h.deps.DB.Exec(ctx, query, tariffID, claims.TenantID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "delete failed")
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) ActivateTariff(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	tariffID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid tariff id")
	}

	ctx := c.Request().Context()
	if err := h.deps.DB.Exec(ctx, `UPDATE tariff_sandbox SET status = 'active' WHERE id = $1 AND tenant_id = $2`, tariffID, claims.TenantID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "activation failed")
	}

	return c.NoContent(http.StatusNoContent)
}

// CDR
func (h *Handler) ListCDR(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	ctx := c.Request().Context()
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	filter := map[string]interface{}{
		"Tenant": strconv.FormatInt(claims.TenantID, 10),
		"Limit":  perPage,
		"Offset": (page - 1) * perPage,
	}

	if msisdn := c.QueryParam("msisdn"); msisdn != "" {
		filter["Account"] = msisdn
	}
	if from := c.QueryParam("from"); from != "" {
		filter["AnswerTimeStart"] = from
	}
	if to := c.QueryParam("to"); to != "" {
		filter["AnswerTimeEnd"] = to
	}

	cdrs, err := h.deps.CGRateS.GetCDRs(ctx, filter)
	if err != nil {
		return c.JSON(http.StatusOK, models.PaginatedResponse{
			Data:       []map[string]interface{}{},
			Pagination: models.Pagination{Page: page, PerPage: perPage, Total: 0, TotalPages: 0},
		})
	}

	return c.JSON(http.StatusOK, models.PaginatedResponse{
		Data: cdrs,
		Pagination: models.Pagination{
			Page:       page,
			PerPage:    perPage,
			Total:      len(cdrs),
			TotalPages: 1,
		},
	})
}

func (h *Handler) ExportCDR(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	ctx := c.Request().Context()
	exportID := uuid.New().String()
	format := c.QueryParam("format")
	if format != "json" {
		format = "csv"
	}

	// Build filter same as ListCDR
	filter := map[string]interface{}{
		"Tenant": strconv.FormatInt(claims.TenantID, 10),
	}
	if msisdn := c.QueryParam("msisdn"); msisdn != "" {
		filter["Account"] = msisdn
	}
	if from := c.QueryParam("from"); from != "" {
		filter["AnswerTimeStart"] = from
	}
	if to := c.QueryParam("to"); to != "" {
		filter["AnswerTimeEnd"] = to
	}

	// Fetch CDRs
	cdrs, err := h.deps.CGRateS.GetCDRs(ctx, filter)
	if err != nil {
		cdrs = []map[string]interface{}{}
	}

	// Generate file
	var buf bytes.Buffer
	var contentType string
	if format == "csv" {
		contentType = "text/csv"
		w := csv.NewWriter(&buf)
		w.Write([]string{"CGRID", "RunID", "Source", "OriginHost", "OriginID", "ToR", "RequestType", "Tenant", "Category", "Account", "Subject", "Destination", "SetupTime", "AnswerTime", "Usage", "Cost", "ExtraFields"})
		for _, cdr := range cdrs {
			extra, _ := json.Marshal(cdr["ExtraFields"])
			w.Write([]string{
				fmt.Sprint(cdr["CGRID"]),
				fmt.Sprint(cdr["RunID"]),
				fmt.Sprint(cdr["Source"]),
				fmt.Sprint(cdr["OriginHost"]),
				fmt.Sprint(cdr["OriginID"]),
				fmt.Sprint(cdr["ToR"]),
				fmt.Sprint(cdr["RequestType"]),
				fmt.Sprint(cdr["Tenant"]),
				fmt.Sprint(cdr["Category"]),
				fmt.Sprint(cdr["Account"]),
				fmt.Sprint(cdr["Subject"]),
				fmt.Sprint(cdr["Destination"]),
				fmt.Sprint(cdr["SetupTime"]),
				fmt.Sprint(cdr["AnswerTime"]),
				fmt.Sprint(cdr["Usage"]),
				fmt.Sprint(cdr["Cost"]),
				string(extra),
			})
		}
		w.Flush()
	} else {
		contentType = "application/json"
		enc := json.NewEncoder(&buf)
		enc.SetIndent("", "  ")
		_ = enc.Encode(cdrs)
	}

	// Upload to MinIO
	objectKey := fmt.Sprintf("exports/%d/cdr-%s.%s", claims.TenantID, exportID, format)
	downloadURL := ""
	if h.deps.Storage != nil {
		if err := h.deps.Storage.Upload(ctx, objectKey, &buf, int64(buf.Len()), contentType); err == nil {
			downloadURL = h.deps.Storage.PublicURL(objectKey)
		}
	}

	// Record in DB
	filterJSON, _ := json.Marshal(filter)
	_ = h.deps.DB.Exec(ctx, `
		INSERT INTO cdr_exports (id, tenant_id, created_by, filter, format, status, object_key, download_url, record_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, exportID, claims.TenantID, claims.UserID, filterJSON, format, "completed", objectKey, downloadURL, len(cdrs))

	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"export_id":    exportID,
		"status":       "completed",
		"format":       format,
		"record_count": len(cdrs),
		"download_url": downloadURL,
	})
}

// Sessions
func (h *Handler) GetActiveSessions(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	ctx := c.Request().Context()
	sessions, err := h.deps.CGRateS.GetActiveSessions(ctx, map[string]interface{}{
		"Tenant": strconv.FormatInt(claims.TenantID, 10),
	})
	if err != nil {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"data":         []map[string]interface{}{},
			"total_active": 0,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data":         sessions,
		"total_active": len(sessions),
	})
}

// Bulk
func (h *Handler) BulkTariffChange(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	var req struct {
		SubscriberIDs []string `json:"subscriber_ids" validate:"required,min=1"`
		NewTariffID   string   `json:"new_tariff_id" validate:"required"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	// Publish to Pulsar for async processing
	if h.deps.Pulsar != nil {
		event := map[string]interface{}{
			"type":           "bulk_tariff_change",
			"subscriber_ids": req.SubscriberIDs,
			"new_tariff_id":  req.NewTariffID,
			"operator_id":    claims.UserID,
			"tenant_id":      claims.TenantID,
		}
		payload, _ := json.Marshal(event)
		go h.deps.Pulsar.PublishTo(context.Background(), "persistent://billing/commands/commands.tariff.update", payload, "")
	}

	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"batch_id": "bulk-" + strconv.FormatInt(time.Now().Unix(), 10),
		"status":   "queued",
		"total_count": len(req.SubscriberIDs),
	})
}

func (h *Handler) BulkBonus(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	var req struct {
		SubscriberIDs []string `json:"subscriber_ids" validate:"required,min=1"`
		Amount        float64  `json:"amount" validate:"required,gt=0"`
		Reason        string   `json:"reason"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	if h.deps.Pulsar != nil {
		event := map[string]interface{}{
			"type":           "bulk_bonus",
			"subscriber_ids": req.SubscriberIDs,
			"amount":         req.Amount,
			"reason":         req.Reason,
			"operator_id":    claims.UserID,
		}
		payload, _ := json.Marshal(event)
		go h.deps.Pulsar.PublishTo(context.Background(), "persistent://billing/commands/commands.balance.adjust", payload, "")
	}

	return c.JSON(http.StatusAccepted, map[string]interface{}{
		"batch_id":    "bulk-bonus-" + strconv.FormatInt(time.Now().Unix(), 10),
		"status":      "queued",
		"total_count": len(req.SubscriberIDs),
	})
}

// Reports
func (h *Handler) UsageReport(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	ctx := c.Request().Context()
	from := c.QueryParam("from")
	to := c.QueryParam("to")

	// Aggregate from balance_history
	query := `
		SELECT balance_type, COUNT(*) as count, SUM(amount_after - COALESCE(amount_before, 0)) as total
		FROM balance_history
		WHERE tenant_id = $1 AND created_at BETWEEN $2 AND $3
		GROUP BY balance_type
	`
	rows, err := h.deps.DB.Query(ctx, query, claims.TenantID, from, to)
	if err != nil {
		return c.JSON(http.StatusOK, map[string]interface{}{})
	}
	defer rows.Close()

	var report []map[string]interface{}
	for rows.Next() {
		var bType string
		var count int
		var total *float64
		if err := rows.Scan(&bType, &count, &total); err != nil {
			continue
		}
		report = append(report, map[string]interface{}{
			"balance_type": bType,
			"count":        count,
			"total":        total,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"data": report, "period": map[string]string{"from": from, "to": to}})
}

func (h *Handler) RevenueReport(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	from := c.QueryParam("from")
	to := c.QueryParam("to")
	if from == "" {
		from = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	}
	if to == "" {
		to = time.Now().Format("2006-01-02")
	}

	ctx := c.Request().Context()
	query := `
		SELECT
			DATE(created_at) as date,
			COALESCE(SUM(CASE WHEN operation = 'charge' THEN COALESCE(amount_after - amount_before, 0) ELSE 0 END), 0) as total_charges,
			COALESCE(SUM(CASE WHEN operation = 'topup' THEN COALESCE(amount_after - amount_before, 0) ELSE 0 END), 0) as total_topups,
			COUNT(DISTINCT subscriber_id) as subscriber_count
		FROM balance_history
		WHERE tenant_id = $1 AND created_at >= $2 AND created_at <= $3::date + interval '1 day'
		GROUP BY DATE(created_at)
		ORDER BY DATE(created_at)
	`
	rows, err := h.deps.DB.Query(ctx, query, claims.TenantID, from, to)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "query failed")
	}
	defer rows.Close()

	var data []map[string]interface{}
	for rows.Next() {
		var date time.Time
		var totalCharges, totalTopups float64
		var subscriberCount int
		if err := rows.Scan(&date, &totalCharges, &totalTopups, &subscriberCount); err != nil {
			continue
		}
		data = append(data, map[string]interface{}{
			"date":             date.Format("2006-01-02"),
			"total_charges":    totalCharges,
			"total_topups":     totalTopups,
			"net_revenue":      totalTopups - totalCharges,
			"subscriber_count": subscriberCount,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data":   data,
		"period": map[string]string{"from": from, "to": to},
	})
}

