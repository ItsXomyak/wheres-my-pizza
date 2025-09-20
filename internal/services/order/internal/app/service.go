package app

// add correct structure

import (
	"context"
	"fmt"
	"time"

	"wheres-my-pizza/internal/services/order/adapter/db"
	"wheres-my-pizza/internal/services/order/internal/domain"
	"wheres-my-pizza/internal/services/order/internal/validation"
)

var (
	orderCounter  int
	lastOrderDate string
)

type OrderService struct {
	db *db.Repository
}

func NewOrderService(db *db.Repository) *OrderService {
	return &OrderService{
		db: db,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, req *domain.OrderRequest) (*domain.OrderResponse, error) {
	// Validate request
	if err := validation.ValidateOrderRequest(req); err != nil {
		return nil, fmt.Errorf("validate order request: %w", err)
	}

	// Create response
	var resp domain.OrderResponse
	resp.OrderNumber = s.generateOrderNumber()
	resp.Status = "received"
	total := s.countTotalPrice(req)
	resp.TotalAmount = total

	// Insert into DB
	priority := s.setOrderPriority(&resp)
	err := s.db.AddOrderInfoTransaction(ctx, req, &resp, priority)
	if err != nil {
		return nil, fmt.Errorf("add order to db: %w", err)
	}

	return &resp, nil
}

func (s *OrderService) generateOrderNumber() string {
	// If service restarts, we need to get today's order count from DB
	if orderCounter == 0 {
		orderCounter, _ = s.db.GetOrderNumber(context.Background())
	}
	if lastOrderDate == "" {
		lastOrderDate = time.Now().UTC().Format("20060102")
	}

	today := time.Now().UTC().Format("20060102")
	if today != lastOrderDate {
		orderCounter = 0
	}

	orderCounter++
	orderNumber := fmt.Sprintf("ORD_%s_%03d", today, orderCounter)
	lastOrderDate = today
	return orderNumber
}

func (s *OrderService) countTotalPrice(req *domain.OrderRequest) float64 {
	var total float64
	for _, item := range req.Items {
		total += item.Price * float64(item.Quantity)
	}
	return total
}

func (s *OrderService) setOrderPriority(resp *domain.OrderResponse) int {
	priority := 0
	if resp.TotalAmount > 100 {
		priority = 10
	} else if resp.TotalAmount > 50 {
		priority = 5
	} else {
		priority = 1
	}

	return priority
}
