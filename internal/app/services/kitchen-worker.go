package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"wheres-my-pizza/internal/adapter/postgres"
	"wheres-my-pizza/internal/adapter/rabbit"
	"wheres-my-pizza/internal/config"
	"wheres-my-pizza/internal/domain/types"
	"wheres-my-pizza/internal/services/kitchen"
	"wheres-my-pizza/pkg/logger"
	postgresclient "wheres-my-pizza/pkg/postgres"
)

var (
	ErrEmptyOrderTypes    = fmt.Errorf("order types cannot be empty")
	ErrDuplicateOrderType = fmt.Errorf("duplicate order type found")
	ErrInvalidOrderType   = errors.New("invalid order type. must be Comma-separated list of order types the worker can handle (e.g., dine_in,takeout)")

	ErrInvalidHeartbeatInterval = errors.New("heartbeat interval must be at least 5 seconds")
)

type KitchenWorker interface {
	Work(ctx context.Context, errCh chan<- error)
	Stop(ctx context.Context)
}

// Feature: Order Service
// The Kitchen Worker is a background service that simulates the kitchen staff. It consumes order
// messages from a queue, processes them, and updates their status in the database. It is the core
// processing engine of the restaurant. Multiple worker instances can run concurrently to handle
// high order volumes and can be specialized to process specific types of orders.
type KitchenService struct {
	postgresDB    *postgresclient.PostgreDB
	kitchenWorker KitchenWorker
	consumer      *rabbit.OrderConsumer
	producer      *rabbit.NotificationProducer

	cfg config.Config
	log logger.Logger
}

func NewKitchen(ctx context.Context, cfg config.Config, log logger.Logger) (*KitchenService, error) {
	// Validating worker name
	if err := validateWorkerName(cfg.Services.Kitchen.WorkerName); err != nil {
		log.Error(ctx, types.ActionValidationFailed, "failed to vailidate worker name", err)
		return nil, fmt.Errorf("failed to validate worker name: %w", err)
	}
	// Validating order-types to handle by worker.
	validOrderTypes, err := ValidateOrderTypes(cfg.Services.Kitchen.OrderTypes)
	if err != nil {
		log.Error(ctx, types.ActionValidationFailed, "failed to vailidate provided order types", err)
		return nil, fmt.Errorf("failed to vailidate provided order types: %w", err)
	}

	// validate heartbeat interval
	heartbeatDuration := time.Duration(cfg.Services.Kitchen.HeartbeatInterval) * time.Second
	if heartbeatDuration <= time.Second*5 {
		return nil, ErrInvalidHeartbeatInterval
	}

	// Postgres database connection
	db, err := postgresclient.New(ctx, cfg.Postgres)
	if err != nil {
		log.Error(ctx, types.ActionDBConnectionFailed, "failed to connect postgres", err)
		return nil, fmt.Errorf("failed to connect postgres: %v", err)
	}
	log.Info(ctx, types.ActionDBConnected, "connected to the database")

	// RabbitMQ connection
	// Initialize order consumer
	consumer, err := rabbit.NewOrderConsumer(ctx, cfg.RabbitMQ, cfg.Services.Kitchen.Prefetch, validOrderTypes, log)
	if err != nil {
		log.Error(ctx, types.ActionRabbitConnectionFailed, "failed to create order consumer", err)
		return nil, fmt.Errorf("failed to create order consumer: %w", err)
	}
	// Initialize notification producer
	producer, err := rabbit.NewProducerNotify(ctx, cfg.RabbitMQ, log)
	if err != nil {
		log.Error(ctx, types.ActionRabbitConnectionFailed, "failed to create notification producer", err)
		return nil, fmt.Errorf("failed to create notification producer: %w", err)
	}

	// Initialize repositories
	workerRepo := postgres.NewWorkerRepo(db.Pool)
	orderRepo := postgres.NewOrderRepo(db.Pool)

	// Initialize kitchen-worker service
	kitchenWorker := kitchen.NewWorker(workerRepo, orderRepo, consumer, producer, cfg.Services.Kitchen.WorkerName, validOrderTypes, heartbeatDuration, log)

	return &KitchenService{
		postgresDB:    db,
		kitchenWorker: kitchenWorker,
		consumer:      consumer,
		producer:      producer,

		cfg: cfg,
		log: log,
	}, nil
}

