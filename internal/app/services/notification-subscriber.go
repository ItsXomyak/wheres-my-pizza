package services

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"wheres-my-pizza/internal/adapter/rabbit"
	"wheres-my-pizza/internal/config"
	"wheres-my-pizza/internal/domain/types"
	"wheres-my-pizza/internal/services/notification"
	"wheres-my-pizza/pkg/logger"
	pkg "wheres-my-pizza/pkg/rabbit"
)

// ## Feature: Notification Service
// The Notification Service is a simple subscriber that demonstrates the fanout
// capabilities of the messaging system. It listens for all order status updates
// published by the Kitchen Workers and displays them. In a real-world scenario,
// this service could be extended to send push notifications, emails, or SMS
// messages to customers.
type NotificationSubsriber struct {
	service Service

	cfg config.Config
	log logger.Logger
}

type Service interface {
	Notify(ctx context.Context, errCh chan error)
	Close() error
}

func NewNotificationSubscriber(ctx context.Context, cfg config.Config, log logger.Logger) (*NotificationSubsriber, error) {
	client, err := pkg.New(ctx, cfg.RabbitMQ.Conn, log)
	if err != nil {
		log.Error(ctx, "rabbit_connect", "failed to connect rabbitmq", err)
		return nil, fmt.Errorf("failed to connect rabbitmq: %v", err)
	}

	reader := rabbit.NewNotificationSubscriber(client, cfg.RabbitMQ, log)
	notifier := notification.NewNotifyPrinter(log)
	service := notification.NewService(reader, notifier, log)

	return &NotificationSubsriber{
		service: service,
		cfg:     cfg,
		log:     log,
	}, nil
}

func (s *NotificationSubsriber) Start(ctx context.Context) error {
	defer func() {
		s.close(ctx)
		s.log.Info(ctx, types.ActionGracefulShutdown, "notification service closed")
	}()

	errCh := make(chan error, 1)
	go s.service.Notify(ctx, errCh)

	// Waiting signal
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)

	s.log.Info(ctx, types.ActionServiceStarted, "service started")

	select {
	case errRun := <-errCh:
		return errRun
	case sig := <-shutdownCh:
		s.log.Info(ctx, types.ActionGracefulShutdown, "shuting down application", "signal", sig.String())
		return nil
	}
}

func (s *NotificationSubsriber) close(ctx context.Context) {
	if err := s.service.Close(); err != nil {
		s.log.Error(ctx, types.ActionGracefulShutdown, "failed to close notification service", err)
	}
}
