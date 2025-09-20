package notification

import (
	"context"

	"wheres-my-pizza/internal/domain/models"
)

type NotificationConsumer interface {
	StartListening(ctx context.Context) (chan models.StatusUpdate, error)
	Close() error
}

type Notifier interface {
	StatusUpdate(ctx context.Context, req models.StatusUpdate)
}
