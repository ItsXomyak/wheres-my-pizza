package rabbit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"wheres-my-pizza/internal/config"
	"wheres-my-pizza/internal/domain/models"
	"wheres-my-pizza/internal/domain/types"
	"wheres-my-pizza/pkg/logger"
	"wheres-my-pizza/pkg/rabbit"
)

var ErrEmptyExchangeName = errors.New("empty exchange name")

// NotificationProducer
type NotificationProducer struct {
	client *rabbit.RabbitMQ

	exchangeName string

	cfg config.RabbitMQ
	log logger.Logger
}

func NewProducerNotify(ctx context.Context, cfg config.RabbitMQ, log logger.Logger) (*NotificationProducer, error) {
	if len(cfg.NotificationsExchange) == 0 {
		return nil, ErrEmptyExchangeName
	}

	// RabbitMQ connection
	client, err := rabbit.New(ctx, cfg.Conn, log)
	if err != nil {
		log.Error(ctx, types.ActionRabbitConnectionFailed, "failed to connect RabbitMQ", err)
		return nil, err
	}

	// declaring notification exchange
	if err := client.Channel.ExchangeDeclare(
		cfg.NotificationsExchange,
		"fanout",
		true, false, false, false, nil,
	); err != nil {
		return nil, fmt.Errorf("failed to declare exchange %s: %w", cfg.NotificationsExchange, err)
	}

	return &NotificationProducer{
		client:       client,
		exchangeName: cfg.NotificationsExchange,

		cfg: cfg,
		log: log,
	}, nil
}

// StatusUpdate publishes event about status change.
func (p *NotificationProducer) StatusUpdate(ctx context.Context, req *models.StatusUpdate) error {
	// Cheking if connected
	if p.client.IsConnectionClosed() {
		p.log.Debug(ctx, types.ActionRabbitReconnect, "trying to recconect to RabbitMQ")
		if err := p.reconnect(ctx); err != nil {
			return err
		}
	}

	// Marshal the struct to JSON
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal StatusUpdate: %w", err)
	}

	// Prepare the message
	msg := amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		DeliveryMode: amqp.Persistent, // Persistent message (2). Means rabbitmq will store message in the disc.
		Timestamp:    time.Now(),
	}

	// Publish to the exchange with empty routing key (fanout ignores it)
	if err := p.client.Channel.PublishWithContext(
		ctx,
		p.exchangeName,
		"",    // routing key is ignored for fanout
		false, // mandatory
		false, // immediate
		msg,
	); err != nil {
		return fmt.Errorf("failed to publish StatusUpdate: %w", err)
	}

	return nil
}

func (r *NotificationProducer) reconnect(ctx context.Context) error {
	fn := func() error {
		conn, err := rabbit.New(ctx, r.cfg.Conn, r.log)
		if err != nil {
			return err
		}
		r.client = conn

		return nil
	}

	if err := retry(ctx, r.cfg.ReconnectAttempt, r.cfg.ReconnectDelay, fn); err != nil {
		return fmt.Errorf("failed to recconect rabbitMQ: %w", err)
	}

	return nil
}

func (r *NotificationProducer) Close(ctx context.Context) error {
	if r.client == nil {
		return nil
	}

	return r.client.Close(ctx)
}
