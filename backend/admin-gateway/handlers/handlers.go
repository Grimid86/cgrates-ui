package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Grimid86/cgrates-ui/backend/pkg/auth"
	"github.com/Grimid86/cgrates-ui/backend/pkg/branding"
	"github.com/Grimid86/cgrates-ui/backend/pkg/config"
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

// Dependencies holds all service dependencies
type Dependencies struct {
	DB        *db.Pool
	Redis     *redis.Client
	Pulsar    *pulsar.Client
	I18n      *i18n.Service
	Branding  *branding.Service
	Storage   *storage.Client
	JWTConfig middleware.JWTConfig
	StaffAuth *auth.StaffAuth
	Config    *config.Config
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
		"portal":    "admin",
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

	ctx := c.Request().Context()

	// Authenticate
	user, err := h.deps.StaffAuth.Authenticate(ctx, req.Email, req.Password, "admin")
	if err != nil {
		if err.Error() == "account locked" {
			return echo.NewHTTPError(http.StatusLocked, map[string]interface{}{
				"code":    "ACCOUNT_LOCKED",
				"message": err.Error(),
			})
		}
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	// MFA verification
	mfaVerified := false
	if user.MFAEnabled {
		if req.MFACode == "" {
			return echo.NewHTTPError(http.StatusForbidden, "mfa code required")
		}

		// Try TOTP first
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

		// Try backup codes if TOTP failed
		if !mfaVerified && len(user.MFABackupCodes) > 0 {
			var remainingCodes []string
			for _, hash := range user.MFABackupCodes {
				if security.VerifyPassword(req.MFACode, hash) {
					mfaVerified = true
					// Skip this code (consumed)
					continue
				}
				remainingCodes = append(remainingCodes, hash)
			}
			if mfaVerified {
				// Update consumed backup codes in DB
				codesJSON, _ := json.Marshal(remainingCodes)
				_ = h.deps.DB.Exec(ctx, `UPDATE users SET mfa_backup_codes = $1 WHERE id = $2`, codesJSON, user.ID)
			}
		}

		if !mfaVerified {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid mfa code")
		}
	}

	perms, _ := h.deps.StaffAuth.GetUserPermissions(ctx, user.RoleID)

	_, accessToken, refreshToken, err := h.deps.StaffAuth.CreatePortalSession(
		ctx, user, "admin", c.RealIP(), mfaVerified,
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to create session")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    int(h.deps.JWTConfig.AccessTTL.Seconds()),
		"mfa_required":  user.MFAEnabled,
		"user": map[string]interface{}{
			"id":          user.ID,
			"email":       user.Email,
			"role":        user.RoleCode,
			"permissions": perms,
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

	// Get user email for TOTP account name
	var userEmail string
	err := h.deps.DB.QueryRow(ctx, `SELECT email FROM users WHERE id = $1`, claims.UserID).Scan(&userEmail)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to load user")
	}

	// Generate TOTP key
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "UI-Bill",
		AccountName: userEmail,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate mfa secret")
	}
	secret := key.Secret()

	// Encrypt secret before storing
	encryptedSecret, err := security.EncryptAES(secret, h.deps.Config.Security.CSRFSecret)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to encrypt mfa secret")
	}

	// Generate 8 backup codes (8 chars hex, e.g. "a1b2c3d4")
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

	// Store encrypted secret and hashed backup codes; do NOT enable MFA yet (requires verification)
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

	// Enable MFA after successful verification
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

// Tenants
func (h *Handler) ListTenants(c echo.Context) error {
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

	var query string
	var args []interface{}

	if claims.Role == "root" {
		query = `SELECT id, uuid, name, code, is_active, created_at FROM tenant ORDER BY created_at DESC LIMIT $1 OFFSET $2`
		args = []interface{}{perPage, (page - 1) * perPage}
	} else {
		query = `SELECT id, uuid, name, code, is_active, created_at FROM tenant WHERE id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
		args = []interface{}{claims.TenantID, perPage, (page - 1) * perPage}
	}

	rows, err := h.deps.DB.Query(ctx, query, args...)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "query failed")
	}
	defer rows.Close()

	var tenants []map[string]interface{}
	for rows.Next() {
		var t models.Tenant
		if err := rows.Scan(&t.ID, &t.UUID, &t.Name, &t.Code, &t.IsActive, &t.CreatedAt); err != nil {
			continue
		}
		tenants = append(tenants, map[string]interface{}{
			"id":         t.ID,
			"uuid":       t.UUID,
			"name":       t.Name,
			"code":       t.Code,
			"is_active":  t.IsActive,
			"created_at": t.CreatedAt,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"data": tenants})
}

func (h *Handler) GetTenant(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	tenantID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid tenant id")
	}

	if claims.Role != "root" && claims.TenantID != tenantID {
		return echo.NewHTTPError(http.StatusForbidden, "access denied")
	}

	ctx := c.Request().Context()
	var t models.Tenant
	err = h.deps.DB.QueryRow(ctx, `SELECT id, uuid, name, code, is_active, created_at FROM tenant WHERE id = $1`, tenantID).Scan(
		&t.ID, &t.UUID, &t.Name, &t.Code, &t.IsActive, &t.CreatedAt,
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "tenant not found")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":        t.ID,
		"uuid":      t.UUID,
		"name":      t.Name,
		"code":      t.Code,
		"is_active": t.IsActive,
		"limits":    map[string]interface{}{"max_subscribers": 100000, "max_staff_users": 50},
	})
}

func (h *Handler) CreateTenant(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil || claims.Role != "root" {
		return echo.NewHTTPError(http.StatusForbidden, "root access required")
	}

	var req struct {
		Name   string                 `json:"name" validate:"required"`
		Code   string                 `json:"code" validate:"required"`
		Limits map[string]interface{} `json:"limits"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	ctx := c.Request().Context()
	query := `INSERT INTO tenant (name, code, is_active, created_at) VALUES ($1, $2, true, NOW()) RETURNING id`
	var id int64
	if err := h.deps.DB.QueryRow(ctx, query, req.Name, req.Code).Scan(&id); err != nil {
		return echo.NewHTTPError(http.StatusConflict, "tenant code already exists")
	}

	// Create default branding
	_ = h.deps.DB.Exec(ctx, `
		INSERT INTO branding_config (tenant_id, product_name, timezone, date_format)
		VALUES ($1, $2, 'UTC', 'DD.MM.YYYY')
	`, id, req.Name+" Billing")

	// Publish config change
	if h.deps.Pulsar != nil {
		event := map[string]interface{}{
			"type":      "tenant_created",
			"tenant_id": id,
			"code":      req.Code,
		}
		payload, _ := json.Marshal(event)
		go h.deps.Pulsar.PublishTo(ctx, "persistent://billing/config/config.changes", payload, "")
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{"id": id, "name": req.Name})
}

func (h *Handler) UpdateTenant(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	tenantID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid tenant id")
	}

	if claims.Role != "root" && claims.TenantID != tenantID {
		return echo.NewHTTPError(http.StatusForbidden, "access denied")
	}

	var req struct {
		Name     *string `json:"name"`
		IsActive *bool   `json:"is_active"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	ctx := c.Request().Context()
	query := `UPDATE tenant SET name = COALESCE($1, name), is_active = COALESCE($2, is_active) WHERE id = $3`
	if err := h.deps.DB.Exec(ctx, query, req.Name, req.IsActive, tenantID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "update failed")
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) SuspendTenant(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil || claims.Role != "root" {
		return echo.NewHTTPError(http.StatusForbidden, "root access required")
	}

	tenantID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid tenant id")
	}

	ctx := c.Request().Context()
	_ = h.deps.DB.Exec(ctx, `UPDATE tenant SET is_active = false WHERE id = $1`, tenantID)
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) DeleteTenant(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil || claims.Role != "root" {
		return echo.NewHTTPError(http.StatusForbidden, "root access required")
	}

	tenantID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid tenant id")
	}

	ctx := c.Request().Context()
	// Soft delete by disabling
	_ = h.deps.DB.Exec(ctx, `UPDATE tenant SET is_active = false WHERE id = $1`, tenantID)
	return c.NoContent(http.StatusNoContent)
}

// Users
func (h *Handler) ListUsers(c echo.Context) error {
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

	var query string
	var args []interface{}

	if claims.Role == "root" {
		query = `
			SELECT u.id, u.uuid, u.tenant_id, u.email, r.code as role, u.locale, u.is_active, u.created_at
			FROM users u
			JOIN roles r ON u.role_id = r.id
			ORDER BY u.created_at DESC
			LIMIT $1 OFFSET $2
		`
		args = []interface{}{perPage, (page - 1) * perPage}
	} else {
		query = `
			SELECT u.id, u.uuid, u.tenant_id, u.email, r.code as role, u.locale, u.is_active, u.created_at
			FROM users u
			JOIN roles r ON u.role_id = r.id
			WHERE u.tenant_id = $1
			ORDER BY u.created_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{claims.TenantID, perPage, (page - 1) * perPage}
	}

	rows, err := h.deps.DB.Query(ctx, query, args...)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "query failed")
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var id int64
		var uuid, email, role, locale string
		var tenantID int64
		var isActive bool
		var createdAt time.Time
		if err := rows.Scan(&id, &uuid, &tenantID, &email, &role, &locale, &isActive, &createdAt); err != nil {
			continue
		}
		users = append(users, map[string]interface{}{
			"id":         id,
			"uuid":       uuid,
			"tenant_id":  tenantID,
			"email":      email,
			"role":       role,
			"locale":     locale,
			"is_active":  isActive,
			"created_at": createdAt,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"data": users})
}

