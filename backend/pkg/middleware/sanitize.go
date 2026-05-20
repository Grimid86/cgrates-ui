package middleware

import (
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/labstack/echo/v4"
)

// SanitizeConfig holds request sanitization rules
type SanitizeConfig struct {
	MaxBodySize     int64
	AllowedPatterns map[string]*regexp.Regexp
}

// DefaultSanitizeConfig returns production defaults
func DefaultSanitizeConfig() SanitizeConfig {
	return SanitizeConfig{
		MaxBodySize: 1024 * 1024, // 1MB
		AllowedPatterns: map[string]*regexp.Regexp{
			"msisdn": regexp.MustCompile(`^[0-9]+$`),
			"imsi":   regexp.MustCompile(`^[0-9]+$`),
			"tariff": regexp.MustCompile(`^[A-Za-z0-9_\-]+$`),
			"uuid":   regexp.MustCompile(`^[a-f0-9\-]{36}$`),
		},
	}
}

// SanitizeMiddleware returns Echo middleware for input sanitization
func SanitizeMiddleware(cfg SanitizeConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Limit body size
			c.Request().Body = http.MaxBytesReader(c.Response().Writer, c.Request().Body, cfg.MaxBodySize)

			// Validate path parameters against patterns
			for name, value := range c.ParamValues() {
				if pattern, ok := cfg.AllowedPatterns[name]; ok {
					if !pattern.MatchString(value) {
						return echo.NewHTTPError(http.StatusBadRequest, "invalid parameter: "+name)
					}
				}
			}

			// Sanitize query parameters
			q := c.Request().URL.Query()
			for key, values := range q {
				for i, v := range values {
					values[i] = strings.TrimSpace(v)
					// Basic XSS prevention
					values[i] = strings.ReplaceAll(values[i], "<", "&lt;")
					values[i] = strings.ReplaceAll(values[i], ">", "&gt;")
				}
				q[key] = values
			}
			c.Request().URL.RawQuery = q.Encode()

			return next(c)
		}
	}
}

// ValidateBalanceType ensures balance type is in whitelist
func ValidateBalanceType(balanceType string) bool {
	allowed := map[string]bool{
		"*monetary": true,
		"*data":     true,
		"*sms":      true,
		"*voice":    true,
		"*bonus":    true,
	}
	return allowed[balanceType]
}

// ReadLimitedBody reads request body with size limit
func ReadLimitedBody(r io.Reader, limit int64) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r, limit))
}
