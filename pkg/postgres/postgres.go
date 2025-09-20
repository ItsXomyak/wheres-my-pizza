package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgreDB struct {
	Pool     *pgxpool.Pool
	DBConfig *pgxpool.Config
}

type Config struct {
	Host         string `env:"POSTGRES_HOST"`
	Port         string `env:"POSTGRES_PORT"`
	User         string `env:"POSTGRES_USER"`
	Password     string `env:"POSTGRES_PASSWORD"`
	DBName       string `env:"POSTGRES_DATABASE"`
	MaxOpenConns int32  `env:"POSTGRES_MAX_OPEN_CONN" default:"25"`
	MaxIdleTime  string `env:"POSTGRES_MAX_IDLE_TIME" default:"15m"`
}

func (c Config) GetDsn() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.Host,
		c.Port,
		c.User,
		c.Password,
		c.DBName,
	)
}

func New(ctx context.Context, config Config) (*PostgreDB, error) {
	dbConfig, err := pgxpool.ParseConfig(config.GetDsn())
	if err != nil {
		return nil, err
	}

	// Setting maxOpenConns
	dbConfig.MaxConns = config.MaxOpenConns

	// Use the time.ParseDuration() function to convert the idle timeout duration string
	// to a time.Duration type.
	duration, err := time.ParseDuration(config.MaxIdleTime)
	if err != nil {
		return nil, err
	}

	// Setting MaxConnIdleTime
	dbConfig.MaxConnIdleTime = duration

	pool, err := pgxpool.NewWithConfig(ctx, dbConfig)
	if err != nil {
		return nil, err
	}

	// Ping the database
	if err = pool.Ping(ctx); err != nil {
		return nil, err
	}

	return &PostgreDB{
		Pool:     pool,
		DBConfig: dbConfig,
	}, nil
}
