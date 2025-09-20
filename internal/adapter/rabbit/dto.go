package rabbit

import (
	"context"
	"encoding/json"

	"wheres-my-pizza/internal/domain/models"
)

// Order represents the structure of an order to be published
type Order struct {
	OrderNumber     string      `json:"order_number"`
	CustomerName    string      `json:"customer_name"`
	OrderType       string      `json:"order_type"`
	TableNumber     *int        `json:"table_number,omitempty"`
	DeliveryAddress *string     `json:"delivery_address,omitempty"`
	Items           []OrderItem `json:"items"`
	TotalAmount     float64     `json:"total_amount"`
	Priority        int         `json:"priority"`
	RequestID       string      `json:"request_id,omitempty"`
}

// OrderItem represents an item in the order
type OrderItem struct {
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

func FromInternalToPublishOrder(ctx context.Context, m *models.CreateOrder) *Order {
	if m == nil {
		return nil
	}

	// Convert order items
	publishItems := make([]OrderItem, 0, len(m.Items))
	for _, item := range m.Items {
		publishItems = append(publishItems, OrderItem{
			Name:     item.Name,
			Quantity: item.Quantity,
			Price:    item.Price,
		})
	}

	var requestID string
	if reqID, ok := ctx.Value(models.GetRequestIDKey()).(string); ok {
		requestID = reqID
	}

	return &Order{
		OrderNumber:     m.Number,
		CustomerName:    m.CustomerName,
		OrderType:       m.Type,
		TableNumber:     m.TableNumber,
		DeliveryAddress: m.DeliveryAddress,
		Items:           publishItems,
		TotalAmount:     m.TotalAmount,
		Priority:        m.Priority,
		RequestID:       requestID,
	}
}

func FromPublishToInternalOrder(m *Order) *models.CreateOrder {
	if m == nil {
		return nil
	}

	// Convert order items
	internalItems := make([]models.CreateOrderItem, 0, len(m.Items))
	for _, item := range m.Items {
		internalItems = append(internalItems, models.CreateOrderItem{
			Name:     item.Name,
			Quantity: item.Quantity,
			Price:    item.Price,
		})
	}

	return &models.CreateOrder{
		Number:          m.OrderNumber,
		CustomerName:    m.CustomerName,
		Type:            m.OrderType,
		Items:           internalItems,
		TableNumber:     m.TableNumber,
		DeliveryAddress: m.DeliveryAddress,
		TotalAmount:     m.TotalAmount,
		Priority:        m.Priority,
		Status:          "", // no status in published message
	}
}

func ToInternalOrder(body []byte) (*Order, error) {
	publishedOrder := &Order{}
	if err := json.Unmarshal(body, publishedOrder); err != nil {
		return nil, err
	}

	// order := FromPublishToInternalOrder(&publishedOrder)
	// if order == nil {
	// 	return nil, errors.New("failed to map request to internal CreateOrder struct")
	// }

	return publishedOrder, nil
}
