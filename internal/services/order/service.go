package order

import (
	"context"
	"fmt"
	"sync"
	"time"

	"restaurant-system/internal/database"
	"restaurant-system/internal/logger"
	"restaurant-system/internal/messaging"
	"restaurant-system/internal/models"
)

// Service provides order management functionality
type Service struct {
	db        *database.DB
	publisher *messaging.Publisher
	logger    *logger.Logger
	
	// Concurrency control
	semaphore chan struct{}
	
	// Order number generation
	orderNumberMutex sync.Mutex
}

// NewService creates a new order service
func NewService(db *database.DB, publisher *messaging.Publisher, logger *logger.Logger, maxConcurrent int) *Service {
	return &Service{
		db:        db,
		publisher: publisher,
		logger:    logger,
		semaphore: make(chan struct{}, maxConcurrent),
	}
}

// CreateOrder creates a new order
func (s *Service) CreateOrder(ctx context.Context, req *models.CreateOrderRequest, requestID string) (*models.CreateOrderResponse, error) {
	// Acquire semaphore for concurrency control
	select {
	case s.semaphore <- struct{}{}:
		defer func() { <-s.semaphore }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	now := time.Now().UTC()
	
	// Calculate order details
	totalAmount := req.CalculateTotalAmount()
	priority := req.CalculatePriority()
	
	// Generate unique order number
	orderNumber, err := s.generateOrderNumber(ctx, now)
	if err != nil {
		s.logger.Error("order_number_generation_failed", "Failed to generate order number", requestID, err, nil)
		return nil, fmt.Errorf("failed to generate order number: %w", err)
	}

	// Start database transaction
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		s.logger.Error("db_transaction_failed", "Failed to start transaction", requestID, err, nil)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert order
	var orderID int
	var createdAt time.Time
	err = tx.QueryRow(ctx, database.InsertOrderSQL,
		orderNumber,
		req.CustomerName,
		req.OrderType,
		req.TableNumber,
		req.DeliveryAddress,
		totalAmount,
		priority,
	).Scan(&orderID, &createdAt)
	
	if err != nil {
		s.logger.Error("db_transaction_failed", "Failed to insert order", requestID, err, map[string]interface{}{
			"order_number": orderNumber,
		})
		return nil, fmt.Errorf("failed to insert order: %w", err)
	}

	// Insert order items
	for _, item := range req.Items {
		_, err = tx.Exec(ctx, database.InsertOrderItemSQL,
			orderID,
			item.Name,
			item.Quantity,
			item.Price,
		)
		
		if err != nil {
			s.logger.Error("db_transaction_failed", "Failed to insert order item", requestID, err, map[string]interface{}{
				"order_id": orderID,
				"item_name": item.Name,
			})
			return nil, fmt.Errorf("failed to insert order item: %w", err)
		}
	}

	// Insert initial status log entry
	_, err = tx.Exec(ctx, database.InsertOrderStatusLogSQL,
		orderID,
		models.StatusReceived,
		"order-service",
		"Order received and processed",
	)
	
	if err != nil {
		s.logger.Error("db_transaction_failed", "Failed to insert status log", requestID, err, map[string]interface{}{
			"order_id": orderID,
		})
		return nil, fmt.Errorf("failed to insert status log: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		s.logger.Error("db_transaction_failed", "Failed to commit transaction", requestID, err, map[string]interface{}{
			"order_id": orderID,
		})
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Publish order message to RabbitMQ
	orderMessage := models.CreateOrderMessageFromRequest(req, orderNumber, priority)
	routingKey := models.GenerateRoutingKey(req.OrderType, priority)
	
	err = s.publisher.PublishOrder(ctx, orderMessage, routingKey, uint8(priority))
	if err != nil {
		s.logger.Error("rabbitmq_publish_failed", "Failed to publish order message", requestID, err, map[string]interface{}{
			"order_number": orderNumber,
			"routing_key":  routingKey,
		})
		// Note: We don't return error here as the order is already saved to database
		// This is a design choice - the order exists even if messaging fails
	} else {
		s.logger.Debug("order_published", "Successfully published order message", requestID, map[string]interface{}{
			"order_number": orderNumber,
			"routing_key":  routingKey,
		})
	}

	return &models.CreateOrderResponse{
		OrderNumber: orderNumber,
		Status:      string(models.StatusReceived),
		TotalAmount: totalAmount,
	}, nil
}

// HealthCheck checks the health of dependencies
func (s *Service) HealthCheck(ctx context.Context) bool {
	// Check database connection
	if err := s.db.Ping(ctx); err != nil {
		s.logger.Error("health_check_failed", "Database ping failed", "", err, nil)
		return false
	}

	// Check messaging connection (basic check)
	if s.publisher != nil {
		// We can't directly access the connection, so we'll just assume it's healthy for now
		// In a production system, you might want to add a health check method to the publisher
	}

	return true
}

// generateOrderNumber generates a unique order number for the current date
func (s *Service) generateOrderNumber(ctx context.Context, date time.Time) (string, error) {
	s.orderNumberMutex.Lock()
	defer s.orderNumberMutex.Unlock()

	// Generate pattern for today's orders
	dateStr := date.Format("20060102")
	pattern := fmt.Sprintf("ORD_%s_%%", dateStr)

	// Get next sequence number for today
	var nextSeq int
	err := s.db.QueryRow(ctx, database.GetNextOrderNumberSQL, pattern).Scan(&nextSeq)
	if err != nil {
		return "", fmt.Errorf("failed to get next order sequence: %w", err)
	}

	return models.GenerateOrderNumber(date, nextSeq), nil
}
