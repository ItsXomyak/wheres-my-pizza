package dto

import (
	"regexp"
	"unicode/utf8"

	"wheres-my-pizza/internal/domain/models"
	"wheres-my-pizza/internal/domain/types"
	"wheres-my-pizza/pkg/validator"
)

// 2. **Input Validation:** The incoming JSON payload is validated against the following rules:

// | `tag`           | `required type` | `description`                                                                                      |
// | --------------- | --------------- | -------------------------------------------------------------------------------------------------- |
// | `customer_name` | string           | 1-100 characters. Must not contain special characters other than spaces, hyphens, and apostrophes. |
// | `order_type`    | string           | Must be one of: `'dine_in'`, `'takeout'`, or `'delivery'`.                                         |
// | `items`         | array            | Must contain between 1 and 20 items.                                                               |
// | `item.name`     | string           | 1-50 characters.                                                                                   |
// | `item.quantity` | integer          | must be between 1 and 10.                                                                          |
// | `item.price`    | decimal          | must be between `0.01` and `999.99`.                                                               |

// - **Conditional Validation:**

// | `order_type` | `required fields`                              | `description`                                 | `must not be present` |
// | ------------ | ---------------------------------------------- | --------------------------------------------- | --------------------- |
// | `'dine_in'`  | `table_number` (integer, 1-100)                | Table number at which the customer is served. | `delivery_address`    |
// | `'delivery'` | `delivery_address` (string, min 10 characters) | Address for delivery of the order by courier. | `table_number`        |

var ValidOrderTypes = []string{
	types.OrderTypeDineIn,
	types.OrderTypeDelivery,
	types.OrderTypeTakeOut,
}

func ValidateCreateOrderRequest(v *validator.Validator, req *models.CreateOrder) {
	if req == nil {
		return
	}

	v.Check(
		isValidCustomerName(req.CustomerName),
		"customer_name",
		"1-100 characters. Must not contain special characters other than spaces, hyphens, and apostrophes.",
	)

	// Check if order_type in request contains in ValidOrderTypes
	v.Check(
		validator.PermittedValue(req.Type, ValidOrderTypes...),
		"order_type",
		"must be one of: 'dine_in', 'takeout', or 'delivery'",
	)

	// Conditional validations based on order_type
	vaildateOrdertype(v, req)

	v.Check(
		len(req.Items) >= 1 && len(req.Items) <= 20,
		"items",
		"must contain between 1 and 20 items.",
	)

	for _, item := range req.Items {
		v.Check(
			isValidItemName(item.Name),
			"item.name",
			"must be between 1-50 characters",
		)

		v.Check(
			item.Quantity >= 1 && item.Quantity <= 10,
			"item.quantity",
			"must be between 1 and 10",
		)

		v.Check(
			item.Price >= 0.01 && item.Price <= 999.99,
			"item.Price",
			"must be between `0.01` and `999.99`",
		)
	}
}

// vaildateOrdertype does conditional validations based on order_type
func vaildateOrdertype(v *validator.Validator, req *models.CreateOrder) {
	// Conditional validations based on order_type
	switch req.Type {
	case types.OrderTypeDineIn:
		v.Check(
			req.TableNumber != nil,
			"table_number",
			"required for dine_in orders",
		)

		if req.TableNumber != nil {
			v.Check(
				*req.TableNumber >= 1 && *req.TableNumber <= 100,
				"table_number",
				"must be between 1 and 100",
			)
		}

		v.Check(
			req.DeliveryAddress == nil,
			"delivery_address",
			"must not be present for dine_in orders",
		)

	case types.OrderTypeDelivery:
		v.Check(
			req.DeliveryAddress != nil,
			"delivery_address",
			"required for delivery orders",
		)

		if req.DeliveryAddress != nil {
			v.Check(
				len(*req.DeliveryAddress) >= 10,
				"delivery_address",
				"must be at least 10 characters",
			)
		}

		v.Check(
			req.TableNumber == nil,
			"table_number",
			"must not be present for delivery orders",
		)

	case types.OrderTypeTakeOut:
		v.Check(
			req.TableNumber == nil,
			"table_number",
			"must not be present for takeout orders",
		)

		v.Check(
			req.DeliveryAddress == nil,
			"delivery_address",
			"must not be present for takeout orders",
		)
	}
}

func isValidCustomerName(name string) bool {
	// Check length (1-100 characters)
	if utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 100 {
		return false
	}

	// Check for allowed characters: letters, spaces, hyphens, and apostrophes
	validPattern := `^[a-zA-Z\s\-']+$`
	matched, err := regexp.MatchString(validPattern, name)
	if err != nil {
		return false
	}

	return matched
}

func isValidItemName(item string) bool {
	return utf8.RuneCountInString(item) >= 1 && utf8.RuneCountInString(item) <= 50
}
