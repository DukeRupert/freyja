package interfaces

import (
	"context"
	"time"
)

// =============================================================================
// Checkout Service Interface
// =============================================================================

type CheckoutService interface {
	// Create checkout session
	CreateCheckoutSession(ctx context.Context, customerID *int32, sessionID *string, successURL, cancelURL string) (*CheckoutSessionResponse, error)
	
	// Handle webhook events from payment providers
	HandleWebhookEvent(ctx context.Context, eventType string, eventData map[string]interface{}) error
	
	// Session management
	// GetCheckoutSession(ctx context.Context, sessionID string) (*CheckoutSessionDetails, error)
	// CancelCheckoutSession(ctx context.Context, sessionID string) error
	
	// Cart to order conversion
	// ConvertCartToOrder(ctx context.Context, customerID int32, paymentIntentID string) (*Order, error)
	
	// Payment processing
	// ProcessPaymentSuccess(ctx context.Context, paymentIntentID string, orderID int32) error
	// ProcessPaymentFailure(ctx context.Context, paymentIntentID string, errorMessage string) error
	
	// Validation
	// ValidateCheckoutEligibility(ctx context.Context, customerID *int32, sessionID *string) (*CheckoutValidationResult, error)
	
	// Analytics and reporting
	// GetCheckoutStats(ctx context.Context, filters CheckoutStatsFilters) (*CheckoutStats, error)
	// GetAbandonedCheckouts(ctx context.Context, filters AbandonedCheckoutFilters) ([]AbandonedCheckout, error)
}

// =============================================================================
// Request/Response Types
// =============================================================================

type CreateCheckoutSessionRequest struct {
	CustomerID *int32  `json:"customer_id,omitempty"`
	SessionID  *string `json:"session_id,omitempty"`
	SuccessURL string  `json:"success_url" validate:"required,url"`
	CancelURL  string  `json:"cancel_url" validate:"required,url"`
}

type CheckoutSessionDetails struct {
	SessionID     string                `json:"session_id"`
	CheckoutURL   string                `json:"checkout_url"`
	Status        string                `json:"status"`
	CustomerID    *int32                `json:"customer_id,omitempty"`
	CartID        int32                 `json:"cart_id"`
	TotalAmount   int32                 `json:"total_amount"`
	Items         []CartItemWithVariant `json:"items"`
	CreatedAt     time.Time             `json:"created_at"`
	ExpiresAt     *time.Time            `json:"expires_at,omitempty"`
	PaymentMethod *string               `json:"payment_method,omitempty"`
}

type CheckoutValidationResult struct {
	Valid        bool                  `json:"valid"`
	Errors       []string              `json:"errors,omitempty"`
	Warnings     []string              `json:"warnings,omitempty"`
	Cart         *CartWithItems        `json:"cart,omitempty"`
	TotalAmount  int32                 `json:"total_amount"`
	ItemCount    int32                 `json:"item_count"`
	OutOfStock   []CartItemWithVariant `json:"out_of_stock,omitempty"`
	PriceChanges []PriceChange         `json:"price_changes,omitempty"`
}

type PriceChange struct {
	ProductID int32  `json:"product_id"`
	ItemID    int32  `json:"item_id"`
	OldPrice  int32  `json:"old_price"`
	NewPrice  int32  `json:"new_price"`
	ProductName string `json:"product_name"`
}

// =============================================================================
// Analytics and Reporting Types
// =============================================================================

type CheckoutStatsFilters struct {
	DateFrom   *time.Time `json:"date_from,omitempty"`
	DateTo     *time.Time `json:"date_to,omitempty"`
	Status     *string    `json:"status,omitempty"`
	CustomerID *int32     `json:"customer_id,omitempty"`
}

type CheckoutStats struct {
	TotalSessions      int64   `json:"total_sessions"`
	CompletedSessions  int64   `json:"completed_sessions"`
	AbandonedSessions  int64   `json:"abandoned_sessions"`
	ConversionRate     float64 `json:"conversion_rate"`
	TotalRevenue       int64   `json:"total_revenue"`
	AverageCartValue   int64   `json:"average_cart_value"`
	TopAbandonReasons  []AbandonReason `json:"top_abandon_reasons"`
	SessionsByStatus   map[string]int64 `json:"sessions_by_status"`
}

type AbandonReason struct {
	Reason string `json:"reason"`
	Count  int64  `json:"count"`
}

type AbandonedCheckoutFilters struct {
	DateFrom     *time.Time `json:"date_from,omitempty"`
	DateTo       *time.Time `json:"date_to,omitempty"`
	MinValue     *int32     `json:"min_value,omitempty"`
	CustomerID   *int32     `json:"customer_id,omitempty"`
	HasEmail     *bool      `json:"has_email,omitempty"`
	DaysAbandoned *int      `json:"days_abandoned,omitempty"`
	Limit        int        `json:"limit,omitempty"`
	Offset       int        `json:"offset,omitempty"`
}

type AbandonedCheckout struct {
	SessionID     string     `json:"session_id"`
	CustomerID    *int32     `json:"customer_id,omitempty"`
	CustomerEmail *string    `json:"customer_email,omitempty"`
	CartValue     int32      `json:"cart_value"`
	ItemCount     int32      `json:"item_count"`
	CreatedAt     time.Time  `json:"created_at"`
	AbandonedAt   time.Time  `json:"abandoned_at"`
	LastActivity  time.Time  `json:"last_activity"`
	RecoveryURL   *string    `json:"recovery_url,omitempty"`
	Items         []CartItemWithVariant `json:"items"`
}

// =============================================================================
// Webhook Event Types
// =============================================================================

type WebhookEventType string

const (
	WebhookCheckoutSessionCompleted WebhookEventType = "checkout.session.completed"
	WebhookPaymentIntentSucceeded   WebhookEventType = "payment_intent.succeeded"
	WebhookPaymentIntentFailed      WebhookEventType = "payment_intent.payment_failed"
	WebhookCustomerCreated          WebhookEventType = "customer.created"
	WebhookCustomerUpdated          WebhookEventType = "customer.updated"
)

type WebhookProcessingResult struct {
	EventType     string                 `json:"event_type"`
	EventID       string                 `json:"event_id"`
	Processed     bool                   `json:"processed"`
	OrderCreated  *int32                 `json:"order_created,omitempty"`
	ErrorMessage  *string                `json:"error_message,omitempty"`
	ProcessedAt   time.Time              `json:"processed_at"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}