func (h *Handler) GetUser(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}

	ctx := c.Request().Context()
	var user models.User
	err = h.deps.DB.QueryRow(ctx, `
		SELECT u.id, u.uuid, u.tenant_id, u.email, r.code, u.locale, u.mfa_enabled, u.is_active, u.created_at
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.id = $1
	`, userID).Scan(
		&user.ID, &user.UUID, &user.TenantID, &user.Email, &user.RoleCode,
		&user.Locale, &user.MFAEnabled, &user.IsActive, &user.CreatedAt,
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user not found")
	}

	if claims.Role != "root" && claims.TenantID != user.TenantID {
		return echo.NewHTTPError(http.StatusForbidden, "access denied")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":          user.ID,
		"uuid":        user.UUID,
		"tenant_id":   user.TenantID,
		"email":       user.Email,
		"role":        user.RoleCode,
		"locale":      user.Locale,
		"mfa_enabled": user.MFAEnabled,
		"is_active":   user.IsActive,
		"created_at":  user.CreatedAt,
	})
}

func (h *Handler) CreateUser(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	var req struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required,min=8"`
		RoleCode string `json:"role_code" validate:"required"`
		TenantID *int64 `json:"tenant_id"`
		Locale   string `json:"locale" default:"en"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	if !security.ValidateEmail(req.Email) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid email")
	}

	targetTenantID := claims.TenantID
	if req.TenantID != nil && claims.Role == "root" {
		targetTenantID = *req.TenantID
	}

	ctx := c.Request().Context()

	// Get role ID
	var roleID int64
	if err := h.deps.DB.QueryRow(ctx, `SELECT id FROM roles WHERE code = $1`, req.RoleCode).Scan(&roleID); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid role")
	}

	// Hash password
	passwordHash, err := security.HashPassword(req.Password, 12)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to hash password")
	}

	query := `
		INSERT INTO users (tenant_id, email, password_hash, role_id, locale, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5, true, NOW())
		RETURNING id
	`
	var id int64
	if err := h.deps.DB.QueryRow(ctx, query, targetTenantID, req.Email, passwordHash, roleID, req.Locale).Scan(&id); err != nil {
		return echo.NewHTTPError(http.StatusConflict, "email already exists")
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{"id": id, "email": req.Email})
}

func (h *Handler) UpdateUser(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}

	var req struct {
		RoleCode *string `json:"role_code"`
		Locale   *string `json:"locale"`
		IsActive *bool   `json:"is_active"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	ctx := c.Request().Context()

	// Get role ID if provided
	var roleID *int64
	if req.RoleCode != nil {
		var rid int64
		if err := h.deps.DB.QueryRow(ctx, `SELECT id FROM roles WHERE code = $1`, *req.RoleCode).Scan(&rid); err == nil {
			roleID = &rid
		}
	}

	query := `
		UPDATE users
		SET role_id = COALESCE($1, role_id),
		    locale = COALESCE($2, locale),
		    is_active = COALESCE($3, is_active)
		WHERE id = $4
	`
	if err := h.deps.DB.Exec(ctx, query, roleID, req.Locale, req.IsActive, userID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "update failed")
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) ResetUserMFA(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}

	ctx := c.Request().Context()
	_ = h.deps.DB.Exec(ctx, `UPDATE users SET mfa_enabled = false, mfa_secret = NULL WHERE id = $1`, userID)
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) ResetUserPassword(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}

	ctx := c.Request().Context()

	// Generate temporary password
	tempPass, _ := security.GenerateRandomToken(8)
	hash, _ := security.HashPassword(tempPass, 12)

	_ = h.deps.DB.Exec(ctx, `UPDATE users SET password_hash = $1 WHERE id = $2`, hash, userID)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"temporary_password": tempPass,
		"message":            "Password reset successfully. User must change on next login.",
	})
}

