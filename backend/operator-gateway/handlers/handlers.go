package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Grimid86/cgrates-ui/backend/pkg/auth"
	"github.com/Grimid86/cgrates-ui/backend/pkg/branding"
	"github.com/Grimid86/cgrates-ui/backend/pkg/cgrates"
	"github.com/Grimid86/cgrates-ui/backend/pkg/db"
	"github.com/Grimid86/cgrates-ui/backend/pkg/i18n"
	"github.com/Grimid86/cgrates-ui/backend/pkg/middleware"
	"github.com/Grimid86/cgrates-ui/backend/pkg/models"
	"github.com/Grimid86/cgrates-ui/backend/pkg/pulsar"
	"github.com/Grimid86/cgrates-ui/backend/pkg/redis"
	"github.com/Grimid86/cgrates-ui/backend/pkg/security"
	"github.com/labstack/echo/v4"
)

// Dependencies holds all service dependencies for handlers
type Dependencies struct {
	DB         *db.Pool
	Redis      *redis.Client
	Pulsar     *pulsar.Client
	CGRateS    *cgrates.Client
	I18n       *i18n.Service
	Branding   *branding.Service
	JWTConfig  middleware.JWTConfig
	StaffAuth  *auth.StaffAuth
}

// Handler contains all HTTP handlers
type Handler struct {
	deps Dependencies
}

func New(deps Dependencies) *Handler {
	if deps.StaffAuth == nil {
		deps.StaffAuth = auth.NewStaffAuth(deps.DB, deps.JWTConfig)
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
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	// Check MFA for admin/reseller roles
	mfaVerified := false
	if user.MFAEnabled && (user.RoleCode == "admin" || user.RoleCode == "reseller") {
		// TODO: Verify TOTP code using user.MFASecret
		if req.MFACode == "" {
			return echo.NewHTTPError(http.StatusForbidden, "mfa code required")
		}
		mfaVerified = true // Replace with real TOTP verification
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
	return c.JSON(http.StatusOK, map[string]interface{}{"access_token": "eyJ...", "expires_in": 900})
}

func (h *Handler) Logout(c echo.Context) error {
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
			go h.deps.Pulsar.PublishTo(ctx, "persistent://billing/commands/commands.balance.adjust", payload, msisdn)
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
	// TODO: Implement balance freeze via CGRateS
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) UnfreezeBalance(c echo.Context) error {
	// TODO: Implement balance unfreeze via CGRateS
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
	return c.JSON(http.StatusOK, map[string]interface{}{"id": c.Param("id")})
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
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) DeleteTariff(c echo.Context) error {
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
	// TODO: Queue export job via Pulsar
	return c.JSON(http.StatusAccepted, map[string]interface{}{"export_id": "export-" + strconv.FormatInt(time.Now().Unix(), 10), "status": "processing"})
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
		go h.deps.Pulsar.PublishTo(ctx, "persistent://billing/commands/commands.tariff.update", payload, "")
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
		go h.deps.Pulsar.PublishTo(ctx, "persistent://billing/commands/commands.balance.adjust", payload, "")
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
	// TODO: Aggregate from CDR data
	return c.JSON(http.StatusOK, map[string]interface{}{"data": []map[string]interface{}{}, "period": map[string]string{}})
}

