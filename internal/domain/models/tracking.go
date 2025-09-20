package models

import "time"

type OrderStatus struct {
	OrderNumber string     `json:"order_number"`
	Status      string     `json:"current_status"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Completion  *time.Time `json:"estimated_completion"` // nullable
	ProcessedBy *string    `json:"processed_by"`         // nullable
}

type OrderHistory struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	ChangedBy string    `json:"changed_by"`
}
