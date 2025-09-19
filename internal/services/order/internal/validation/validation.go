package validation

import (
	"fmt"
	"where-is-my-pizza/internal/services/order/internal/domain"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func ValidateOrderRequest(req *domain.OrderRequest) error {
	if err := validateCustomerName(req.CustomerName); err != nil {
		return err
	}

	if err := validateOrderType(req.OrderType); err != nil {
		return err
	}

	if err := validateOrderTypeConditions(req); err != nil {
		return err
	}

	if err := validateItems(req.Items); err != nil {
		return err
	}

	return nil
}

func validateCustomerName(name string) error {
	if name == "" {
		return ValidationError{
			Field:   "customer_name",
			Message: "customer name is required",
		}
	}

	if len(name) > 100 {
		return ValidationError{
			Field:   "customer_name",
			Message: "customer name must be less than 100 characters",
		}
	}
	return nil
}

func validateOrderType(orderType string) error {
	if orderType == "" {
		return ValidationError{
			Field:   "order_type",
			Message: "order type is required",
		}
	}
	allowedTypes := map[string]bool{
		"takeout":  true,
		"dine_in":  true,
		"delivery": true,
	}

	if !allowedTypes[orderType] {
		return ValidationError{
			Field:   "order_type",
			Message: "invalid order type",
		}
	}
	return nil
}

func validateOrderTypeConditions(req *domain.OrderRequest) error {
	if req.OrderType == "delivery" && req.DeliveryAddr == "" {
		return ValidationError{
			Field:   "delivery_address",
			Message: "delivery address is required for delivery orders",
		}
	}

	if req.OrderType == "dine_in" && req.TableNumber == 0 {
		return ValidationError{
			Field:   "table_number",
			Message: "table number is required for dine-in orders",
		}
	}

	return nil
}

func validateItems(items []domain.OrderItem) error {
	if len(items) == 0 {
		return ValidationError{
			Field:   "items",
			Message: "items cannot be empty",
		}
	}

	if len(items) > 20 {
		return ValidationError{
			Field:   "items",
			Message: "a maximum of 20 items is allowed",
		}
	}

	for i, item := range items {
		if err := validateItem(item, i); err != nil {
			return err
		}
	}
	return nil
}

func validateItem(item domain.OrderItem, index int) error {
	if item.Name == "" {
		return ValidationError{
			Field:   fmt.Sprintf("items[%d].name", index),
			Message: "item name is required",
		}
	}

	if len(item.Name) > 50 {
		return ValidationError{
			Field:   fmt.Sprintf("items[%d].name", index),
			Message: "item name must be less than 50 characters",
		}
	}

	if item.Quantity <= 0 {
		return ValidationError{
			Field:   fmt.Sprintf("items[%d].quantity", index),
			Message: "item quantity must be greater than 0",
		}
	}

	if item.Quantity > 10 {
		return ValidationError{
			Field:   fmt.Sprintf("items[%d].quantity", index),
			Message: "item quantity must be less than or equal to 10",
		}
	}

	if item.Price < 0.01 {
		return ValidationError{
			Field:   fmt.Sprintf("items[%d].price", index),
			Message: "item price must be at least 0.01",
		}
	}

	if item.Price > 999.99 {
		return ValidationError{
			Field:   fmt.Sprintf("items[%d].price", index),
			Message: "item price must be less than or equal to 999.99",
		}
	}
	return nil
}
