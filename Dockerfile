# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o restaurant-system .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS calls
RUN apk --no-cache add ca-certificates tzdata

# Create app directory
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/restaurant-system .

# Copy configuration file
COPY --from=builder /app/config.yaml .

# Copy migrations
COPY --from=builder /app/migrations ./migrations/

# Make sure the binary is executable
RUN chmod +x ./restaurant-system

# Expose port (will be overridden by docker-compose for different services)
EXPOSE 3000

# Default command
CMD ["./restaurant-system"]
