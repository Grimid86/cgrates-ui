// Cache Invalidator — consumes config.changes and invalidates Redis cache
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/Grimid86/cgrates-ui/backend/pkg/pulsar"
	"github.com/Grimid86/cgrates-ui/backend/pkg/redis"
)

// ConfigChangeEvent represents a configuration change event
type ConfigChangeEvent struct {
	Type     string                 `json:"type"`
	TenantID int64                  `json:"tenant_id,omitempty"`
	Locale   string                 `json:"locale,omitempty"`
	Payload  map[string]interface{} `json:"payload,omitempty"`
	Timestamp time.Time             `json:"timestamp"`
}

func main() {
	log.Println("Starting Cache Invalidator...")

	redisClient, err := redis.New(redis.Config{
		Addr: getEnv("REDIS_URL", "localhost:6379"),
	})
	if err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}
	defer redisClient.Close()

	pulsarClient, err := pulsar.New(pulsar.Config{
		URL:    getEnv("PULSAR_URL", "pulsar://localhost:6650"),
		Tenant: getEnv("PULSAR_TENANT", "billing"),
	})
	if err != nil {
		log.Fatalf("Pulsar connection failed: %v", err)
	}
	defer pulsarClient.Close()

	consumer, err := pulsarClient.CreateConsumer(
		"persistent://billing/config/config.changes",
		"cache-invalidator",
	)
	if err != nil {
		log.Fatalf("Consumer creation failed: %v", err)
	}
	defer consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			msg, err := consumer.Receive(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("Receive error: %v", err)
				continue
			}

			var event ConfigChangeEvent
			if err := json.Unmarshal(msg.Payload(), &event); err != nil {
				consumer.Ack(msg)
				continue
			}

			// Process cache invalidation based on event type
			processEvent(ctx, redisClient, event)

			consumer.Ack(msg)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Cache Invalidator...")
}

func processEvent(ctx context.Context, redisClient *redis.Client, event ConfigChangeEvent) {
	switch event.Type {
	case "translation_updated":
		if event.Locale != "" {
			key := fmt.Sprintf("i18n:%s", event.Locale)
			if err := redisClient.Delete(ctx, key); err != nil {
				log.Printf("Failed to invalidate i18n cache for %s: %v", event.Locale, err)
			} else {
				log.Printf("Invalidated i18n cache for locale %s", event.Locale)
			}
		}

	case "branding_updated":
		if event.TenantID > 0 {
			key := fmt.Sprintf("branding:%d", event.TenantID)
			if err := redisClient.Delete(ctx, key); err != nil {
				log.Printf("Failed to invalidate branding cache for tenant %d: %v", event.TenantID, err)
			} else {
				log.Printf("Invalidated branding cache for tenant %d", event.TenantID)
			}
		}

	case "tenant_created", "tenant_updated":
		// Invalidate tenant list caches
		if err := redisClient.Delete(ctx, "tenants:list"); err != nil {
			log.Printf("Failed to invalidate tenant list cache: %v", err)
		}

	case "user_updated", "user_created":
		// Invalidate user caches
		if event.TenantID > 0 {
			key := fmt.Sprintf("users:tenant:%d", event.TenantID)
			_ = redisClient.Delete(ctx, key)
		}

	case "role_permissions_updated":
		// Invalidate RBAC matrix cache
		_ = redisClient.Delete(ctx, "rbac:matrix")

	case "api_key_rotated":
		_ = redisClient.Delete(ctx, "api_keys:list")

	default:
		log.Printf("Unknown config change type: %s", event.Type)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
