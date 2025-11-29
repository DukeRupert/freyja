package billing

import (
	"context"
	"time"
)

// Provider defines the interface for payment processing.
// Implementations can use Stripe, PayPal, Square, etc.
type Provider interface {
	// CreatePaymentIntent creates a payment intent for one-time charges.
	// Required for MVP checkout flow.
	// Returns payment intent with client_secret for frontend confirmation.
	CreatePaymentIntent(ctx context.Context, params CreatePaymentIntentParams) (*PaymentIntent, error)

	// GetPaymentIntent retrieves an existing payment intent.
	// Required for MVP checkout flow to verify payment before creating order.
	GetPaymentIntent(ctx context.Context, params GetPaymentIntentParams) (*PaymentIntent, error)

	// UpdatePaymentIntent updates a payment intent before confirmation.
	// Required for MVP to handle cart changes during checkout.
	UpdatePaymentIntent(ctx context.Context, params UpdatePaymentIntentParams) (*PaymentIntent, error)

	// CancelPaymentIntent cancels a payment intent that hasn't been confirmed.
	// Required for MVP to clean up abandoned checkouts.
	CancelPaymentIntent(ctx context.Context, paymentIntentID string, tenantID string) error

	// VerifyWebhookSignature verifies that a webhook request is authentic.
	// Required for MVP to process async payment confirmations.
	VerifyWebhookSignature(payload []byte, signature string, secret string) error

	// CreateCustomer creates a customer record in the billing provider.
	// Post-MVP: For saving payment methods and subscriptions.
	CreateCustomer(ctx context.Context, params CreateCustomerParams) (*Customer, error)

	// GetCustomer retrieves an existing customer.
	// Post-MVP: For subscription management.
	GetCustomer(ctx context.Context, customerID string) (*Customer, error)

	// UpdateCustomer updates customer information.
	// Post-MVP: For account management.
	UpdateCustomer(ctx context.Context, customerID string, params UpdateCustomerParams) (*Customer, error)

	// CreateSubscription creates a recurring subscription.
	// Post-MVP: For coffee subscriptions.
	CreateSubscription(ctx context.Context, params SubscriptionParams) (*Subscription, error)

	// CancelSubscription cancels a subscription.
	// Post-MVP: For subscription management.
	CancelSubscription(ctx context.Context, subscriptionID string, cancelAtPeriodEnd bool) error

	// RefundPayment refunds a completed payment.
	// Post-MVP: For order cancellations and returns.
	RefundPayment(ctx context.Context, params RefundParams) (*Refund, error)
}

// CreateCustomerParams contains parameters for creating a customer.
type CreateCustomerParams struct {
	Email       string
	Name        string
	Phone       string
	Description string
	Metadata    map[string]string
}

// Customer represents a billing customer.
type Customer struct {
	ID        string
	Email     string
	Name      string
	CreatedAt time.Time
}

// CreatePaymentIntentParams contains parameters for creating a payment intent.
// Extended from existing type to support checkout flow requirements.
type CreatePaymentIntentParams struct {
	// AmountCents is the amount in smallest currency unit (cents for USD)
	AmountCents int32

	// Currency code (ISO 4217) - e.g., "usd", "eur"
	Currency string

	// CustomerID is optional - if provided, links payment to existing customer
	CustomerID string

	// CustomerEmail is used to prefill customer email in payment sheet
	CustomerEmail string

	// Description appears on customer's statement and in Stripe dashboard
	Description string

	// Metadata for filtering and reporting (always include tenant_id, cart_id)
	Metadata map[string]string

	// IdempotencyKey prevents duplicate payment intents
	// Typically use cart_id or a unique checkout session identifier
	IdempotencyKey string

	// ShippingAddress is used for Stripe Tax calculation (if enabled)
	ShippingAddress *PaymentAddress

	// LineItems are used for Stripe Tax calculation (if enabled)
	// Each line item should include tax_code for accurate tax calculation
	LineItems []PaymentLineItem

	// EnableStripeTax determines if Stripe should calculate tax automatically
	// If false, tax should be calculated separately and included in Amount
	EnableStripeTax bool

	// CaptureMethod: "automatic" (default) or "manual"
	// Use "manual" for authorizations that capture later (wholesale, preorders)
	CaptureMethod string

	// SetupFutureUsage: "on_session" or "off_session"
	// Use "on_session" to save payment method for future subscriptions
	SetupFutureUsage string
}

