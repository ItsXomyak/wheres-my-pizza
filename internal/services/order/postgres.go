package order

import (
	"context"
	"where-is-my-pizza/adapter/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DBPool *pgxpool.Pool
}

func (r *Repository) InitPostgres(ctx context.Context) Repository {
	DBPool := db.NewDBPool(ctx)
	return Repository{DBPool: DBPool}
}

func (r *Repository) ClosePostgres() {
	db.CloseDB(r.DBPool)
}

func (r *Repository) InsertOrder(ctx context.Context, orderRequest *OrderRequest, orderResponse *OrderResponse, priority int) error {

	return nil
}

func (r *Repository) QueryRow(query string, args ...interface{}) pgxpool.Row {
	return r.DBPool.QueryRow(context.Background(), query, args...)
}
