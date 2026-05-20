// Balance Monitor — consumes charges/topups, checks action triggers
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

// BalanceEvent represents a balance change event
type BalanceEvent struct {
	Type        string  `json:"type"` // charge, topup
	TenantID    int64   `json:"tenant_id"`
	SubscriberID int64  `json:"subscriber_id"`
	MSISDN      string  `json:"msisdn"`
	BalanceType string  `json:"balance_type"`
	Amount      float64 `json:"amount"`
	Timestamp   time.Time `json:"timestamp"`
}

// ActionTrigger represents a configured trigger from DB
type ActionTrigger struct {
	ID         int64                  `json:"id"`
	TenantID   int64                  `json:"tenant_id"`
	Name       string                 `json:"name"`
	Conditions map[string]interface{} `json:"conditions"`
	Actions    map[string]interface{} `json:"actions"`
}

func main() {
	log.Println("Starting Balance Monitor...")

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
		"persistent://billing/events/billing.events.charges",
		"balance-monitor",
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

			var event BalanceEvent
			if err := json.Unmarshal(msg.Payload(), &event); err != nil {
				consumer.Ack(msg)
				continue
			}

			// Check triggers for this tenant
			triggers, err := loadTriggers(ctx, dbPool, event.TenantID)
			if err != nil {
				log.Printf("Load triggers error: %v", err)
			} else {
				for _, trigger := range triggers {
					if shouldTrigger(trigger, event) {
						executeTrigger(ctx, pulsarClient, trigger, event)
					}
				}
			}

			consumer.Ack(msg)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Balance Monitor...")
}

func loadTriggers(ctx context.Context, dbPool *db.Pool, tenantID int64) ([]ActionTrigger, error) {
	query := `
		SELECT id, tenant_id, name, conditions, actions
		FROM action_trigger_presets
		WHERE tenant_id = $1 AND is_active = true
	`
	rows, err := dbPool.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var triggers []ActionTrigger
	for rows.Next() {
		var t ActionTrigger
		var condJSON, actJSON []byte
		if err := rows.Scan(&t.ID, &t.TenantID, &t.Name, &condJSON, &actJSON); err != nil {
			continue
		}
		_ = json.Unmarshal(condJSON, &t.Conditions)
		_ = json.Unmarshal(actJSON, &t.Actions)
		triggers = append(triggers, t)
	}
	return triggers, nil
}

func shouldTrigger(trigger ActionTrigger, event BalanceEvent) bool {
	// Simple threshold check
	if threshold, ok := trigger.Conditions["threshold"].(float64); ok {
		if event.Amount >= threshold {
			return true
		}
	}
	if event.Type == "charge" {
		if minBalance, ok := trigger.Conditions["min_balance"].(float64); ok {
			// In real implementation, fetch current balance from DB
			if event.Amount > minBalance {
				return true
			}
		}
	}
	return false
}

func executeTrigger(ctx context.Context, pulsarClient *pulsar.Client, trigger ActionTrigger, event BalanceEvent) {
	actionType, _ := trigger.Actions["type"].(string)
	log.Printf("Trigger %d fired for subscriber %d: action=%s", trigger.ID, event.SubscriberID, actionType)

	switch actionType {
	case "notification":
		// Send notification
		if pulsarClient != nil {
			payload, _ := json.Marshal(map[string]interface{}{
				"tenant_id":     event.TenantID,
				"to":            event.MSISDN,
				"template_type": "balance_alert",
				"variables": map[string]string{
					"msisdn": event.MSISDN,
					"amount": strconv.FormatFloat(event.Amount, 'f', 2, 64),
				},
			})
			go pulsarClient.PublishTo(ctx, "persistent://billing/notifications/notifications.email", payload, event.MSISDN)
		}
	case "adjust_balance":
		// Queue balance adjustment
		if pulsarClient != nil {
			payload, _ := json.Marshal(map[string]interface{}{
				"type":         "auto_adjust",
				"subscriber_id": event.SubscriberID,
				"msisdn":       event.MSISDN,
				"amount":       trigger.Actions["amount"],
			})
			go pulsarClient.PublishTo(ctx, "persistent://billing/commands/commands.balance.adjust", payload, event.MSISDN)
		}
	}
}
