package rabbit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"

	"wheres-my-pizza/internal/config"
	"wheres-my-pizza/internal/domain/models"
	"wheres-my-pizza/internal/domain/types"
	"wheres-my-pizza/pkg/logger"
	"wheres-my-pizza/pkg/rabbit"
)

type OrderProducer struct {
	client *rabbit.RabbitMQ

	exchangeOrder string

	cfg config.RabbitMQ
	log logger.Logger
}

func NewOrderProducer(ctx context.Context, cfg config.RabbitMQ, log logger.Logger) (*OrderProducer, error) {
	// RabbitMQ connection
	client, err := rabbit.New(ctx, cfg.Conn, log)
	if err != nil {
		log.Error(ctx, types.ActionRabbitConnectionFailed, "failed to connect RabbitMQ", err)
		return nil, err
	}

	// Creating exchange.
	if err := client.Channel.ExchangeDeclare(
		cfg.OrderExchange,
		"topic", // exchange type of 'topic'
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// creating all possible queues and binding them to order exchange.
	if err := InitQueuesForOrderTypes(client, cfg.OrderExchange, types.AllOrderTypes); err != nil {
		return nil, fmt.Errorf("failed to init order queues: %w", err)
	}

	return &OrderProducer{
		client:        client,
		exchangeOrder: cfg.OrderExchange,
		cfg:           cfg,
		log:           log,
	}, nil
}

// PublishCreateOrder publishes an order message to the orders_topic exchange
func (r *OrderProducer) PublishCreateOrder(ctx context.Context, order *models.CreateOrder) error {
	if order == nil {
		return errors.New("nil order")
	}

	if r.client.IsConnectionClosed() {
		r.log.Debug(ctx, types.ActionRabbitReconnect, "trying to recconect to RabbitMQ")
		if err := r.reconnect(ctx); err != nil {
			return err
		}
	}

	// Marshal order to JSON
	body, err := json.Marshal(FromInternalToPublishOrder(ctx, order))
	if err != nil {
		r.log.Error(ctx, types.ActionValidationFailed, "failed to marshal", err)
		return fmt.Errorf("failed to marshal order: %w", err)
	}

	routingKey := createOrderPublishedKey(order)

	// Create the message with persistent delivery mode
	msg := amqp091.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp091.Persistent, // Persistent message (2). Means rabbitmq will store message in the disc.
		Priority:     uint8(order.Priority),
		Timestamp:    time.Now(),
		Body:         body,
	}

	// Publish to the orders_topic exchange
	err = r.client.Channel.PublishWithContext(
		ctx,
		r.exchangeOrder, // exchange name
		routingKey,
		true,  // mandatory
		false, // immediate
		msg,
	)
	if err != nil {
		r.log.Error(ctx, types.ActionRabbitMQPublishFailed, "failed to publish order", err)
		return fmt.Errorf("failed to publish order: %w", err)
	}

	return nil
}

func (r *OrderProducer) reconnect(ctx context.Context) error {
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

func (r *OrderProducer) Close(ctx context.Context) error {
	if r.client == nil {
		return nil
	}

	return r.client.Close(ctx)
}
