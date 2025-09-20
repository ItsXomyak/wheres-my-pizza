package order

import (
	"context"
	"time"

	"wheres-my-pizza/internal/domain/models"
)

type OrderRepository interface {
	Create(ctx context.Context, req *models.CreateOrder, changedBy, notes string) (*models.Order, error)
	GetAndIncrementSequence(ctx context.Context, date string) (int, error)
}

type MessageBroker interface {
	PublishCreateOrder(ctx context.Context, order *models.CreateOrder) error
}

type Semaphore interface {
	TryAcquire(timeout time.Duration) bool
	Release()
	Available() int
	Used() int
}
