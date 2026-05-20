package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/Grimid86/cgrates-ui/backend/pkg/db"
	"github.com/Grimid86/cgrates-ui/backend/pkg/middleware"
	"github.com/Grimid86/cgrates-ui/backend/pkg/models"
	"github.com/Grimid86/cgrates-ui/backend/pkg/security"
	"github.com/google/uuid"
)

// StaffAuth handles operator and admin authentication
type StaffAuth struct {
	DB        *db.Pool
	JWTConfig middleware.JWTConfig
}

// NewStaffAuth creates a new staff authentication service
func NewStaffAuth(dbPool *db.Pool, jwtCfg middleware.JWTConfig) *StaffAuth {
	return &StaffAuth{DB: dbPool, JWTConfig: jwtCfg}
}

// Authenticate validates email + password and returns user with permissions
func (a *StaffAuth) Authenticate(ctx context.Context, email, password string, portalType string) (*models.User, error) {
	var user models.User
	query := `
		SELECT u.id, u.uuid, u.tenant_id, u.email, u.password_hash, u.role_id, r.code,
		       u.locale, u.mfa_enabled, u.is_active, u.failed_login_attempts, u.locked_until
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.email = $1 AND u.is_active = true
	`
	var pinHash string
	err := a.DB.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.UUID, &user.TenantID, &user.Email, &pinHash,
		&user.RoleID, &user.RoleCode, &user.Locale, &user.MFAEnabled,
		&user.IsActive, &user.FailedLoginAttempts, &user.LockedUntil,
	)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return nil, fmt.Errorf("account locked")
	}

	if !security.VerifyPassword(password, pinHash) {
		_ = a.incrementFailedAttempts(ctx, user.ID)
		return nil, fmt.Errorf("invalid credentials")
	}

	_ = a.resetFailedAttempts(ctx, user.ID)
	return &user, nil
}

// GetUserPermissions loads permissions for a user
func (a *StaffAuth) GetUserPermissions(ctx context.Context, roleID int64) ([]string, error) {
	query := `SELECT permissions FROM roles WHERE id = $1`
	var permsJSON []byte
	if err := a.DB.QueryRow(ctx, query, roleID).Scan(&permsJSON); err != nil {
		return nil, err
	}
	var perms []string
	// Simple JSON parsing - in production use proper JSON unmarshaling
	// For now return empty, will be populated from roles table
	_ = permsJSON
	return perms, nil
}

// CreatePortalSession creates a staff session
func (a *StaffAuth) CreatePortalSession(ctx context.Context, user *models.User, portalType, ip string, mfaVerified bool) (*models.PortalSession, string, string, error) {
	claims := middleware.Claims{
		UserID:      user.ID,
		TenantID:    user.TenantID,
		Role:        user.RoleCode,
		Locale:      user.Locale,
		MFAVerified: mfaVerified,
		Portal:      portalType,
	}

	accessToken, err := a.JWTConfig.GenerateAccessToken(claims)
	if err != nil {
		return nil, "", "", err
	}

	refreshToken, err := a.JWTConfig.GenerateRefreshToken(claims)
	if err != nil {
		return nil, "", "", err
	}

	sessionID := uuid.New().String()
	expiresAt := time.Now().Add(a.JWTConfig.RefreshTTL)

	query := `
		INSERT INTO portal_sessions (id, user_id, portal_type, ip_address, mfa_verified, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	if err := a.DB.Exec(ctx, query, sessionID, user.ID, portalType, ip, mfaVerified, expiresAt); err != nil {
		return nil, "", "", err
	}

	session := &models.PortalSession{
		ID:          sessionID,
		UserID:      user.ID,
		PortalType:  portalType,
		IPAddress:   &ip,
		MFAVerified: mfaVerified,
		ExpiresAt:   expiresAt,
	}

	return session, accessToken, refreshToken, nil
}

func (a *StaffAuth) incrementFailedAttempts(ctx context.Context, userID int64) error {
	query := `
		UPDATE users
		SET failed_login_attempts = failed_login_attempts + 1,
		    locked_until = CASE WHEN failed_login_attempts >= 4 THEN NOW() + INTERVAL '15 minutes' ELSE locked_until END
		WHERE id = $1
	`
	return a.DB.Exec(ctx, query, userID)
}

func (a *StaffAuth) resetFailedAttempts(ctx context.Context, userID int64) error {
	query := `
		UPDATE users
		SET failed_login_attempts = 0,
		    locked_until = NULL,
		    last_login_at = NOW()
		WHERE id = $1
	`
	return a.DB.Exec(ctx, query, userID)
}
