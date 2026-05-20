// Package models defines shared data structures used across all gateways and workers.
package models

import (
	"time"
)

// Tenant represents a billing tenant
type Tenant struct {
	ID        int64     `json:"id" db:"id"`
	UUID      string    `json:"uuid" db:"uuid"`
	Name      string    `json:"name" db:"name"`
	Code      string    `json:"code" db:"code"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// User represents a staff user (operator/admin)
type User struct {
	ID                  int64     `json:"id" db:"id"`
	UUID                string    `json:"uuid" db:"uuid"`
	TenantID            int64     `json:"tenant_id" db:"tenant_id"`
	Email               string    `json:"email" db:"email"`
	PasswordHash        string    `json:"-" db:"password_hash"`
	RoleID              int64     `json:"role_id" db:"role_id"`
	RoleCode            string    `json:"role" db:"code"`
	Locale              string    `json:"locale" db:"locale"`
	MFAEnabled          bool      `json:"mfa_enabled" db:"mfa_enabled"`
	IsActive            bool      `json:"is_active" db:"is_active"`
	FailedLoginAttempts int       `json:"failed_login_attempts" db:"failed_login_attempts"`
	LockedUntil         *time.Time `json:"locked_until,omitempty" db:"locked_until"`
	LastLoginAt         *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
}

// Subscriber represents a SelfCare subscriber
type Subscriber struct {
	ID                  int64      `json:"id" db:"id"`
	TenantID            int64      `json:"tenant_id" db:"tenant_id"`
	MSISDN              string     `json:"msisdn" db:"msisdn"`
	IMSI                *string    `json:"imsi,omitempty" db:"imsi"`
	Email               *string    `json:"email,omitempty" db:"email"`
	Category            string     `json:"category" db:"category"`
	IsActive            bool       `json:"is_active" db:"is_active"`
	FailedLoginAttempts int        `json:"failed_login_attempts" db:"failed_login_attempts"`
	LockedUntil         *time.Time `json:"locked_until,omitempty" db:"locked_until"`
	LastLoginAt         *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
}

// SubscriberSession represents an active SelfCare session
type SubscriberSession struct {
	ID           string     `json:"id" db:"id"`
	SubscriberID int64      `json:"subscriber_id" db:"subscriber_id"`
	TokenHash    string     `json:"-" db:"token_hash"`
	IPAddress    *string    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent    *string    `json:"user_agent,omitempty" db:"user_agent"`
	IssuedAt     time.Time  `json:"issued_at" db:"issued_at"`
	ExpiresAt    time.Time  `json:"expires_at" db:"expires_at"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
}

// PortalSession represents a staff user session
type PortalSession struct {
	ID          string     `json:"id" db:"id"`
	UserID      int64      `json:"user_id" db:"user_id"`
	PortalType  string     `json:"portal_type" db:"portal_type"`
	TokenHash   string     `json:"-" db:"token_hash"`
	IPAddress   *string    `json:"ip_address,omitempty" db:"ip_address"`
	MFAVerified bool       `json:"mfa_verified" db:"mfa_verified"`
	IssuedAt    time.Time  `json:"issued_at" db:"issued_at"`
	ExpiresAt   time.Time  `json:"expires_at" db:"expires_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
}

// Balance represents a subscriber balance item
type Balance struct {
	Type       string  `json:"type"`
	Value      float64 `json:"value"`
	Currency   string  `json:"currency,omitempty"`
	Unit       string  `json:"unit,omitempty"`
	ExpiryDate *string `json:"expiry_date,omitempty"`
}

// CDR represents a Call Detail Record
type CDR struct {
	ID              string  `json:"id"`
	Type            string  `json:"type"`
	Destination     string  `json:"destination,omitempty"`
	DurationSeconds int     `json:"duration_seconds,omitempty"`
	Cost            float64 `json:"cost"`
	Currency        string  `json:"currency"`
	StartedAt       string  `json:"started_at"`
	Status          string  `json:"status"`
}

// PaginatedResponse is a generic paginated API response
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination metadata
type Pagination struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}
