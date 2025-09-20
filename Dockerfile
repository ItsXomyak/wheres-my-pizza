FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o restaurant-system .

# Final stage
FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/restaurant-system .
COPY config.yaml config.yaml  

CMD ["./restaurant-system"]
