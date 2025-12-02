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

	// GetCustomerByEmail searches for an existing customer by email.
	// Used for reconciliation - linking existing Stripe customers to local users.
	// Returns nil, nil if no customer found (not an error).
	GetCustomerByEmail(ctx context.Context, email string) (*Customer, error)

	// UpdateCustomer updates customer information.
	// Post-MVP: For account management.
	UpdateCustomer(ctx context.Context, customerID string, params UpdateCustomerParams) (*Customer, error)

	// CreateSubscription creates a recurring subscription.
	// Post-MVP: For coffee subscriptions.
	CreateSubscription(ctx context.Context, params CreateSubscriptionParams) (*Subscription, error)

	// CreateProduct creates a billing provider product.
	// Required for subscriptions to create Stripe Products for each subscribable SKU.
	// Returns existing product if one already exists with matching metadata.
	CreateProduct(ctx context.Context, params CreateProductParams) (*Product, error)

	// CreateRecurringPrice creates a Stripe Price for recurring subscriptions.
	// Required for subscriptions to define pricing per product SKU.
	CreateRecurringPrice(ctx context.Context, params CreateRecurringPriceParams) (*Price, error)

	// GetSubscription retrieves an existing subscription.
	// SECURITY: Validates tenant_id in subscription metadata before returning.
	GetSubscription(ctx context.Context, params GetSubscriptionParams) (*Subscription, error)

	// PauseSubscription pauses a subscription until explicitly resumed.
	// SECURITY: Validates tenant_id ownership before pausing.
	PauseSubscription(ctx context.Context, params PauseSubscriptionParams) (*Subscription, error)

	// ResumeSubscription resumes a paused subscription immediately.
	// SECURITY: Validates tenant_id ownership before resuming.
	ResumeSubscription(ctx context.Context, params ResumeSubscriptionParams) (*Subscription, error)

	// CancelSubscription cancels a subscription.
	// Post-MVP: For subscription management.
	CancelSubscription(ctx context.Context, params CancelSubscriptionParams) error

	// CreateCustomerPortalSession creates a Stripe Customer Portal session.
	// Returns session URL where customer can manage subscriptions and payment methods.
	CreateCustomerPortalSession(ctx context.Context, params CreatePortalSessionParams) (*PortalSession, error)

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

	// ReceiptEmail is the email where Stripe sends receipts
	// Used for guest checkout to create user account
	ReceiptEmail string
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

// CreateSubscriptionParams contains parameters for creating a subscription.
type CreateSubscriptionParams struct {
	TenantID               string
	CustomerID             string // Stripe customer ID (cus_...)
	PriceID                string // Stripe price ID (price_...)
	Quantity               int32
	DefaultPaymentMethodID string // pm_...
	CollectionMethod       string // "charge_automatically" (default) or "send_invoice"
	Metadata               map[string]string
	IdempotencyKey         string
}

// SubscriptionParams contains parameters for creating a subscription.
type SubscriptionParams struct {
	CustomerID string
	PriceID    string // Provider's price/plan identifier
	Quantity   int
	Metadata   map[string]string
}

// CreateRecurringPriceParams contains parameters for creating a recurring price.
type CreateRecurringPriceParams struct {
	// Currency code (ISO 4217 lowercase) - e.g., "usd"
	Currency string

	// UnitAmountCents is the amount per billing period in smallest currency unit
	UnitAmountCents int32

	// BillingInterval is the frequency: "week", "month"
	BillingInterval string

	// IntervalCount is the multiplier for interval (e.g., 2 for biweekly)
	IntervalCount int32

	// ProductID is the Stripe product ID (prod_...) to attach price to
	ProductID string

	// Metadata for filtering and reporting (should include tenant_id)
	Metadata map[string]string

	// Nickname for the price (e.g., "Colombia Supremo - Monthly")
	Nickname string
}

// CreateProductParams contains parameters for creating a product.
type CreateProductParams struct {
	// Name is the product name (e.g., "Colombia Supremo - 12oz Whole Bean")
	Name string

	// Description is optional product description
	Description string

	// Metadata for filtering and reporting (must include tenant_id and product_sku_id)
	Metadata map[string]string

	// Active determines if product is available for purchase
	Active bool
}

// Product represents a billing provider product.
type Product struct {
	ID          string
	Name        string
	Description string
	Active      bool
	Metadata    map[string]string
	CreatedAt   time.Time
}

// Price represents a Stripe price (one-time or recurring).
type Price struct {
	ID              string
	ProductID       string
	Currency        string
	UnitAmountCents int32
	Type            string // "one_time" or "recurring"
	Recurring       *PriceRecurring
	Active          bool
	Metadata        map[string]string
	CreatedAt       time.Time
}

// PriceRecurring contains recurring price details.
type PriceRecurring struct {
	Interval      string // "day", "week", "month", "year"
	IntervalCount int32
}

// GetSubscriptionParams contains parameters for retrieving a subscription.
type GetSubscriptionParams struct {
	SubscriptionID string
	TenantID       string
	Expand         []string
}

// PauseSubscriptionParams contains parameters for pausing a subscription.
type PauseSubscriptionParams struct {
	SubscriptionID string
	TenantID       string
	Behavior       string     // "void", "keep_as_draft", "mark_uncollectible"
	ResumesAt      *time.Time // nil for manual resume
}

// ResumeSubscriptionParams contains parameters for resuming a subscription.
type ResumeSubscriptionParams struct {
	SubscriptionID string
	TenantID       string
}

// CancelSubscriptionParams contains parameters for canceling a subscription.
type CancelSubscriptionParams struct {
	SubscriptionID     string
	TenantID           string
	CancelAtPeriodEnd  bool
	CancellationReason string
}

// CreatePortalSessionParams contains parameters for creating a customer portal session.
type CreatePortalSessionParams struct {
	CustomerID string
	TenantID   string
	ReturnURL  string
}

// PortalSession represents a Stripe Customer Portal session.
type PortalSession struct {
	ID        string
	URL       string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// SubscriptionItem represents a line item in a subscription.
type SubscriptionItem struct {
	ID       string
	PriceID  string
	Quantity int32
	Metadata map[string]string
}

// SubscriptionPauseCollection contains pause settings for a subscription.
type SubscriptionPauseCollection struct {
	Behavior  string
	ResumesAt *time.Time
}

// Subscription represents a recurring subscription.
type Subscription struct {
	ID                     string
	CustomerID             string
	Status                 string // "active", "past_due", "canceled", "incomplete", etc.
	Items                  []SubscriptionItem
	DefaultPaymentMethodID string
	CurrentPeriodStart     time.Time
	CurrentPeriodEnd       time.Time
	CancelAtPeriodEnd      bool
	CanceledAt             *time.Time
	PauseCollection        *SubscriptionPauseCollection
	Metadata               map[string]string
	CreatedAt              time.Time
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
