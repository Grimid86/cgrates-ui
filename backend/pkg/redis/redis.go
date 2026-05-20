// Package redis provides a Redis client wrapper for caching, sessions,
// rate limiting, and idempotency key storage.
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps go-redis
type Client struct {
	rdb *redis.Client
}

// Config for Redis connection
type Config struct {
	Addr     string
	Password string
	DB       int
}

// New creates a new Redis client
func New(cfg Config) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	return &Client{rdb: rdb}, nil
}

// Close closes the Redis connection
func (c *Client) Close() error {
	return c.rdb.Close()
}

// Get retrieves a value by key
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.rdb.Get(ctx, key).Result()
}

// Set stores a value with TTL
func (c *Client) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, value, ttl).Err()
}

// Delete removes keys
func (c *Client) Delete(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

// Incr increments a counter
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	return c.rdb.Incr(ctx, key).Result()
}

// Expire sets TTL on a key
func (c *Client) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.rdb.Expire(ctx, key, ttl).Err()
}

// HSet sets hash field
func (c *Client) HSet(ctx context.Context, key string, values ...interface{}) error {
	return c.rdb.HSet(ctx, key, values...).Err()
}

// HGetAll gets all hash fields
func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return c.rdb.HGetAll(ctx, key).Result()
}

// BlacklistToken adds a token JTI to the revocation list with TTL
func (c *Client) BlacklistToken(ctx context.Context, jti string, ttl time.Duration) error {
	return c.rdb.Set(ctx, "blacklist:"+jti, "1", ttl).Err()
}

// IsTokenBlacklisted checks if a token JTI is revoked
func (c *Client) IsTokenBlacklisted(ctx context.Context, jti string) bool {
	exists, err := c.rdb.Exists(ctx, "blacklist:"+jti).Result()
	return err == nil && exists > 0
}

// SlidingWindowRateLimit checks if request is within rate limit
func (c *Client) SlidingWindowRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	pipe := c.rdb.Pipeline()
	now := time.Now().Unix()
	windowStart := now - int64(window.Seconds())

	// Remove old entries
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))
	// Count current entries
	pipe.ZCard(ctx, key)
	// Add current request
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})
	// Set expiry on the key
	pipe.Expire(ctx, key, window)

	cmders, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	count := cmders[1].(*redis.IntCmd).Val()
	return count < int64(limit), nil
}
