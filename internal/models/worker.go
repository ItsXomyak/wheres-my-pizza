package models

import (
	"strings"
	"time"
)

// WorkerStatus represents the status of a worker
type WorkerStatus string

const (
	WorkerOnline  WorkerStatus = "online"
	WorkerOffline WorkerStatus = "offline"
)

// WorkerType represents the type/specialization of a worker
type WorkerType string

const (
	GeneralWorker WorkerType = "general"
	DineInWorker  WorkerType = "dine_in"
	TakeoutWorker WorkerType = "takeout"
	DeliveryWorker WorkerType = "delivery"
)

// Worker represents a kitchen worker
type Worker struct {
	ID               int          `json:"id,omitempty" db:"id"`
	CreatedAt        time.Time    `json:"created_at,omitempty" db:"created_at"`
	Name             string       `json:"worker_name" db:"name"`
	Type             WorkerType   `json:"type,omitempty" db:"type"`
	Status           WorkerStatus `json:"status" db:"status"`
	LastSeen         time.Time    `json:"last_seen" db:"last_seen"`
	OrdersProcessed  int          `json:"orders_processed" db:"orders_processed"`
}

// WorkerStatusResponse represents the response for worker status queries
type WorkerStatusResponse struct {
	WorkerName      string    `json:"worker_name"`
	Status          string    `json:"status"`
	OrdersProcessed int       `json:"orders_processed"`
	LastSeen        time.Time `json:"last_seen"`
}

// ParseOrderTypes parses a comma-separated string of order types into a slice
func ParseOrderTypes(orderTypesStr string) []OrderType {
	if orderTypesStr == "" {
		return nil
	}
	
	var orderTypes []OrderType
	parts := strings.Split(orderTypesStr, ",")
	
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		switch trimmed {
		case "dine_in":
			orderTypes = append(orderTypes, DineIn)
		case "takeout":
			orderTypes = append(orderTypes, Takeout)
		case "delivery":
			orderTypes = append(orderTypes, Delivery)
		}
	}
	
	return orderTypes
}

// CanHandleOrderType checks if a worker can handle a specific order type
func (w *Worker) CanHandleOrderType(orderType OrderType, specializations []OrderType) bool {
	// If no specializations are set, worker can handle all types
	if len(specializations) == 0 {
		return true
	}
	
	// Check if the order type is in the worker's specializations
	for _, specialization := range specializations {
		if specialization == orderType {
			return true
		}
	}
	
	return false
}

// IsOnline checks if a worker is considered online based on heartbeat interval
func (w *Worker) IsOnline(heartbeatInterval time.Duration) bool {
	if w.Status == WorkerOffline {
		return false
	}
	
	// Consider worker offline if last seen is more than 2 * heartbeat interval ago
	threshold := 2 * heartbeatInterval
	return time.Since(w.LastSeen) <= threshold
}
