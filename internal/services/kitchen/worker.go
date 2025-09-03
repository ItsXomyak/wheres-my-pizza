package kitchen

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"restaurant-system/internal/database"
	"restaurant-system/internal/logger"
	"restaurant-system/internal/messaging"
	"restaurant-system/internal/models"
)

// Worker represents a kitchen worker
type Worker struct {
	name              string
	orderTypes        []models.OrderType
	heartbeatInterval time.Duration
	prefetch          int
	
	db        *database.DB
	consumer  *messaging.Consumer
	publisher *messaging.Publisher
	logger    *logger.Logger
	
	// Graceful shutdown
	shutdown chan os.Signal
	done     chan bool
}

// NewWorker creates a new kitchen worker
func NewWorker(name string, orderTypes []models.OrderType, heartbeatInterval time.Duration, prefetch int,
	db *database.DB, consumer *messaging.Consumer, publisher *messaging.Publisher, logger *logger.Logger) *Worker {
	
	return &Worker{
		name:              name,
		orderTypes:        orderTypes,
		heartbeatInterval: heartbeatInterval,
		prefetch:          prefetch,
		db:                db,
		consumer:          consumer,
		publisher:         publisher,
		logger:            logger,
		shutdown:          make(chan os.Signal, 1),
		done:              make(chan bool, 1),
	}
}

// Start starts the kitchen worker
func (w *Worker) Start(ctx context.Context) error {
	requestID := logger.GenerateRequestID()
	
	// Register worker in database
	if err := w.registerWorker(ctx, requestID); err != nil {
		return fmt.Errorf("failed to register worker: %w", err)
	}

	// Set up graceful shutdown
	signal.Notify(w.shutdown, syscall.SIGINT, syscall.SIGTERM)
	
	// Start heartbeat goroutine
	go w.heartbeatLoop(ctx)
	
	// Start message processing
	go func() {
		if err := w.consumer.StartConsuming(ctx, w.handleMessage); err != nil {
			w.logger.Error("consumer_failed", "Message consumer failed", requestID, err, nil)
		}
		w.done <- true
	}()
	
	w.logger.Info("worker_started", fmt.Sprintf("Kitchen worker %s started", w.name), requestID, map[string]interface{}{
		"worker_name":        w.name,
		"order_types":        w.orderTypes,
		"heartbeat_interval": w.heartbeatInterval.Seconds(),
		"prefetch":           w.prefetch,
	})
	
	// Wait for shutdown signal or consumer to finish
	select {
	case <-w.shutdown:
		w.logger.Info("graceful_shutdown", "Received shutdown signal", requestID, nil)
		return w.gracefulShutdown(ctx, requestID)
	case <-w.done:
		return nil
	}
}

// registerWorker registers the worker in the database
func (w *Worker) registerWorker(ctx context.Context, requestID string) error {
	// Check if worker with same name is already online
	var count int
	err := w.db.QueryRow(ctx, database.CheckWorkerOnlineSQL, w.name).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check worker status: %w", err)
	}
	
	if count > 0 {
		w.logger.Error("worker_registration_failed", "Worker with same name is already online", requestID, nil, map[string]interface{}{
			"worker_name": w.name,
		})
		return fmt.Errorf("worker %s is already online", w.name)
	}
	
	// Register or update worker
	var workerID int
	err = w.db.QueryRow(ctx, database.InsertWorkerSQL, w.name, "general").Scan(&workerID)
	if err != nil {
		return fmt.Errorf("failed to register worker: %w", err)
	}
	
	w.logger.Info("worker_registered", fmt.Sprintf("Worker %s registered successfully", w.name), requestID, map[string]interface{}{
		"worker_id":   workerID,
		"worker_name": w.name,
	})
	
	return nil
}

