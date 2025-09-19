package validation

import (
	"testing"
	"where-is-my-pizza/internal/services/order/internal/domain"
)

func TestValidateOrderRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *domain.OrderRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &domain.OrderRequest{
				CustomerName: "John Doe",
				OrderType:    "delivery",
				DeliveryAddr: "123 Main St",
				TableNumber:  0,
				Items: []domain.OrderItem{
					{Name: "Pizza", Quantity: 1, Price: 9.99},
				},
			},
			wantErr: false,
		},
		{
			name: "missing customer name",
			req: &domain.OrderRequest{
				CustomerName: "",
				OrderType:    "delivery",
				DeliveryAddr: "123 Main St",
				TableNumber:  0,
				Items: []domain.OrderItem{
					{Name: "Pizza", Quantity: 1, Price: 9.99},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid order type",
			req: &domain.OrderRequest{
				CustomerName: "John Doe",
				OrderType:    "invalid",
				DeliveryAddr: "123 Main St",
				TableNumber:  0,
				Items: []domain.OrderItem{
					{Name: "Pizza", Quantity: 1, Price: 9.99},
				},
			},
			wantErr: true,
		},
		{
			name: "missing delivery address for delivery order",
			req: &domain.OrderRequest{
				CustomerName: "John Doe",
				OrderType:    "delivery",
				DeliveryAddr: "",
				TableNumber:  0,
				Items: []domain.OrderItem{
					{Name: "Pizza", Quantity: 1, Price: 9.99},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOrderRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateOrderRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
