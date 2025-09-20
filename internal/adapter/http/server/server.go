package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"wheres-my-pizza/internal/config"
	"wheres-my-pizza/internal/adapter/http/handler"
	"wheres-my-pizza/internal/domain/types"
	"wheres-my-pizza/pkg/logger"
)

const serverIPAddress = "%s:%d"

type API struct {
	mode   types.ServiceMode
	mux    *http.ServeMux
	server *http.Server
	routes *handlers // routes/handlers

	addr string
	cfg  config.HTTPServer
	log  logger.Logger
}

type handlers struct {
	order    *handler.Order
	tracking *handler.Tracking
}

func New(cfg config.Config, orderService handler.OrderService, trackingService handler.TrackingService, logger logger.Logger) *API {
	addr := fmt.Sprintf(serverIPAddress, "0.0.0.0", cfg.HTTPServer.Port)

	handlers := &handlers{
		order:    handler.NewOrder(orderService, logger),
		tracking: handler.NewTracking(trackingService, logger),
	}

	api := &API{
		mode: cfg.Mode,

		mux:    http.NewServeMux(),
		routes: handlers,

		addr: addr,
		cfg:  cfg.HTTPServer,
		log:  logger,
	}

	api.server = &http.Server{
		Addr:    api.addr,
		Handler: api.mux,
	}

	api.setupRoutes()

	return api
}

func (a *API) Stop(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	a.log.Debug(ctx, "http_server_stop", "shutting down HTTP server...", "address", a.addr)
	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("error shutting down server: %w", err)
	}
	a.log.Debug(ctx, "http_server_stop", "shutting down HTTP server completed")

	return nil
}

func (a *API) Run(ctx context.Context, errCh chan<- error) {
	go func() {
		a.log.Info(ctx, "http_server_run", "started http server", "address", a.addr)
		if err := http.ListenAndServe(a.addr, a.withMiddleware()); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("failed to start HTTP server: %w", err)
			return
		}
	}()
}
