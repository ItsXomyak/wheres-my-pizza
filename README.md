# Restaurant Order Management System

A distributed restaurant order management system built with Go, PostgreSQL, and RabbitMQ. The system demonstrates microservices architecture, message queues, and concurrent programming concepts.

## Architecture

The system consists of the following services:

- **Order Service**: HTTP API for receiving and processing customer orders
- **Kitchen Workers**: Background workers that process orders from the message queue
- **Tracking Service**: Read-only API for order status and history queries  
- **Notification Subscriber**: Listens for order status updates and displays notifications
- **PostgreSQL**: Database for persistent order storage
- **RabbitMQ**: Message broker for asynchronous communication

## Quick Start with Docker

### Prerequisites

- Docker and Docker Compose installed on your system
- At least 4GB of RAM available for containers

### Running the System

1. **Clone and navigate to the project directory:**
   ```bash
   cd where-is-my-pizza
   ```

2. **Start all services using Docker Compose:**
   ```bash
   docker-compose up --build
   ```

   This will:
   - Build the restaurant system application
   - Start PostgreSQL database
   - Start RabbitMQ message broker
   - Start the Order Service (port 3000)
   - Start two Kitchen Workers (chef_mario and chef_luigi)
   - Start the Tracking Service (port 3002)
   - Start the Notification Subscriber

3. **Wait for all services to be healthy** (check the logs for "service_started" messages)

### Testing the System

#### Using the Test Script (Linux/Mac/WSL):

```bash
chmod +x test-order.sh
./test-order.sh
```

#### Manual Testing with curl:

**Create a takeout order:**
```bash
curl -X POST http://localhost:3000/orders \
  -H "Content-Type: application/json" \
  -d '{
        "customer_name": "John Doe",
        "order_type": "takeout",
        "items": [
          {"name": "Margherita Pizza", "quantity": 1, "price": 15.99},
          {"name": "Caesar Salad", "quantity": 1, "price": 8.99}
        ]
      }'
```

**Create a delivery order:**
```bash
curl -X POST http://localhost:3000/orders \
  -H "Content-Type: application/json" \
  -d '{
        "customer_name": "Jane Smith",
        "order_type": "delivery",
        "delivery_address": "123 Main Street, Downtown",
        "items": [
          {"name": "Pepperoni Pizza", "quantity": 2, "price": 18.99}
        ]
      }'
```

**Check order service health:**
```bash
curl http://localhost:3000/health
```

### Monitoring

- **Application Logs**: View logs for each service using:
  ```bash
  docker-compose logs -f [service-name]
  ```
  
- **RabbitMQ Management UI**: http://localhost:15672 (guest/guest)

- **Database**: Connect to PostgreSQL at localhost:5432
  - Database: restaurant_db
  - Username: restaurant_user  
  - Password: restaurant_pass

### Service Endpoints

- **Order Service**: http://localhost:3000
  - POST `/orders` - Create new order
  - GET `/health` - Health check

- **Tracking Service**: http://localhost:3002 *(Not implemented yet)*
  - GET `/orders/{order_number}/status` - Get order status
  - GET `/orders/{order_number}/history` - Get order history
  - GET `/workers/status` - Get worker status

## Development

### Building Locally

```bash
go build -o restaurant-system .
```

### Running Individual Services

```bash
# Order Service
./restaurant-system --mode=order-service --port=3000

# Kitchen Worker
./restaurant-system --mode=kitchen-worker --worker-name=chef_mario

# Tracking Service
./restaurant-system --mode=tracking-service --port=3002

# Notification Subscriber
./restaurant-system --mode=notification-subscriber
```

## System Features

### Implemented ✅

- ✅ Project structure and foundation
- ✅ Database layer with migrations  
- ✅ RabbitMQ messaging layer
- ✅ Core data models and validation
- ✅ Order Service (HTTP API)
- ✅ Configuration management
- ✅ Docker containerization

### In Progress / TODO 🚧

- 🚧 Kitchen Worker Service
- 🚧 Tracking Service  
- 🚧 Notification Service
- 🚧 End-to-end testing

### Order Processing Flow

1. Customer submits order via HTTP POST to Order Service
2. Order Service validates request and stores in PostgreSQL  
3. Order message published to RabbitMQ with routing key based on type/priority
4. Kitchen Workers consume messages and process orders
5. Status updates published to notification exchange
6. Notification Subscribers display order status changes
7. Tracking Service provides read-only API for order status queries

### Message Queue Design

- **Topic Exchange**: `orders_topic` - Routes orders to appropriate kitchen workers
- **Fanout Exchange**: `notifications_fanout` - Broadcasts status updates
- **Queues**: Specialized queues for different order types
- **Priority**: High-value orders get higher priority processing

## Troubleshooting

**Services fail to start:**
- Ensure Docker has enough memory allocated
- Check if ports 3000, 3002, 5432, 5672, 15672 are available
- View logs: `docker-compose logs [service-name]`

**Database connection issues:**
- Wait for PostgreSQL health check to pass
- Verify database credentials in config.yaml

**RabbitMQ connection issues:**
- Wait for RabbitMQ health check to pass
- Check RabbitMQ management UI for queue status

**Reset everything:**
```bash
docker-compose down -v
docker-compose up --build
```

# where-is-my-pizza