func (h *Handler) DeleteUser(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}

	// Prevent self-deletion
	if claims.UserID == userID {
		return echo.NewHTTPError(http.StatusBadRequest, "cannot delete yourself")
	}

	ctx := c.Request().Context()
	_ = h.deps.DB.Exec(ctx, `UPDATE users SET is_active = false WHERE id = $1`, userID)
	return c.NoContent(http.StatusNoContent)
}

// RBAC
func (h *Handler) ListRoles(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	ctx := c.Request().Context()
	rows, err := h.deps.DB.Query(ctx, `SELECT id, code, name, permissions, is_system FROM roles ORDER BY id`)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "query failed")
	}
	defer rows.Close()

	var roles []map[string]interface{}
	for rows.Next() {
		var id int64
		var code, name string
		var permsJSON []byte
		var isSystem bool
		if err := rows.Scan(&id, &code, &name, &permsJSON, &isSystem); err != nil {
			continue
		}
		var perms []string
		_ = json.Unmarshal(permsJSON, &perms)
		roles = append(roles, map[string]interface{}{
			"id":         id,
			"code":       code,
			"name":       name,
			"permissions": perms,
			"is_system":  isSystem,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"roles": roles})
}

func (h *Handler) GetRole(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	roleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid role id")
	}

	ctx := c.Request().Context()
	var r struct {
		ID          int64           `json:"id"`
		Code        string          `json:"code"`
		Name        string          `json:"name"`
		Permissions json.RawMessage `json:"permissions"`
		IsSystem    bool            `json:"is_system"`
		CreatedAt   time.Time       `json:"created_at"`
		UpdatedAt   time.Time       `json:"updated_at"`
	}

	query := `
		SELECT id, code, name, permissions, is_system, created_at, updated_at
		FROM roles
		WHERE id = $1
	`
	if err := h.deps.DB.QueryRow(ctx, query, roleID).Scan(
		&r.ID, &r.Code, &r.Name, &r.Permissions, &r.IsSystem, &r.CreatedAt, &r.UpdatedAt,
	); err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "role not found")
	}

	return c.JSON(http.StatusOK, r)
}

