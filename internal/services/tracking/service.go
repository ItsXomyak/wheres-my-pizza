package tracking

import (
	"context"
	"errors"
	"fmt"
	"time"

	"wheres-my-pizza/internal/domain/models"
	"wheres-my-pizza/internal/domain/types"
	"wheres-my-pizza/pkg/logger"
)

type Service struct {
	statusRepo   StatusRepo
	workerRepo   WorkerRepo
	heartbeatInt int

	log logger.Logger
}

func NewService(statusRepo StatusRepo, workerRepo WorkerRepo, heartbeatInt int, log logger.Logger) *Service {
	return &Service{
		statusRepo:   statusRepo,
		workerRepo:   workerRepo,
		heartbeatInt: heartbeatInt,
		log:          log,
	}
}

// GetOrderStatus — возвращает статус заказа по его номеру.
func (s *Service) GetOrderStatus(ctx context.Context, orderNumber string) (models.OrderStatus, error) {
	const op = "Service.GetOrderStatus"

	statusInfo, err := s.statusRepo.GetCurrent(ctx, orderNumber)
	if err != nil {
		if errors.Is(err, models.ErrOrderNotFound) {
			return models.OrderStatus{}, models.ErrOrderNotFound
		}

		s.log.Error(ctx, types.ActionDBQueryFailed, "failed to get current order status", err)
		return models.OrderStatus{}, fmt.Errorf("%s: %v", op, err)
	}

	return statusInfo, nil
}

// ListWorkers — возвращает список работников, задействованных в процессе.
func (s *Service) ListWorkers(ctx context.Context) ([]models.Worker, error) {
	const op = "Service.ListWorkers"

	now := time.Now()
	workers, err := s.workerRepo.List(ctx)
	if err != nil {
		if errors.Is(err, models.ErrWorkerNotFound) {
			return nil, models.ErrWorkerNotFound
		}

		s.log.Error(ctx, types.ActionDBQueryFailed, "failed to get workers list", err)
		return nil, fmt.Errorf("%s: %v", op, err)
	}

	threshold := time.Duration(s.heartbeatInt) * time.Second
	for i := range workers {
		if now.Sub(workers[i].LastSeen) > threshold {
			workers[i].Status = types.WorkerOffline
		}
		continue
	}

	return workers, nil
}

// GetTrackingHistory — возвращает историю изменений по заказу.
func (s *Service) GetTrackingHistory(ctx context.Context, orderNumber string) ([]models.OrderHistory, error) {
	const op = "Service.GetTrackingHistory"

	historyList, err := s.statusRepo.ListOrderHistory(ctx, orderNumber)
	if err != nil {
		if errors.Is(err, models.ErrOrderNotFound) {
			return nil, models.ErrOrderNotFound
		}

		s.log.Error(ctx, types.ActionDBQueryFailed, "failed to get tracking history", err)
		return nil, fmt.Errorf("%s: %v", op, err)
	}

	return historyList, nil
}
