package server

import (
	"encoding/json"
	"net/http"

	"wheres-my-pizza/internal/domain/types"
)

// setupRoutes - setups http routes
func (a *API) setupRoutes() {
	a.setupDefaultRoutes()

	switch a.mode {
	case types.ModeOrder:
		a.setupOrderRoutes()
	case types.ModeTracking:
		a.setupTrackingRoutes()
	}
}

// setupDefaultRoutes - setups default http routes
func (a *API) setupDefaultRoutes() {
	// System Health
	a.mux.HandleFunc("/health", a.HealthCheck)
}

// setupOrderRoutes setups routes for order service
func (a *API) setupOrderRoutes() {
	a.mux.HandleFunc("POST /orders", a.routes.order.CreateOrder)
}

// setupTrackingRoutes setups routes for tracking service
func (a *API) setupTrackingRoutes() {
	a.mux.HandleFunc("GET /orders/{order_number}/status", a.routes.tracking.GetOrderStatus)
	a.mux.HandleFunc("GET /orders/{order_number}/history", a.routes.tracking.GetTrackingHistory)
	a.mux.HandleFunc("GET /workers/status", a.routes.tracking.ListWorkers)
}

// HealthCheck - returns system information.
func (a *API) HealthCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]any{
		"status": "available",
		"system_info": map[string]string{
			"address": a.addr,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		a.log.Error(r.Context(), "healthcheck", "failed to encode", err)
		return
	}
}
