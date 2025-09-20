package notification

import (
	"context"
	"fmt"

	"wheres-my-pizza/internal/domain/models"
	"wheres-my-pizza/internal/domain/types"
	"wheres-my-pizza/pkg/logger"
)

type NotifyPrinter struct {
	log logger.Logger
}

func NewNotifyPrinter(log logger.Logger) *NotifyPrinter {
	return &NotifyPrinter{
		log: log,
	}
}

// StatusUpdate just prints status update information to the console
func (s *NotifyPrinter) StatusUpdate(ctx context.Context, update models.StatusUpdate) {
	fmt.Printf("Notification for order %s: Status changed from '%s' to '%s' by %s\n",
		update.OrderNumber,
		update.OldStatus,
		update.NewStatus,
		update.ChangedBy,
	)

	details := struct {
		Number    string `json:"order_number"`
		NewStatus string `json:"new_status"`
	}{
		Number:    update.OrderNumber,
		NewStatus: update.NewStatus,
	}

	s.log.Info(ctx, types.ActionNotificationReceived,
		fmt.Sprintf("Received status update for order %s", update.OrderNumber),
		"order_timestamp", update.Timestamp,
		"request_id", update.RequestID,
		"details", details,
	)
}
