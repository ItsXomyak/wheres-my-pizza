package order

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"restaurant-system/internal/logger"
	"restaurant-system/internal/models"
)

// Handler handles HTTP requests for the order service
type Handler struct {
	service *Service
	logger  *logger.Logger
}

// NewHandler creates a new order handler
func NewHandler(service *Service, log *logger.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  log,
	}
}

// CreateOrder handles POST /orders requests
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	requestID := logger.GenerateRequestID()
	
	// Only accept POST method
	if r.Method != http.MethodPost {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", requestID)
		return
	}

	// Only accept JSON content
	if r.Header.Get("Content-Type") != "application/json" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Content-Type must be application/json", requestID)
		return
	}

	h.logger.Debug("order_received", "Received order creation request", requestID, map[string]interface{}{
		"content_length": r.ContentLength,
		"remote_addr":    r.RemoteAddr,
	})

	// Parse request body
	var req models.CreateOrderRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	
	if err := decoder.Decode(&req); err != nil {
		h.logger.Error("validation_failed", "Failed to parse request body", requestID, err, nil)
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON format", requestID)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		h.logger.Error("validation_failed", "Request validation failed", requestID, err, map[string]interface{}{
			"customer_name": req.CustomerName,
			"order_type":    req.OrderType,
		})
		h.writeErrorResponse(w, http.StatusBadRequest, err.Error(), requestID)
		return
	}

	// Process order with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	response, err := h.service.CreateOrder(ctx, &req, requestID)
	if err != nil {
		h.logger.Error("order_creation_failed", "Failed to create order", requestID, err, map[string]interface{}{
			"customer_name": req.CustomerName,
			"order_type":    req.OrderType,
		})
		h.writeErrorResponse(w, http.StatusInternalServerError, "Internal server error", requestID)
		return
	}

	h.logger.Debug("order_created", "Order created successfully", requestID, map[string]interface{}{
		"order_number": response.OrderNumber,
		"total_amount": response.TotalAmount,
	})

	// Write successful response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("response_encoding_failed", "Failed to encode response", requestID, err, nil)
	}
}

// HealthCheck handles GET /health requests
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", "")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Check database and messaging health
	healthy := h.service.HealthCheck(ctx)
	
	response := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "order-service",
		"healthy":   healthy,
	}

	w.Header().Set("Content-Type", "application/json")
	
	if healthy {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		response["status"] = "unhealthy"
	}

	json.NewEncoder(w).Encode(response)
}

// writeErrorResponse writes an error response in JSON format
func (h *Handler) writeErrorResponse(w http.ResponseWriter, statusCode int, message, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResponse := map[string]interface{}{
		"error":      message,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
		"request_id": requestID,
	}

	json.NewEncoder(w).Encode(errorResponse)
}

// SetupRoutes sets up the HTTP routes
func (h *Handler) SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	
	// Add request logging middleware
	mux.HandleFunc("/orders", h.withLogging(h.CreateOrder))
	mux.HandleFunc("/health", h.withLogging(h.HealthCheck))
	
	return mux
}

// withLogging adds request logging middleware
func (h *Handler) withLogging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := logger.GenerateRequestID()
		
		// Add request ID to context
		ctx := context.WithValue(r.Context(), "request_id", requestID)
		r = r.WithContext(ctx)
		
		// Log request
		h.logger.Debug("request_started", 
			fmt.Sprintf("%s %s", r.Method, r.URL.Path),
			requestID, 
			map[string]interface{}{
				"method":     r.Method,
				"path":       r.URL.Path,
				"remote_addr": r.RemoteAddr,
				"user_agent": r.Header.Get("User-Agent"),
			})
		
		// Create a response writer that captures status code
		rw := &responseWriter{ResponseWriter: w, statusCode: 200}
		
		// Call the handler
		next(rw, r)
		
		// Log response
		duration := time.Since(start)
		h.logger.Debug("request_completed",
			fmt.Sprintf("%s %s - %d", r.Method, r.URL.Path, rw.statusCode),
			requestID,
			map[string]interface{}{
				"method":      r.Method,
				"path":        r.URL.Path,
				"status_code": rw.statusCode,
				"duration_ms": duration.Milliseconds(),
			})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
