// Email Consumer — consumes notifications.email and sends via SMTP
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Grimid86/cgrates-ui/backend/pkg/db"
	"github.com/Grimid86/cgrates-ui/backend/pkg/pulsar"
)

// EmailEvent represents an email notification from Pulsar
type EmailEvent struct {
	TenantID    int64             `json:"tenant_id"`
	To          string            `json:"to"`
	TemplateType string           `json:"template_type"`
	Locale      string            `json:"locale"`
	Variables   map[string]string `json:"variables"`
	Timestamp   time.Time         `json:"timestamp"`
}

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

func loadSMTPConfig() SMTPConfig {
	return SMTPConfig{
		Host:     getEnv("SMTP_HOST", "smtp.example.com"),
		Port:     getEnv("SMTP_PORT", "587"),
		Username: getEnv("SMTP_USER", ""),
		Password: getEnv("SMTP_PASSWORD", ""),
		From:     getEnv("SMTP_FROM", "noreply@billing.com"),
	}
}

func main() {
	log.Println("Starting Email Consumer...")

	dbPool, err := db.New(db.Config{
		DSN: os.Getenv("DB_DSN"),
	})
	if err != nil {
		log.Printf("DB connection failed (non-critical for email): %v", err)
	} else {
		defer dbPool.Close()
	}

	smtpCfg := loadSMTPConfig()

	pulsarClient, err := pulsar.New(pulsar.Config{
		URL:    os.Getenv("PULSAR_URL"),
		Tenant: os.Getenv("PULSAR_TENANT"),
	})
	if err != nil {
		log.Fatalf("Pulsar connection failed: %v", err)
	}
	defer pulsarClient.Close()

	consumer, err := pulsarClient.CreateConsumer(
		"persistent://billing/notifications/notifications.email",
		"email-consumer",
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

			var event EmailEvent
			if err := json.Unmarshal(msg.Payload(), &event); err != nil {
				log.Printf("Unmarshal error: %v", err)
				consumer.Ack(msg)
				continue
			}

			if err := sendEmail(ctx, dbPool, smtpCfg, event); err != nil {
				log.Printf("Send email error: %v", err)
				// Retry with backoff or send to DLQ
				continue
			}

			consumer.Ack(msg)
			log.Printf("Email sent to %s", event.To)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Email Consumer...")
}

func sendEmail(ctx context.Context, dbPool *db.Pool, cfg SMTPConfig, event EmailEvent) error {
	// Load template from DB
	subject, body, err := loadTemplate(ctx, dbPool, event.TenantID, event.TemplateType, event.Locale)
	if err != nil {
		// Use default template
		subject = "Notification"
		body = fmt.Sprintf("<html><body><h1>%s</h1></body></html>", event.TemplateType)
	}

	// Render template with variables
	for key, value := range event.Variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		subject = replaceAll(subject, placeholder, value)
		body = replaceAll(body, placeholder, value)
	}

	// Send via SMTP if configured
	if cfg.Host == "smtp.example.com" || cfg.Username == "" {
		log.Printf("[MOCK EMAIL] To: %s, Subject: %s", event.To, subject)
		return nil
	}

	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n%s\r\n%s", event.To, subject, mime, body))

	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)

	return smtp.SendMail(addr, auth, cfg.From, []string{event.To}, msg)
}

func loadTemplate(ctx context.Context, dbPool *db.Pool, tenantID int64, templateType, locale string) (string, string, error) {
	if dbPool == nil {
		return "", "", fmt.Errorf("db not available")
	}

	var subject, body string
	query := `
		SELECT e.subject_template, e.body_html_template
		FROM email_template e
		JOIN language l ON e.language_id = l.id
		WHERE e.tenant_id = $1 AND e.template_type = $2 AND l.code = $3
	`
	err := dbPool.QueryRow(ctx, query, tenantID, templateType, locale).Scan(&subject, &body)
	return subject, body, err
}

func replaceAll(s, old, new string) string {
	// Simple string replacement - in production use proper template engine
	result := ""
	start := 0
	for {
		i := 0
		for i < len(s)-len(old)+1 {
			if s[i:i+len(old)] == old {
				result += s[start:i] + new
				start = i + len(old)
				i = start
				break
			}
			i++
		}
		if i >= len(s)-len(old)+1 {
			break
		}
	}
	result += s[start:]
	return result
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