// handleMessage processes incoming order messages
func (w *Worker) handleMessage(ctx context.Context, body []byte) error {
	requestID := logger.GenerateRequestID()
	
	// Parse order message
	var orderMsg models.OrderMessage
	if err := json.Unmarshal(body, &orderMsg); err != nil {
		w.logger.Error("message_parsing_failed", "Failed to parse order message", requestID, err, nil)
		return fmt.Errorf("failed to parse message: %w", err)
	}
	
	w.logger.Debug("order_processing_started", fmt.Sprintf("Processing order %s", orderMsg.OrderNumber), requestID, map[string]interface{}{
		"order_number": orderMsg.OrderNumber,
		"customer_name": orderMsg.CustomerName,
		"order_type": orderMsg.OrderType,
		"total_amount": orderMsg.TotalAmount,
	})
	
	// Check if worker can handle this order type
	if !w.canHandleOrderType(models.OrderType(orderMsg.OrderType)) {
		w.logger.Debug("order_rejected", fmt.Sprintf("Worker %s cannot handle order type %s", w.name, orderMsg.OrderType), requestID, map[string]interface{}{
			"order_number": orderMsg.OrderNumber,
			"order_type": orderMsg.OrderType,
			"worker_specializations": w.orderTypes,
		})
		// Return error to nack and requeue the message
		return fmt.Errorf("worker cannot handle order type %s", orderMsg.OrderType)
	}
	
	// Process the order
	return w.processOrder(ctx, &orderMsg, requestID)
}

// processOrder processes a single order through its lifecycle
func (w *Worker) processOrder(ctx context.Context, orderMsg *models.OrderMessage, requestID string) error {
	// Step 1: Update order status to 'cooking'
	if err := w.updateOrderStatus(ctx, orderMsg.OrderNumber, models.StatusCooking, requestID); err != nil {
		return fmt.Errorf("failed to update order status to cooking: %w", err)
	}
	
	// Step 2: Publish status update notification
	estimatedCompletion := time.Now().UTC().Add(models.GetCookingTime(orderMsg.OrderType))
	statusUpdate := models.CreateStatusUpdateMessage(
		orderMsg.OrderNumber, 
		string(models.StatusReceived), 
		string(models.StatusCooking), 
		w.name, 
		&estimatedCompletion,
	)
	
	if err := w.publisher.PublishNotification(ctx, statusUpdate); err != nil {
		w.logger.Error("notification_publish_failed", "Failed to publish cooking notification", requestID, err, map[string]interface{}{
			"order_number": orderMsg.OrderNumber,
		})
		// Don't fail the order processing if notification fails
	}
	
	// Step 3: Simulate cooking
	cookingTime := models.GetCookingTime(orderMsg.OrderType)
	w.logger.Debug("cooking_started", fmt.Sprintf("Cooking order %s for %v", orderMsg.OrderNumber, cookingTime), requestID, map[string]interface{}{
		"order_number": orderMsg.OrderNumber,
		"cooking_time_seconds": cookingTime.Seconds(),
	})
	
	time.Sleep(cookingTime)
	
	// Step 4: Update order status to 'ready'
	if err := w.updateOrderStatusReady(ctx, orderMsg.OrderNumber, requestID); err != nil {
		return fmt.Errorf("failed to update order status to ready: %w", err)
	}
	
	// Step 5: Publish final status update notification
	finalStatusUpdate := models.CreateStatusUpdateMessage(
		orderMsg.OrderNumber, 
		string(models.StatusCooking), 
		string(models.StatusReady), 
		w.name, 
		nil, // No estimated completion for ready orders
	)
	
	if err := w.publisher.PublishNotification(ctx, finalStatusUpdate); err != nil {
		w.logger.Error("notification_publish_failed", "Failed to publish ready notification", requestID, err, map[string]interface{}{
			"order_number": orderMsg.OrderNumber,
		})
		// Don't fail the order processing if notification fails
	}
	
	w.logger.Debug("order_completed", fmt.Sprintf("Successfully processed order %s", orderMsg.OrderNumber), requestID, map[string]interface{}{
		"order_number": orderMsg.OrderNumber,
		"processed_by": w.name,
	})
	
	return nil
}

