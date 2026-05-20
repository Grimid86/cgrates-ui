package middleware

import (
	"encoding/json"
	"time"

	"github.com/Grimid86/cgrates-ui/backend/pkg/pulsar"
	"github.com/labstack/echo/v4"
)

// AuditConfig holds audit logging configuration
type AuditConfig struct {
	Pulsar      *pulsar.Client
	PortalType  string
	Async       bool
	SyncForAuth bool
}

// AuditEntry represents a single audit log entry
type AuditEntry struct {
	ID         int64                  `json:"id,omitempty"`
	TenantID   int64                  `json:"tenant_id"`
	UserID     int64                  `json:"user_id,omitempty"`
	PortalType string                 `json:"portal_type"`
	Action     string                 `json:"action"`
	EntityType string                 `json:"entity_type,omitempty"`
	EntityID   string                 `json:"entity_id,omitempty"`
	OldData    map[string]interface{} `json:"old_data,omitempty"`
	NewData    map[string]interface{} `json:"new_data,omitempty"`
	IPAddress  string                 `json:"ip_address"`
	UserAgent  string                 `json:"user_agent"`
	RequestID  string                 `json:"request_id"`
	Timestamp  time.Time              `json:"timestamp"`
}

// AuditMiddleware returns Echo middleware for audit logging
func AuditMiddleware(cfg AuditConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			duration := time.Since(start)

			// Skip health checks and static files
			path := c.Request().URL.Path
			if path == "/health" || path == "/metrics" || path == "/favicon.ico" {
				return err
			}

			claims := GetClaims(c)
			var userID int64
			var tenantID int64
			if claims != nil {
				userID = claims.UserID
				tenantID = claims.TenantID
			}

			entry := AuditEntry{
				TenantID:   tenantID,
				UserID:     userID,
				PortalType: cfg.PortalType,
				Action:     c.Request().Method + " " + path,
				IPAddress:  c.RealIP(),
				UserAgent:  c.Request().UserAgent(),
				RequestID:  c.Response().Header().Get("X-Request-ID"),
				Timestamp:  time.Now(),
			}

			// Add response status
			if err != nil {
				he := err.(*echo.HTTPError)
				if he != nil {
					entry.NewData = map[string]interface{}{
						"status_code": he.Code,
						"error":       he.Message,
						"duration_ms": duration.Milliseconds(),
					}
				}
			} else {
				entry.NewData = map[string]interface{}{
					"status_code": c.Response().Status,
					"duration_ms": duration.Milliseconds(),
				}
			}

			// Publish to Pulsar (async)
			if cfg.Pulsar != nil {
				payload, _ := json.Marshal(entry)
				go cfg.Pulsar.Publish(c.Request().Context(), payload, "")
			}

			return err
		}
	}
}
