// Audit Consumer — consumes audit.events from Pulsar and batch inserts to PostgreSQL
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

	"github.com/Grimid86/cgrates-ui/backend/pkg/db"
	"github.com/Grimid86/cgrates-ui/backend/pkg/pulsar"
)

// AuditEvent represents an audit log event from Pulsar
type AuditEvent struct {
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

func main() {
	log.Println("Starting Audit Consumer...")

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
		"persistent://billing/audit/audit.events",
		"audit-consumer",
	)
	if err != nil {
		log.Fatalf("Consumer creation failed: %v", err)
	}
	defer consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Batch buffer
	batchSize := 100
	flushInterval := 5 * time.Second
	batch := make([]AuditEvent, 0, batchSize)
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	flushBatch := func() {
		if len(batch) == 0 {
			return
		}
		if err := insertBatch(ctx, dbPool, batch); err != nil {
			log.Printf("Batch insert error: %v", err)
		} else {
			log.Printf("Inserted %d audit records", len(batch))
		}
		batch = batch[:0]
	}

	go func() {
		for {
			select {
			case <-ticker.C:
				flushBatch()
			case <-ctx.Done():
				return
			}
		}
	}()

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

			var event AuditEvent
			if err := json.Unmarshal(msg.Payload(), &event); err != nil {
				log.Printf("Unmarshal error: %v", err)
				consumer.Ack(msg)
				continue
			}

			batch = append(batch, event)
			consumer.Ack(msg)

			if len(batch) >= batchSize {
				flushBatch()
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Audit Consumer...")
	cancel()
	flushBatch()
}

func insertBatch(ctx context.Context, dbPool *db.Pool, events []AuditEvent) error {
	// Use COPY for efficient batch insert
	// Fallback to individual inserts for simplicity in this implementation
	for _, e := range events {
		oldJSON, _ := json.Marshal(e.OldData)
		newJSON, _ := json.Marshal(e.NewData)

		query := `
			INSERT INTO audit_log (tenant_id, user_id, portal_type, action, entity_type, entity_id, old_data, new_data, ip_address, user_agent, request_id, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		`
		if err := dbPool.Exec(ctx, query,
			e.TenantID, e.UserID, e.PortalType, e.Action, e.EntityType, e.EntityID,
			oldJSON, newJSON, e.IPAddress, e.UserAgent, e.RequestID, e.Timestamp,
		); err != nil {
			log.Printf("Insert audit error: %v", err)
		}
	}
	return nil
}
