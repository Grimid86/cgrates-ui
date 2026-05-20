// Package auth provides authentication services for subscribers and staff users.
package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/Grimid86/cgrates-ui/backend/pkg/config"
	"github.com/Grimid86/cgrates-ui/backend/pkg/db"
	"github.com/Grimid86/cgrates-ui/backend/pkg/middleware"
	"github.com/Grimid86/cgrates-ui/backend/pkg/models"
	"github.com/Grimid86/cgrates-ui/backend/pkg/security"
	"github.com/google/uuid"
)

// SelfCareAuth handles subscriber authentication
type SelfCareAuth struct {
	DB                *db.Pool
	JWTConfig         middleware.JWTConfig
	MaxFailedAttempts int
	LockoutDuration   time.Duration
}

// NewSelfCareAuth creates a new SelfCare authentication service
func NewSelfCareAuth(dbPool *db.Pool, jwtCfg middleware.JWTConfig, cfg *config.Config) *SelfCareAuth {
	return &SelfCareAuth{
		DB:                dbPool,
		JWTConfig:         jwtCfg,
		MaxFailedAttempts: cfg.Security.MaxFailedLoginAttempts,
		LockoutDuration:   cfg.Security.LockoutDuration,
	}
}

// Authenticate validates MSISDN + PIN and returns subscriber data
func (a *SelfCareAuth) Authenticate(ctx context.Context, msisdn, pin string) (*models.Subscriber, error) {
	var sub models.Subscriber
	query := `
		SELECT id, tenant_id, msisdn, imsi, email, category, is_active,
		       failed_login_attempts, locked_until, pin_hash
		FROM subscriber_credentials
		WHERE msisdn = $1
	`
	var pinHash string
	err := a.DB.QueryRow(ctx, query, msisdn).Scan(
		&sub.ID, &sub.TenantID, &sub.MSISDN, &sub.IMSI, &sub.Email,
		&sub.Category, &sub.IsActive, &sub.FailedLoginAttempts,
		&sub.LockedUntil, &pinHash,
	)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check if account is locked
	if sub.LockedUntil != nil && sub.LockedUntil.After(time.Now()) {
		return nil, fmt.Errorf("account locked")
	}

	// Check if account is active
	if !sub.IsActive {
		return nil, fmt.Errorf("account disabled")
	}

	// Verify PIN
	if !security.VerifyPassword(pin, pinHash) {
		_ = a.incrementFailedAttempts(ctx, sub.ID)
		return nil, fmt.Errorf("invalid credentials")
	}

	// Reset failed attempts and update last login
	_ = a.resetFailedAttempts(ctx, sub.ID)

	return &sub, nil
}