func (s *KitchenService) Start(ctx context.Context) error {
	defer func() {
		s.close(ctx)
		s.log.Info(ctx, types.ActionGracefulShutdown, "kitchen-worker service closed")
	}()

	errCh := make(chan error, 1)

	// kitchen worker starts to work in goroutine
	go s.kitchenWorker.Work(ctx, errCh)

	// Waiting signal
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)

	s.log.Info(ctx, types.ActionServiceStarted, "service started")

	for {
		select {
		case <-ctx.Done():
			s.log.Info(ctx, types.ActionGracefulShutdown, "context cancelled")
			return ctx.Err()
		case errRun := <-errCh:
			if errors.Is(errRun, kitchen.ErrWorkerStopped) {
				if err := s.reconnect(ctx, shutdownCh, s.cfg.Services.Kitchen.ReconnectAttempt, s.cfg.Services.Kitchen.ReconnectDelay); err != nil {
					return err
				}
				continue
			}
			return errRun
		case sig := <-shutdownCh:
			s.log.Info(ctx, types.ActionGracefulShutdown, "shutting down application", "signal", sig.String())
			return nil
		}
	}
}

// close stops worker and closes connections.
func (s *KitchenService) close(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	s.kitchenWorker.Stop(ctx)

	if err := s.consumer.Close(ctx); err != nil {
		s.log.Error(ctx, types.ActionGracefulShutdown, "failed to close rabbit connection", err)
	}

	if err := s.producer.Close(ctx); err != nil {
		s.log.Error(ctx, types.ActionGracefulShutdown, "failed to close rabbit connection", err)
	}

	s.postgresDB.Pool.Close()
}

func (s *KitchenService) reconnect(ctx context.Context, shutdownCh chan os.Signal, attempts int, delay time.Duration) error {
	var lastErr error

	const action = "kitchen-create-attempt"
	const failedAction = "kitchen-create-attempt-failed"

	for i := 1; i <= attempts; i++ {
		s.log.Info(ctx, action, fmt.Sprintf("attempt %d to recreate service", i))

		newSvc, err := NewKitchen(ctx, s.cfg, s.log)
		if err == nil {
			// Closing old service
			s.close(ctx)

			*s = *newSvc

			// Starting a new worker
			go s.kitchenWorker.Work(ctx, make(chan error, 1))
			return nil
		}

		lastErr = err
		s.log.Error(ctx, failedAction, "failed to recreate kitchen service", err, "attempt", i)

		select {
		case <-shutdownCh:
			s.log.Info(ctx, action, "reconnect was stopped externally")
			return fmt.Errorf("reconnect stopped externally")
		default:
			time.Sleep(delay)
		}
	}

	s.log.Warn(ctx, failedAction, "failed to recreate kitchen-worker", "total-attempts", attempts)
	return lastErr
}

// ValidateOrderTypes handles all validation cases for the --order-types flag
// Input examples: "", "dine_in", "dine_in,takeout", "dine_in,takeout,dine_in" (invalid)
func ValidateOrderTypes(input string) ([]string, error) {
	// Handle empty input (means all types)
	if input == "" {
		return types.AllOrderTypes, nil
	}

	// Split and clean input
	rawTypes := strings.Split(input, ",")
	orderTypes := make([]string, 0, len(rawTypes))
	seen := make(map[string]struct{})

	for _, rawType := range rawTypes {
		trimmed := strings.TrimSpace(rawType)
		if trimmed == "" {
			continue // skip empty entries
		}

		// Check for duplicates
		if _, exists := seen[trimmed]; exists {
			return nil, fmt.Errorf("%q: %w", trimmed, ErrDuplicateOrderType)
		}
		seen[trimmed] = struct{}{}

		// Validate against available types
		if !types.IsValidOrderType(trimmed) {
			return nil, fmt.Errorf("%q: %w (available: %v)", trimmed, ErrInvalidOrderType, types.AllOrderTypes)
		}

		orderTypes = append(orderTypes, trimmed)
	}

	// After cleaning, check if we have any types left
	if len(orderTypes) == 0 {
		return nil, ErrEmptyOrderTypes
	}

	return orderTypes, nil
}

func validateWorkerName(name string) error {
	// Check length (1-100 characters)
	if utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 100 {
		return errors.New("worker name length must be between 1-100 characters")
	}
	return nil
}
