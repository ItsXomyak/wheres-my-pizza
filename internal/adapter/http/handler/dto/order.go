// dto - Data Transfer Obeject
package dto

import "wheres-my-pizza/internal/domain/models"

type CreateOrderRequest struct {
	CustomerName    string      `json:"customer_name"`
	OrderType       string      `json:"order_type"`
	Items           []OrderItem `json:"items"`
	TableNumber     *int        `json:"table_number,omitempty"`     // Only for dine_in
	DeliveryAddress *string     `json:"delivery_address,omitempty"` // Only for delivery
}

type OrderItem struct {
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

func FromRequestToInternalCreateOrder(req CreateOrderRequest) *models.CreateOrder {
	// Convert OrderItems to CreateOrderItems
	items := make([]models.CreateOrderItem, len(req.Items))
	for i, item := range req.Items {
		items[i] = models.CreateOrderItem{
			Name:     item.Name,
			Quantity: item.Quantity,
			Price:    item.Price,
		}
	}

	return &models.CreateOrder{
		CustomerName:    req.CustomerName,
		Type:            req.OrderType,
		Items:           items,
		TableNumber:     req.TableNumber,
		DeliveryAddress: req.DeliveryAddress,
		// These fields will be set later in the business logic
		Number:      "",
		TotalAmount: 0,
		Priority:    0,
		Status:      "",
	}
}

type CreateOrderResponse struct {
	OrderNumber string  `json:"order_number"`
	Status      string  `json:"status"`
	TotalAmount float64 `json:"total_amount"`
}
