package domain

import (
	"context"

	"github.com/dukerupert/freyja/internal/address"
	"github.com/dukerupert/freyja/internal/billing"
	"github.com/dukerupert/freyja/internal/shipping"
	"github.com/dukerupert/freyja/internal/tax"
	"github.com/jackc/pgx/v5/pgtype"
)

// CheckoutService provides business logic for checkout operations.
// Implementations should be tenant-scoped.
type CheckoutService interface {
	// ValidateAndNormalizeAddress validates a shipping or billing address.
	ValidateAndNormalizeAddress(ctx context.Context, addr address.Address) (*address.ValidationResult, error)

	// GetShippingRates calculates available shipping options for the cart.
	GetShippingRates(ctx context.Context, cartID string, shippingAddr address.Address) ([]shipping.Rate, error)

	// CalculateOrderTotal computes the complete order total including tax and shipping.
	CalculateOrderTotal(ctx context.Context, params OrderTotalParams) (*OrderTotal, error)

	// CreatePaymentIntent initiates a Stripe Payment Intent.
	CreatePaymentIntent(ctx context.Context, params PaymentIntentParams) (*billing.PaymentIntent, error)

	// CompleteCheckout converts a cart to an order after successful payment (idempotent).
	CompleteCheckout(ctx context.Context, params CompleteCheckoutParams) (*CheckoutOrder, error)
}

// OrderTotalParams contains parameters for calculating order totals.
type OrderTotalParams struct {
	CartID               string
	ShippingAddress      address.Address
	BillingAddress       address.Address
	SelectedShippingRate shipping.Rate
	DiscountCode         string
}

// OrderTotal contains the breakdown of an order's total cost.
type OrderTotal struct {
	SubtotalCents int32
	ShippingCents int32
	TaxCents      int32
	DiscountCents int32
	TotalCents    int32

	TaxBreakdown      []tax.TaxBreakdown
	TaxCalculationID  string // Stripe tax calculation ID for audit trail
	ShippingRateID    string
	DiscountCodeID    pgtype.UUID
	DiscountCodeValue string
}

// PaymentIntentParams contains parameters for creating a payment intent.
type PaymentIntentParams struct {
	CartID          string
	OrderTotal      *OrderTotal
	ShippingAddress address.Address
	BillingAddress  address.Address
	CustomerEmail   string
	IdempotencyKey  string
}

// CompleteCheckoutParams contains parameters for completing checkout.
type CompleteCheckoutParams struct {
	CartID          string
	PaymentIntentID string
	ShippingAddress address.Address
	BillingAddress  address.Address
	ShippingRateID  string
	DiscountCodeID  pgtype.UUID
	CustomerNotes   string
	IdempotencyKey  string
}

// CheckoutOrder represents a completed order from checkout.
type CheckoutOrder struct {
	ID            pgtype.UUID
	OrderNumber   string
	Status        string
	SubtotalCents int32
	TaxCents      int32
	ShippingCents int32
	DiscountCents int32
	TotalCents    int32
	Currency      string
	CreatedAt     pgtype.Timestamptz

	PaymentID         pgtype.UUID
	ShippingAddressID pgtype.UUID
	BillingAddressID  pgtype.UUID
}
