package order

import (
	"context"
	"where-is-my-pizza/adapter/db"

	"github.com/jackc/pgx/v5"
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
	var id int
	err := r.DBPool.QueryRow(ctx,
		`"INSERT INTO orders (number, customer_name, type, table_number, delivery_address, total_amount, 
	priority, status, processed_by) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id"`,
		orderResponse.OrderNumber, orderRequest.CustomerName, orderRequest.OrderType, orderRequest.TableNumber,
		orderRequest.DeliveryAddr, orderResponse.TotalAmount, priority, orderResponse.Status, "order service").Scan(&id)
	if err != nil {
	}
	return nil
}

func (r *Repository) GetOrderNumber(ctx context.Context) (int, error) {
	var orderCounter int
	err := r.DBPool.QueryRow(ctx, "SELECT COUNT(*) FROM orders WHERE DATE(created_at) = CURRENT_DATE").Scan(&orderCounter)
	if err != nil {
		return 0, err
	}
	return orderCounter, nil
}

func (r *Repository) QueryRow(query string, args ...interface{}) pgx.Row {
	return r.DBPool.QueryRow(context.Background(), query, args...)
}
