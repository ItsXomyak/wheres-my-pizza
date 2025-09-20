package config

import (
	"flag"
	"fmt"
)

const HelpMessage = `Usage: ./restaurant-system --mode=<service> [options]

Available Services:
  order-service           - HTTP API for order management
  kitchen-worker          - Kitchen order processing service
  tracking-service        - Order tracking API
  notification-subscriber - Status update subscriber

Common Flags:
  --help                  - Show this help message
  --config                - Path to config file (default: config.yaml)
  --log-level             - Defines logger level (DEBUG, INFO, WARN, ERROR)

Service-Specific Flags:

Order Service:
  --port           - HTTP port (default: 3000)
  --max-concurrent - Max concurrent orders (default: 50)

Kitchen Worker:
  --worker-name        - Unique worker identifier (required)
  --order-types        - Comma-separated order types (dine_in,takeout,delivery)
  --heartbeat-interval - Worker heartbeat in seconds (default: 30)
  --prefetch           - RabbitMQ prefetch count (default: 1)

Tracking Service:
  --port - HTTP port (default: 3002)

Examples:
  ./restaurant-system --mode=order-service --port=3000 --max-concurrent 50

  ./restaurant-system --mode=kitchen-worker --worker-name="gordon_ramsay" --order-types="dine_in" --heartbeat-interval=30 --prefetch=1
  ./restaurant-system --mode=kitchen-worker --worker-name="gordon_ramsay" --order-types="dine_in,takeout" --heartbeat-interval=30 --prefetch=1
  ./restaurant-system --mode=kitchen-worker --worker-name="gordon_ramsay" --order-types="dine_in,takeout,delivery" --heartbeat-interval=30 --prefetch=1

  ./restaurant-system --mode=tracking-service --port=3002
  ./restaurant-system --mode=notification-subscriber
`

func PrintHelp() {
	if HelpMessage != "" {
		fmt.Printf("%s", HelpMessage)
	} else {
		flag.Usage()
	}
}
