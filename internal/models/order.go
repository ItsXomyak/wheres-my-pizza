package models

import (
	"fmt"
	"regexp"
	"time"
)

// OrderType represents the type of an order
type OrderType string

const (
	DineIn   OrderType = "dine_in"
	Takeout  OrderType = "takeout"
	Delivery OrderType = "delivery"
)

// OrderStatus represents the status of an order
type OrderStatus string

const (
	StatusReceived  OrderStatus = "received"
	StatusCooking   OrderStatus = "cooking"
	StatusReady     OrderStatus = "ready"
	StatusCompleted OrderStatus = "completed"
	StatusCancelled OrderStatus = "cancelled"
)

// OrderItem represents an item in an order
type OrderItem struct {
	ID       int     `json:"id,omitempty" db:"id"`
	OrderID  int     `json:"order_id,omitempty" db:"order_id"`
	Name     string  `json:"name" db:"name"`
	Quantity int     `json:"quantity" db:"quantity"`
	Price    float64 `json:"price" db:"price"`
}

// Order represents a customer order
type Order struct {
	ID              int          `json:"id,omitempty" db:"id"`
	CreatedAt       time.Time    `json:"created_at,omitempty" db:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at,omitempty" db:"updated_at"`
	Number          string       `json:"order_number" db:"number"`
	CustomerName    string       `json:"customer_name" db:"customer_name"`
	Type            OrderType    `json:"order_type" db:"type"`
	TableNumber     *int         `json:"table_number,omitempty" db:"table_number"`
	DeliveryAddress *string      `json:"delivery_address,omitempty" db:"delivery_address"`
	Items           []OrderItem  `json:"items"`
	TotalAmount     float64      `json:"total_amount" db:"total_amount"`
	Priority        int          `json:"priority" db:"priority"`
	Status          OrderStatus  `json:"status" db:"status"`
	ProcessedBy     *string      `json:"processed_by,omitempty" db:"processed_by"`
	CompletedAt     *time.Time   `json:"completed_at,omitempty" db:"completed_at"`
}

// CreateOrderRequest represents the request to create a new order
type CreateOrderRequest struct {
	CustomerName    string      `json:"customer_name"`
	OrderType       string      `json:"order_type"`
	TableNumber     *int        `json:"table_number,omitempty"`
	DeliveryAddress *string     `json:"delivery_address,omitempty"`
	Items           []OrderItem `json:"items"`
}

// CreateOrderResponse represents the response after creating an order
type CreateOrderResponse struct {
	OrderNumber string  `json:"order_number"`
	Status      string  `json:"status"`
	TotalAmount float64 `json:"total_amount"`
}

// OrderStatusHistory represents an entry in the order status log
type OrderStatusHistory struct {
	Status    OrderStatus `json:"status" db:"status"`
	ChangedBy string      `json:"changed_by" db:"changed_by"`
	ChangedAt time.Time   `json:"timestamp" db:"changed_at"`
	Notes     *string     `json:"notes,omitempty" db:"notes"`
}

// OrderTrackingResponse represents the response for order tracking
type OrderTrackingResponse struct {
	OrderNumber         string     `json:"order_number"`
	CurrentStatus       string     `json:"current_status"`
	UpdatedAt           time.Time  `json:"updated_at"`
	EstimatedCompletion *time.Time `json:"estimated_completion,omitempty"`
	ProcessedBy         *string    `json:"processed_by,omitempty"`
}

// ValidateCreateOrderRequest validates the create order request
func (req *CreateOrderRequest) Validate() error {
	// Validate customer name
	if err := validateCustomerName(req.CustomerName); err != nil {
		return err
	}

	// Validate order type
	orderType, err := validateOrderType(req.OrderType)
	if err != nil {
		return err
	}

	// Validate conditional fields based on order type
	if err := validateConditionalFields(orderType, req.TableNumber, req.DeliveryAddress); err != nil {
		return err
	}

	// Validate items
	if err := validateItems(req.Items); err != nil {
		return err
	}

	return nil
}

