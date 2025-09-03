package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"restaurant-system/internal/logger"
)

// MessageHandler defines the interface for processing messages
type MessageHandler func(ctx context.Context, body []byte) error

// Consumer handles message consumption from RabbitMQ
type Consumer struct {
	conn        *Connection
	logger      *logger.Logger
	queueName   string
	consumerTag string
	prefetch    int
}

// NewConsumer creates a new message consumer
func NewConsumer(conn *Connection, log *logger.Logger, queueName, consumerTag string, prefetch int) *Consumer {
	return &Consumer{
		conn:        conn,
		logger:      log,
		queueName:   queueName,
		consumerTag: consumerTag,
		prefetch:    prefetch,
	}
}

// StartConsuming starts consuming messages from the queue
func (c *Consumer) StartConsuming(ctx context.Context, handler MessageHandler) error {
	// Check if connection is alive
	if c.conn.IsClosed() {
		if err := c.conn.Reconnect(); err != nil {
			return fmt.Errorf("failed to reconnect: %w", err)
		}
	}

	// Set QoS for prefetch
	err := c.conn.Channel().Qos(
		c.prefetch, // prefetch count
		0,          // prefetch size
		false,      // global
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	// Start consuming messages
	msgs, err := c.conn.Channel().Consume(
		c.queueName,   // queue
		c.consumerTag, // consumer
		false,         // auto-ack (we'll ack manually)
		false,         // exclusive
		false,         // no-local
		false,         // no-wait
		nil,           // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	c.logger.Info("consumer_started", 
		fmt.Sprintf("Started consuming from queue %s", c.queueName),
		"", map[string]interface{}{
			"queue":     c.queueName,
			"consumer":  c.consumerTag,
			"prefetch":  c.prefetch,
		})

	// Process messages
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("consumer_stopped", "Consumer stopped by context", "", nil)
			return ctx.Err()
		case d, ok := <-msgs:
			if !ok {
				// Channel closed, try to reconnect
				c.logger.Error("consumer_channel_closed", "Message channel closed, attempting to reconnect", "", nil, nil)
				if err := c.conn.Reconnect(); err != nil {
					return fmt.Errorf("failed to reconnect after channel closed: %w", err)
				}
				// Restart consuming after reconnect
				return c.StartConsuming(ctx, handler)
			}

			c.processMessage(ctx, d, handler)
		}
	}
}

// processMessage handles a single message
func (c *Consumer) processMessage(ctx context.Context, delivery amqp091.Delivery, handler MessageHandler) {
	startTime := time.Now()
	
	c.logger.Debug("message_received", 
		"Processing message",
		"", map[string]interface{}{
			"queue":          c.queueName,
			"routing_key":    delivery.RoutingKey,
			"message_size":   len(delivery.Body),
			"delivery_tag":   delivery.DeliveryTag,
		})

	// Process the message with timeout
	processingCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err := handler(processingCtx, delivery.Body)
	
	duration := time.Since(startTime)
	
	if err != nil {
		c.logger.Error("message_processing_failed", 
			"Failed to process message",
			"", err, map[string]interface{}{
				"queue":        c.queueName,
				"routing_key":  delivery.RoutingKey,
				"duration_ms":  duration.Milliseconds(),
				"delivery_tag": delivery.DeliveryTag,
			})
		
		// Negative acknowledgment with requeue
		if nackErr := delivery.Nack(false, true); nackErr != nil {
			c.logger.Error("message_nack_failed", "Failed to nack message", "", nackErr, nil)
		}
	} else {
		c.logger.Debug("message_processed", 
			"Successfully processed message",
			"", map[string]interface{}{
				"queue":        c.queueName,
				"routing_key":  delivery.RoutingKey,
				"duration_ms":  duration.Milliseconds(),
				"delivery_tag": delivery.DeliveryTag,
			})
		
		// Positive acknowledgment
		if ackErr := delivery.Ack(false); ackErr != nil {
			c.logger.Error("message_ack_failed", "Failed to ack message", "", ackErr, nil)
		}
	}
}

// ParseMessage parses a JSON message into the provided struct
func ParseMessage(body []byte, v interface{}) error {
	return json.Unmarshal(body, v)
}

// Close stops consuming messages
func (c *Consumer) Close() error {
	if c.conn != nil && !c.conn.IsClosed() {
		// Cancel consumer
		err := c.conn.Channel().Cancel(c.consumerTag, false)
		if err != nil {
			c.logger.Error("consumer_cancel_failed", "Failed to cancel consumer", "", err, nil)
		}
		return c.conn.Close()
	}
	return nil
}
