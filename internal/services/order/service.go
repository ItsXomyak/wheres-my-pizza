package order

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

var (
	orderCounter  int
	lastOrderDate string
)

type OrderService struct {
	db *sql.DB
}

func NewOrderService(db *sql.DB) *OrderService {
	return &OrderService{
		db: db,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, req *OrderRequest) (*OrderResponse, error) {
	
	
	
	// Create response
	var resp OrderResponse
	resp.OrderNumber = s.generateOrderNumber()
	resp.Status = "received"
	total := s.countTotalPrice(req)
	resp.TotalAmount = total
	
	return &resp, nil
}

func (s *OrderService) generateOrderNumber() string {
	// If service restarts, we need to get today's order count from DB
	if orderCounter == 0 {
		s.db.QueryRow("SELECT COUNT(*) FROM orders WHERE DATE(created_at) = CURRENT_DATE").Scan(&orderCounter)
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

func (s *OrderService) countTotalPrice(req *OrderRequest) float64 {
	var total float64
	for _, item := range req.Items {
		total += item.Price * float64(item.Quantity)
	}
	return total
}

func (s *OrderService) setOrderPriority(resp *OrderResponse) int {
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
