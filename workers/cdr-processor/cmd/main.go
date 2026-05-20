// CDR Processor — consumes CDR raw events, normalizes and writes to PostgreSQL
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

	"github.com/Grimid86/cgrates-ui/backend/pkg/db"
	"github.com/Grimid86/cgrates-ui/backend/pkg/pulsar"
)

// CDREvent represents a raw CDR event from CGRateS
type CDREvent struct {
	CGRID       string  `json:"CGRID"`
	RunID       string  `json:"RunID"`
	OriginHost  string  `json:"OriginHost"`
	Source      string  `json:"Source"`
	OriginID    string  `json:"OriginID"`
	ToR         string  `json:"ToR"`
	RequestType string  `json:"RequestType"`
	Tenant      string  `json:"Tenant"`
	Category    string  `json:"Category"`
	Account     string  `json:"Account"`
	Subject     string  `json:"Subject"`
	Destination string  `json:"Destination"`
	SetupTime   string  `json:"SetupTime"`
	AnswerTime  string  `json:"AnswerTime"`
	Usage       float64 `json:"Usage"`
	Cost        float64 `json:"Cost"`
	ExtraFields map[string]string `json:"ExtraFields"`
}

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

	batchSize := 500
	flushInterval := 2 * time.Second
	batch := make([]CDREvent, 0, batchSize)
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	flushBatch := func() {
		if len(batch) == 0 {
			return
		}
		start := time.Now()
		if err := processBatch(ctx, dbPool, batch); err != nil {
			log.Printf("CDR batch processing error: %v", err)
		} else {
			log.Printf("Processed %d CDRs in %v", len(batch), time.Since(start))
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

			var event CDREvent
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

	log.Println("Shutting down CDR Processor...")
	cancel()
	flushBatch()
}

func processBatch(ctx context.Context, dbPool *db.Pool, events []CDREvent) error {
	for _, e := range events {
		// Normalize CDR
		cdrType := map[string]string{
			"*voice": "voice",
			"*data":  "data",
			"*sms":   "sms",
			"*mms":   "mms",
		}[e.ToR]
		if cdrType == "" {
			cdrType = "other"
		}

		tenantID, _ := strconv.ParseInt(e.Tenant, 10, 64)
		usage := int(e.Usage)

		query := `
			INSERT INTO balance_history (tenant_id, subscriber_id, balance_type, amount_after, operation, extra_data, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`
		extraData, _ := json.Marshal(map[string]interface{}{
			"cgrid":       e.CGRID,
			"destination": e.Destination,
			"usage":       usage,
			"cost":        e.Cost,
		})

		// Find subscriber_id by account (MSISDN)
		var subID int64
		err := dbPool.QueryRow(ctx, `SELECT id FROM subscriber_credentials WHERE msisdn = $1`, e.Account).Scan(&subID)
		if err != nil {
			// Log and skip if subscriber not found
			log.Printf("Subscriber not found for MSISDN %s", e.Account)
			continue
		}

		_ = dbPool.Exec(ctx, query, tenantID, subID, "*"+cdrType, e.Cost, "charge", extraData, time.Now())
	}
	return nil
}
