// Package middleware provides HTTP middleware for all gateways.
package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// Claims defines JWT claims structure
type Claims struct {
	UserID       int64    `json:"user_id,omitempty"`
	SubscriberID int64    `json:"sub_id,omitempty"`
	TenantID     int64    `json:"tenant_id"`
	MSISDN       string   `json:"msisdn,omitempty"`
	Role         string   `json:"role,omitempty"`
	Permissions  []string `json:"permissions,omitempty"`
	Locale       string   `json:"locale"`
	BrandingID   int64    `json:"branding_id,omitempty"`
	MFAVerified  bool     `json:"mfa_verified,omitempty"`
	Portal       string   `json:"portal,omitempty"`
	Type         string   `json:"type,omitempty"`
	jwt.RegisteredClaims
}

// JWTConfig holds JWT middleware configuration
type JWTConfig struct {
	Secret      []byte
	AccessTTL   time.Duration
	RefreshTTL  time.Duration
	ContextKey  string
}

// NewJWTConfig creates default JWT config for a gateway
func NewJWTConfig(secret string, accessTTL, refreshTTL time.Duration) JWTConfig {
	return JWTConfig{
		Secret:     []byte(secret),
		AccessTTL:  accessTTL,
		RefreshTTL: refreshTTL,
		ContextKey: "claims",
	}
}

// GenerateAccessToken creates a new access token
func (cfg JWTConfig) GenerateAccessToken(claims Claims) (string, error) {
	claims.Type = "access"
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(cfg.AccessTTL))
	claims.IssuedAt = jwt.NewNumericDate(time.Now())

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(cfg.Secret)
}

// GenerateRefreshToken creates a new refresh token
func (cfg JWTConfig) GenerateRefreshToken(claims Claims) (string, error) {
	claims.Type = "refresh"
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(cfg.RefreshTTL))
	claims.IssuedAt = jwt.NewNumericDate(time.Now())

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(cfg.Secret)
}

// JWTMiddleware returns Echo middleware for JWT validation
func JWTMiddleware(cfg JWTConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			auth := c.Request().Header.Get("Authorization")
			if auth == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
			}

			parts := strings.SplitN(auth, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization format")
			}

			tokenStr := parts[1]
			token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, echo.NewHTTPError(http.StatusUnauthorized, "unexpected signing method")
				}
				return cfg.Secret, nil
			})
			if err != nil || !token.Valid {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid or expired token")
			}

			claims, ok := token.Claims.(*Claims)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid claims")
			}

			// Verify token type
			if claims.Type != "access" {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token type")
			}

			c.Set(cfg.ContextKey, claims)
			return next(c)
		}
	}
}

// GetClaims extracts claims from Echo context
func GetClaims(c echo.Context) *Claims {
	claims, ok := c.Get("claims").(*Claims)
	if !ok {
		return nil
	}
	return claims
}
