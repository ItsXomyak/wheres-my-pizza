package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"restaurant-system/internal/config"
	"restaurant-system/internal/database"
	"restaurant-system/internal/logger"
	"restaurant-system/internal/messaging"
	"restaurant-system/internal/services/order"
)

func main() {
	// Parse command line flags
	var (
		mode            = flag.String("mode", "", "Service mode (order-service, kitchen-worker, tracking-service, notification-subscriber)")
		port            = flag.Int("port", 3000, "HTTP port")
		maxConcurrent   = flag.Int("max-concurrent", 50, "Maximum concurrent operations")
		workerName      = flag.String("worker-name", "", "Worker name (required for kitchen-worker mode)")
		orderTypes      = flag.String("order-types", "", "Comma-separated order types for worker specialization")
		heartbeatInterval = flag.Int("heartbeat-interval", 30, "Heartbeat interval in seconds")
		prefetch        = flag.Int("prefetch", 1, "RabbitMQ prefetch count")
	)
	flag.Parse()

	// Validate required mode flag
	if *mode == "" {
		fmt.Fprintf(os.Stderr, "Error: --mode flag is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Create logger
	log := logger.New(*mode)
	requestID := logger.GenerateRequestID()

	log.Info("service_started", fmt.Sprintf("Starting %s", *mode), requestID, map[string]interface{}{
		"mode":            *mode,
		"port":            *port,
		"max_concurrent":  *maxConcurrent,
	})

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info("graceful_shutdown", "Received shutdown signal", requestID, nil)
		cancel()
	}()

	// Route to appropriate service
	switch *mode {
	case "order-service":
		if err := runOrderService(ctx, cfg, log, *port, *maxConcurrent); err != nil {
			log.Error("service_failed", "Order service failed", requestID, err, nil)
			os.Exit(1)
		}
	case "kitchen-worker":
		if *workerName == "" {
			log.Error("validation_failed", "worker-name is required for kitchen-worker mode", requestID, nil, nil)
			os.Exit(1)
		}
		if err := runKitchenWorker(ctx, cfg, log, *workerName, *orderTypes, *heartbeatInterval, *prefetch); err != nil {
			log.Error("service_failed", "Kitchen worker failed", requestID, err, nil)
			os.Exit(1)
		}
	case "tracking-service":
		if err := runTrackingService(ctx, cfg, log, *port); err != nil {
			log.Error("service_failed", "Tracking service failed", requestID, err, nil)
			os.Exit(1)
		}
	case "notification-subscriber":
		if err := runNotificationSubscriber(ctx, cfg, log); err != nil {
			log.Error("service_failed", "Notification subscriber failed", requestID, err, nil)
			os.Exit(1)
		}
	default:
		log.Error("validation_failed", fmt.Sprintf("Unknown mode: %s", *mode), requestID, nil, nil)
		os.Exit(1)
	}

	log.Info("service_stopped", "Service stopped gracefully", requestID, nil)
}

// runOrderService runs the order service
func runOrderService(ctx context.Context, cfg *config.Config, log *logger.Logger, port, maxConcurrent int) error {
	requestID := logger.GenerateRequestID()

	// Initialize database
	db, err := database.New(cfg, log)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer db.Close()

	log.Info("db_connected", "Connected to PostgreSQL database", requestID, nil)

	// Run database migrations
	if err := db.RunMigrations(ctx, "migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize messaging
	conn, err := messaging.New(cfg, log)
	if err != nil {
		return fmt.Errorf("failed to initialize messaging: %w", err)
	}
	defer conn.Close()

	log.Info("rabbitmq_connected", "Connected to RabbitMQ", requestID, nil)

	publisher := messaging.NewPublisher(conn, log)

	// Initialize service and handler
	service := order.NewService(db, publisher, log, maxConcurrent)
	handler := order.NewHandler(service, log)

	// Setup HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: handler.SetupRoutes(),
	}

	// Start HTTP server in goroutine
	go func() {
		log.Info("service_started", fmt.Sprintf("Order Service started on port %d", port), requestID, map[string]interface{}{
			"port":           port,
			"max_concurrent": maxConcurrent,
		})
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server_failed", "HTTP server failed", requestID, err, nil)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	return server.Shutdown(shutdownCtx)
}

// Placeholder functions for other services
func runKitchenWorker(ctx context.Context, cfg *config.Config, log *logger.Logger, workerName, orderTypes string, heartbeatInterval, prefetch int) error {
	log.Info("service_not_implemented", "Kitchen worker not yet implemented", "", nil)
	<-ctx.Done()
	return nil
}

func runTrackingService(ctx context.Context, cfg *config.Config, log *logger.Logger, port int) error {
	log.Info("service_not_implemented", "Tracking service not yet implemented", "", nil)
	<-ctx.Done()
	return nil
}

func runNotificationSubscriber(ctx context.Context, cfg *config.Config, log *logger.Logger) error {
	log.Info("service_not_implemented", "Notification subscriber not yet implemented", "", nil)
	<-ctx.Done()
	return nil
}
