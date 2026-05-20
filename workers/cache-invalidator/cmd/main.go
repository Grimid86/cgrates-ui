// Cache Invalidator — consumes config.changes and invalidates Redis cache
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Grimid86/cgrates-ui/backend/pkg/pulsar"
	"github.com/Grimid86/cgrates-ui/backend/pkg/redis"
)

func main() {
	log.Println("Starting Cache Invalidator...")

	redisClient, err := redis.New(redis.Config{
		Addr: os.Getenv("REDIS_URL"),
	})
	if err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}
	defer redisClient.Close()

	pulsarClient, err := pulsar.New(pulsar.Config{
		URL:    os.Getenv("PULSAR_URL"),
		Tenant: os.Getenv("PULSAR_TENANT"),
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
				log.Printf("Receive error: %v", err)
				continue
			}

			var event map[string]interface{}
			if err := json.Unmarshal(msg.Payload(), &event); err != nil {
				consumer.Ack(msg)
				continue
			}

			// Invalidate caches based on event type
			eventType, _ := event["type"].(string)
			switch eventType {
			case "translation_updated":
				locale, _ := event["locale"].(string)
				redisClient.Delete(ctx, "i18n:"+locale)
				log.Printf("Invalidated i18n cache for %s", locale)
			case "branding_updated":
				tenantID, _ := event["tenant_id"].(float64)
				redisClient.Delete(ctx, "branding:"+string(rune(tenantID)))
				log.Printf("Invalidated branding cache for tenant %v", tenantID)
			}

			consumer.Ack(msg)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Cache Invalidator...")
}
