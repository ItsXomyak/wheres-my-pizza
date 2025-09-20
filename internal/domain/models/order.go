package models

import (
	"fmt"
	"time"
)

type Order struct {
	ID              int
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Number          string
	CustomerName    string
	Type            string  // 'dine_in', 'takeout', or 'delivery'
	TableNumber     *int    // nullable
	DeliveryAddress *string // nullable
	TotalAmount     float64 // decimal(10,2)
	Priority        int
	Status          string
	ProcessedBy     *string    // nullable
	CompletedAt     *time.Time // nullable
}

type OrderItem struct {
	ID        int
	CreatedAt time.Time
	OrderID   int
	Name      string
	Quantity  int
	Price     float64 // decimal(8,2)
}

type CreateOrder struct {
	Number          string
	CustomerName    string
	Type            string
	Items           []CreateOrderItem
	TableNumber     *int    // Only for dine_in
	DeliveryAddress *string // Only for delivery
	TotalAmount     float64
	Priority        int
	Status          string
}
type CreateOrderItem struct {
	Name     string
	Quantity int
	Price    float64
}

// CalucalteTotalAmount sets total amount. Sum the price * quantity for all items in the order.
func (m *CreateOrder) CalucalteTotalAmount() {
	var total float64

	for _, item := range m.Items {
		total += item.Price * float64(item.Quantity)
	}

	m.TotalAmount = total
}

// CalculatePriority sets priority
// '10' if Order total amount is greater than $100.
// '5'	if Order total amount is between $50 and $100.
// '1'	if All other standard orders.
func (m *CreateOrder) CalculatePriority() {
	switch {
	case m.TotalAmount > 100:
		m.Priority = 10
	case m.TotalAmount > 50:
		m.Priority = 5
	default:
		m.Priority = 1
	}
}

// - **Generate `order_number`:** Create a unique order number using the format `ORD_YYYYMMDD_NNN`.
// The `NNN` sequence should reset to `001` daily (based on UTC).
func (m *CreateOrder) SetNumber(date string, sequence int) {
	// Automatically uses minimum required digits
	format := "_%03d" // default
	if sequence > 999 {
		format = "_%04d" // 4 digits
	}
	if sequence > 9999 {
		format = "_%05d" // 5 digits
	}
	m.Number = fmt.Sprintf("ORD_%s"+format, date, sequence)
}

type OrderCreatedInfo struct {
	Number      string
	Status      string
	TotalAmount float64
}
