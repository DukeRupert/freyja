package domain

import (
	"context"

	"github.com/dukerupert/freyja/internal/repository"
)

// Order-related domain errors.
var (
	ErrOrderNotFound           = &Error{Code: ENOTFOUND, Message: "Order not found"}
	ErrPaymentNotSucceeded     = &Error{Code: EPAYMENT, Message: "Payment has not succeeded"}
	ErrCartAlreadyConverted    = &Error{Code: ECONFLICT, Message: "Cart already converted to order"}
	ErrInsufficientStock       = &Error{Code: ECONFLICT, Message: "Insufficient stock for one or more items"}
	ErrMissingCartID           = &Error{Code: EINVALID, Message: "Cart ID missing from payment metadata"}
	ErrPaymentAlreadyProcessed = &Error{Code: ECONFLICT, Message: "Payment intent already processed"}
	ErrEmptyCart               = &Error{Code: EINVALID, Message: "Cart is empty"}
	ErrMissingShippingAddress  = &Error{Code: EINVALID, Message: "Shipping address missing from payment metadata"}
	ErrMissingBillingAddress   = &Error{Code: EINVALID, Message: "Billing address missing from payment metadata"}
	ErrMissingCustomerEmail    = &Error{Code: EINVALID, Message: "Customer email required for guest checkout"}
	ErrInvalidAddressJSON      = &Error{Code: EINVALID, Message: "Address JSON is empty or invalid"}
)

// OrderService provides business logic for order operations.
// Implementations should be tenant-scoped.
type OrderService interface {
	// CreateOrderFromPaymentIntent creates an order from a successful payment.
	// This is the primary order creation flow for retail purchases.
	// Implements idempotency via payment_intent_id to prevent duplicate orders.
	CreateOrderFromPaymentIntent(ctx context.Context, paymentIntentID string) (*OrderDetail, error)

	// GetOrder retrieves a single order by ID with tenant scoping.
	GetOrder(ctx context.Context, orderID string) (*OrderDetail, error)

	// GetOrderByNumber retrieves a single order by order number with tenant scoping.
	GetOrderByNumber(ctx context.Context, orderNumber string) (*OrderDetail, error)
}

// OrderDetail aggregates order information with items and addresses.
type OrderDetail struct {
	Order           repository.Order
	Items           []repository.OrderItem
	ShippingAddress repository.Address
	BillingAddress  repository.Address
	Payment         repository.Payment
}
