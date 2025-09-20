package app

import (
	"context"
	"errors"
	"fmt"

	svc "wheres-my-pizza/internal/app/services"
	"wheres-my-pizza/internal/config"
	"wheres-my-pizza/internal/domain/types"
	"wheres-my-pizza/pkg/logger"
)

var ErrInvalidMode = errors.New("invalid mode")

type Service interface {
	Start(ctx context.Context) error
}

type App struct {
	mode    types.ServiceMode
	service Service

	cfg config.Config
	log logger.Logger
}

// NewApplication
func NewApplication(ctx context.Context, cfg config.Config, log logger.Logger) (*App, error) {
	app := &App{
		mode: cfg.Mode,
		cfg:  cfg,
		log:  log,
	}

	if err := app.initService(ctx, app.mode); err != nil {
		return nil, err
	}

	return app, nil
}

func (app *App) Run(ctx context.Context) error {
	if app.service == nil {
		if err := app.initService(ctx, app.mode); err != nil {
			return err
		}
	}

	if err := app.service.Start(ctx); err != nil {
		return err
	}

	return nil
}

func (app *App) initService(ctx context.Context, mode types.ServiceMode) error {
	var (
		service Service
		err     error
	)
	switch mode {
	case types.ModeOrder:
		service, err = svc.NewOrder(ctx, app.cfg, app.log)
	case types.ModeKitchenWorker:
		service, err = svc.NewKitchen(ctx, app.cfg, app.log)
	case types.ModeTracking:
		service, err = svc.NewTracking(ctx, app.cfg, app.log)
	case types.ModeNotificationSubscriber:
		service, err = svc.NewNotificationSubscriber(ctx, app.cfg, app.log)
	default:
		return ErrInvalidMode
	}

	if err != nil {
		return fmt.Errorf("failed to init service: %w", err)
	}

	app.service = service

	return nil
}
