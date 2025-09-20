package models

import "time"

type Worker struct {
	Name            string    `json:"worker_name"`
	Status          string    `json:"status"`
	ProcessedOrders int       `json:"orders_processed"`
	LastSeen        time.Time `json:"last_seen"`
}
