package web

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"where-is-my-pizza/internal/services/order/internal/app"
	"where-is-my-pizza/internal/services/order/internal/domain"
	"where-is-my-pizza/internal/services/order/internal/validation"
)

type OrderHandler struct {
	service *app.OrderService
}

func NewOrderHandler(service *app.OrderService) *OrderHandler {
	return &OrderHandler{
		service: service,
	}
}

func (h *OrderHandler) StartServer(ctx context.Context) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", h.CreateOrder)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	server.ListenAndServe()
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		slog.Error("Invalid Content-Type", "Content-Type", r.Header.Get("Content-Type")) // replace with your logger
		http.Error(w, "Content-Type must be application/json", http.StatusBadRequest)
		return
	}

	var req domain.OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Failed to decode request body", "error", err) // replace with your logger
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	reqErr := validation.ValidateOrderRequest(&req)
	if reqErr != nil {
		slog.Error("Validation error", "error", reqErr) // replace with your logger
		http.Error(w, reqErr.Error(), http.StatusBadRequest)
		return
	}

	// resp, err := h.service.CreateOrder(r.Context(), &req)
	// if err != nil {
	// 	slog.Error("Failed to create order", "error", err) // replace with your logger
	// 	http.Error(w, "Failed to create order", http.StatusInternalServerError)
	// 	return
	// }
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// json.NewEncoder(w).Encode(resp)
}
