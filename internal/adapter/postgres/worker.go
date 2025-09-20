package postgres

import (
	"context"
	"fmt"
	"time"

	"wheres-my-pizza/internal/domain/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type workerRepository struct {
	pool *pgxpool.Pool
}

func NewWorkerRepo(pool *pgxpool.Pool) *workerRepository {
	return &workerRepository{
		pool: pool,
	}
}

func (repo *workerRepository) List(ctx context.Context) ([]models.Worker, error) {
	const op = "workerRepository.List"

	query := `
	SELECT 
		name,
		status,
		orders_processed,
		last_seen
	FROM 
		workers;`

	rows, err := repo.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", op, err)
	}

	workers, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (models.Worker, error) {
		var worker models.Worker
		if err := row.Scan(&worker.Name, &worker.Status, &worker.ProcessedOrders, &worker.LastSeen); err != nil {
			return models.Worker{}, err
		}
		return worker, nil
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %v", op, err)
	}

	if len(workers) == 0 {
		return nil, models.ErrWorkerNotFound
	}
	return workers, nil
}

// MarkOnline marks a worker as online by inserting or updating its record.
// If the worker already exists and is online but last_seen is recent (within heartbeat), registration fails.
// if worker marker 'online' worker still will be successfully marked if last_seen < Now() - heartbeat * 2
func (repo *workerRepository) MarkOnline(ctx context.Context, name, orderTypes string, heartbeat time.Duration) error {
	const op = "workerRepository.MarkOnline"

	query := `
		INSERT INTO workers (name, type, status, last_seen)
		VALUES ($1, $2, 'online', now())
		ON CONFLICT (name)
		DO UPDATE
		SET 
			status = 'online',
			type = $2,
			last_seen = now()
		WHERE 
			workers.name = $1
			AND (
				workers.status = 'offline' 
				OR workers.last_seen < now() - make_interval(secs => $3)
			);
		`

	res, err := repo.pool.Exec(ctx, query, name, orderTypes, int64(heartbeat.Seconds())*2)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("%s: %v", op, models.ErrWorkerAlreadyOnline)
	}

	return nil
}

func (repo *workerRepository) UpdateLastSeen(ctx context.Context, name string) error {
	const op = "workerRepository.UpdateLastSeen"

	query := `
		UPDATE 
			workers
		SET 
			last_seen = now()
		WHERE 
			name = $1;`

	res, err := repo.pool.Exec(ctx, query, name)
	if err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}

	if res.RowsAffected() == 0 {
		return models.ErrOrderNotFound
	}

	return nil
}

func (repo *workerRepository) IncrOrdersProcessed(ctx context.Context, name string) error {
	const op = "workerRepository.IncrOrdersProcessed"

	query := `
		UPDATE 
			workers
		SET 
			orders_processed = orders_processed + 1
		WHERE 
			name = $1;`

	res, err := repo.pool.Exec(ctx, query, name)
	if err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}

	if res.RowsAffected() == 0 {
		return models.ErrOrderNotFound
	}

	return nil
}

func (repo *workerRepository) MarkOffline(ctx context.Context, name string) error {
	const op = "workerRepository.MarkOffline"

	query := `
		UPDATE 
			workers
		SET 
			status = 'offline',
			last_seen = now()
		WHERE 
			name = $1;`

	res, err := repo.pool.Exec(ctx, query, name)
	if err != nil {
		return fmt.Errorf("%s: %v", op, err)
	}

	if res.RowsAffected() == 0 {
		return models.ErrOrderNotFound
	}

	return nil
}
