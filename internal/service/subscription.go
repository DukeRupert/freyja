package service

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// SubscriptionService provides business logic for subscription operations.
type SubscriptionService interface {
	// CreateSubscription creates a new subscription for a customer.
	//
	// Flow:
	//  1. Validates customer has payment method on file
	//  2. Creates Stripe recurring price for product SKU + interval
	//  3. Creates local subscription record (status: "pending")
	//  4. Creates Stripe subscription
	//  5. Updates local record with Stripe subscription ID
	//  6. Returns subscription details
	//
	// First payment is charged immediately. Subsequent charges happen
	// automatically based on billing_interval.
	//
	// Returns ErrNoPaymentMethod if customer has no saved payment method.
	// Returns ErrInvalidProduct if product SKU is inactive or unavailable.
	CreateSubscription(ctx context.Context, params CreateSubscriptionParams) (*SubscriptionDetail, error)

	// GetSubscription retrieves subscription details.
	GetSubscription(ctx context.Context, params GetSubscriptionParams) (*SubscriptionDetail, error)

	// ListSubscriptionsForUser retrieves all subscriptions for a customer.
	ListSubscriptionsForUser(ctx context.Context, params ListSubscriptionsParams) ([]SubscriptionSummary, error)

	// PauseSubscription pauses a subscription until manually resumed.
	//
	// Paused subscriptions stop billing but retain all settings.
	// Customer can resume at any time via portal or service.
	PauseSubscription(ctx context.Context, params PauseSubscriptionParams) (*SubscriptionDetail, error)

	// ResumeSubscription resumes a paused subscription immediately.
	//
	// Resumed subscriptions bill immediately for current period.
	ResumeSubscription(ctx context.Context, params ResumeSubscriptionParams) (*SubscriptionDetail, error)

	// CancelSubscription cancels a subscription.
	//
	// Supports immediate or end-of-period cancellation.
	CancelSubscription(ctx context.Context, params CancelSubscriptionParams) (*SubscriptionDetail, error)

	// CreateCustomerPortalSession creates a Stripe Customer Portal session.
	//
	// Returns URL where customer can manage subscriptions, payment methods, and invoices.
	CreateCustomerPortalSession(ctx context.Context, params PortalSessionParams) (string, error)

	// SyncSubscriptionFromWebhook updates local subscription from Stripe webhook.
	//
	// Called from webhook handler when subscription status changes.
	// Idempotent - safe to call multiple times for same event.
	SyncSubscriptionFromWebhook(ctx context.Context, params SyncSubscriptionParams) error

	// CreateOrderFromSubscriptionInvoice creates an order when subscription invoice is paid.
	//
	// Called from webhook handler for "invoice.payment_succeeded" events
	// where invoice.subscription is set.
	CreateOrderFromSubscriptionInvoice(ctx context.Context, invoiceID string, tenantID pgtype.UUID) (*OrderDetail, error)
}

// CreateSubscriptionParams contains parameters for creating a subscription.
type CreateSubscriptionParams struct {
	// TenantID is the roaster's tenant ID
	TenantID pgtype.UUID

	// UserID is the customer's user ID
	UserID pgtype.UUID

	// ProductSKUID is the product variant to subscribe to
	ProductSKUID pgtype.UUID

	// Quantity of items per billing period (default: 1)
	Quantity int32

	// BillingInterval: "weekly", "biweekly", "monthly", "every_6_weeks", "every_2_months"
	BillingInterval string

	// ShippingAddressID is the address to ship to each period
	ShippingAddressID pgtype.UUID

	// ShippingMethodID is optional - defaults to standard shipping
	ShippingMethodID pgtype.UUID

	// PaymentMethodID is the saved payment method to use
	// Required - customer must have saved payment method
	PaymentMethodID pgtype.UUID
}

// GetSubscriptionParams contains parameters for retrieving a subscription.
type GetSubscriptionParams struct {
	// TenantID is required for multi-tenant isolation
	TenantID pgtype.UUID

	// SubscriptionID is our database subscription ID
	SubscriptionID pgtype.UUID

	// UserID is optional - if provided, validates ownership before returning
	// Use this for customer-facing requests to ensure users can only access their own subscriptions
	UserID pgtype.UUID

	// IncludeUpcomingInvoice includes next invoice preview if true
	IncludeUpcomingInvoice bool
}

// ListSubscriptionsParams contains parameters for listing subscriptions.
type ListSubscriptionsParams struct {
	// TenantID is required for multi-tenant isolation
	TenantID pgtype.UUID

	// UserID filters to specific customer
	UserID pgtype.UUID

	// Status filters by subscription status (nil = all statuses)
	// Values: "active", "paused", "past_due", "cancelled"
	Status *string

	// Limit is max results to return (default: 50)
	Limit int32

	// Offset for pagination
	Offset int32
}

// PauseSubscriptionParams contains parameters for pausing a subscription.
type PauseSubscriptionParams struct {
	// TenantID is required for multi-tenant isolation
	TenantID pgtype.UUID

	// SubscriptionID is our database subscription ID
	SubscriptionID pgtype.UUID

	// ResumesAt is optional auto-resume timestamp (nil for manual resume)
	ResumesAt *time.Time
}

