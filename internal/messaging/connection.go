package messaging

import (
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"restaurant-system/internal/config"
	"restaurant-system/internal/logger"
)

// Connection wraps RabbitMQ connection with reconnection logic
type Connection struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
	config  *config.Config
	logger  *logger.Logger
	url     string
}

// New creates a new RabbitMQ connection
func New(cfg *config.Config, log *logger.Logger) (*Connection, error) {
	url := cfg.RabbitMQURL()
	
	conn := &Connection{
		config: cfg,
		logger: log,
		url:    url,
	}
	
	err := conn.connect()
	if err != nil {
		return nil, fmt.Errorf("failed to establish initial connection: %w", err)
	}
	
	return conn, nil
}

// connect establishes connection to RabbitMQ with retry logic
func (c *Connection) connect() error {
	maxRetries := 5
	var err error
	
	for i := 0; i < maxRetries; i++ {
		c.conn, err = amqp091.Dial(c.url)
		if err == nil {
			c.channel, err = c.conn.Channel()
			if err == nil {
				// Set up exchanges and queues
				if setupErr := c.setupTopology(); setupErr != nil {
					c.logger.Error("rabbitmq_setup_failed", "Failed to set up topology", "startup", setupErr, nil)
					c.close()
					err = setupErr
				} else {
					return nil
				}
			} else {
				c.conn.Close()
			}
		}
		
		if i < maxRetries-1 {
			waitTime := time.Duration(i+1) * 2 * time.Second
			c.logger.Error("rabbitmq_connection_failed", 
				fmt.Sprintf("Failed to connect to RabbitMQ, retrying in %v", waitTime),
				"startup", err, nil)
			time.Sleep(waitTime)
		}
	}
	
	return fmt.Errorf("failed to connect to RabbitMQ after %d attempts: %w", maxRetries, err)
}

// setupTopology creates exchanges and queues
func (c *Connection) setupTopology() error {
	// Declare orders topic exchange
	err := c.channel.ExchangeDeclare(
		"orders_topic",  // name
		"topic",         // type
		true,            // durable
		false,           // auto-deleted
		false,           // internal
		false,           // no-wait
		nil,             // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare orders_topic exchange: %w", err)
	}

	// Declare notifications fanout exchange
	err = c.channel.ExchangeDeclare(
		"notifications_fanout", // name
		"fanout",              // type
		true,                  // durable
		false,                 // auto-deleted
		false,                 // internal
		false,                 // no-wait
		nil,                   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare notifications_fanout exchange: %w", err)
	}

	// Declare kitchen queues
	queues := []string{
		"kitchen_queue",
		"kitchen_dine_in_queue",
		"kitchen_takeout_queue",
		"kitchen_delivery_queue",
	}

	for _, queueName := range queues {
		_, err = c.channel.QueueDeclare(
			queueName, // name
			true,      // durable
			false,     // delete when unused
			false,     // exclusive
			false,     // no-wait
			amqp091.Table{
				"x-message-ttl": 300000, // 5 minutes TTL
			},
		)
		if err != nil {
			return fmt.Errorf("failed to declare queue %s: %w", queueName, err)
		}
	}

	// Bind kitchen queues to orders_topic exchange
	bindings := []struct {
		queue      string
		routingKey string
	}{
		{"kitchen_queue", "kitchen.*"},
		{"kitchen_dine_in_queue", "kitchen.dine_in.*"},
		{"kitchen_takeout_queue", "kitchen.takeout.*"},
		{"kitchen_delivery_queue", "kitchen.delivery.*"},
	}

	for _, binding := range bindings {
		err = c.channel.QueueBind(
			binding.queue,      // queue name
			binding.routingKey, // routing key
			"orders_topic",     // exchange
			false,              // no-wait
			nil,                // arguments
		)
		if err != nil {
			return fmt.Errorf("failed to bind queue %s with routing key %s: %w", binding.queue, binding.routingKey, err)
		}
	}

	// Declare notifications queue and bind to fanout exchange
	_, err = c.channel.QueueDeclare(
		"notifications_queue", // name
		true,                  // durable
		false,                 // delete when unused
		false,                 // exclusive
		false,                 // no-wait
		nil,                   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare notifications queue: %w", err)
	}

	err = c.channel.QueueBind(
		"notifications_queue",  // queue name
		"",                     // routing key (ignored for fanout)
		"notifications_fanout", // exchange
		false,                  // no-wait
		nil,                    // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to bind notifications queue: %w", err)
	}

	return nil
}

// Channel returns the current channel
func (c *Connection) Channel() *amqp091.Channel {
	return c.channel
}

// Close closes the connection
func (c *Connection) Close() error {
	return c.close()
}

// close internal close method
func (c *Connection) close() error {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// IsClosed checks if the connection is closed
func (c *Connection) IsClosed() bool {
	return c.conn == nil || c.conn.IsClosed()
}

// Reconnect attempts to reconnect to RabbitMQ
func (c *Connection) Reconnect() error {
	c.close()
	return c.connect()
}
