package domain

import (
	"errors"
	"time"
)

const (
	OrderPending   = "pending"
	OrderConfirmed = "confirmed"
	OrderShipped   = "shipped"
	OrderDelivered = "delivered"
	OrderCancelled = "cancelled"
)

type Order struct {
	ID         int64
	UserID     int64
	UserEmail  string // filled only by admin queries
	Status     string
	TotalMinor int64
	CreatedAt  time.Time
	Items      []OrderItem
}

type OrderItem struct {
	ID         int64
	VariantID  int64
	Name       string // snapshot at purchase time
	Label      string
	PriceMinor int64
	Qty        int
}

var ErrInvalidTransition = errors.New("invalid order status transition")

// The order lifecycle as data: pending → confirmed → shipped → delivered,
// with cancellation possible while not yet shipped.
var orderTransitions = map[string][]string{
	OrderPending:   {OrderConfirmed, OrderCancelled},
	OrderConfirmed: {OrderShipped, OrderCancelled},
	OrderShipped:   {OrderDelivered},
}

func ValidOrderTransition(from, to string) bool {
	for _, allowed := range orderTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

func ValidOrderStatus(s string) bool {
	switch s {
	case OrderPending, OrderConfirmed, OrderShipped, OrderDelivered, OrderCancelled:
		return true
	}
	return false
}
