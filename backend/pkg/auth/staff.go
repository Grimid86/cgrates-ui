package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Grimid86/cgrates-ui/backend/pkg/config"
	"github.com/Grimid86/cgrates-ui/backend/pkg/db"
	"github.com/Grimid86/cgrates-ui/backend/pkg/middleware"
	"github.com/Grimid86/cgrates-ui/backend/pkg/models"
	"github.com/Grimid86/cgrates-ui/backend/pkg/security"
	"github.com/google/uuid"
)

// StaffAuth handles operator and admin authentication
type StaffAuth struct {
	DB                *db.Pool
	JWTConfig         middleware.JWTConfig
	MaxFailedAttempts int
	LockoutDuration   time.Duration
}

// NewStaffAuth creates a new staff authentication service
func NewStaffAuth(dbPool *db.Pool, jwtCfg middleware.JWTConfig, cfg *config.Config) *StaffAuth {
	return &StaffAuth{
		DB:                dbPool,
		JWTConfig:         jwtCfg,
		MaxFailedAttempts: cfg.Security.MaxFailedLoginAttempts,
		LockoutDuration:   cfg.Security.LockoutDuration,
	}
}

// Authenticate validates email + password and returns user with permissions
func (a *StaffAuth) Authenticate(ctx context.Context, email, password string, portalType string) (*models.User, error) {
	var user models.User
	query := `
		SELECT u.id, u.uuid, u.tenant_id, u.email, u.password_hash, u.role_id, r.code,
		       u.locale, u.mfa_enabled, u.mfa_secret, u.mfa_backup_codes,
		       u.is_active, u.failed_login_attempts, u.locked_until
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.email = $1 AND u.is_active = true
	`
	var pinHash string
	var backupCodesRaw []byte
	err := a.DB.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.UUID, &user.TenantID, &user.Email, &pinHash,
		&user.RoleID, &user.RoleCode, &user.Locale, &user.MFAEnabled, &user.MFASecret,
		&backupCodesRaw, &user.IsActive, &user.FailedLoginAttempts, &user.LockedUntil,
	)
	if err == nil && len(backupCodesRaw) > 0 {
		_ = json.Unmarshal(backupCodesRaw, &user.MFABackupCodes)
	}
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return nil, fmt.Errorf("account locked")
	}
	// If lock expired, reset counter implicitly on next wrong attempt via SQL logic

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
	sessionID := uuid.New().String()

	claims := middleware.Claims{
		UserID:      user.ID,
		TenantID:    user.TenantID,
		Role:        user.RoleCode,
		Locale:      user.Locale,
		MFAVerified: mfaVerified,
		Portal:      portalType,
	}
	claims.ID = sessionID

	accessToken, err := a.JWTConfig.GenerateAccessToken(claims)
	if err != nil {
		return nil, "", "", err
	}

	refreshToken, err := a.JWTConfig.GenerateRefreshToken(claims)
	if err != nil {
		return nil, "", "", err
	}
	expiresAt := time.Now().Add(a.JWTConfig.RefreshTTL)

	// Hash refresh token for storage
	hash := sha256.Sum256([]byte(refreshToken))
	tokenHash := hex.EncodeToString(hash[:])

	query := `
		INSERT INTO portal_sessions (id, user_id, portal_type, token_hash, ip_address, mfa_verified, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	if err := a.DB.Exec(ctx, query, sessionID, user.ID, portalType, tokenHash, ip, mfaVerified, expiresAt); err != nil {
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
		SET failed_login_attempts = CASE 
				WHEN locked_until IS NOT NULL AND locked_until < NOW() THEN 1
				ELSE failed_login_attempts + 1 
			END,
		    locked_until = CASE 
				WHEN locked_until IS NOT NULL AND locked_until < NOW() THEN
					CASE WHEN 1 >= $2 THEN NOW() + ($3 || ' seconds')::interval ELSE NULL END
				WHEN failed_login_attempts + 1 >= $2 THEN NOW() + ($3 || ' seconds')::interval
				ELSE locked_until 
			END
		WHERE id = $1
	`
	return a.DB.Exec(ctx, query, userID, a.MaxFailedAttempts, fmt.Sprintf("%d", int(a.LockoutDuration.Seconds())))
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

// RevokeSession marks a portal session as revoked
func (a *StaffAuth) RevokeSession(ctx context.Context, sessionID string) error {
	query := `UPDATE portal_sessions SET revoked_at = NOW() WHERE id = $1`
	return a.DB.Exec(ctx, query, sessionID)
}

// ValidateRefreshToken checks if a refresh token hash matches an active session
func (a *StaffAuth) ValidateRefreshToken(ctx context.Context, tokenHash string) (*models.PortalSession, error) {
	query := `
		SELECT id, user_id, portal_type, ip_address, mfa_verified, issued_at, expires_at
		FROM portal_sessions
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`
	var s models.PortalSession
	err := a.DB.QueryRow(ctx, query, tokenHash).Scan(
		&s.ID, &s.UserID, &s.PortalType, &s.IPAddress, &s.MFAVerified, &s.IssuedAt, &s.ExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired refresh token")
	}
	return &s, nil
}

// RotatePortalSession validates an old refresh token, revokes it, and issues a new pair
func (a *StaffAuth) RotatePortalSession(ctx context.Context, sessionID, oldTokenHash, ip string) (string, string, error) {
	query := `SELECT user_id, portal_type, mfa_verified FROM portal_sessions WHERE id = $1 AND token_hash = $2 AND revoked_at IS NULL AND expires_at > NOW()`
	var userID int64
	var portalType string
	var mfaVerified bool
	err := a.DB.QueryRow(ctx, query, sessionID, oldTokenHash).Scan(&userID, &portalType, &mfaVerified)
	if err != nil {
		return "", "", fmt.Errorf("invalid or expired refresh token")
	}

	var user models.User
	var pinHash string
	var backupCodesRaw []byte
	err = a.DB.QueryRow(ctx, `
		SELECT u.id, u.uuid, u.tenant_id, u.email, u.password_hash, u.role_id, r.code,
		       u.locale, u.mfa_enabled, u.mfa_secret, u.mfa_backup_codes,
		       u.is_active, u.failed_login_attempts, u.locked_until
		FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.id = $1
	`, userID).Scan(
		&user.ID, &user.UUID, &user.TenantID, &user.Email, &pinHash,
		&user.RoleID, &user.RoleCode, &user.Locale, &user.MFAEnabled, &user.MFASecret,
		&backupCodesRaw, &user.IsActive, &user.FailedLoginAttempts, &user.LockedUntil,
	)
	if err == nil && len(backupCodesRaw) > 0 {
		_ = json.Unmarshal(backupCodesRaw, &user.MFABackupCodes)
	}
	if err != nil {
		return "", "", fmt.Errorf("user not found")
	}

	_ = a.RevokeSession(ctx, sessionID)

	_, accessToken, refreshToken, err := a.CreatePortalSession(ctx, &user, portalType, ip, mfaVerified)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}