// ResumeSubscriptionParams contains parameters for resuming a subscription.
type ResumeSubscriptionParams struct {
	// TenantID is required for multi-tenant isolation
	TenantID pgtype.UUID

	// SubscriptionID is our database subscription ID
	SubscriptionID pgtype.UUID
}

// CancelSubscriptionParams contains parameters for canceling a subscription.
type CancelSubscriptionParams struct {
	// TenantID is required for multi-tenant isolation
	TenantID pgtype.UUID

	// SubscriptionID is our database subscription ID
	SubscriptionID pgtype.UUID

	// CancelAtPeriodEnd controls cancellation timing
	// true: cancel at end of current period (customer retains access)
	// false: cancel immediately (customer loses access now)
	CancelAtPeriodEnd bool

	// CancellationReason is optional customer feedback
	CancellationReason string
}

// PortalSessionParams contains parameters for creating portal session.
type PortalSessionParams struct {
	// TenantID is required for multi-tenant isolation
	TenantID pgtype.UUID

	// UserID is the customer requesting portal access
	UserID pgtype.UUID

	// ReturnURL is where to redirect after portal session
	ReturnURL string
}

// SyncSubscriptionParams contains parameters for syncing subscription from webhook.
type SyncSubscriptionParams struct {
	// TenantID is required for multi-tenant isolation
	TenantID pgtype.UUID

	// ProviderSubscriptionID is the Stripe subscription ID (sub_...)
	ProviderSubscriptionID string

	// EventType is the webhook event type (e.g., "customer.subscription.updated")
	EventType string

	// EventID is the Stripe event ID for idempotency
	EventID string
}

// SubscriptionDetail contains complete subscription information.
type SubscriptionDetail struct {
	// Database fields
	ID              pgtype.UUID
	TenantID        pgtype.UUID
	UserID          pgtype.UUID
	Status          string
	BillingInterval string
	SubtotalCents   int32
	TaxCents        int32
	ShippingCents   int32
	TotalCents      int32
	Currency        string

	// Stripe integration
	ProviderSubscriptionID string
	ProviderCustomerID     string
	Provider               string

	// Dates
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
	NextBillingDate    time.Time
	CancelAtPeriodEnd  bool
	CancelledAt        *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time

	// Related entities
	Items           []SubscriptionItemDetail
	ShippingAddress *AddressDetail
	PaymentMethod   *PaymentMethodDetail

	// Upcoming invoice preview (optional)
	UpcomingInvoice *UpcomingInvoiceDetail
}

// SubscriptionSummary contains subscription list item information.
type SubscriptionSummary struct {
	ID                pgtype.UUID
	Status            string
	BillingInterval   string
	TotalCents        int32
	Currency          string
	NextBillingDate   time.Time
	CancelAtPeriodEnd bool
	ProductName       string
	ProductImageURL   string
	CreatedAt         time.Time
}

// SubscriptionItemDetail contains subscription item details.
type SubscriptionItemDetail struct {
	ID             pgtype.UUID
	ProductSKUID   pgtype.UUID
	ProductName    string
	SKU            string
	Quantity       int32
	UnitPriceCents int32
	ImageURL       string
	WeightValue    string
	WeightUnit     string
	Grind          string
}

// AddressDetail contains address information.
type AddressDetail struct {
	ID           pgtype.UUID
	FullName     string
	Company      string
	AddressLine1 string
	AddressLine2 string
	City         string
	State        string
	PostalCode   string
	Country      string
	Phone        string
}

// PaymentMethodDetail contains payment method display information.
type PaymentMethodDetail struct {
	ID              pgtype.UUID
	MethodType      string
	DisplayBrand    string
	DisplayLast4    string
	DisplayExpMonth int32
	DisplayExpYear  int32
}

// UpcomingInvoiceDetail contains next invoice preview.
type UpcomingInvoiceDetail struct {
	AmountDueCents int32
	Currency       string
	PeriodStart    time.Time
	PeriodEnd      time.Time
}

// Billing interval constants for validation
const (
	BillingIntervalWeekly       = "weekly"
	BillingIntervalBiweekly     = "biweekly"
	BillingIntervalMonthly      = "monthly"
	BillingIntervalEvery6Weeks  = "every_6_weeks"
	BillingIntervalEvery2Months = "every_2_months"
)

// ValidBillingIntervals lists all valid billing interval values.
var ValidBillingIntervals = []string{
	BillingIntervalWeekly,
	BillingIntervalBiweekly,
	BillingIntervalMonthly,
	BillingIntervalEvery6Weeks,
	BillingIntervalEvery2Months,
}

// IsValidBillingInterval checks if the given interval is valid.
func IsValidBillingInterval(interval string) bool {
	for _, v := range ValidBillingIntervals {
		if v == interval {
			return true
		}
	}
	return false
}

// MapBillingIntervalToStripe converts our billing interval to Stripe's interval format.
// Returns interval (week/month) and interval_count.
func MapBillingIntervalToStripe(interval string) (stripeInterval string, intervalCount int32, err error) {
	switch interval {
	case BillingIntervalWeekly:
		return "week", 1, nil
	case BillingIntervalBiweekly:
		return "week", 2, nil
	case BillingIntervalMonthly:
		return "month", 1, nil
	case BillingIntervalEvery6Weeks:
		return "week", 6, nil
	case BillingIntervalEvery2Months:
		return "month", 2, nil
	default:
		return "", 0, ErrInvalidBillingInterval
	}
}
