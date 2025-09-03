package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"restaurant-system/internal/logger"
)

// Publisher handles message publishing to RabbitMQ
type Publisher struct {
	conn   *Connection
	logger *logger.Logger
}

// NewPublisher creates a new message publisher
func NewPublisher(conn *Connection, log *logger.Logger) *Publisher {
	return &Publisher{
		conn:   conn,
		logger: log,
	}
}

// PublishOrder publishes an order message to the orders topic exchange
func (p *Publisher) PublishOrder(ctx context.Context, orderMsg interface{}, routingKey string, priority uint8) error {
	return p.publishMessage(ctx, "orders_topic", routingKey, orderMsg, priority, true)
}

// PublishNotification publishes a status update message to the notifications fanout exchange
func (p *Publisher) PublishNotification(ctx context.Context, notificationMsg interface{}) error {
	return p.publishMessage(ctx, "notifications_fanout", "", notificationMsg, 0, false)
}

// publishMessage is the generic message publishing function
func (p *Publisher) publishMessage(ctx context.Context, exchange, routingKey string, message interface{}, priority uint8, persistent bool) error {
	// Check if connection is alive
	if p.conn.IsClosed() {
		if err := p.conn.Reconnect(); err != nil {
			return fmt.Errorf("failed to reconnect: %w", err)
		}
	}

	// Serialize message to JSON
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Prepare publishing options
	deliveryMode := uint8(1) // Non-persistent by default
	if persistent {
		deliveryMode = 2 // Persistent
	}

	publishing := amqp091.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: deliveryMode,
		Timestamp:    time.Now(),
	}

	// Set priority if specified
	if priority > 0 {
		publishing.Priority = priority
	}

	// Publish with timeout
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err = p.conn.Channel().PublishWithContext(
		ctx,
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		publishing,
	)

	if err != nil {
		p.logger.Error("message_publish_failed", 
			fmt.Sprintf("Failed to publish message to exchange %s", exchange),
			"", err, map[string]interface{}{
				"exchange":    exchange,
				"routing_key": routingKey,
			})
		return fmt.Errorf("failed to publish message: %w", err)
	}

	p.logger.Debug("message_published", 
		fmt.Sprintf("Published message to exchange %s", exchange),
		"", map[string]interface{}{
			"exchange":    exchange,
			"routing_key": routingKey,
			"message_size": len(body),
		})

	return nil
}

// Close closes the publisher
func (p *Publisher) Close() error {
	return p.conn.Close()
}
