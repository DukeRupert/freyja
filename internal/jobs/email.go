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
func EnqueuePasswordResetEmail(ctx context.Context, q repository.Querier, tenantID uuid.UUID, payload PasswordResetPayload) error {
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
func EnqueueOrderConfirmationEmail(ctx context.Context, q repository.Querier, tenantID uuid.UUID, payload OrderConfirmationPayload) error {
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
func EnqueueShippingConfirmationEmail(ctx context.Context, q repository.Querier, tenantID uuid.UUID, payload ShippingConfirmationPayload) error {
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
func EnqueueSubscriptionWelcomeEmail(ctx context.Context, q repository.Querier, tenantID uuid.UUID, payload SubscriptionWelcomePayload) error {
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
func EnqueueSubscriptionPaymentFailedEmail(ctx context.Context, q repository.Querier, tenantID uuid.UUID, payload SubscriptionPaymentFailedPayload) error {
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
func EnqueueSubscriptionCancelledEmail(ctx context.Context, q repository.Querier, tenantID uuid.UUID, payload SubscriptionCancelledPayload) error {
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
	switch job.JobType {
	case JobTypePasswordReset:
		var payload PasswordResetPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal password reset payload: %w", err)
		}

		emailData := email.PasswordResetEmail{
			Email:     payload.Email,
			FirstName: payload.FirstName,
			ResetURL:  payload.ResetURL,
			ExpiresAt: payload.ExpiresAt,
		}

		return emailService.SendPasswordReset(ctx, emailData)

	case JobTypeOrderConfirmation:
		var payload OrderConfirmationPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal order confirmation payload: %w", err)
		}

		emailData := email.OrderConfirmationEmail{
			OrderNumber:   payload.OrderNumber,
			CustomerName:  payload.Email,
			OrderDate:     payload.OrderDate,
			Items:         []email.OrderItem{},
			SubtotalCents: payload.SubtotalCents,
			ShippingCents: payload.ShippingCents,
			TaxCents:      payload.TaxCents,
			TotalCents:    payload.TotalCents,
			ShippingAddr:  email.Address{},
			BillingAddr:   email.Address{},
		}

		return emailService.SendOrderConfirmation(ctx, emailData)

	case JobTypeShippingConfirmation:
		var payload ShippingConfirmationPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal shipping confirmation payload: %w", err)
		}

		emailData := email.ShippingConfirmationEmail{
			OrderNumber:    payload.OrderNumber,
			CustomerName:   payload.Email,
			ShippedDate:    time.Now(),
			Items:          []email.OrderItem{},
			ShippingAddr:   email.Address{},
			Carrier:        payload.Carrier,
			TrackingNumber: payload.TrackingNumber,
			TrackingURL:    payload.TrackingURL,
		}

		return emailService.SendShippingConfirmation(ctx, emailData)

	case JobTypeSubscriptionWelcome:
		var payload SubscriptionWelcomePayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal subscription welcome payload: %w", err)
		}

		emailData := email.SubscriptionWelcomeEmail{
			CustomerName:      payload.Email,
			ProductName:       payload.ProductName,
			Frequency:         payload.Frequency,
			NextDeliveryDate:  time.Now().AddDate(0, 0, 14),
			ManagementURL:     "/account/subscriptions",
			ShippingAddr:      email.Address{},
			SubscriptionTotal: 0,
		}

		return emailService.SendSubscriptionWelcome(ctx, emailData)

	case JobTypeSubscriptionPaymentFailed:
		var payload SubscriptionPaymentFailedPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal subscription payment failed payload: %w", err)
		}

		emailData := email.SubscriptionPaymentFailedEmail{
			CustomerName:     payload.Email,
			ProductName:      payload.ProductName,
			FailedDate:       payload.FailedDate,
			RetryDate:        payload.RetryDate,
			UpdatePaymentURL: "/account/payment-methods",
			ManagementURL:    "/account/subscriptions",
		}

		return emailService.SendSubscriptionPaymentFailed(ctx, emailData)

	case JobTypeSubscriptionCancelled:
		var payload SubscriptionCancelledPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal subscription cancelled payload: %w", err)
		}

		emailData := email.SubscriptionCancelledEmail{
			CustomerName:      payload.Email,
			ProductName:       payload.ProductName,
			CancelledDate:     payload.CancelledDate,
			FinalDeliveryDate: time.Time{},
			HasFinalDelivery:  false,
			ReactivationURL:   "/subscriptions",
		}

		return emailService.SendSubscriptionCancelled(ctx, emailData)

	default:
		return fmt.Errorf("unknown job type: %s", job.JobType)
	}
}
