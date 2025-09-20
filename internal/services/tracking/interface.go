package tracking

import (
	"context"

	"wheres-my-pizza/internal/domain/models"
)

type StatusRepo interface {
	GetCurrent(ctx context.Context, orderNumber string) (models.OrderStatus, error)
	ListOrderHistory(ctx context.Context, orderNumber string) ([]models.OrderHistory, error)
}

type WorkerRepo interface {
	List(ctx context.Context) ([]models.Worker, error)
}
