package middleware

import (
	"net/http"

	"github.com/Grimid86/cgrates-ui/backend/pkg/redis"
	"github.com/labstack/echo/v4"
)

// TokenBlacklistMiddleware checks if the token's JTI is in the Redis revocation list
func TokenBlacklistMiddleware(redisClient *redis.Client) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := GetClaims(c)
			if claims != nil && claims.ID != "" {
				if redisClient.IsTokenBlacklisted(c.Request().Context(), claims.ID) {
					return echo.NewHTTPError(http.StatusUnauthorized, "token revoked")
				}
			}
			return next(c)
		}
	}
}
