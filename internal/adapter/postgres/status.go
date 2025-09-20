package postgres

import (
	"context"
	"fmt"

	"wheres-my-pizza/internal/domain/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type statusRepository struct {
	pool *pgxpool.Pool
}

func NewStatusRepo(pool *pgxpool.Pool) *statusRepository {
	return &statusRepository{
		pool: pool,
	}
}

func (repo *statusRepository) GetCurrent(ctx context.Context, orderNumber string) (models.OrderStatus, error) {
	const op = "statusRepository.GetCurrent"
	query := `
	SELECT 
		o.number,
		COALESCE(s.status,''),
		s.changed_at,
		o.completed_at,
		o.processed_by  
	FROM 
		order_status_log s
	INNER JOIN orders o on s.order_id = o.id
	WHERE 
		o.number = $1
	ORDER BY 
		s.created_at DESC;`

	var statusInfo models.OrderStatus

	if err := repo.pool.QueryRow(ctx, query, orderNumber).
		Scan(&statusInfo.OrderNumber, &statusInfo.Status, &statusInfo.UpdatedAt, &statusInfo.Completion, &statusInfo.ProcessedBy); err != nil {
		if err == pgx.ErrNoRows {
			return models.OrderStatus{}, models.ErrOrderNotFound
		}
		return models.OrderStatus{}, fmt.Errorf("%s: %v", op, err)
	}

	return statusInfo, nil
}

func (repo *statusRepository) ListOrderHistory(ctx context.Context, orderNumber string) ([]models.OrderHistory, error) {
	const op = "statusRepository.ListStatusHistory"

	query := `
	SELECT 
		COALESCE(s.status,''),
		s.changed_at,
		COALESCE(s.changed_by,'') 
	FROM 
		order_status_log s
	INNER JOIN orders o ON s.order_id = o.id
	WHERE 
		o.number = $1;`

	rows, err := repo.pool.Query(ctx, query, orderNumber)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", op, err)
	}

	historyList, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.OrderHistory, error) {
		var history models.OrderHistory
		if err := row.Scan(&history.Status, &history.Timestamp, &history.ChangedBy); err != nil {
			return models.OrderHistory{}, err
		}
		return history, nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %v", op, err)
	}

	if len(historyList) == 0 {
		return nil, models.ErrOrderNotFound
	}

	return historyList, nil
}
