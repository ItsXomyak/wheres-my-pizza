package database

// Order queries
const (
	InsertOrderSQL = `
		INSERT INTO orders (number, customer_name, type, table_number, delivery_address, total_amount, priority)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`

	InsertOrderItemSQL = `
		INSERT INTO order_items (order_id, name, quantity, price)
		VALUES ($1, $2, $3, $4)`

	InsertOrderStatusLogSQL = `
		INSERT INTO order_status_log (order_id, status, changed_by, notes)
		VALUES ($1, $2, $3, $4)`

	UpdateOrderStatusSQL = `
		UPDATE orders SET status = $1, processed_by = $2, updated_at = NOW()
		WHERE number = $3`

	UpdateOrderCompletedSQL = `
		UPDATE orders SET status = $1, completed_at = NOW(), updated_at = NOW()
		WHERE number = $2`

	GetOrderByNumberSQL = `
		SELECT id, number, customer_name, type, table_number, delivery_address, 
			   total_amount, priority, status, processed_by, created_at, updated_at, completed_at
		FROM orders WHERE number = $1`

	GetOrderStatusHistorySQL = `
		SELECT status, changed_by, changed_at, notes
		FROM order_status_log
		WHERE order_id = (SELECT id FROM orders WHERE number = $1)
		ORDER BY changed_at ASC`

	GetNextOrderNumberSQL = `
		SELECT COALESCE(MAX(CAST(SUBSTRING(number FROM 'ORD_[0-9]{8}_([0-9]{3})') AS INTEGER)), 0) + 1
		FROM orders 
		WHERE number LIKE $1`
)

// Worker queries
const (
	InsertWorkerSQL = `
		INSERT INTO workers (name, type, status)
		VALUES ($1, $2, 'online')
		ON CONFLICT (name) DO UPDATE SET
			status = 'online',
			last_seen = NOW()
		RETURNING id`

	UpdateWorkerStatusSQL = `
		UPDATE workers SET status = $1, last_seen = NOW()
		WHERE name = $2`

	UpdateWorkerHeartbeatSQL = `
		UPDATE workers SET last_seen = NOW(), orders_processed = orders_processed + $1
		WHERE name = $2`

	GetWorkerByNameSQL = `
		SELECT id, name, type, status, last_seen, orders_processed
		FROM workers WHERE name = $1`

	GetAllWorkersSQL = `
		SELECT name, type, status, orders_processed, last_seen, created_at
		FROM workers
		ORDER BY created_at ASC`

	CheckWorkerOnlineSQL = `
		SELECT COUNT(*) FROM workers WHERE name = $1 AND status = 'online'`
)
