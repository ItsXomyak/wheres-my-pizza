package kitchen

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"wheres-my-pizza/internal/domain/models"
	"wheres-my-pizza/internal/domain/types"
	"wheres-my-pizza/pkg/logger"
	"wheres-my-pizza/pkg/utils"
)

var (
	ErrWorkerStopped = errors.New("worker stopped")
	ErrNilOrder      = errors.New("nil order")
)

type (
	KitchenWorker struct {
		workerRepo WorkerRepository
		orderRepo  OrderRepository
		consumer   Consumer
		producer   Producer

		isWorking bool
		worker    *worker

		mu           sync.Mutex
		cancel       func()
		activeOrders sync.WaitGroup // activeOrders for monitor processing orders
		stopping     chan struct{}  // stopping channel to stop signal for proccessing orders

		log logger.Logger
	}

	worker struct {
		name       string
		orderTypes []string // Comma-separated list of order types the worker can handle (e.g., `dine_in,takeout`). If omitted, handles all.
		heartbeat  time.Duration
	}
)

// NewWorker creates new instance of kitchen-worker service
func NewWorker(
	workerRepo WorkerRepository,
	orderRepo OrderRepository,
	consumer Consumer,
	producer Producer,
	workerName string,
	orderTypes []string,
	heartbeat time.Duration,
	log logger.Logger,
) *KitchenWorker {
	return &KitchenWorker{
		workerRepo: workerRepo,
		orderRepo:  orderRepo,
		consumer:   consumer,
		producer:   producer,
		mu:         sync.Mutex{},

		worker: &worker{
			name:       workerName,
			orderTypes: orderTypes,
			heartbeat:  heartbeat,
		},

		activeOrders: sync.WaitGroup{},
		stopping:     make(chan struct{}),

		log: log,
	}
}

// Work starts consuming orders and proccesses them
func (s *KitchenWorker) Work(ctx context.Context, errCh chan<- error) {
	defer func() {
		// stop the worker from consuming and updating database.
		s.Stop(ctx)
		errCh <- ErrWorkerStopped
	}()

	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	// marking kitchen-worker as 'online'
	if err := s.markOnline(ctx); err != nil {
		s.log.Error(ctx, types.ActionWorkerRegistrationFailed, "failed to mark online worker", err, "worker-name", s.worker.name)
		errCh <- fmt.Errorf("failed to mark online: %w", err)
		return
	}

	wg := sync.WaitGroup{}
	for _, ot := range s.worker.orderTypes {
		wg.Add(1)
		go func(ot string) {
			defer func() {
				wg.Done()
				s.log.Info(ctx, "kitchen_worker_stop_consume", "stopped consuming orders", "order-type", ot)
			}()

			// Start consuming
			err := s.consumer.Consume(ctx, ot, s.processOrderWrapper)
			if err != nil {
				select {
				case errCh <- fmt.Errorf("failed to start consuming: %w", err):
				default:
					s.log.Error(ctx, "error_channel_full", "failed to send error to channel", err)
				}
			}
		}(ot)
	}

	go func() {
		s.heartbeatLoop(ctx, s.worker.heartbeat)
	}()

	wg.Wait()
}

// Stop worker from work
func (s *KitchenWorker) Stop(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isWorking {
		return
	}

	// stop proccessing new orders
	close(s.stopping)
	s.log.Debug(ctx, types.ActionWorkerStop, "waiting for active orders to finish", "worker-name", s.worker.name)
	// waiting for active proccesses to finish
	s.activeOrders.Wait()

	// Cancel context
	if s.cancel != nil {
		s.cancel()
	}

	// Mark worker offline
	if err := s.workerRepo.MarkOffline(ctx, s.worker.name); err != nil {
		s.log.Error(ctx, types.ActionWorkerStop, "failed to mark worker offline", err, "worker-name", s.worker.name)
		return
	}
	s.isWorking = false

	s.log.Info(ctx, "worker_stop", "stopping worker", "worker-name", s.worker.name)
}

func (s *KitchenWorker) processOrderWrapper(ctx context.Context, req *models.CreateOrder) error {
	// Check if order proccessing stoppped.
	select {
	case <-s.stopping:
		s.log.Info(ctx, types.ActionWorkerStop, "rejecting new order due to worker stopping", "order-number", req.Number)
		return errors.New("worker is stopping, cannot process new orders")
	default:
	}

	s.activeOrders.Add(1)
	defer s.activeOrders.Done()

	return s.proccessOrder(ctx, req)
}