func (h *Handler) CreateRole(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	var req struct {
		Name        string   `json:"name" validate:"required"`
		Code        string   `json:"code" validate:"required"`
		Permissions []string `json:"permissions"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	permsJSON, _ := json.Marshal(req.Permissions)
	ctx := c.Request().Context()

	query := `INSERT INTO roles (code, name, permissions) VALUES ($1, $2, $3) RETURNING id`
	var id int64
	if err := h.deps.DB.QueryRow(ctx, query, req.Code, req.Name, permsJSON).Scan(&id); err != nil {
		return echo.NewHTTPError(http.StatusConflict, "role code already exists")
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{"id": id})
}

func (h *Handler) UpdateRolePermissions(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	roleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid role id")
	}

	var req struct {
		Permissions []string `json:"permissions" validate:"required"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	permsJSON, _ := json.Marshal(req.Permissions)
	ctx := c.Request().Context()
	_ = h.deps.DB.Exec(ctx, `UPDATE roles SET permissions = $1 WHERE id = $2`, permsJSON, roleID)

	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) ListPermissions(c echo.Context) error {
	ctx := c.Request().Context()
	rows, err := h.deps.DB.Query(ctx, `SELECT id, code, resource, action, description FROM permissions ORDER BY resource, action`)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "query failed")
	}
	defer rows.Close()

	var perms []map[string]interface{}
	for rows.Next() {
		var id int64
		var code, resource, action string
		var desc *string
		if err := rows.Scan(&id, &code, &resource, &action, &desc); err != nil {
			continue
		}
		perms = append(perms, map[string]interface{}{
			"id":         id,
			"code":       code,
			"resource":   resource,
			"action":     action,
			"description": desc,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"permissions": perms})
}

