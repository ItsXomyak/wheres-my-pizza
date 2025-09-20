package handler

import (
	"context"
	"errors"
	"net/http"

	"wheres-my-pizza/internal/adapter/http/handler/dto"
	"wheres-my-pizza/internal/domain/models"
	"wheres-my-pizza/internal/domain/types"
	"wheres-my-pizza/internal/services/order"
	"wheres-my-pizza/pkg/logger"
	"wheres-my-pizza/pkg/validator"
)

type OrderService interface {
	CreateOrder(ctx context.Context, req *models.CreateOrder) (*models.OrderCreatedInfo, error)
}

type Order struct {
	service OrderService
	log     logger.Logger
}

func NewOrder(service OrderService, log logger.Logger) *Order {
	return &Order{
		service: service,
		log:     log,
	}
}

// CreateOrder creates new order
func (h *Order) CreateOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req dto.CreateOrderRequest
	if err := readJSON(w, r, &req); err != nil {
		h.log.Error(ctx, types.ActionValidationFailed, "failed to decode request", err)
		errorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	createOrder := dto.FromRequestToInternalCreateOrder(req)

	v := validator.New()
	dto.ValidateCreateOrderRequest(v, createOrder)
	if !v.Valid() {
		h.log.Error(ctx, types.ActionValidationFailed, "failed to validate request", v)
		failedValidationResponse(w, v.Errors)
		return
	}

	info, err := h.service.CreateOrder(ctx, createOrder)
	if err != nil {
		if errors.Is(err, order.ErrTooManyRequest) {
			errorResponse(w, http.StatusTooManyRequests, err.Error())
			return
		}
		internalErrorResponse(w, err.Error())
		return
	}

	response := envelope{
		"customer_name": req.CustomerName,
		"order_info": dto.CreateOrderResponse{
			OrderNumber: info.Number,
			Status:      info.Status,
			TotalAmount: info.TotalAmount,
		},
	}

	if err := writeJSON(w, http.StatusCreated, response, nil); err != nil {
		h.log.Error(ctx, types.ActionValidationFailed, "failed to write response", err)
		internalErrorResponse(w, err.Error())
	}
}

// Post request to create order. TODO: delete
//	{
//	    "customer_name": "John",
//	    "order_type": "delivery",
//	    "items": [
//	        {
//	            "name": "pizza",
//	            "quantity": 10,
//	            "price": 999.99
//	        },
//	        {
//	            "name": "Caesar Salad",
//	            "quantity": 1,
//	            "price": 8.99
//	        }
//	    ],
//	    "delivery_address": "Kabanbay batyra 66"
//	}