// proccessOrder processes created order
func (s *KitchenWorker) proccessOrder(ctx context.Context, req *models.CreateOrder) error {
	if req == nil {
		s.log.Error(ctx, types.ActionValidationFailed, "nil order to proccess", ErrNilOrder, "worker-name", s.worker.name)
		return ErrNilOrder
	}

	cookingTime := types.GetSimulateCookingDuration(req.Type) // Simulated time

	s.log.Debug(
		ctx,
		types.ActionOrderProcessingStarted,
		"kitchen worker started proccessing order",
		"worker-name", s.worker.name,
		"order-number", req.Number,
		"cooking-time", utils.PrettyDuration(cookingTime))

	// Set status cooking
	oldStatus, err := s.orderRepo.SetStatus(ctx, req.Number, s.worker.name, types.StatusOrderCooking, "")
	if err != nil {
		s.log.Error(ctx, types.ActionMessageProcessingFailed, "failed to set cooking status for order", err, "worker-name", s.worker.name)
		return fmt.Errorf("failed to set cooking status for order : %w", err)
	}

	timestamp := time.Now()
	completion := timestamp.Add(cookingTime)

	// RequestID
	var requestID string
	if reqID, ok := ctx.Value(models.GetRequestIDKey()).(string); ok {
		requestID = reqID
	}

	// Publish status update message
	if err := s.producer.StatusUpdate(ctx, &models.StatusUpdate{
		OrderNumber: req.Number,
		OldStatus:   oldStatus,
		NewStatus:   types.StatusOrderCooking,
		ChangedBy:   s.worker.name,
		Timestamp:   timestamp,
		Completion:  completion,
		RequestID:   requestID,
	}); err != nil {
		s.log.Error(ctx, types.ActionRabbitMQPublishFailed, "failed to publish status update", err)
		s.log.Warn(ctx, types.ActionMessageProcessingFailed, "order status changed to cooking, but could not increment number of proccessed order for worker in the database", "worker-name", s.worker.name)
	}

	// Simulating working process with context cancellation support
	select {
	case <-time.After(cookingTime):
		// cooked
	case <-ctx.Done():
		s.log.Warn(ctx, types.ActionMessageProcessingFailed, "order processing interrupted but completing", "order-number", req.Number, "context-error", ctx.Err())
	}

	// Set status ready
	oldStatus, err = s.orderRepo.SetStatus(ctx, req.Number, s.worker.name, types.StatusOrderReady, "")
	if err != nil {
		s.log.Error(ctx, types.ActionMessageProcessingFailed, "failed to set ready status for order", err, "worker-name", s.worker.name)
		return fmt.Errorf("failed to set ready status for order: %w", err)
	}

	// Publish status update message
	timestamp = time.Now()
	if err := s.producer.StatusUpdate(ctx, &models.StatusUpdate{
		OrderNumber: req.Number,
		OldStatus:   oldStatus,
		NewStatus:   types.StatusOrderReady,
		ChangedBy:   s.worker.name,
		Timestamp:   timestamp,
		Completion:  completion,
		RequestID:   requestID,
	}); err != nil {
		s.log.Error(ctx, types.ActionRabbitMQPublishFailed, "failed to publish status update", err, "worker-name", s.worker.name)
		s.log.Warn(ctx, types.ActionRabbitMQPublishFailed, "order has been proccessed, but failed to publish status update", "worker-name", s.worker.name)
	}

	// Increment number of proccessed orders by the worker.
	if err := s.workerRepo.IncrOrdersProcessed(ctx, s.worker.name); err != nil {
		s.log.Error(ctx, types.ActionMessageProcessingFailed, "failed to increment number of ordered", err, "worker-name", s.worker.name)
		s.log.Warn(ctx, types.ActionMessageProcessingFailed, "order has been proccessed, but could not increment number of proccessed order for worker in the database", "worker-name", s.worker.name)
	}

	s.log.Debug(ctx, types.ActionOrderCompleted, "order proccess finished", "worker-name", s.worker.name)
	return nil
}

// heartbeatLoop tries to update last seen field in database each heartbeat interval.
func (s *KitchenWorker) heartbeatLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			s.log.Info(ctx, "worker_hearbeat_stop", "stopped hearbeat loop")
			return
		case <-ticker.C:
			if err := s.workerRepo.UpdateLastSeen(ctx, s.worker.name); err != nil {
				s.log.Error(ctx, types.ActionDBQueryFailed, "failed to update last seen on worker", err, "worker-name", s.worker.name)
				continue
			}
			s.log.Debug(ctx, types.ActionHeartbeatSent, "heartbeat was sent", "worker-name", s.worker.name)
		}
	}
}

// markOnline marks kitchen-worker as online
func (s *KitchenWorker) markOnline(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// check if it's already working.
	if s.isWorking {
		return models.ErrWorkerAlreadyOnline
	}

	// turning all order types that worker can handle into string to store in database.
	workerOrderTypes := strings.Join(s.worker.orderTypes, ",")

	// Marking worker as online
	if err := s.workerRepo.MarkOnline(ctx, s.worker.name, workerOrderTypes, s.worker.heartbeat); err != nil {
		return err
	}
	s.isWorking = true

	s.log.Info(ctx,
		types.ActionWorkerRegistered,
		"worker was successfully registered",
		"worker-name", s.worker.name,
		"order-types", workerOrderTypes,
		"heartbeat-interval", utils.PrettyDuration(s.worker.heartbeat),
	)

	return nil
}