// updateOrderStatus updates the order status to cooking
func (w *Worker) updateOrderStatus(ctx context.Context, orderNumber string, status models.OrderStatus, requestID string) error {
	tx, err := w.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	
	// Update order status
	_, err = tx.Exec(ctx, database.UpdateOrderStatusSQL, status, w.name, orderNumber)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	
	// Get order ID for status log
	var orderID int
	err = tx.QueryRow(ctx, "SELECT id FROM orders WHERE number = $1", orderNumber).Scan(&orderID)
	if err != nil {
		return fmt.Errorf("failed to get order ID: %w", err)
	}
	
	// Insert status log entry
	_, err = tx.Exec(ctx, database.InsertOrderStatusLogSQL, orderID, status, w.name, fmt.Sprintf("Order status changed to %s by %s", status, w.name))
	if err != nil {
		return fmt.Errorf("failed to insert status log: %w", err)
	}
	
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// updateOrderStatusReady updates the order status to ready and increments processed count
func (w *Worker) updateOrderStatusReady(ctx context.Context, orderNumber string, requestID string) error {
	tx, err := w.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	
	// Update order status to ready with completion time
	_, err = tx.Exec(ctx, database.UpdateOrderCompletedSQL, models.StatusReady, orderNumber)
	if err != nil {
		return fmt.Errorf("failed to update order to ready: %w", err)
	}
	
	// Get order ID for status log
	var orderID int
	err = tx.QueryRow(ctx, "SELECT id FROM orders WHERE number = $1", orderNumber).Scan(&orderID)
	if err != nil {
		return fmt.Errorf("failed to get order ID: %w", err)
	}
	
	// Insert status log entry
	_, err = tx.Exec(ctx, database.InsertOrderStatusLogSQL, orderID, models.StatusReady, w.name, fmt.Sprintf("Order completed and ready for pickup/delivery"))
	if err != nil {
		return fmt.Errorf("failed to insert status log: %w", err)
	}
	
	// Increment worker's processed count
	_, err = tx.Exec(ctx, database.UpdateWorkerHeartbeatSQL, 1, w.name)
	if err != nil {
		return fmt.Errorf("failed to update worker processed count: %w", err)
	}
	
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// canHandleOrderType checks if worker can handle the given order type
func (w *Worker) canHandleOrderType(orderType models.OrderType) bool {
	// If no specializations, can handle all types
	if len(w.orderTypes) == 0 {
		return true
	}
	
	// Check if order type is in specializations
	for _, specialization := range w.orderTypes {
		if specialization == orderType {
			return true
		}
	}
	
	return false
}

// heartbeatLoop sends periodic heartbeats to update last_seen
func (w *Worker) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(w.heartbeatInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.shutdown:
			return
		case <-ticker.C:
			if err := w.sendHeartbeat(ctx); err != nil {
				w.logger.Error("heartbeat_failed", "Failed to send heartbeat", "", err, nil)
			} else {
				w.logger.Debug("heartbeat_sent", "Heartbeat sent successfully", "", nil)
			}
		}
	}
}

// sendHeartbeat updates the worker's last_seen timestamp
func (w *Worker) sendHeartbeat(ctx context.Context) error {
	_, err := w.db.Pool.Exec(ctx, database.UpdateWorkerStatusSQL, "online", w.name)
	return err
}

// gracefulShutdown handles graceful shutdown of the worker
func (w *Worker) gracefulShutdown(ctx context.Context, requestID string) error {
	w.logger.Info("graceful_shutdown", "Starting graceful shutdown", requestID, nil)
	
	// Update worker status to offline
	_, err := w.db.Pool.Exec(ctx, database.UpdateWorkerStatusSQL, "offline", w.name)
	if err != nil {
		w.logger.Error("shutdown_failed", "Failed to update worker status to offline", requestID, err, nil)
	}
	
	// Close consumer
	if w.consumer != nil {
		w.consumer.Close()
	}
	
	w.logger.Info("graceful_shutdown", "Graceful shutdown completed", requestID, nil)
	return nil
}
