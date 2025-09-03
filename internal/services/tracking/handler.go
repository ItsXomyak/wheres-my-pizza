package tracking

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"restaurant-system/internal/logger"
)

// Handler handles HTTP requests for the tracking service
type Handler struct {
	service *Service
	logger  *logger.Logger
}

// NewHandler creates a new tracking handler
func NewHandler(service *Service, log *logger.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  log,
	}
}

// GetOrderStatus handles GET /orders/{order_number}/status requests
func (h *Handler) GetOrderStatus(w http.ResponseWriter, r *http.Request) {
	requestID := logger.GenerateRequestID()
	
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", requestID)
		return
	}

	// Extract order number from URL path
	orderNumber := h.extractOrderNumber(r.URL.Path, "/orders/", "/status")
	if orderNumber == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid order number", requestID)
		return
	}

	h.logger.Debug("request_received", "Get order status request", requestID, map[string]interface{}{
		"order_number": orderNumber,
		"endpoint":     "status",
	})

	// Get order status from service
	status, err := h.service.GetOrderStatus(r.Context(), orderNumber, requestID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.writeErrorResponse(w, http.StatusNotFound, "Order not found", requestID)
		} else {
			h.logger.Error("db_query_failed", "Failed to get order status", requestID, err, map[string]interface{}{
				"order_number": orderNumber,
			})
			h.writeErrorResponse(w, http.StatusInternalServerError, "Internal server error", requestID)
		}
		return
	}

	// Write successful response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	if err := json.NewEncoder(w).Encode(status); err != nil {
		h.logger.Error("response_encoding_failed", "Failed to encode response", requestID, err, nil)
	}
}

// GetOrderHistory handles GET /orders/{order_number}/history requests
func (h *Handler) GetOrderHistory(w http.ResponseWriter, r *http.Request) {
	requestID := logger.GenerateRequestID()
	
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", requestID)
		return
	}

	// Extract order number from URL path
	orderNumber := h.extractOrderNumber(r.URL.Path, "/orders/", "/history")
	if orderNumber == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid order number", requestID)
		return
	}

	h.logger.Debug("request_received", "Get order history request", requestID, map[string]interface{}{
		"order_number": orderNumber,
		"endpoint":     "history",
	})

	// Get order history from service
	history, err := h.service.GetOrderHistory(r.Context(), orderNumber, requestID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.writeErrorResponse(w, http.StatusNotFound, "Order not found", requestID)
		} else {
			h.logger.Error("db_query_failed", "Failed to get order history", requestID, err, map[string]interface{}{
				"order_number": orderNumber,
			})
			h.writeErrorResponse(w, http.StatusInternalServerError, "Internal server error", requestID)
		}
		return
	}

	// Write successful response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	if err := json.NewEncoder(w).Encode(history); err != nil {
		h.logger.Error("response_encoding_failed", "Failed to encode response", requestID, err, nil)
	}
}

// GetWorkerStatus handles GET /workers/status requests
func (h *Handler) GetWorkerStatus(w http.ResponseWriter, r *http.Request) {
	requestID := logger.GenerateRequestID()
	
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", requestID)
		return
	}

	h.logger.Debug("request_received", "Get worker status request", requestID, map[string]interface{}{
		"endpoint": "workers/status",
	})

	// Get worker status from service
	workers, err := h.service.GetWorkerStatus(r.Context(), requestID)
	if err != nil {
		h.logger.Error("db_query_failed", "Failed to get worker status", requestID, err, nil)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Internal server error", requestID)
		return
	}

	// Write successful response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	if err := json.NewEncoder(w).Encode(workers); err != nil {
		h.logger.Error("response_encoding_failed", "Failed to encode response", requestID, err, nil)
	}
}

// HealthCheck handles GET /health requests
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", "")
		return
	}

	healthy := h.service.HealthCheck(r.Context())
	
	response := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "tracking-service",
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

// extractOrderNumber extracts order number from URL path
func (h *Handler) extractOrderNumber(path, prefix, suffix string) string {
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return ""
	}
	
	orderNumber := strings.TrimPrefix(path, prefix)
	orderNumber = strings.TrimSuffix(orderNumber, suffix)
	
	// Basic validation - order number should follow ORD_YYYYMMDD_NNN format
	if len(orderNumber) < 15 || !strings.HasPrefix(orderNumber, "ORD_") {
		return ""
	}
	
	return orderNumber
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
	mux.HandleFunc("/orders/", h.withLogging(h.routeOrderRequests))
	mux.HandleFunc("/workers/status", h.withLogging(h.GetWorkerStatus))
	mux.HandleFunc("/health", h.withLogging(h.HealthCheck))
	
	return mux
}

// routeOrderRequests routes order-related requests to appropriate handlers
func (h *Handler) routeOrderRequests(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/status") {
		h.GetOrderStatus(w, r)
	} else if strings.HasSuffix(r.URL.Path, "/history") {
		h.GetOrderHistory(w, r)
	} else {
		h.writeErrorResponse(w, http.StatusNotFound, "Endpoint not found", "")
	}
}

// withLogging adds request logging middleware
func (h *Handler) withLogging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := logger.GenerateRequestID()
		
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
