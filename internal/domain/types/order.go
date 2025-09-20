package types

import (
	"slices"
	"time"
)

const (
	StatusOrderReceived  = "received"
	StatusOrderCooking   = "cooking"
	StatusOrderReady     = "ready"
	StatusOrderCompleted = "completed"
	StatusOrderCancelled = "cancelled"
)

// Must be one of: `'dine_in'`, `'takeout'`, or `'delivery'`.

const (
	OrderTypeDineIn   = "dine_in"
	OrderTypeTakeOut  = "takeout"
	OrderTypeDelivery = "delivery"

	CookingTimeDineIn   time.Duration = time.Second * 8
	CookingTimeTakeOut  time.Duration = time.Second * 10
	CookingTimeDelivery time.Duration = time.Second * 12
	DefaultCookingTime  time.Duration = time.Second * 5
)

// All order types
var AllOrderTypes = []string{
	OrderTypeDineIn,
	OrderTypeTakeOut,
	OrderTypeDelivery,
}

// Checks if given string is order type
func IsValidOrderType(s string) bool {
	return slices.Contains(AllOrderTypes, s)
}

// GetSimulateCookingDuration returns cooking time depending on order type
func GetSimulateCookingDuration(orderType string) time.Duration {
	switch orderType {
	case OrderTypeDineIn:
		return CookingTimeDineIn
	case OrderTypeTakeOut:
		return CookingTimeTakeOut
	case OrderTypeDelivery:
		return CookingTimeDelivery
	default:
		return DefaultCookingTime
	}
}
