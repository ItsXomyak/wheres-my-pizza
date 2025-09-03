package tracking

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"restaurant-system/internal/database"
	"restaurant-system/internal/logger"
	"restaurant-system/internal/models"
)

// Service provides tracking functionality
type Service struct {
	db     *database.DB
	logger *logger.Logger
}

// NewService creates a new tracking service
func NewService(db *database.DB, logger *logger.Logger) *Service {
	return &Service{
		db:     db,
		logger: logger,
	}
}

// GetOrderStatus retrieves the current status of an order
func (s *Service) GetOrderStatus(ctx context.Context, orderNumber, requestID string) (*models.OrderTrackingResponse, error) {
	var order models.Order
	
	err := s.db.QueryRow(ctx, database.GetOrderByNumberSQL, orderNumber).Scan(
		&order.ID,
		&order.Number,
		&order.CustomerName,
		&order.Type,
		&order.TableNumber,
		&order.DeliveryAddress,
		&order.TotalAmount,
		&order.Priority,
		&order.Status,
		&order.ProcessedBy,
		&order.CreatedAt,
		&order.UpdatedAt,
		&order.CompletedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("order not found")
		}
		s.logger.Error("db_query_failed", "Failed to query order", requestID, err, map[string]interface{}{
			"order_number": orderNumber,
		})
		return nil, fmt.Errorf("database error: %w", err)
	}
	
	// Calculate estimated completion if order is still cooking
	var estimatedCompletion *time.Time
	if order.Status == models.StatusCooking {
		cookingTime := models.GetCookingTime(string(order.Type))
		estimated := order.UpdatedAt.Add(cookingTime)
		estimatedCompletion = &estimated
	}
	
	response := &models.OrderTrackingResponse{
		OrderNumber:         order.Number,
		CurrentStatus:       string(order.Status),
		UpdatedAt:          order.UpdatedAt,
		EstimatedCompletion: estimatedCompletion,
		ProcessedBy:        order.ProcessedBy,
	}
	
	return response, nil
}

// GetOrderHistory retrieves the complete status history of an order
func (s *Service) GetOrderHistory(ctx context.Context, orderNumber, requestID string) ([]models.OrderStatusHistory, error) {
	// First check if order exists
	var orderExists bool
	err := s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM orders WHERE number = $1)", orderNumber).Scan(&orderExists)
	if err != nil {
		s.logger.Error("db_query_failed", "Failed to check order existence", requestID, err, map[string]interface{}{
			"order_number": orderNumber,
		})
		return nil, fmt.Errorf("database error: %w", err)
	}
	
	if !orderExists {
		return nil, fmt.Errorf("order not found")
	}
	
	// Get order status history
	rows, err := s.db.Query(ctx, database.GetOrderStatusHistorySQL, orderNumber)
	if err != nil {
		s.logger.Error("db_query_failed", "Failed to query order history", requestID, err, map[string]interface{}{
			"order_number": orderNumber,
		})
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer rows.Close()
	
	var history []models.OrderStatusHistory
	for rows.Next() {
		var entry models.OrderStatusHistory
		err := rows.Scan(
			&entry.Status,
			&entry.ChangedBy,
			&entry.ChangedAt,
			&entry.Notes,
		)
		if err != nil {
			s.logger.Error("db_scan_failed", "Failed to scan order history row", requestID, err, nil)
			return nil, fmt.Errorf("database error: %w", err)
		}
		
		history = append(history, entry)
	}
	
	if err = rows.Err(); err != nil {
		s.logger.Error("db_rows_failed", "Error iterating order history rows", requestID, err, nil)
		return nil, fmt.Errorf("database error: %w", err)
	}
	
	return history, nil
}

// GetWorkerStatus retrieves the status of all workers
func (s *Service) GetWorkerStatus(ctx context.Context, requestID string) ([]models.WorkerStatusResponse, error) {
	rows, err := s.db.Query(ctx, database.GetAllWorkersSQL)
	if err != nil {
		s.logger.Error("db_query_failed", "Failed to query worker status", requestID, err, nil)
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer rows.Close()
	
	var workers []models.WorkerStatusResponse
	heartbeatThreshold := 2 * 30 * time.Second // 2 * default heartbeat interval
	
	for rows.Next() {
		var worker models.Worker
		var createdAt time.Time
		
		err := rows.Scan(
			&worker.Name,
			&worker.Type,
			&worker.Status,
			&worker.OrdersProcessed,
			&worker.LastSeen,
			&createdAt,
		)
		if err != nil {
			s.logger.Error("db_scan_failed", "Failed to scan worker row", requestID, err, nil)
			return nil, fmt.Errorf("database error: %w", err)
		}
		
		// Determine if worker is actually online based on heartbeat
		actualStatus := string(worker.Status)
		if worker.Status == models.WorkerOnline {
			if time.Since(worker.LastSeen) > heartbeatThreshold {
				actualStatus = "offline"
			}
		}
		
		response := models.WorkerStatusResponse{
			WorkerName:      worker.Name,
			Status:          actualStatus,
			OrdersProcessed: worker.OrdersProcessed,
			LastSeen:        worker.LastSeen,
		}
		
		workers = append(workers, response)
	}
	
	if err = rows.Err(); err != nil {
		s.logger.Error("db_rows_failed", "Error iterating worker rows", requestID, err, nil)
		return nil, fmt.Errorf("database error: %w", err)
	}
	
	return workers, nil
}

// HealthCheck checks the health of dependencies
func (s *Service) HealthCheck(ctx context.Context) bool {
	// Check database connection
	if err := s.db.Ping(ctx); err != nil {
		s.logger.Error("health_check_failed", "Database ping failed", "", err, nil)
		return false
	}
	
	return true
}