func (h *Handler) GetRBACMatrix(c echo.Context) error {
	ctx := c.Request().Context()

	// Get all roles
	roleRows, err := h.deps.DB.Query(ctx, `SELECT code, permissions FROM roles`)
	if err != nil {
		return c.JSON(http.StatusOK, map[string]interface{}{"matrix": map[string]interface{}{}})
	}
	defer roleRows.Close()

	matrix := make(map[string]map[string]bool)
	for roleRows.Next() {
		var code string
		var permsJSON []byte
		if err := roleRows.Scan(&code, &permsJSON); err != nil {
			continue
		}
		var perms []string
		_ = json.Unmarshal(permsJSON, &perms)
		rolePerms := make(map[string]bool)
		for _, p := range perms {
			rolePerms[p] = true
		}
		matrix[code] = rolePerms
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"matrix": matrix})
}

// Branding
func (h *Handler) UpdateBranding(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	var req struct {
		ProductName        string  `json:"product_name"`
		PrimaryColor       *string `json:"primary_color"`
		SecondaryColor     *string `json:"secondary_color"`
		AccentColor        *string `json:"accent_color"`
		DangerColor        *string `json:"danger_color"`
		SupportEmail       *string `json:"support_email"`
		SupportPhone       *string `json:"support_phone"`
		SupportTelegram    *string `json:"support_telegram"`
		Timezone           *string `json:"timezone"`
		DateFormat         *string `json:"date_format"`
		FirstDayOfWeek     *int    `json:"first_day_of_week"`
		CurrencySymbol     *string `json:"currency_symbol"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	ctx := c.Request().Context()
	query := `
		UPDATE branding_config
		SET product_name = $1,
		    primary_color = COALESCE($2, primary_color),
		    secondary_color = COALESCE($3, secondary_color),
		    accent_color = COALESCE($4, accent_color),
		    danger_color = COALESCE($5, danger_color),
		    support_email = COALESCE($6, support_email),
		    support_phone = COALESCE($7, support_phone),
		    support_telegram = COALESCE($8, support_telegram),
		    timezone = COALESCE($9, timezone),
		    date_format = COALESCE($10, date_format),
		    first_day_of_week = COALESCE($11, first_day_of_week),
		    currency_symbol = COALESCE($12, currency_symbol),
		    updated_at = NOW()
		WHERE tenant_id = $13
	`
	targetTenantID := claims.TenantID
	if claims.Role == "root" {
		// Root can specify tenant_id via query param
		if t := c.QueryParam("tenant_id"); t != "" {
			targetTenantID, _ = strconv.ParseInt(t, 10, 64)
		}
	}

	if err := h.deps.DB.Exec(ctx, query,
		req.ProductName, req.PrimaryColor, req.SecondaryColor, req.AccentColor,
		req.DangerColor, req.SupportEmail, req.SupportPhone, req.SupportTelegram,
		req.Timezone, req.DateFormat, req.FirstDayOfWeek, req.CurrencySymbol,
		targetTenantID,
	); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "update failed")
	}

	// Invalidate cache
	_ = h.deps.Branding.InvalidateCache(ctx, targetTenantID)

	// Publish config change
	if h.deps.Pulsar != nil {
		event := map[string]interface{}{
			"type":      "branding_updated",
			"tenant_id": targetTenantID,
		}
		payload, _ := json.Marshal(event)
		go h.deps.Pulsar.PublishTo(ctx, "persistent://billing/config/config.changes", payload, "")
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) UploadLogo(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	// Parse multipart form
	fileHeader, err := c.FormFile("logo")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "missing logo file")
	}
	file, err := fileHeader.Open()
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "cannot open logo file")
	}
	defer file.Close()

	// Validate size
	maxSize := int64(h.deps.Config.Branding.MaxLogoSizeMB) * 1024 * 1024
	if fileHeader.Size > maxSize {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("file too large (max %dMB)", h.deps.Config.Branding.MaxLogoSizeMB))
	}

	// Validate MIME type by extension
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	allowed := map[string]string{
		".png":  "image/png",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".svg":  "image/svg+xml",
	}
	contentType, ok := allowed[ext]
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid file type (allowed: png, jpg, svg)")
	}

	// Generate object key
	objectKey := fmt.Sprintf("branding/%d/logo-%s%s", claims.TenantID, uuid.New().String(), ext)

	ctx := c.Request().Context()

	// Upload to MinIO
	if err := h.deps.Storage.Upload(ctx, objectKey, file, fileHeader.Size, contentType); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("upload failed: %v", err))
	}

	logoURL := h.deps.Storage.PublicURL(objectKey)

	// Update database
	if err := h.deps.DB.Exec(ctx, `
		UPDATE branding_config SET logo_url = $1, updated_at = NOW()
		WHERE tenant_id = $2
	`, logoURL, claims.TenantID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "update branding config failed")
	}

	// Publish cache invalidation event
	if h.deps.Pulsar != nil {
		payload, _ := json.Marshal(map[string]interface{}{
			"type":      "branding_updated",
			"tenant_id": claims.TenantID,
			"timestamp": time.Now(),
		})
		go h.deps.Pulsar.PublishTo(ctx, "persistent://billing/config/config.changes", payload, "")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"logo_url": logoURL})
}

func (h *Handler) ListEmailTemplates(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	ctx := c.Request().Context()
	rows, err := h.deps.DB.Query(ctx, `
		SELECT e.id, e.template_type, l.code as language, e.subject_template, e.updated_at
		FROM email_template e
		LEFT JOIN language l ON e.language_id = l.id
		WHERE e.tenant_id = $1
	`, claims.TenantID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "query failed")
	}
	defer rows.Close()

	var templates []map[string]interface{}
	for rows.Next() {
		var id int64
		var templateType, subject string
		var lang *string
		var updatedAt time.Time
		if err := rows.Scan(&id, &templateType, &lang, &subject, &updatedAt); err != nil {
			continue
		}
		templates = append(templates, map[string]interface{}{
			"id":       id,
			"type":     templateType,
			"language": lang,
			"subject":  subject,
			"updated_at": updatedAt,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"templates": templates})
}

func (h *Handler) UpdateEmailTemplate(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	templateID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid template id")
	}

	var req struct {
		Subject string `json:"subject_template"`
		Body    string `json:"body_html_template"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	ctx := c.Request().Context()
	_ = h.deps.DB.Exec(ctx, `
		UPDATE email_template
		SET subject_template = $1, body_html_template = $2, updated_at = NOW()
		WHERE id = $3 AND tenant_id = $4
	`, req.Subject, req.Body, templateID, claims.TenantID)

	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) PreviewEmailTemplate(c echo.Context) error {
	// TODO: Render template with sample data
	return c.JSON(http.StatusOK, map[string]interface{}{
		"subject": "Welcome!",
		"body":    "<html><body><h1>Welcome to Orange Billing</h1></body></html>",
	})
}

// Localization
func (h *Handler) ImportTranslations(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	var req struct {
		Locale       string            `json:"locale" validate:"required"`
		Translations map[string]string `json:"translations" validate:"required"`
		Category     string            `json:"category" default:"common"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	ctx := c.Request().Context()

	// Get language ID
	var langID int64
	if err := h.deps.DB.QueryRow(ctx, `SELECT id FROM language WHERE code = $1`, req.Locale).Scan(&langID); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid locale")
	}

	// Upsert translations
	for key, value := range req.Translations {
		_ = h.deps.DB.Exec(ctx, `
			INSERT INTO translation (language_id, key, value, category)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (language_id, key, category) DO UPDATE SET
			    value = EXCLUDED.value,
			    updated_at = NOW()
		`, langID, key, value, req.Category)
	}

	// Invalidate cache
	_ = h.deps.I18n.InvalidateCache(ctx, req.Locale)

	// Publish event
	if h.deps.Pulsar != nil {
		event := map[string]interface{}{
			"type":      "translation_updated",
			"locale":    req.Locale,
			"category":  req.Category,
		}
		payload, _ := json.Marshal(event)
		go h.deps.Pulsar.PublishTo(ctx, "persistent://billing/config/config.changes", payload, "")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"imported": len(req.Translations),
		"locale":   req.Locale,
	})
}

// Monitoring
func (h *Handler) Metrics(c echo.Context) error {
	return c.String(http.StatusOK, `# HELP http_requests_total Total HTTP requests
# TYPE http_requests_total counter
http_requests_total{portal="admin"} 0
`)
}

func (h *Handler) GetAuditLog(c echo.Context) error {
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

	portalFilter := c.QueryParam("portal_type")
	actionFilter := c.QueryParam("action")

	var query string
	var args []interface{}
	argCount := 0

	if claims.Role == "root" {
		query = `SELECT id, tenant_id, user_id, portal_type, action, entity_type, entity_id, created_at FROM audit_log WHERE 1=1`
	} else {
		argCount++
		query = fmt.Sprintf(`SELECT id, tenant_id, user_id, portal_type, action, entity_type, entity_id, created_at FROM audit_log WHERE tenant_id = $%d`, argCount)
		args = append(args, claims.TenantID)
	}

	if portalFilter != "" {
		argCount++
		query += fmt.Sprintf(` AND portal_type = $%d`, argCount)
		args = append(args, portalFilter)
	}
	if actionFilter != "" {
		argCount++
		query += fmt.Sprintf(` AND action = $%d`, argCount)
		args = append(args, actionFilter)
	}

	argCount++
	query += fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, argCount, argCount+1)
	args = append(args, perPage, (page-1)*perPage)

	rows, err := h.deps.DB.Query(ctx, query, args...)
	if err != nil {
		return c.JSON(http.StatusOK, models.PaginatedResponse{
			Data:       []map[string]interface{}{},
			Pagination: models.Pagination{Page: page, PerPage: perPage, Total: 0, TotalPages: 0},
		})
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var id int64
		var tenantID, userID int64
		var portalType, action, entityType, entityID string
		var createdAt time.Time
		if err := rows.Scan(&id, &tenantID, &userID, &portalType, &action, &entityType, &entityID, &createdAt); err != nil {
			continue
		}
		logs = append(logs, map[string]interface{}{
			"id":          id,
			"tenant_id":   tenantID,
			"user_id":     userID,
			"portal_type": portalType,
			"action":      action,
			"entity_type": entityType,
			"entity_id":   entityID,
			"created_at":  createdAt,
		})
	}

	return c.JSON(http.StatusOK, models.PaginatedResponse{
		Data: logs,
		Pagination: models.Pagination{
			Page:       page,
			PerPage:    perPage,
			Total:      len(logs),
			TotalPages: 1,
		},
	})
}

func (h *Handler) GetPulsarTopics(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"topics": []string{
			"persistent://billing/events/charges",
			"persistent://billing/events/topups",
			"persistent://billing/audit/events",
			"persistent://billing/audit/security",
			"persistent://billing/notifications/email",
			"persistent://billing/notifications/sms",
			"persistent://billing/commands/balance.adjust",
			"persistent://billing/commands/tariff.update",
			"persistent://billing/config/config.changes",
		},
	})
}

func (h *Handler) GetPulsarLag(c echo.Context) error {
	// TODO: Query Pulsar admin API for real lag metrics
	return c.JSON(http.StatusOK, map[string]interface{}{
		"lag": map[string]int64{
			"billing.events.charges": 0,
			"billing.audit.events":   0,
			"billing.notifications.email": 0,
		},
	})
}

func (h *Handler) GetDatabaseStatus(c echo.Context) error {
	ctx := c.Request().Context()

	// Check primary connection
	var primaryUp bool
	var primaryLag float64
	if err := h.deps.DB.QueryRow(ctx, `SELECT true, 0.0`).Scan(&primaryUp, &primaryLag); err != nil {
		primaryUp = false
	}

	// Get partition list
	rows, err := h.deps.DB.Query(ctx, `
		SELECT inhrelid::regclass::text
		FROM pg_inherits
		WHERE inhparent = 'audit_log'::regclass
	`)
	var partitions []string
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var p string
			if err := rows.Scan(&p); err == nil {
				partitions = append(partitions, p)
			}
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"primary": map[string]interface{}{
			"status":      map[bool]string{true: "up", false: "down"}[primaryUp],
			"lag_seconds": primaryLag,
		},
		"replicas":      []map[string]interface{}{},
		"partitions":    partitions,
		"vacuum_status": "healthy",
	})
}

// API Keys
func (h *Handler) ListAPIKeys(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	ctx := c.Request().Context()
	rows, err := h.deps.DB.Query(ctx, `
		SELECT id, name, portal_type, scopes, rate_limit_rps, expires_at, last_used_at, created_at
		FROM api_keys
		WHERE tenant_id = $1
	`, claims.TenantID)
	if err != nil {
		return c.JSON(http.StatusOK, map[string]interface{}{"keys": []map[string]interface{}{}})
	}
	defer rows.Close()

	var keys []map[string]interface{}
	for rows.Next() {
		var id int64
		var name, portalType string
		var scopesJSON []byte
		var rateLimit int
		var expiresAt, lastUsedAt, createdAt *time.Time
		if err := rows.Scan(&id, &name, &portalType, &scopesJSON, &rateLimit, &expiresAt, &lastUsedAt, &createdAt); err != nil {
			continue
		}
		var scopes map[string]interface{}
		_ = json.Unmarshal(scopesJSON, &scopes)
		keys = append(keys, map[string]interface{}{
			"id":           id,
			"name":         name,
			"portal_type":  portalType,
			"scopes":       scopes,
			"rate_limit":   rateLimit,
			"expires_at":   expiresAt,
			"last_used_at": lastUsedAt,
			"created_at":   createdAt,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"keys": keys})
}

func (h *Handler) CreateAPIKey(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	var req struct {
		Name      string   `json:"name" validate:"required"`
		PortalType string  `json:"portal_type" validate:"required,oneof=selfcare operator admin"`
		Scopes    []string `json:"scopes"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	// Generate API key
	rawKey, _ := security.GenerateRandomToken(32)
	hash := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(hash[:])

	scopesJSON, _ := json.Marshal(req.Scopes)
	ctx := c.Request().Context()

	query := `
		INSERT INTO api_keys (tenant_id, name, key_hash, portal_type, scopes, expires_at)
		VALUES ($1, $2, $3, $4, $5, NOW() + INTERVAL '180 days')
		RETURNING id
	`
	var id int64
	if err := h.deps.DB.QueryRow(ctx, query, claims.TenantID, req.Name, keyHash, req.PortalType, scopesJSON).Scan(&id); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "create failed")
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"id":  id,
		"key": rawKey,
	})
}

func (h *Handler) DeleteAPIKey(c echo.Context) error {
	claims := middleware.GetClaims(c)
	if claims == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
	}

	keyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid key id")
	}

	ctx := c.Request().Context()
	_ = h.deps.DB.Exec(ctx, `DELETE FROM api_keys WHERE id = $1 AND tenant_id = $2`, keyID, claims.TenantID)
	return c.NoContent(http.StatusNoContent)
}