// PaymentAddress represents an address for tax calculation or verification.
type PaymentAddress struct {
	Line1      string
	Line2      string
	City       string
	State      string
	PostalCode string
	Country    string
}

// PaymentLineItem represents a line item for tax calculation.
type PaymentLineItem struct {
	// ProductID links to our product (stored in metadata)
	ProductID string

	// Description shown to customer
	Description string

	// Quantity of this line item
	Quantity int32

	// AmountCents is the total amount for this line item (unit price * quantity)
	AmountCents int32

	// TaxCode is Stripe's tax code for this product type
	// See: https://stripe.com/docs/tax/tax-categories
	// Examples: "txcd_99999999" (general), "txcd_30011000" (food and beverages)
	TaxCode string
}

// PaymentIntent represents a payment intent (extended from existing).
type PaymentIntent struct {
	// ID is the Stripe payment intent ID (pi_...)
	ID string

	// ClientSecret is used by Stripe.js on frontend to confirm payment
	ClientSecret string

	// AmountCents is the amount in smallest currency unit (cents)
	// If Stripe Tax is enabled, this includes calculated tax
	AmountCents int32

	// Currency code
	Currency string

	// Status: requires_payment_method, requires_confirmation, succeeded, etc.
	Status string

	// TaxCents is the calculated tax amount (if Stripe Tax enabled)
	TaxCents int32

	// ShippingCents is the shipping amount (from metadata if provided)
	ShippingCents int32

	// Metadata passed during creation
	Metadata map[string]string

	// CreatedAt is when payment intent was created
	CreatedAt time.Time

	// LastPaymentError contains details if payment failed
	LastPaymentError *PaymentError
}

// PaymentError contains details about a failed payment attempt.
type PaymentError struct {
	Code        string // Stripe error code
	Message     string // Human-readable message
	DeclineCode string // Reason card was declined (if applicable)
}

// GetPaymentIntentParams contains parameters for retrieving a payment intent.
type GetPaymentIntentParams struct {
	// PaymentIntentID is the Stripe payment intent ID
	PaymentIntentID string

	// TenantID is required for multi-tenant isolation
	// Must match the tenant_id in payment intent metadata
	TenantID string

	// Expand specifies related objects to include in response
	// Example: []string{"latest_charge", "customer"}
	Expand []string
}

// UpdatePaymentIntentParams contains parameters for updating a payment intent.
// Used when cart changes before payment is confirmed.
type UpdatePaymentIntentParams struct {
	// PaymentIntentID is the Stripe payment intent ID
	PaymentIntentID string

	// TenantID is required for multi-tenant isolation
	// Must match the tenant_id in payment intent metadata
	TenantID string

	// AmountCents updates the amount (must be before confirmation)
	AmountCents int32

	// Metadata updates or adds metadata fields
	Metadata map[string]string

	// Description updates the description
	Description string
}

// SubscriptionParams contains parameters for creating a subscription.
type SubscriptionParams struct {
	CustomerID string
	PriceID    string // Provider's price/plan identifier
	Quantity   int
	Metadata   map[string]string
}

// Subscription represents a recurring subscription.
type Subscription struct {
	ID                string
	CustomerID        string
	Status            string // active, past_due, canceled, etc.
	CurrentPeriodEnd  time.Time
	CancelAtPeriodEnd bool
	CreatedAt         time.Time
}

// UpdateCustomerParams contains parameters for updating a customer.
type UpdateCustomerParams struct {
	Email       string
	Name        string
	Phone       string
	Description string
	Metadata    map[string]string
}

// RefundParams contains parameters for creating a refund.
type RefundParams struct {
	PaymentIntentID string
	AmountCents     int32  // If 0, refunds full amount
	Reason          string // "duplicate", "fraudulent", "requested_by_customer"
	Metadata        map[string]string
}

// Refund represents a payment refund.
type Refund struct {
	ID        string
	PaymentID string
	Amount    int64
	Status    string // succeeded, pending, failed
	CreatedAt time.Time
}
