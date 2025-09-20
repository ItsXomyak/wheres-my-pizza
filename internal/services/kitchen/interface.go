package kitchen

import (
	"context"
	"time"

	"wheres-my-pizza/internal/domain/models"
)

// Repository contract
type WorkerRepository interface {
	// MarkOnline marks worker by inserting (or updating) a record in the
	// workers table with its unique name and type, marking it online.
	MarkOnline(ctx context.Context, name, orderTypes string, heartbeat time.Duration) error

	// MarkOffline marks worker offline.
	MarkOffline(ctx context.Context, name string) error

	// UpdateLastSeen updates last seen timestamp
	UpdateLastSeen(ctx context.Context, name string) error

	// Incerements number of proccessed orders for worker.
	IncrOrdersProcessed(ctx context.Context, name string) error
}

type OrderRepository interface {
	// SetStatus sets new status and returns old status
	SetStatus(ctx context.Context, orderNumber, workerName, status string, notes string) (string, error)
}

type Consumer interface {
	Consume(ctx context.Context, orderType string, handler func(ctx context.Context, req *models.CreateOrder) error) error
}

type Producer interface {
	StatusUpdate(ctx context.Context, req *models.StatusUpdate) error
}
