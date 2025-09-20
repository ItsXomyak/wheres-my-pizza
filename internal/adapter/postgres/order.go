package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"wheres-my-pizza/internal/domain/models"
	"wheres-my-pizza/internal/domain/types"
)

type orderRepository struct {
	pool *pgxpool.Pool
}

func NewOrderRepo(pool *pgxpool.Pool) *orderRepository {
	return &orderRepository{
		pool: pool,
	}
}

func (r *orderRepository) Create(ctx context.Context, req *models.CreateOrder, changedBy, notes string) (*models.Order, error) {
	var order models.Order

	// Start a transaction
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert the order
	err = tx.QueryRow(ctx,
		`INSERT INTO orders (
			number, 
			customer_name, 
			type, 
			table_number, 
			delivery_address, 
			total_amount, 
			priority, 
			status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING 
			id, created_at, updated_at, number, customer_name, 
			type, table_number, delivery_address, total_amount, 
			priority, status, processed_by, completed_at`,
		req.Number,
		req.CustomerName,
		req.Type,
		req.TableNumber,
		req.DeliveryAddress,
		req.TotalAmount,
		req.Priority,
		req.Status,
	).Scan(
		&order.ID,
		&order.CreatedAt,
		&order.UpdatedAt,
		&order.Number,
		&order.CustomerName,
		&order.Type,
		&order.TableNumber,
		&order.DeliveryAddress,
		&order.TotalAmount,
		&order.Priority,
		&order.Status,
		&order.ProcessedBy,
		&order.CompletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Insert order items
	for _, item := range req.Items {
		_, err := tx.Exec(ctx,
			`INSERT INTO order_items (
				order_id, 
				name, 
				quantity, 
				price
			) VALUES ($1, $2, $3, $4)`,
			order.ID,
			item.Name,
			item.Quantity,
			item.Price,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create order item: %w", err)
		}
	}

	// Log initial status
	_, err = tx.Exec(ctx,
		`INSERT INTO order_status_log (
			order_id,
			status,
			changed_by,
			notes
		) VALUES ($1, $2, $3, $4)`,
		order.ID,
		types.StatusOrderReceived, // 'received'
		changedBy,
		notes,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to log initial order status: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &order, nil
}

func (r *orderRepository) GetAndIncrementSequence(ctx context.Context, date string) (int, error) {
	var seq int

	// Configure transaction with serializable isolation level
	txOptions := pgx.TxOptions{
		IsoLevel:       pgx.Serializable,  // Highest isolation level
		AccessMode:     pgx.ReadWrite,     // Default, but explicit is better
		DeferrableMode: pgx.NotDeferrable, // Not deferrable for this case
	}

	// Start a transaction
	tx, err := r.pool.BeginTx(ctx, txOptions)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Use the new pgx syntax for querying
	err = tx.QueryRow(ctx,
		`INSERT INTO order_sequences (date, last_value) 
		VALUES ($1, 1)
		ON CONFLICT (date) DO UPDATE 
		SET last_value = order_sequences.last_value + 1,
		    updated_at = NOW()
		RETURNING last_value`,
		date,
	).Scan(&seq)
	if err != nil {
		return 0, fmt.Errorf("failed to get/increment sequence: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return seq, nil
}

// SetStatus updates order status and logs it in one transaction.
func (r *orderRepository) SetStatus(ctx context.Context, orderNumber, workerName, status, notes string) (string, error) {
	const op = "orderRepository.SetStatus"

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("%s: %v", op, err)
	}

	query := `
	UPDATE orders AS o
	SET 
		status = $1,
		processed_by = $2,
		updated_at = now()`

	if status == types.StatusOrderReady {
		query += ", completed_at = now()"
	}

	query += `
	FROM orders AS old
	WHERE o.id = old.id
	  AND o.number = $3
	RETURNING old.status AS old_status, o.id;`

	var (
		orderID   int
		oldStatus string
	)
	if err := tx.QueryRow(ctx, query, status, workerName, orderNumber).Scan(&oldStatus, &orderID); err != nil {
		tx.Rollback(ctx)
		if err == pgx.ErrNoRows {
			return "", models.ErrOrderNotFound
		}
		return "", fmt.Errorf("%s: %v", op, err)
	}

	query = `
		INSERT INTO
			order_status_log (order_id, status, changed_by, notes)
		VALUES
			($1, $2, $3, $4);`

	if _, err := tx.Exec(ctx, query, orderID, status, workerName, notes); err != nil {
		tx.Rollback(ctx)
		if err == pgx.ErrNoRows {
			return "", models.ErrOrderNotFound
		}
		return "", fmt.Errorf("%s: %v", op, err)
	}

	return oldStatus, tx.Commit(ctx)
}
