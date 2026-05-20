package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/Grimid86/cgrates-ui/backend/pkg/config"
	"github.com/Grimid86/cgrates-ui/backend/pkg/db"
	"github.com/Grimid86/cgrates-ui/backend/pkg/redis"
	"github.com/labstack/echo/v4"
)

// IdempotencyMiddleware deduplicates mutating requests using Idempotency-Key header
func IdempotencyMiddleware(dbPool *db.Pool, redisClient *redis.Client, cfg *config.Config) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			method := c.Request().Method
			if method == "GET" || method == "HEAD" || method == "OPTIONS" {
				return next(c)
			}
			key := c.Request().Header.Get("Idempotency-Key")
			if key == "" {
				return next(c)
			}

			portal := string(cfg.GatewayType)
			redisKey := "idempotency:" + portal + ":" + key
			ctx := c.Request().Context()

			// Check Redis first
			if cached, err := redisClient.Get(ctx, redisKey); err == nil && cached != "" {
				var cachedResp map[string]interface{}
				if json.Unmarshal([]byte(cached), &cachedResp) == nil {
					return c.JSON(http.StatusOK, cachedResp)
				}
			}

			// Read request body for hash
			body, err := io.ReadAll(c.Request().Body)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "failed to read request body")
			}
			c.Request().Body = io.NopCloser(bytes.NewBuffer(body))
			hash := sha256.Sum256(body)
			bodyHash := hex.EncodeToString(hash[:])

			// Check database
			var existingHash string
			var responseData []byte
			err = dbPool.QueryRow(ctx,
				`SELECT request_hash, response_data FROM rpc_idempotency_keys WHERE key = $1 AND portal_type = $2 AND expires_at > NOW()`,
				key, portal,
			).Scan(&existingHash, &responseData)
			if err == nil {
				if existingHash != bodyHash {
					return echo.NewHTTPError(http.StatusConflict, "idempotency key reused with different request body")
				}
				// Cache hit in DB — store in Redis and return
				_ = redisClient.Set(ctx, redisKey, string(responseData), cfg.Security.IdempotencyTTL)
				var resp map[string]interface{}
				_ = json.Unmarshal(responseData, &resp)
				return c.JSON(http.StatusOK, resp)
			}

			// Execute handler and capture response
			originalWriter := c.Response().Writer
			rec := newResponseRecorder(originalWriter)
			c.Response().Writer = rec
			err = next(c)

			// Restore original writer immediately
			c.Response().Writer = originalWriter
			if err != nil {
				return err
			}

			// Only cache successful responses (2xx)
			if rec.status >= 200 && rec.status < 300 {
				var respData map[string]interface{}
				if json.Unmarshal(rec.body.Bytes(), &respData) == nil {
					raw, _ := json.Marshal(respData)
					_ = dbPool.Exec(ctx,
						`INSERT INTO rpc_idempotency_keys (key, portal_type, request_hash, response_data, expires_at) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (key, portal_type) DO UPDATE SET request_hash = EXCLUDED.request_hash, response_data = EXCLUDED.response_data, expires_at = EXCLUDED.expires_at`,
						key, portal, bodyHash, raw, time.Now().Add(cfg.Security.IdempotencyTTL),
					)
					_ = redisClient.Set(ctx, redisKey, string(raw), cfg.Security.IdempotencyTTL)
				}
			}

			// Write captured response to real response writer
			if rec.status > 0 {
				c.Response().WriteHeader(rec.status)
			}
			_, _ = rec.body.WriteTo(originalWriter)
			return nil
		}
	}
}

type responseRecorder struct {
	original http.ResponseWriter
	status   int
	body     *bytes.Buffer
	header   http.Header
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{
		original: w,
		status:   http.StatusOK,
		body:     &bytes.Buffer{},
		header:   w.Header().Clone(),
	}
}

func (r *responseRecorder) Header() http.Header {
	return r.header
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	return r.body.Write(b)
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
}
