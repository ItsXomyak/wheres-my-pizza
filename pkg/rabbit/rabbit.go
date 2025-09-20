package rabbit

import (
	"context"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"wheres-my-pizza/internal/domain/types"
	"wheres-my-pizza/pkg/logger"
)

type RabbitMQ struct {
	Conn      *amqp.Connection
	Channel   *amqp.Channel
	closeChan chan *amqp.Error
	isClosed  bool

	log logger.Logger
}

type Config struct {
	Host     string `env:"RABBITMQ_HOST" default:"localhost"`
	Port     string `env:"RABBITMQ_PORT" default:"5672"`
	User     string `env:"RABBITMQ_USER" default:"guest"`
	Password string `env:"RABBITMQ_PASSWORD" default:"guest"`
}

func (c Config) GetDSN() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%s/",
		c.User,
		c.Password,
		c.Host,
		c.Port,
	)
}

func New(ctx context.Context, config Config, log logger.Logger) (*RabbitMQ, error) {
	conn, err := amqp.DialConfig(config.GetDSN(), amqp.Config{
		Heartbeat: 10 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	// Create a channel
	channel, err := conn.Channel()
	if err != nil {
		conn.Close() // Close connection if channel creation fails
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	// Create close notification channel
	closeChan := make(chan *amqp.Error, 1)
	conn.NotifyClose(closeChan)

	// Verify the connection is alive
	select {
	case closeErr := <-closeChan:
		if closeErr != nil {
			return nil, fmt.Errorf("rabbitmq connection is closed: %v", closeErr)
		}
		return nil, fmt.Errorf("rabbitmq connection is closed")
	default:
		// Connection is good
	}

	log.Info(ctx, types.ActionRabbitMQConnected, "connected to rabbitMQ")

	r := &RabbitMQ{
		Conn:      conn,
		Channel:   channel,
		closeChan: closeChan,
		isClosed:  false,
		log:       log,
	}

	// Start monitoring connection in background
	go r.monitorConnection()

	return r, nil
}

// monitorConnection monitors the connection status
func (r *RabbitMQ) monitorConnection() {
	closeErr := <-r.closeChan
	r.isClosed = true
	if closeErr != nil {
		r.log.Error(context.Background(), types.ActionRabbitConnectionClosed, "RabbitMQ connection closed with error", closeErr)
	} else {
		r.log.Debug(context.Background(), types.ActionRabbitConnectionClosed, "RabbitMQ connection closed gracefully")
	}
}

// IsConnectionClosed checks if the connection is closed
func (r *RabbitMQ) IsConnectionClosed() bool {
	return r.isClosed || r.Conn.IsClosed()
}

// Close closes rabbit connection
func (r *RabbitMQ) Close(ctx context.Context) error {
	return r.CloseWithContext(ctx)
}

// CloseWithContext - closes RabbitMQ connection using context for cancellation
func (r *RabbitMQ) CloseWithContext(ctx context.Context) error {
	op := "rabbitMQ:CloseWithContext"

	r.log.Debug(ctx, types.ActionRabbitConnectionClosing, "closing channel", "op", op)

	// If connection is already closed, don't try to close channel/connection
	if r.IsConnectionClosed() {
		return nil
	}

	// Close channel with context
	done := make(chan error, 1)
	go func() {
		if r.Channel != nil {
			done <- r.Channel.Close()
		} else {
			done <- nil
		}
	}()

	select {
	case err := <-done:
		if err != nil {
			r.log.Error(ctx, types.ActionRabbitConnectionClosing, "error closing channel", err, "op", op)
		}
	case <-ctx.Done():
		r.log.Debug(ctx, types.ActionRabbitConnectionClosing, "context cancelled, forcing channel close", "op", op)
	}

	r.log.Debug(ctx, types.ActionRabbitConnectionClosing, "closing RabbitMQ connection", "op", op)

	go func() {
		if r.Conn != nil {
			done <- r.Conn.Close()
		} else {
			done <- nil
		}
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("failed to close connection: %w", err)
		}
	case <-ctx.Done():
		r.log.Debug(ctx, types.ActionRabbitConnectionClosing, "context cancelled, forcing connection close", "op", op)
	}

	return nil
}
