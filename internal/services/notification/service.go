package notification

import (
	"context"
	"errors"

	"wheres-my-pizza/pkg/logger"
)

var ErrNotificationStopped = errors.New("notification subscriber stopped")

type Service struct {
	reader NotificationConsumer
	writer Notifier
	log    logger.Logger
}

func NewService(reader NotificationConsumer, notifier Notifier, log logger.Logger) *Service {
	return &Service{
		reader: reader,
		writer: notifier,
		log:    log,
	}
}

func (s *Service) Notify(ctx context.Context, errCh chan error) {
	updateCh, err := s.reader.StartListening(ctx)
	if err != nil {
		s.log.Error(ctx, "rabbit_queue_listening", "failed to start listening", err)
		errCh <- err
		return
	}

	defer func() {
		s.reader.Close()
		errCh <- ErrNotificationStopped
	}()

	for update := range updateCh {
		s.writer.StatusUpdate(ctx, update)
	}
}

func (s *Service) Close() error {
	return s.reader.Close()
}
