package db

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DBPool *pgxpool.Pool

func InitDB(ctx context.Context) {
	poolConfig, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to parse DATABASE_URL: %v\n", err) // rewrite with logger
		os.Exit(1)
	}
	poolConfig.MinConns = int32(4)

	DBPool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create connection pool: %v\n", err) // rewrite with logger
		os.Exit(1)
	}
	defer DBPool.Close()

	if err := DBPool.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err) // rewrite with logger
		os.Exit(1)
	}

	slog.Info("Connected to database") // rewrite with logger
}

func CloseDB() {
	if DBPool != nil {
		DBPool.Close()
	}
}