// CalculateTotalAmount calculates the total amount for the order
func (req *CreateOrderRequest) CalculateTotalAmount() float64 {
	total := 0.0
	for _, item := range req.Items {
		total += item.Price * float64(item.Quantity)
	}
	return total
}

// CalculatePriority calculates the priority based on total amount
func (req *CreateOrderRequest) CalculatePriority() int {
	total := req.CalculateTotalAmount()
	if total > 100.0 {
		return 10
	}
	if total >= 50.0 {
		return 5
	}
	return 1
}

// GenerateOrderNumber generates a unique order number in format ORD_YYYYMMDD_NNN
func GenerateOrderNumber(date time.Time, sequence int) string {
	dateStr := date.Format("20060102")
	return fmt.Sprintf("ORD_%s_%03d", dateStr, sequence)
}

// validateCustomerName validates the customer name field
func validateCustomerName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("customer_name is required")
	}
	if len(name) > 100 {
		return fmt.Errorf("customer_name must not exceed 100 characters")
	}
	
	// Allow letters, spaces, hyphens, and apostrophes
	validNamePattern := regexp.MustCompile(`^[a-zA-Z\s\-']+$`)
	if !validNamePattern.MatchString(name) {
		return fmt.Errorf("customer_name contains invalid characters")
	}
	
	return nil
}

// validateOrderType validates the order type field
func validateOrderType(orderType string) (OrderType, error) {
	switch OrderType(orderType) {
	case DineIn, Takeout, Delivery:
		return OrderType(orderType), nil
	default:
		return "", fmt.Errorf("order_type must be one of: dine_in, takeout, delivery")
	}
}

// validateConditionalFields validates fields based on order type
func validateConditionalFields(orderType OrderType, tableNumber *int, deliveryAddress *string) error {
	switch orderType {
	case DineIn:
		if tableNumber == nil {
			return fmt.Errorf("table_number is required for dine_in orders")
		}
		if *tableNumber < 1 || *tableNumber > 100 {
			return fmt.Errorf("table_number must be between 1 and 100")
		}
		if deliveryAddress != nil {
			return fmt.Errorf("delivery_address must not be present for dine_in orders")
		}
	case Delivery:
		if deliveryAddress == nil || *deliveryAddress == "" {
			return fmt.Errorf("delivery_address is required for delivery orders")
		}
		if len(*deliveryAddress) < 10 {
			return fmt.Errorf("delivery_address must be at least 10 characters")
		}
		if tableNumber != nil {
			return fmt.Errorf("table_number must not be present for delivery orders")
		}
	case Takeout:
		if tableNumber != nil {
			return fmt.Errorf("table_number must not be present for takeout orders")
		}
		if deliveryAddress != nil {
			return fmt.Errorf("delivery_address must not be present for takeout orders")
		}
	}
	
	return nil
}

// validateItems validates the order items
func validateItems(items []OrderItem) error {
	if len(items) == 0 {
		return fmt.Errorf("items array cannot be empty")
	}
	if len(items) > 20 {
		return fmt.Errorf("items array cannot contain more than 20 items")
	}
	
	for i, item := range items {
		if err := validateItem(item, i); err != nil {
			return err
		}
	}
	
	return nil
}

// validateItem validates a single order item
func validateItem(item OrderItem, index int) error {
	prefix := fmt.Sprintf("items[%d]", index)
	
	// Validate name
	if len(item.Name) == 0 {
		return fmt.Errorf("%s.name is required", prefix)
	}
	if len(item.Name) > 50 {
		return fmt.Errorf("%s.name must not exceed 50 characters", prefix)
	}
	
	// Validate quantity
	if item.Quantity < 1 || item.Quantity > 10 {
		return fmt.Errorf("%s.quantity must be between 1 and 10", prefix)
	}
	
	// Validate price
	if item.Price < 0.01 || item.Price > 999.99 {
		return fmt.Errorf("%s.price must be between 0.01 and 999.99", prefix)
	}
	
	return nil
}
