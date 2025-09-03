package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"restaurant-system/internal/config"
	"restaurant-system/internal/logger"
)

// DB wraps the PostgreSQL connection pool
type DB struct {
	Pool   *pgxpool.Pool
	logger *logger.Logger
}

// New creates a new database connection
func New(cfg *config.Config, log *logger.Logger) (*DB, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL())
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	// Configure connection pool settings
	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = time.Minute * 30

	// Try to connect with retries
	var pool *pgxpool.Pool
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		pool, err = pgxpool.NewWithConfig(context.Background(), poolConfig)
		if err == nil {
			// Test the connection
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err = pool.Ping(ctx)
			cancel()
			if err == nil {
				break
			}
			pool.Close()
		}
		
		if i < maxRetries-1 {
			waitTime := time.Duration(i+1) * 2 * time.Second
			log.Error("db_connection_failed", 
				fmt.Sprintf("Failed to connect to database, retrying in %v", waitTime),
				"startup", err, nil)
			time.Sleep(waitTime)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
	}

	return &DB{
		Pool:   pool,
		logger: log,
	}, nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

// Ping tests the database connection
func (db *DB) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// Begin starts a new transaction
func (db *DB) Begin(ctx context.Context) (pgx.Tx, error) {
	return db.Pool.Begin(ctx)
}

// Exec executes a query without returning any rows
func (db *DB) Exec(ctx context.Context, sql string, args ...interface{}) error {
	_, err := db.Pool.Exec(ctx, sql, args...)
	return err
}

// Query executes a query that returns rows
func (db *DB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return db.Pool.Query(ctx, sql, args...)
}

// QueryRow executes a query that is expected to return at most one row
func (db *DB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return db.Pool.QueryRow(ctx, sql, args...)
}
