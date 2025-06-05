// internal/interfaces/payment.go
package interfaces

import (
	"context"
	"time"

	"github.com/dukerupert/freyja/internal/database"
)

// =============================================================================
// Payment Provider Interface
// =============================================================================

type PaymentProvider interface {
	// Create a checkout session for the customer
	CreateCheckoutSession(ctx context.Context, req CheckoutSessionRequest) (*CheckoutSessionResponse, error)
	
	// Verify webhook signature and parse event
	VerifyWebhook(payload []byte, signature string) (*PaymentWebhookEvent, error)
	
	// Create a customer in the payment provider
	CreateCustomer(ctx context.Context, customer database.Customers) (*PaymentCustomer, error)
	
	// Get customer from payment provider
	GetCustomer(ctx context.Context, customerID string) (*PaymentCustomer, error)
	
	// Refund a payment
	RefundPayment(ctx context.Context, paymentID string, amount int) (*RefundResponse, error)
}

// Request/Response types for checkout sessions
type CheckoutSessionRequest struct {
	CustomerID    *int32     `json:"customer_id,omitempty"`
	Items         []CartItem `json:"items" validate:"required,min=1"`
	SuccessURL    string     `json:"success_url" validate:"required,url"`
	CancelURL     string     `json:"cancel_url" validate:"required,url"`
	CustomerEmail *string    `json:"customer_email,omitempty"`
}

type CheckoutSessionResponse struct {
	SessionID   string `json:"session_id"`
	CheckoutURL string `json:"checkout_url"`
}

// Webhook event from payment provider
type PaymentWebhookEvent struct {
	Type      string                 `json:"type"`
	ID        string                 `json:"id"`
	Data      map[string]interface{} `json:"data"`
	CreatedAt time.Time              `json:"created_at"`
}

// Payment provider customer representation
type PaymentCustomer struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// Refund response
type RefundResponse struct {
	ID     string `json:"id"`
	Amount int    `json:"amount"`
	Status string `json:"status"`
}