// CreateSession creates a new subscriber session and returns tokens
func (a *SelfCareAuth) CreateSession(ctx context.Context, sub *models.Subscriber, ip, userAgent string) (*models.SubscriberSession, string, string, error) {
	sessionID := uuid.New().String()

	claims := middleware.Claims{
		SubscriberID: sub.ID,
		TenantID:     sub.TenantID,
		MSISDN:       sub.MSISDN,
		Locale:       "ru", // Default or from subscriber profile
	}
	claims.ID = sessionID

	accessToken, err := a.JWTConfig.GenerateAccessToken(claims)
	if err != nil {
		return nil, "", "", fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := a.JWTConfig.GenerateRefreshToken(claims)
	if err != nil {
		return nil, "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	// Hash refresh token for storage
	hash := sha256.Sum256([]byte(refreshToken))
	tokenHash := hex.EncodeToString(hash[:])
	expiresAt := time.Now().Add(a.JWTConfig.RefreshTTL)

	query := `
		INSERT INTO subscriber_sessions (id, subscriber_id, token_hash, ip_address, user_agent, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	if err := a.DB.Exec(ctx, query, sessionID, sub.ID, tokenHash, ip, userAgent, expiresAt); err != nil {
		return nil, "", "", fmt.Errorf("create session: %w", err)
	}

	session := &models.SubscriberSession{
		ID:           sessionID,
		SubscriberID: sub.ID,
		TokenHash:    tokenHash,
		IPAddress:    &ip,
		UserAgent:    &userAgent,
		ExpiresAt:    expiresAt,
	}

	return session, accessToken, refreshToken, nil
}

// RevokeSession marks a session as revoked
func (a *SelfCareAuth) RevokeSession(ctx context.Context, subscriberID int64, sessionID string) error {
	query := `
		UPDATE subscriber_sessions
		SET revoked_at = NOW()
		WHERE subscriber_id = $1 AND id = $2
	`
	return a.DB.Exec(ctx, query, subscriberID, sessionID)
}

// ValidateRefreshToken checks if a refresh token hash matches an active session
func (a *SelfCareAuth) ValidateRefreshToken(ctx context.Context, tokenHash string) (*models.SubscriberSession, error) {
	query := `
		SELECT id, subscriber_id, ip_address, user_agent, issued_at, expires_at
		FROM subscriber_sessions
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`
	var s models.SubscriberSession
	err := a.DB.QueryRow(ctx, query, tokenHash).Scan(
		&s.ID, &s.SubscriberID, &s.IPAddress, &s.UserAgent, &s.IssuedAt, &s.ExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("invalid or expired refresh token")
	}
	return &s, nil
}

// RotateSubscriberSession validates an old refresh token, revokes it, and issues a new pair
func (a *SelfCareAuth) RotateSubscriberSession(ctx context.Context, sessionID, oldTokenHash, ip, userAgent string) (string, string, error) {
	query := `SELECT subscriber_id FROM subscriber_sessions WHERE id = $1 AND token_hash = $2 AND revoked_at IS NULL AND expires_at > NOW()`
	var subscriberID int64
	err := a.DB.QueryRow(ctx, query, sessionID, oldTokenHash).Scan(&subscriberID)
	if err != nil {
		return "", "", fmt.Errorf("invalid or expired refresh token")
	}

	var sub models.Subscriber
	err = a.DB.QueryRow(ctx, `
		SELECT id, tenant_id, msisdn, imsi, email, category, is_active,
		       failed_login_attempts, locked_until
		FROM subscriber_credentials
		WHERE id = $1
	`, subscriberID).Scan(
		&sub.ID, &sub.TenantID, &sub.MSISDN, &sub.IMSI, &sub.Email,
		&sub.Category, &sub.IsActive, &sub.FailedLoginAttempts,
		&sub.LockedUntil,
	)
	if err != nil {
		return "", "", fmt.Errorf("subscriber not found")
	}

	_ = a.RevokeSession(ctx, subscriberID, sessionID)

	_, accessToken, refreshToken, err := a.CreateSession(ctx, &sub, ip, userAgent)
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

// GetActiveSessions returns all active sessions for a subscriber
func (a *SelfCareAuth) GetActiveSessions(ctx context.Context, subscriberID int64) ([]models.SubscriberSession, error) {
	query := `
		SELECT id, subscriber_id, ip_address, user_agent, issued_at, expires_at
		FROM subscriber_sessions
		WHERE subscriber_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		ORDER BY issued_at DESC
	`
	rows, err := a.DB.Query(ctx, query, subscriberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []models.SubscriberSession
	for rows.Next() {
		var s models.SubscriberSession
		if err := rows.Scan(&s.ID, &s.SubscriberID, &s.IPAddress, &s.UserAgent, &s.IssuedAt, &s.ExpiresAt); err != nil {
			continue
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (a *SelfCareAuth) incrementFailedAttempts(ctx context.Context, subscriberID int64) error {
	query := `
		UPDATE subscriber_credentials
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
	return a.DB.Exec(ctx, query, subscriberID, a.MaxFailedAttempts, fmt.Sprintf("%d", int(a.LockoutDuration.Seconds())))
}

func (a *SelfCareAuth) resetFailedAttempts(ctx context.Context, subscriberID int64) error {
	query := `
		UPDATE subscriber_credentials
		SET failed_login_attempts = 0,
		    locked_until = NULL,
		    last_login_at = NOW()
		WHERE id = $1
	`
	return a.DB.Exec(ctx, query, subscriberID)
}

// ChangePIN updates subscriber PIN after verifying old PIN
func (a *SelfCareAuth) ChangePIN(ctx context.Context, subscriberID int64, oldPIN, newPIN string) error {
	var pinHash string
	query := `SELECT pin_hash FROM subscriber_credentials WHERE id = $1`
	if err := a.DB.QueryRow(ctx, query, subscriberID).Scan(&pinHash); err != nil {
		return fmt.Errorf("subscriber not found")
	}

	if !security.VerifyPassword(oldPIN, pinHash) {
		return fmt.Errorf("invalid old PIN")
	}

	newHash, err := security.HashPassword(newPIN, 12)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	updateQuery := `UPDATE subscriber_credentials SET pin_hash = $1 WHERE id = $2`
	return a.DB.Exec(ctx, updateQuery, newHash, subscriberID)
}
