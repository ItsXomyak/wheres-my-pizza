package services

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpserver "wheres-my-pizza/internal/adapter/http/server"
	"wheres-my-pizza/internal/adapter/postgres"
	"wheres-my-pizza/internal/config"
	"wheres-my-pizza/internal/domain/types"
	"wheres-my-pizza/internal/services/tracking"
	"wheres-my-pizza/pkg/logger"
	postgresclient "wheres-my-pizza/pkg/postgres"
)

// ## Feature: Tracking Service
// The Tracking Service provides visibility into the restaurant's operations.
// It offers a read-only HTTP API for external clients (like a customer-facing
// app or an internal dashboard) to query the current status of orders, view an
// order's history, and monitor the status of all kitchen workers. It directly
// queries the database and does not interact with RabbitMQ.
type Tracking struct {
	postgresDB *postgresclient.PostgreDB
	httpServer *httpserver.API

	cfg config.Config
	log logger.Logger
}

func NewTracking(ctx context.Context, cfg config.Config, log logger.Logger) (*Tracking, error) {
	// Postgres database
	db, err := postgresclient.New(ctx, cfg.Postgres)
	if err != nil {
		log.Error(ctx, "db_connect", "failed to connect postgres", err)
		return nil, fmt.Errorf("failed to connect postgres: %v", err)
	}
	log.Info(ctx, types.ActionDBConnected, "connected to the database")

	workerRepo := postgres.NewWorkerRepo(db.Pool)
	statusRepo := postgres.NewStatusRepo(db.Pool)

	trackingService := tracking.NewService(statusRepo, workerRepo, cfg.Services.Tracking.HeartbeatInterval, log)

	api := httpserver.New(cfg, nil, trackingService, log)

	return &Tracking{
		postgresDB: db,
		httpServer: api,
		cfg:        cfg,

		log: log,
	}, nil
}

func (s *Tracking) Start(ctx context.Context) error {
	errCh := make(chan error, 1)

	s.httpServer.Run(ctx, errCh)

	defer func() {
		s.close(ctx)
		s.log.Info(ctx, types.ActionGracefulShutdown, "tracking service closed!")
	}()

	// Waiting signal
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)

	s.log.Info(ctx, types.ActionServiceStarted, "service started")

	select {
	case errRun := <-errCh:
		return errRun
	case sig := <-shutdownCh:
		s.log.Info(ctx, types.ActionGracefulShutdown, "shuting down application", "signal", sig.String())
		return nil
	}
}

func (s *Tracking) close(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	if err := s.httpServer.Stop(ctx); err != nil {
		s.log.Warn(ctx, types.ActionGracefulShutdown, "failed to shutdown HTTP server")
	}

	s.postgresDB.Pool.Close()
}
