package billing

import (
	"errors"
	"fmt"
)

var (
	// ErrNotImplemented is returned when a provider method is not yet implemented.
	// Used for subscription and advanced features not needed in MVP.
	ErrNotImplemented = errors.New("billing: method not implemented")

	// ErrInvalidAPIKey is returned when Stripe API key is invalid or missing.
	ErrInvalidAPIKey = errors.New("billing: invalid or missing API key")

	// ErrPaymentIntentNotFound is returned when payment intent does not exist.
	ErrPaymentIntentNotFound = errors.New("billing: payment intent not found")

	// ErrPaymentFailed is returned when payment fails (card declined, etc.)
	ErrPaymentFailed = errors.New("billing: payment failed")

	// ErrInvalidWebhookSignature is returned when webhook signature verification fails.
	ErrInvalidWebhookSignature = errors.New("billing: invalid webhook signature")

	// ErrIdempotencyConflict is returned when idempotency key matches a different request.
	ErrIdempotencyConflict = errors.New("billing: idempotency key conflict")

	// ErrAmountTooSmall is returned when payment amount is below Stripe's minimum.
	ErrAmountTooSmall = errors.New("billing: amount too small (minimum $0.50 USD)")

	// ErrSubscriptionNotFound is returned when subscription does not exist or tenant mismatch.
	ErrSubscriptionNotFound = errors.New("billing: subscription not found")
)

// StripeError wraps a Stripe API error with additional context.
type StripeError struct {
	Message       string // Human-readable error message
	Code          string // Stripe error code (e.g., "card_declined")
	DeclineCode   string // Card decline reason (if applicable)
	StripeCode    string // HTTP status code from Stripe
	RequestID     string // Stripe request ID for debugging
	OriginalError error  // Original error from Stripe SDK
}

func (e *StripeError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("stripe: %s (code: %s)", e.Message, e.Code)
	}
	return fmt.Sprintf("stripe: %s", e.Message)
}

func (e *StripeError) Unwrap() error {
	return e.OriginalError
}

// IsDeclined returns true if error is due to card decline.
func (e *StripeError) IsDeclined() bool {
	return e.Code == "card_declined" || e.DeclineCode != ""
}

// IsTemporary returns true if error is likely transient and retryable.
func (e *StripeError) IsTemporary() bool {
	return e.Code == "rate_limit" || e.Code == "api_connection_error"
}
