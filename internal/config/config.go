package config

import (
	"errors"
	"flag"
	"fmt"
	"time"

	"wheres-my-pizza/internal/domain/types"
	"wheres-my-pizza/pkg/configparser"
	"wheres-my-pizza/pkg/logger"
	"wheres-my-pizza/pkg/postgres"
	"wheres-my-pizza/pkg/rabbit"
)

const (
	DefaultOrderServicePort    = 3000
	DefaultTrackingServicePort = 3002
)

var (
	// General
	modeFlag = flag.String("mode", "", "application mode")
	portFlag = flag.Int("port", -1, "The HTTP port for the API")
	logLevel = flag.String("log-level", logger.LevelDebug, "Logger level. (DEBUG, INFO, WARN, ERROR)")

	// Order service
	maxConcurrent = flag.Int("max-concurrent", 50, "Maximum number of concurrent orders to process.")

	// Kitchen service
	workerName   = flag.String("worker-name", "", "unique name for the worker (e.g., chef_mario) (required)")
	orderTypes   = flag.String("order-types", "", "comma-separated list of order types the worker can handle (e.g., dine_in,takeout)")
	heartbeatInt = flag.Int("heartbeat-interval", 30, "interval (seconds) between heartbeats")
	prefetch     = flag.Int("prefetch", 1, "RabbitMQ prefetch count")
)

var (
	ErrModeNotProvided = errors.New("mode flag not provided")
	ErrInvalidModeFlag = errors.New("invalid mode flag")
)

type (
	// Config
	Config struct {
		Mode       types.ServiceMode
		Services   Services
		HTTPServer HTTPServer
		Postgres   postgres.Config
		RabbitMQ   RabbitMQ

		LogLevel string
	}

	Services struct {
		Order    OrderService
		Kitchen  KitchenService
		Tracking TrackingService
	}

	// HTTP service
	HTTPServer struct {
		Port int
	}

	OrderService struct {
		MaxConcurrent int
		SemWait       time.Duration `env:"ORDER_SEMWAIT" default:"1s"`
	}

	TrackingService struct {
		HeartbeatInterval int
	}

	KitchenService struct {
		WorkerName        string
		OrderTypes        string
		Prefetch          int
		HeartbeatInterval int
		ReconnectAttempt  int           `env:"KITCHEN_RECONNECT_ATTEMPT" default:"5"`
		ReconnectDelay    time.Duration `env:"KITCHEN_RECONNECT_DELAY" default:"1s"`
	}

	RabbitMQ struct {
		Conn                  rabbit.Config
		OrderExchange         string        `env:"RABBITMQ_ORDER_EXCHANGE" default:"orders_topic"`
		NotificationsExchange string        `env:"RABBITMQ_NOTIFICATIONS_EXCHANGE" default:"notifications_fanout"`
		ReconnectAttempt      int           `env:"RABBITMQ_RECONNECT_ATTEMPT" default:"5"`
		ReconnectDelay        time.Duration `env:"RABBITMQ_RECONNECT_DELAY" default:"1s"`
	}
)

func New(filepath string) (*Config, error) {
	cfg := &Config{}

	if err := configparser.LoadAndParseYaml(filepath, cfg); err != nil {
		return nil, err
	}

	if err := parseFlags(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func parseFlags(cfg *Config) error {
	if modeFlag == nil {
		return ErrModeNotProvided
	}

	cfg.LogLevel = *logLevel
	if logLevel == nil || *logLevel == "" {
		cfg.LogLevel = logger.LevelDebug
	}

	if err := validateLogLevel(cfg.LogLevel); err != nil {
		return err
	}

	cfg.Mode = types.ServiceMode(*modeFlag)

	switch cfg.Mode {
	case types.ModeOrder:
		if portFlag == nil || *portFlag < 1024 || *portFlag > 65535 {
			cfg.HTTPServer.Port = DefaultOrderServicePort
		} else {
			cfg.HTTPServer.Port = *portFlag
		}

		if maxConcurrent != nil && !(*maxConcurrent >= 0 && *maxConcurrent <= 1000) {
			return errors.New("--max-concurrent flag must be between 1 and 1000")
		}
		cfg.Services.Order.MaxConcurrent = *maxConcurrent
	case types.ModeKitchenWorker:
		if workerName == nil || *workerName == "" {
			return errors.New("missing required flag: --worker-name")
		}

		cfg.Services.Kitchen.WorkerName = *workerName
		cfg.Services.Kitchen.OrderTypes = *orderTypes
		cfg.Services.Kitchen.HeartbeatInterval = *heartbeatInt
		cfg.Services.Kitchen.Prefetch = *prefetch
	case types.ModeTracking:
		if portFlag != nil {
			cfg.HTTPServer.Port = *portFlag
		} else {
			cfg.HTTPServer.Port = DefaultTrackingServicePort
		}
		cfg.Services.Tracking.HeartbeatInterval = *heartbeatInt
	case types.ModeNotificationSubscriber:
	default:
		return ErrInvalidModeFlag
	}

	return nil
}

func validateLogLevel(lvl string) error {
	switch lvl {
	case logger.LevelDebug, logger.LevelError, logger.LevelWarn, logger.LevelInfo:
		return nil
	default:
		return fmt.Errorf("invalid log level: %s", lvl)
	}
}
