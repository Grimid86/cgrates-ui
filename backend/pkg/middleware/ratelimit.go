package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Grimid86/cgrates-ui/backend/pkg/redis"
	"github.com/labstack/echo/v4"
)

// RateLimiterConfig holds rate limiting configuration
type RateLimiterConfig struct {
	Redis       *redis.Client
	LimitRPS    int
	LimitRPM    int
	Window      time.Duration
	KeyPrefix   string
	KeyExtractor func(c echo.Context) string
}

// DefaultKeyExtractor uses IP + user ID (if authenticated)
func DefaultKeyExtractor(c echo.Context) string {
	claims := GetClaims(c)
	if claims != nil {
		return fmt.Sprintf("ratelimit:user:%d", claims.UserID)
	}
	return fmt.Sprintf("ratelimit:ip:%s", c.RealIP())
}

// LoginKeyExtractor uses IP only for login endpoints
func LoginKeyExtractor(c echo.Context) string {
	return fmt.Sprintf("ratelimit:login:%s", c.RealIP())
}

// RateLimiter returns Echo middleware for rate limiting
func RateLimiter(cfg RateLimiterConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if cfg.Redis == nil {
				return next(c)
			}

			key := cfg.KeyPrefix + cfg.KeyExtractor(c)
			limit := cfg.LimitRPS
			if cfg.LimitRPM > 0 {
				limit = cfg.LimitRPM
			}
			window := cfg.Window
			if window == 0 {
				window = time.Minute
			}

			allowed, err := cfg.Redis.SlidingWindowRateLimit(c.Request().Context(), key, limit, window)
			if err != nil {
				// Fail open or closed based on policy; here we log and allow
				return next(c)
			}

			if !allowed {
				retryAfter := int(window.Seconds())
				c.Response().Header().Set("Retry-After", strconv.Itoa(retryAfter))
				c.Response().Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
				c.Response().Header().Set("X-RateLimit-Remaining", "0")
				return echo.NewHTTPError(http.StatusTooManyRequests, "rate limit exceeded")
			}

			c.Response().Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			c.Response().Header().Set("X-RateLimit-Remaining", strconv.Itoa(limit-1))
			return next(c)
		}
	}
}

// EndpointRateLimiter creates a per-endpoint rate limiter
func EndpointRateLimiter(redisClient *redis.Client, limit int, window time.Duration, keyExtractor func(c echo.Context) string) echo.MiddlewareFunc {
	return RateLimiter(RateLimiterConfig{
		Redis:        redisClient,
		LimitRPS:     limit,
		Window:       window,
		KeyPrefix:    "rl:",
		KeyExtractor: keyExtractor,
	})
}
