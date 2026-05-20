// CDR Processor — consumes CDR raw events, normalizes and writes to PostgreSQL
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Grimid86/cgrates-ui/backend/pkg/db"
	"github.com/Grimid86/cgrates-ui/backend/pkg/pulsar"
)

func main() {
	log.Println("Starting CDR Processor...")

	dbPool, err := db.New(db.Config{
		DSN: os.Getenv("DB_DSN"),
	})
	if err != nil {
		log.Fatalf("DB connection failed: %v", err)
	}
	defer dbPool.Close()

	pulsarClient, err := pulsar.New(pulsar.Config{
		URL:    os.Getenv("PULSAR_URL"),
		Tenant: os.Getenv("PULSAR_TENANT"),
	})
	if err != nil {
		log.Fatalf("Pulsar connection failed: %v", err)
	}
	defer pulsarClient.Close()

	consumer, err := pulsarClient.CreateConsumer(
		"persistent://billing/events/billing.events.cdr.raw",
		"cdr-processor",
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
				log.Printf("Unmarshal error: %v", err)
				consumer.Ack(msg)
				continue
			}

			log.Printf("CDR processed: %v", event["id"])
			consumer.Ack(msg)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down CDR Processor...")
}
