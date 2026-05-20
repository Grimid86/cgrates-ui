// Package pulsar provides Apache Pulsar producer/consumer utilities
// for async event publishing and consumption.
package pulsar

import (
	"context"
	"fmt"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
)

// Client wraps Pulsar client
type Client struct {
	client   pulsar.Client
	producer pulsar.Producer
	tenant   string
}

// Config for Pulsar connection
type Config struct {
	URL      string
	Token    string
	Tenant   string
	Topic    string // default topic for producer
}

// New creates a new Pulsar client with producer
func New(cfg Config) (*Client, error) {
	clientOpts := pulsar.ClientOptions{
		URL:               cfg.URL,
		OperationTimeout:  30 * time.Second,
		ConnectionTimeout: 30 * time.Second,
	}

	if cfg.Token != "" {
		clientOpts.Authentication = pulsar.NewAuthenticationToken(cfg.Token)
	}

	client, err := pulsar.NewClient(clientOpts)
	if err != nil {
		return nil, fmt.Errorf("create pulsar client: %w", err)
	}

	var producer pulsar.Producer
	if cfg.Topic != "" {
		producer, err = client.CreateProducer(pulsar.ProducerOptions{
			Topic:                   cfg.Topic,
			BatchingMaxPublishDelay: 10 * time.Millisecond,
		})
		if err != nil {
			client.Close()
			return nil, fmt.Errorf("create pulsar producer: %w", err)
		}
	}

	return &Client{
		client:   client,
		producer: producer,
		tenant:   cfg.Tenant,
	}, nil
}

// Close closes the Pulsar client and producer
func (c *Client) Close() {
	if c.producer != nil {
		c.producer.Close()
	}
	c.client.Close()
}

// Publish sends a message to the default topic
func (c *Client) Publish(ctx context.Context, payload []byte, key string) error {
	if c.producer == nil {
		return fmt.Errorf("no producer configured")
	}

	msg := &pulsar.ProducerMessage{
		Payload: payload,
		Key:     key,
		Properties: map[string]string{
			"tenant":    c.tenant,
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	_, err := c.producer.Send(ctx, msg)
	return err
}

// PublishTo sends a message to a specific topic
func (c *Client) PublishTo(ctx context.Context, topic string, payload []byte, key string) error {
	producer, err := c.client.CreateProducer(pulsar.ProducerOptions{
		Topic: topic,
	})
	if err != nil {
		return fmt.Errorf("create producer for topic %s: %w", topic, err)
	}
	defer producer.Close()

	msg := &pulsar.ProducerMessage{
		Payload: payload,
		Key:     key,
	}

	_, err = producer.Send(ctx, msg)
	return err
}

// CreateConsumer creates a consumer for a topic
func (c *Client) CreateConsumer(topic, subscription string) (pulsar.Consumer, error) {
	return c.client.Subscribe(pulsar.ConsumerOptions{
		Topic:            topic,
		SubscriptionName: subscription,
		Type:             pulsar.Shared,
	})
}

// Event represents a standard event structure for Pulsar messages
type Event struct {
	Type      string                 `json:"type"`
	Portal    string                 `json:"portal"`
	TenantID  int64                  `json:"tenant_id"`
	UserID    int64                  `json:"user_id,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}
