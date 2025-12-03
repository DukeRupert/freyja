package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/dukerupert/freyja/internal/email"
	"github.com/dukerupert/freyja/internal/repository"
)

// Job type constants for email jobs
const (
	JobTypePasswordReset             = "email:password_reset"
	JobTypeOrderConfirmation         = "email:order_confirmation"
	JobTypeShippingConfirmation      = "email:shipping_confirmation"
	JobTypeSubscriptionWelcome       = "email:subscription_welcome"
	JobTypeSubscriptionPaymentFailed = "email:subscription_payment_failed"
	JobTypeSubscriptionCancelled     = "email:subscription_cancelled"
)

// Email job payloads (JSON-serializable)

// PasswordResetPayload represents the payload for a password reset email job
type PasswordResetPayload struct {
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	ResetURL  string    `json:"reset_url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// OrderConfirmationPayload represents the payload for an order confirmation email job
type OrderConfirmationPayload struct {
	OrderID       uuid.UUID `json:"order_id"`
	Email         string    `json:"email"`
	CustomerName  string    `json:"customer_name"`
	OrderNumber   string    `json:"order_number"`
	OrderDate     time.Time `json:"order_date"`
	SubtotalCents int64     `json:"subtotal_cents"`
	ShippingCents int64     `json:"shipping_cents"`
	TaxCents      int64     `json:"tax_cents"`
	TotalCents    int64     `json:"total_cents"`
}

// ShippingConfirmationPayload represents the payload for a shipping confirmation email job
type ShippingConfirmationPayload struct {
	OrderID        uuid.UUID `json:"order_id"`
	Email          string    `json:"email"`
	CustomerName   string    `json:"customer_name"`
	OrderNumber    string    `json:"order_number"`
	Carrier        string    `json:"carrier"`
	TrackingNumber string    `json:"tracking_number"`
	TrackingURL    string    `json:"tracking_url"`
}

// SubscriptionWelcomePayload represents the payload for a subscription welcome email job
type SubscriptionWelcomePayload struct {
	SubscriptionID uuid.UUID `json:"subscription_id"`
	Email          string    `json:"email"`
	CustomerName   string    `json:"customer_name"`
	ProductName    string    `json:"product_name"`
	Frequency      string    `json:"frequency"`
}

// SubscriptionPaymentFailedPayload represents the payload for a subscription payment failed email job
type SubscriptionPaymentFailedPayload struct {
	SubscriptionID uuid.UUID `json:"subscription_id"`
	Email          string    `json:"email"`
	CustomerName   string    `json:"customer_name"`
	ProductName    string    `json:"product_name"`
	FailedDate     time.Time `json:"failed_date"`
	RetryDate      time.Time `json:"retry_date"`
}

// SubscriptionCancelledPayload represents the payload for a subscription cancelled email job
type SubscriptionCancelledPayload struct {
	SubscriptionID uuid.UUID `json:"subscription_id"`
	Email          string    `json:"email"`
	CustomerName   string    `json:"customer_name"`
	ProductName    string    `json:"product_name"`
	CancelledDate  time.Time `json:"cancelled_date"`
}

// Job enqueueing functions

// EnqueuePasswordResetEmail enqueues a password reset email job
func EnqueuePasswordResetEmail(ctx context.Context, q *repository.Queries, tenantID uuid.UUID, payload PasswordResetPayload) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = q.EnqueueJob(ctx, repository.EnqueueJobParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		JobType:    JobTypePasswordReset,
		Queue:      "email",
		Payload:    payloadJSON,
		Priority:   50, // Higher priority for password resets
		MaxRetries: 3,
		ScheduledAt: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
		TimeoutSeconds: 30,
		Metadata:       []byte("{}"),
	})

	return err
}

// EnqueueOrderConfirmationEmail enqueues an order confirmation email job
func EnqueueOrderConfirmationEmail(ctx context.Context, q *repository.Queries, tenantID uuid.UUID, payload OrderConfirmationPayload) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = q.EnqueueJob(ctx, repository.EnqueueJobParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		JobType:    JobTypeOrderConfirmation,
		Queue:      "email",
		Payload:    payloadJSON,
		Priority:   100,
		MaxRetries: 3,
		ScheduledAt: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
		TimeoutSeconds: 30,
		Metadata:       []byte("{}"),
	})

	return err
}

// EnqueueShippingConfirmationEmail enqueues a shipping confirmation email job
func EnqueueShippingConfirmationEmail(ctx context.Context, q *repository.Queries, tenantID uuid.UUID, payload ShippingConfirmationPayload) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = q.EnqueueJob(ctx, repository.EnqueueJobParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		JobType:    JobTypeShippingConfirmation,
		Queue:      "email",
		Payload:    payloadJSON,
		Priority:   100,
		MaxRetries: 3,
		ScheduledAt: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
		TimeoutSeconds: 30,
		Metadata:       []byte("{}"),
	})

	return err
}

// EnqueueSubscriptionWelcomeEmail enqueues a subscription welcome email job
func EnqueueSubscriptionWelcomeEmail(ctx context.Context, q *repository.Queries, tenantID uuid.UUID, payload SubscriptionWelcomePayload) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = q.EnqueueJob(ctx, repository.EnqueueJobParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		JobType:    JobTypeSubscriptionWelcome,
		Queue:      "email",
		Payload:    payloadJSON,
		Priority:   100,
		MaxRetries: 3,
		ScheduledAt: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
		TimeoutSeconds: 30,
		Metadata:       []byte("{}"),
	})

	return err
}

// EnqueueSubscriptionPaymentFailedEmail enqueues a subscription payment failed email job
func EnqueueSubscriptionPaymentFailedEmail(ctx context.Context, q *repository.Queries, tenantID uuid.UUID, payload SubscriptionPaymentFailedPayload) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = q.EnqueueJob(ctx, repository.EnqueueJobParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		JobType:    JobTypeSubscriptionPaymentFailed,
		Queue:      "email",
		Payload:    payloadJSON,
		Priority:   75, // Higher priority for payment issues
		MaxRetries: 3,
		ScheduledAt: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
		TimeoutSeconds: 30,
		Metadata:       []byte("{}"),
	})

	return err
}

// EnqueueSubscriptionCancelledEmail enqueues a subscription cancelled email job
func EnqueueSubscriptionCancelledEmail(ctx context.Context, q *repository.Queries, tenantID uuid.UUID, payload SubscriptionCancelledPayload) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = q.EnqueueJob(ctx, repository.EnqueueJobParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		JobType:    JobTypeSubscriptionCancelled,
		Queue:      "email",
		Payload:    payloadJSON,
		Priority:   100,
		MaxRetries: 3,
		ScheduledAt: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
		TimeoutSeconds: 30,
		Metadata:       []byte("{}"),
	})

	return err
}

// ProcessEmailJob processes an email job based on its type
func ProcessEmailJob(ctx context.Context, job *repository.Job, emailService *email.Service, queries *repository.Queries) error {
	// TODO: Implement job processing logic
	// 1. Switch on job.JobType
	// 2. Unmarshal job.Payload into appropriate payload struct
	// 3. Fetch additional data from database if needed (order details, addresses, etc.)
	// 4. Call appropriate emailService.Send* method
	// 5. Return error if sending fails (job will be retried)

	switch job.JobType {
	case JobTypePasswordReset:
		// var payload PasswordResetPayload
		// json.Unmarshal(job.Payload, &payload)
		// return emailService.SendPasswordReset(ctx, ...)
		return fmt.Errorf("not implemented: %s", job.JobType)

	case JobTypeOrderConfirmation:
		// var payload OrderConfirmationPayload
		// json.Unmarshal(job.Payload, &payload)
		// Fetch order details, line items, addresses from database
		// return emailService.SendOrderConfirmation(ctx, ...)
		return fmt.Errorf("not implemented: %s", job.JobType)

	case JobTypeShippingConfirmation:
		// var payload ShippingConfirmationPayload
		// json.Unmarshal(job.Payload, &payload)
		// Fetch order details, line items, shipping address from database
		// return emailService.SendShippingConfirmation(ctx, ...)
		return fmt.Errorf("not implemented: %s", job.JobType)

	case JobTypeSubscriptionWelcome:
		// var payload SubscriptionWelcomePayload
		// json.Unmarshal(job.Payload, &payload)
		// Fetch subscription details from database
		// return emailService.SendSubscriptionWelcome(ctx, ...)
		return fmt.Errorf("not implemented: %s", job.JobType)

	case JobTypeSubscriptionPaymentFailed:
		// var payload SubscriptionPaymentFailedPayload
		// json.Unmarshal(job.Payload, &payload)
		// return emailService.SendSubscriptionPaymentFailed(ctx, ...)
		return fmt.Errorf("not implemented: %s", job.JobType)

	case JobTypeSubscriptionCancelled:
		// var payload SubscriptionCancelledPayload
		// json.Unmarshal(job.Payload, &payload)
		// return emailService.SendSubscriptionCancelled(ctx, ...)
		return fmt.Errorf("not implemented: %s", job.JobType)

	default:
		return fmt.Errorf("unknown job type: %s", job.JobType)
	}
}
