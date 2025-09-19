package domain

type Order struct {
	OrderRequest  *OrderRequest
	OrderResponse *OrderResponse
	Priority      int
}

// Request structs
type OrderItem struct {
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

type OrderRequest struct {
	CustomerName string      `json:"customer_name"`
	OrderType    string      `json:"order_type"`
	TableNumber  int         `json:"table_number,omitempty"`
	DeliveryAddr string      `json:"delivery_address,omitempty"`
	Items        []OrderItem `json:"items"`
}

// Response structs
type OrderResponse struct {
	OrderNumber string  `json:"order_number"`
	Status      string  `json:"status"`
	TotalAmount float64 `json:"total_amount"`
}
