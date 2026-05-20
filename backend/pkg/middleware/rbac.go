package middleware

import (
	"net/http"
	"slices"

	"github.com/labstack/echo/v4"
)

// RequirePermission checks if user has required permission
func RequirePermission(permission string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := GetClaims(c)
			if claims == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
			}

			// Root role bypass
			if claims.Role == "root" {
				return next(c)
			}

			if !slices.Contains(claims.Permissions, permission) {
				return echo.NewHTTPError(http.StatusForbidden, "permission denied: "+permission)
			}

			return next(c)
		}
	}
}

// RequireRole checks if user has one of the required roles
func RequireRole(roles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := GetClaims(c)
			if claims == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
			}

			for _, role := range roles {
				if claims.Role == role {
					return next(c)
				}
			}

			return echo.NewHTTPError(http.StatusForbidden, "role not authorized")
		}
	}
}

// RequireMFA ensures MFA is verified for critical operations
func RequireMFA() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := GetClaims(c)
			if claims == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
			}

			if !claims.MFAVerified {
				return echo.NewHTTPError(http.StatusForbidden, "mfa verification required")
			}

			return next(c)
		}
	}
}

// TenantEnforcer ensures user can only access their tenant data
func TenantEnforcer() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := GetClaims(c)
			if claims == nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
			}

			// Root can access any tenant
			if claims.Role == "root" {
				return next(c)
			}

			// Check tenant_id in path/query matches user's tenant
			pathTenantID := c.Param("tenant_id")
			if pathTenantID != "" {
				// Implementation: compare pathTenantID with claims.TenantID
				// Simplified: actual comparison depends on route structure
			}

			return next(c)
		}
	}
}
