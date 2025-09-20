package order

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"wheres-my-pizza/internal/config"
	"wheres-my-pizza/internal/domain/models"
	"wheres-my-pizza/internal/domain/types"
	"wheres-my-pizza/pkg/logger"
)

var ErrTooManyRequest = errors.New("too many requests")

const servicename = "order-service"

type Service struct {
	orderRepo OrderRepository
	writer    MessageBroker
	sem       Semaphore
	semWait   time.Duration

	cfg config.Config
	log logger.Logger
}

func NewService(cfg config.Config, repo OrderRepository, writer MessageBroker, sem Semaphore, semWait time.Duration, log logger.Logger) *Service {
	return &Service{
		orderRepo: repo,
		writer:    writer,
		sem:       sem,
		semWait:   time.Second,

		cfg: cfg,
		log: log,
	}
}

// CreateOrder creates new order
func (s *Service) CreateOrder(ctx context.Context, req *models.CreateOrder) (*models.OrderCreatedInfo, error) {
	s.log.Debug(
		ctx,
		types.ActionOrderReceived,
		"creating new order",
		"slots-available", s.sem.Available(),
		"slots-used", s.sem.Used(),
	)

	// Trying to take slot under s.semWait seconds if not returning error.
	if !s.sem.TryAcquire(s.semWait) {
		s.log.Error(ctx, types.ActionOrderProccessingFailed, "failed to proccess order", ErrTooManyRequest)
		return nil, ErrTooManyRequest
	}
	defer s.sem.Release()

	today := todayDate()
	number, err := s.orderRepo.GetAndIncrementSequence(ctx, today)
	if err != nil {
		s.log.Error(ctx, types.ActionDBQueryFailed, "failed to get next order sequence. generating random order_number", err)
		number = getRandomOrderNumber()
	}

	req.SetNumber(today, number)
	req.CalucalteTotalAmount()
	req.CalculatePriority()
	req.Status = types.StatusOrderReceived

	// Store order to database
	order, err := s.orderRepo.Create(ctx, req, servicename, "")
	if err != nil {
		s.log.Error(ctx, types.ActionDBTransactionFailed, "failed to create new order", err)
		return nil, fmt.Errorf("failed to create new order: %w", err)
	}

	// Send request info about publishing order with retry
	if err := retry(s.cfg.RabbitMQ.ReconnectAttempt, s.cfg.RabbitMQ.ReconnectDelay, func() error {
		return s.writer.PublishCreateOrder(ctx, req)
	}); err != nil {
		s.log.Error(ctx, types.ActionDBQueryFailed, "order stored to database, but not published", err)
		return nil, fmt.Errorf("failed to publish order: %w", err)
	}

	return &models.OrderCreatedInfo{
		Number:      order.Number,
		Status:      order.Status,
		TotalAmount: order.TotalAmount,
	}, nil
}

// Generate a random number between 10000 and 99999 (inclusive)
func getRandomOrderNumber() int {
	return rand.Intn(90000) + 10000
}

func todayDate() string {
	return time.Now().UTC().Format("20060102") // Go's reference time format
}

// retry funciton to attempt request multiple times
func retry(attempts int, delay time.Duration, fn func() error) error {
	var err error

	for i := range attempts {
		err = fn()
		if err == nil {
			return nil
		}

		if i < attempts-1 {
			time.Sleep(delay)
		}
	}

	return err
}
