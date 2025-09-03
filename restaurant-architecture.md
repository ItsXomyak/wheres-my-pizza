# Restaurant Order Management System - Architecture & Implementation Guide

## 1. System Architecture Overview

### High-Level Architecture Principles
- **Single Responsibility**: Each service has one clear purpose
- **Event-Driven Architecture**: Services communicate via message queues
- **Database Per Service**: Each service owns its data operations
- **Graceful Degradation**: System continues operating even if some components fail

### Component Interaction Flow
```
HTTP Client → Order Service → Database + RabbitMQ
                ↓
Kitchen Workers ← RabbitMQ Queue → Database Updates
                ↓
RabbitMQ Fanout → Notification Subscribers
                ↓
Tracking Service ← Database (Read-Only)
```

## 2. Project Structure Design

```
restaurant-system/
├── cmd/
│   └── main.go                    # Entry point with flag parsing
├── internal/
│   ├── config/
│   │   └── config.go              # YAML configuration loading
│   ├── database/
│   │   ├── connection.go          # PostgreSQL connection management
│   │   ├── migrations.go          # Database migration runner
│   │   └── queries.go             # SQL query constants
│   ├── messaging/
│   │   ├── connection.go          # RabbitMQ connection management
│   │   ├── publisher.go           # Message publishing utilities
│   │   └── consumer.go            # Message consumption utilities
│   ├── logger/
│   │   └── logger.go              # Structured JSON logging
│   ├── models/
│   │   ├── order.go               # Order-related structs
│   │   ├── worker.go              # Worker-related structs
│   │   └── message.go             # Message format structs
│   └── services/
│       ├── order/
│       │   ├── handler.go         # HTTP handlers
│       │   ├── service.go         # Business logic
│       │   └── validation.go      # Input validation
│       ├── kitchen/
│       │   ├── worker.go          # Worker implementation
│       │   └── processor.go       # Order processing logic
│       ├── tracking/
│       │   ├── handler.go         # HTTP handlers
│       │   └── service.go         # Query logic
│       └── notification/
│           └── subscriber.go      # Notification handling
├── migrations/
│   ├── 001_create_orders.sql
│   ├── 002_create_order_items.sql
│   ├── 003_create_order_status_log.sql
│   └── 004_create_workers.sql
├── config.yaml
├── go.mod
└── README.md
```

## 3. Core Components Architecture

### 3.1 Configuration Management
- **Pattern**: Centralized configuration with environment overrides
- **Implementation**: 
  - YAML file parsing with `gopkg.in/yaml.v3`
  - Environment variable overrides for sensitive data
  - Validation of required fields at startup
  - Connection string builders for DB and RabbitMQ

### 3.2 Database Layer
- **Pattern**: Repository pattern with transaction management
- **Key Components**:
  - Connection pool management with retry logic
  - Transaction wrapper for multi-table operations
  - Migration runner for schema management
  - Query builders for complex operations (order number generation)

### 3.3 Messaging Layer
- **Pattern**: Publisher/Subscriber with connection recovery
- **Exchange Strategy**:
  - `orders_topic` (Topic Exchange) - for routing orders to specialized workers
  - `notifications_fanout` (Fanout Exchange) - for broadcasting status updates
- **Queue Strategy**:
  - `kitchen_queue` - general orders
  - `kitchen_dine_in_queue`, `kitchen_takeout_queue`, `kitchen_delivery_queue` - specialized
  - `notifications_queue` - for subscribers

### 3.4 Logging System
- **Pattern**: Structured logging with correlation IDs
- **Implementation**:
  - JSON formatter with required fields
  - Request ID propagation across services
  - Error object with stack traces
  - Performance metrics logging

## 4. Step-by-Step Implementation Guide

### Phase 1: Foundation (Days 1-2)
1. **Project Setup**
   - Initialize Go module
   - Set up project structure
   - Create basic configuration system
   - Implement structured logging

2. **Database Foundation**
   - Create migration system
   - Implement connection management
   - Set up transaction utilities
   - Test database connectivity

3. **Message Broker Foundation**
   - Implement RabbitMQ connection management
   - Create exchange and queue declarations
   - Build publisher and consumer abstractions
   - Test messaging connectivity

### Phase 2: Core Models & Validation (Day 3)
1. **Data Models**
   - Define order, worker, and message structs
   - Implement JSON serialization tags
   - Create validation functions
   - Build helper methods (order number generation)

