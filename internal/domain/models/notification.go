package models

import "time"

type StatusUpdate struct {
	OrderNumber string    `json:"order_number"`
	OldStatus   string    `json:"old_status"`
	NewStatus   string    `json:"new_status"`
	ChangedBy   string    `json:"changed_by"`
	Timestamp   time.Time `json:"timestamp"`
	Completion  time.Time `json:"estimated_completion"`
	RequestID   string    `json:"request_id"`
}
