package cmd

import (
	"context"
	"flag"
	"log"
	"os"

	"wheres-my-pizza/internal/app"
	"wheres-my-pizza/internal/config"
	"wheres-my-pizza/pkg/logger"
)

var (
	helpFlag   = flag.Bool("help", false, "Show help message")
	configPath = flag.String("config-path", "config.yaml", "Path to the config yaml file")
)

func Run() {
	flag.Parse()
	if *helpFlag {
		config.PrintHelp()
		return
	}

	ctx := context.Background()

	// Init config
	cfg, err := config.New(*configPath)
	if err != nil {
		log.Fatal("failed to configure application", err)
		config.PrintHelp()
	}

	// Init logger
	logger := logger.InitLogger(string(cfg.Mode), cfg.LogLevel)

	config.PrintConfig(cfg)

	// Creating application
	app, err := app.NewApplication(ctx, *cfg, logger)
	if err != nil {
		logger.Error(ctx, "app_init", "failed to init application", err)
		os.Exit(1)
	}

	// Running the apllication
	err = app.Run(ctx)
	if err != nil {
		logger.Error(ctx, "app_run", "failed to run application", err)
		os.Exit(1)
	}
}
