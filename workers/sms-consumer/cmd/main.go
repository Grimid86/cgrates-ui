// SMS Consumer — consumes notifications.sms and sends via SMS gateway
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Grimid86/cgrates-ui/backend/pkg/pulsar"
)

// SMSEvent represents an SMS notification from Pulsar
type SMSEvent struct {
	TenantID  int64  `json:"tenant_id"`
	To        string `json:"to"`
	Message   string `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	log.Println("Starting SMS Consumer...")

	pulsarClient, err := pulsar.New(pulsar.Config{
		URL:    os.Getenv("PULSAR_URL"),
		Tenant: os.Getenv("PULSAR_TENANT"),
	})
	if err != nil {
		log.Fatalf("Pulsar connection failed: %v", err)
	}
	defer pulsarClient.Close()

	consumer, err := pulsarClient.CreateConsumer(
		"persistent://billing/notifications/notifications.sms",
		"sms-consumer",
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

			var event SMSEvent
			if err := json.Unmarshal(msg.Payload(), &event); err != nil {
				consumer.Ack(msg)
				continue
			}

			// TODO: Integrate with real SMS gateway (Twilio, AWS SNS, etc.)
			log.Printf("[MOCK SMS] To: %s, Message: %s", event.To, event.Message)

			consumer.Ack(msg)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down SMS Consumer...")
}
