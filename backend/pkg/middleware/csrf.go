package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// CSRFConfig holds CSRF protection configuration
type CSRFConfig struct {
	Secret        []byte
	TokenLength   int
	CookieName    string
	HeaderName    string
	CookieMaxAge  int
}

// DefaultCSRFConfig returns production-ready defaults
func DefaultCSRFConfig(secret string) CSRFConfig {
	return CSRFConfig{
		Secret:       []byte(secret),
		TokenLength:  32,
		CookieName:   "csrf_token",
		HeaderName:   "X-CSRF-Token",
		CookieMaxAge: 86400,
	}
}

// generateCSRFToken creates a signed CSRF token
func generateCSRFToken(secret []byte, sessionID string) string {
	timestamp := time.Now().Unix()
	data := []byte(sessionID + string(rune(timestamp)))
	mac := hmac.New(sha256.New, secret)
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))[:32]
}

// CSRFMiddleware returns Echo middleware for CSRF protection
func CSRFMiddleware(cfg CSRFConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip for safe methods
			if c.Request().Method == http.MethodGet ||
				c.Request().Method == http.MethodHead ||
				c.Request().Method == http.MethodOptions {
				return next(c)
			}

			// Read token from header
			headerToken := c.Request().Header.Get(cfg.HeaderName)
			if headerToken == "" {
				return echo.NewHTTPError(http.StatusForbidden, "missing csrf token")
			}

			// Read token from cookie (double submit)
			cookieToken, err := c.Cookie(cfg.CookieName)
			if err != nil || cookieToken.Value == "" {
				return echo.NewHTTPError(http.StatusForbidden, "missing csrf cookie")
			}

			// Validate double submit
			if headerToken != cookieToken.Value {
				return echo.NewHTTPError(http.StatusForbidden, "invalid csrf token")
			}

			return next(c)
		}
	}
}

// SetCSRFCookie sets the CSRF cookie for a request
func SetCSRFCookie(c echo.Context, cfg CSRFConfig, token string) {
	cookie := &http.Cookie{
		Name:     cfg.CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   cfg.CookieMaxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
	c.SetCookie(cookie)
	c.Response().Header().Set(cfg.HeaderName, token)
}
