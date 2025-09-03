package models

import (
	"fmt"
	"time"
)

// OrderMessage represents a message sent to kitchen workers
type OrderMessage struct {
	OrderNumber     string      `json:"order_number"`
	CustomerName    string      `json:"customer_name"`
	OrderType       string      `json:"order_type"`
	TableNumber     *int        `json:"table_number"`
	DeliveryAddress *string     `json:"delivery_address"`
	Items           []OrderItem `json:"items"`
	TotalAmount     float64     `json:"total_amount"`
	Priority        int         `json:"priority"`
}

// StatusUpdateMessage represents a status update notification
type StatusUpdateMessage struct {
	OrderNumber         string    `json:"order_number"`
	OldStatus           string    `json:"old_status"`
	NewStatus           string    `json:"new_status"`
	ChangedBy           string    `json:"changed_by"`
	Timestamp           time.Time `json:"timestamp"`
	EstimatedCompletion *time.Time `json:"estimated_completion,omitempty"`
}

// CreateOrderMessageFromRequest creates an OrderMessage from a CreateOrderRequest and order details
func CreateOrderMessageFromRequest(req *CreateOrderRequest, orderNumber string, priority int) *OrderMessage {
	return &OrderMessage{
		OrderNumber:     orderNumber,
		CustomerName:    req.CustomerName,
		OrderType:       req.OrderType,
		TableNumber:     req.TableNumber,
		DeliveryAddress: req.DeliveryAddress,
		Items:           req.Items,
		TotalAmount:     req.CalculateTotalAmount(),
		Priority:        priority,
	}
}

// CreateStatusUpdateMessage creates a StatusUpdateMessage for order status changes
func CreateStatusUpdateMessage(orderNumber, oldStatus, newStatus, changedBy string, estimatedCompletion *time.Time) *StatusUpdateMessage {
	return &StatusUpdateMessage{
		OrderNumber:         orderNumber,
		OldStatus:           oldStatus,
		NewStatus:           newStatus,
		ChangedBy:           changedBy,
		Timestamp:           time.Now().UTC(),
		EstimatedCompletion: estimatedCompletion,
	}
}

// GetCookingTime returns the cooking time duration for different order types
func GetCookingTime(orderType string) time.Duration {
	switch orderType {
	case "dine_in":
		return 8 * time.Second
	case "takeout":
		return 10 * time.Second
	case "delivery":
		return 12 * time.Second
	default:
		return 10 * time.Second
	}
}

// GenerateRoutingKey generates a routing key for order messages
func GenerateRoutingKey(orderType string, priority int) string {
	return fmt.Sprintf("kitchen.%s.%d", orderType, priority)
}