2. **Business Logic Layer**
   - Order validation with all business rules
   - Priority calculation logic
   - Status transition management
   - Error handling strategies

### Phase 3: Order Service (Days 4-5)
1. **HTTP Server Setup**
   - Implement HTTP router (use standard library)
   - Create middleware for logging and request IDs
   - Set up graceful shutdown
   - Add health check endpoint

2. **Order Processing Pipeline**
   - HTTP request validation
   - Database transaction handling
   - Message publishing
   - Error response formatting

3. **Testing & Integration**
   - Unit tests for validation logic
   - Integration tests with database
   - Message publishing verification

### Phase 4: Kitchen Worker Service (Days 6-7)
1. **Worker Registration System**
   - Database registration with duplicate checking
   - Worker specialization logic
   - Heartbeat mechanism
   - Graceful shutdown handling

2. **Message Processing Pipeline**
   - Queue consumption setup
   - Order type filtering
   - Status update transactions
   - Cooking simulation
   - Message acknowledgment

3. **Error Handling & Recovery**
   - Connection recovery
   - Message redelivery logic
   - Dead letter queue handling
   - Idempotency checks

### Phase 5: Tracking Service (Day 8)
1. **Read-Only API Implementation**
   - HTTP handlers for order status
   - Order history retrieval
   - Worker status monitoring
   - JSON response formatting

2. **Query Optimization**
   - Efficient database queries
   - Worker offline detection logic
   - Error handling for missing orders

### Phase 6: Notification Service (Day 9)
1. **Subscriber Implementation**
   - Fanout queue consumption
   - Message parsing and display
   - Multiple subscriber support
   - Graceful shutdown

### Phase 7: Integration & Testing (Day 10)
1. **End-to-End Testing**
   - Full workflow testing
   - Multiple worker scenarios
   - Error condition testing
   - Performance verification

2. **Documentation & Deployment**
   - README updates
   - Configuration examples
   - Docker setup (optional)
   - Monitoring guidelines

## 5. Key Implementation Strategies

### 5.1 Connection Management
- **Database**: Use `pgxpool` for connection pooling
- **RabbitMQ**: Implement connection recovery with exponential backoff
- **Health Checks**: Regular connectivity testing

### 5.2 Error Handling Strategy
- **Validation Errors**: Return structured error responses
- **Database Errors**: Log and return generic 500 responses
- **Message Errors**: Use nack with requeue for recoverable errors
- **Dead Letter Queues**: For permanently failed messages

### 5.3 Concurrency Patterns
- **Order Service**: Use semaphore to limit concurrent requests
- **Kitchen Workers**: Prefetch count for load balancing
- **Tracking Service**: Read-only queries with connection pooling

### 5.4 Data Consistency
- **Order Creation**: Single transaction for order + items + status log
- **Status Updates**: Transactional updates with message publishing
- **Worker Registration**: Use database constraints for uniqueness

### 5.5 Message Design Patterns
- **Work Queue**: Kitchen workers compete for orders
- **Fanout**: All notification subscribers receive updates
- **Topic Routing**: Orders routed by type and priority
- **Dead Letter**: Failed messages for manual inspection

## 6. Advanced Considerations

### 6.1 Scalability Patterns
- Multiple instances of each service
- Queue partitioning for high throughput
- Database read replicas for tracking service
- Load balancing strategies

### 6.2 Monitoring & Observability
- Structured logging with correlation IDs
- Metrics collection (order processing times, queue depths)
- Health check endpoints
- Error rate monitoring

### 6.3 Security Considerations
- Input validation and sanitization
- SQL injection prevention
- Message authentication
- Connection security (TLS)

### 6.4 Testing Strategy
- Unit tests for business logic
- Integration tests for database operations
- Message flow testing
- Load testing for concurrent scenarios

## 7. Implementation Tips

### 7.1 Code Organization
- Keep business logic separate from HTTP/messaging layers
- Use interfaces for testability
- Implement proper error wrapping
- Follow Go naming conventions

### 7.2 Performance Optimization
- Use prepared statements for repeated queries
- Implement connection pooling properly
- Batch operations where possible
- Monitor and optimize slow queries

### 7.3 Operational Excellence
- Implement proper logging levels
- Use configuration for all environment-specific values
- Handle graceful shutdowns
- Implement circuit breakers for external dependencies

This architecture provides a solid foundation for a production-ready distributed system while keeping the implementation manageable and educational.