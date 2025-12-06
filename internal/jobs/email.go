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
	JobTypeEmailVerification         = "email:email_verification"
	JobTypeOrderConfirmation         = "email:order_confirmation"
	JobTypeShippingConfirmation      = "email:shipping_confirmation"
	JobTypeSubscriptionWelcome       = "email:subscription_welcome"
	JobTypeSubscriptionPaymentFailed = "email:subscription_payment_failed"
	JobTypeSubscriptionCancelled     = "email:subscription_cancelled"
	JobTypeInvoiceSent               = "email:invoice_sent"
	JobTypeInvoiceReminder           = "email:invoice_reminder"
	JobTypeInvoiceOverdue            = "email:invoice_overdue"

	// SaaS platform email jobs
	JobTypeOperatorSetup         = "email:operator_setup"
	JobTypeOperatorPasswordReset = "email:operator_password_reset"
	JobTypePlatformPaymentFailed = "email:platform_payment_failed"
	JobTypePlatformSuspended     = "email:platform_suspended"

	// Wholesale application email jobs
	JobTypeWholesaleApproved = "email:wholesale_approved"
	JobTypeWholesaleRejected = "email:wholesale_rejected"
)

// Email job payloads (JSON-serializable)

// PasswordResetPayload represents the payload for a password reset email job
type PasswordResetPayload struct {
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	ResetURL  string    `json:"reset_url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// EmailVerificationPayload represents the payload for an email verification job
type EmailVerificationPayload struct {
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	VerifyURL string    `json:"verify_url"`
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

// InvoiceSentPayload represents the payload for an invoice sent email job
type InvoiceSentPayload struct {
	InvoiceID     uuid.UUID          `json:"invoice_id"`
	Email         string             `json:"email"`
	CustomerName  string             `json:"customer_name"`
	InvoiceNumber string             `json:"invoice_number"`
	InvoiceDate   time.Time          `json:"invoice_date"`
	DueDate       time.Time          `json:"due_date"`
	PaymentTerms  string             `json:"payment_terms"`
	Items         []InvoiceItemData  `json:"items"`
	SubtotalCents int64              `json:"subtotal_cents"`
	ShippingCents int64              `json:"shipping_cents"`
	TaxCents      int64              `json:"tax_cents"`
	DiscountCents int64              `json:"discount_cents"`
	TotalCents    int64              `json:"total_cents"`
	PaymentURL    string             `json:"payment_url"`
}

// InvoiceItemData represents a line item in an invoice email
type InvoiceItemData struct {
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
	UnitCents   int64  `json:"unit_cents"`
	TotalCents  int64  `json:"total_cents"`
}

// InvoiceReminderPayload represents the payload for an invoice reminder email job
type InvoiceReminderPayload struct {
	InvoiceID     uuid.UUID `json:"invoice_id"`
	Email         string    `json:"email"`
	CustomerName  string    `json:"customer_name"`
	InvoiceNumber string    `json:"invoice_number"`
	DueDate       time.Time `json:"due_date"`
	BalanceCents  int64     `json:"balance_cents"`
	ReminderType  string    `json:"reminder_type"` // "approaching_due" or "past_due"
	DaysBefore    int       `json:"days_before"`
	DaysOverdue   int       `json:"days_overdue"`
	PaymentURL    string    `json:"payment_url"`
}

// InvoiceOverduePayload represents the payload for an invoice overdue email job
type InvoiceOverduePayload struct {
	InvoiceID     uuid.UUID `json:"invoice_id"`
	Email         string    `json:"email"`
	CustomerName  string    `json:"customer_name"`
	InvoiceNumber string    `json:"invoice_number"`
	DueDate       time.Time `json:"due_date"`
	BalanceCents  int64     `json:"balance_cents"`
	DaysOverdue   int       `json:"days_overdue"`
	PaymentURL    string    `json:"payment_url"`
}

// SaaS Platform Email Payloads

// OperatorSetupPayload represents the payload for an operator setup email job
type OperatorSetupPayload struct {
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	SetupURL  string    `json:"setup_url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// OperatorPasswordResetPayload represents the payload for an operator password reset email job
type OperatorPasswordResetPayload struct {
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	ResetURL  string    `json:"reset_url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// PlatformPaymentFailedPayload represents the payload for a platform payment failed email job
type PlatformPaymentFailedPayload struct {
	Email            string `json:"email"`
	Name             string `json:"name"`
	UpdatePaymentURL string `json:"update_payment_url"`
}

// PlatformSuspendedPayload represents the payload for a platform suspended email job
type PlatformSuspendedPayload struct {
	Email            string `json:"email"`
	Name             string `json:"name"`
	UpdatePaymentURL string `json:"update_payment_url"`
}

// Wholesale Application Email Payloads

// WholesaleApprovedPayload represents the payload for a wholesale approved email job
type WholesaleApprovedPayload struct {
	Email        string `json:"email"`
	CustomerName string `json:"customer_name"`
	LoginURL     string `json:"login_url"`
}

// WholesaleRejectedPayload represents the payload for a wholesale rejected email job
type WholesaleRejectedPayload struct {
	Email           string `json:"email"`
	CustomerName    string `json:"customer_name"`
	RejectionReason string `json:"rejection_reason"`
	ShopURL         string `json:"shop_url"`
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

// EnqueueEmailVerification enqueues an email verification email job
func EnqueueEmailVerification(ctx context.Context, q repository.Querier, tenantID uuid.UUID, payload EmailVerificationPayload) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = q.EnqueueJob(ctx, repository.EnqueueJobParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		JobType:    JobTypeEmailVerification,
		Queue:      "email",
		Payload:    payloadJSON,
		Priority:   50, // Higher priority for email verification
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

// EnqueueInvoiceSentEmail enqueues an invoice sent email job
func EnqueueInvoiceSentEmail(ctx context.Context, q repository.Querier, tenantID uuid.UUID, payload InvoiceSentPayload) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = q.EnqueueJob(ctx, repository.EnqueueJobParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		JobType:    JobTypeInvoiceSent,
		Queue:      "email",
		Payload:    payloadJSON,
		Priority:   75, // Higher priority for invoice emails
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

// EnqueueInvoiceReminderEmail enqueues an invoice reminder email job
func EnqueueInvoiceReminderEmail(ctx context.Context, q repository.Querier, tenantID uuid.UUID, payload InvoiceReminderPayload) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = q.EnqueueJob(ctx, repository.EnqueueJobParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		JobType:    JobTypeInvoiceReminder,
		Queue:      "email",
		Payload:    payloadJSON,
		Priority:   75,
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

// EnqueueInvoiceOverdueEmail enqueues an invoice overdue email job
func EnqueueInvoiceOverdueEmail(ctx context.Context, q repository.Querier, tenantID uuid.UUID, payload InvoiceOverduePayload) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = q.EnqueueJob(ctx, repository.EnqueueJobParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		JobType:    JobTypeInvoiceOverdue,
		Queue:      "email",
		Payload:    payloadJSON,
		Priority:   80, // Higher priority for overdue notifications
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

// SaaS Platform Email Enqueue Functions

// EnqueueOperatorSetupEmail enqueues an operator setup email job
func EnqueueOperatorSetupEmail(ctx context.Context, q repository.Querier, tenantID uuid.UUID, payload OperatorSetupPayload) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = q.EnqueueJob(ctx, repository.EnqueueJobParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		JobType:    JobTypeOperatorSetup,
		Queue:      "email",
		Payload:    payloadJSON,
		Priority:   50, // Higher priority for setup emails
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

// EnqueueOperatorPasswordResetEmail enqueues an operator password reset email job
func EnqueueOperatorPasswordResetEmail(ctx context.Context, q repository.Querier, tenantID uuid.UUID, payload OperatorPasswordResetPayload) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = q.EnqueueJob(ctx, repository.EnqueueJobParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		JobType:    JobTypeOperatorPasswordReset,
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

// EnqueuePlatformPaymentFailedEmail enqueues a platform payment failed email job
func EnqueuePlatformPaymentFailedEmail(ctx context.Context, q repository.Querier, tenantID uuid.UUID, payload PlatformPaymentFailedPayload) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = q.EnqueueJob(ctx, repository.EnqueueJobParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		JobType:    JobTypePlatformPaymentFailed,
		Queue:      "email",
		Payload:    payloadJSON,
		Priority:   80, // High priority for payment issues
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

// EnqueuePlatformSuspendedEmail enqueues a platform suspended email job
func EnqueuePlatformSuspendedEmail(ctx context.Context, q repository.Querier, tenantID uuid.UUID, payload PlatformSuspendedPayload) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = q.EnqueueJob(ctx, repository.EnqueueJobParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		JobType:    JobTypePlatformSuspended,
		Queue:      "email",
		Payload:    payloadJSON,
		Priority:   90, // Very high priority for suspension notices
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

// Wholesale Application Email Enqueue Functions

// EnqueueWholesaleApprovedEmail enqueues a wholesale approved email job
func EnqueueWholesaleApprovedEmail(ctx context.Context, q repository.Querier, tenantID uuid.UUID, payload WholesaleApprovedPayload) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = q.EnqueueJob(ctx, repository.EnqueueJobParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		JobType:    JobTypeWholesaleApproved,
		Queue:      "email",
		Payload:    payloadJSON,
		Priority:   75, // Higher priority for approval emails
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

// EnqueueWholesaleRejectedEmail enqueues a wholesale rejected email job
func EnqueueWholesaleRejectedEmail(ctx context.Context, q repository.Querier, tenantID uuid.UUID, payload WholesaleRejectedPayload) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = q.EnqueueJob(ctx, repository.EnqueueJobParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		JobType:    JobTypeWholesaleRejected,
		Queue:      "email",
		Payload:    payloadJSON,
		Priority:   75, // Higher priority for rejection emails
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

	case JobTypeEmailVerification:
		var payload EmailVerificationPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal email verification payload: %w", err)
		}

		emailData := email.EmailVerificationEmail{
			Email:     payload.Email,
			FirstName: payload.FirstName,
			VerifyURL: payload.VerifyURL,
			ExpiresAt: payload.ExpiresAt,
		}

		return emailService.SendEmailVerification(ctx, emailData)

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

	case JobTypeInvoiceSent:
		var payload InvoiceSentPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal invoice sent payload: %w", err)
		}

		// Convert items
		items := make([]email.InvoiceItem, len(payload.Items))
		for i, item := range payload.Items {
			items[i] = email.InvoiceItem{
				Description: item.Description,
				Quantity:    item.Quantity,
				UnitCents:   item.UnitCents,
				TotalCents:  item.TotalCents,
			}
		}

		emailData := email.InvoiceSentEmail{
			Email:         payload.Email,
			CustomerName:  payload.CustomerName,
			InvoiceNumber: payload.InvoiceNumber,
			InvoiceDate:   payload.InvoiceDate,
			DueDate:       payload.DueDate,
			PaymentTerms:  payload.PaymentTerms,
			Items:         items,
			SubtotalCents: payload.SubtotalCents,
			ShippingCents: payload.ShippingCents,
			TaxCents:      payload.TaxCents,
			DiscountCents: payload.DiscountCents,
			TotalCents:    payload.TotalCents,
			PaymentURL:    payload.PaymentURL,
		}

		return emailService.SendInvoiceSent(ctx, emailData)

	case JobTypeInvoiceReminder:
		var payload InvoiceReminderPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal invoice reminder payload: %w", err)
		}

		emailData := email.InvoiceReminderEmail{
			Email:         payload.Email,
			CustomerName:  payload.CustomerName,
			InvoiceNumber: payload.InvoiceNumber,
			DueDate:       payload.DueDate,
			BalanceCents:  payload.BalanceCents,
			ReminderType:  payload.ReminderType,
			DaysBefore:    payload.DaysBefore,
			DaysOverdue:   payload.DaysOverdue,
			PaymentURL:    payload.PaymentURL,
		}

		return emailService.SendInvoiceReminder(ctx, emailData)

	case JobTypeInvoiceOverdue:
		var payload InvoiceOverduePayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal invoice overdue payload: %w", err)
		}

		emailData := email.InvoiceOverdueEmail{
			Email:         payload.Email,
			CustomerName:  payload.CustomerName,
			InvoiceNumber: payload.InvoiceNumber,
			DueDate:       payload.DueDate,
			BalanceCents:  payload.BalanceCents,
			DaysOverdue:   payload.DaysOverdue,
			PaymentURL:    payload.PaymentURL,
		}

		return emailService.SendInvoiceOverdue(ctx, emailData)

	// SaaS Platform Email Jobs
	case JobTypeOperatorSetup:
		var payload OperatorSetupPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal operator setup payload: %w", err)
		}

		emailData := email.OperatorSetupEmail{
			Email:     payload.Email,
			Name:      payload.Name,
			SetupURL:  payload.SetupURL,
			ExpiresAt: payload.ExpiresAt,
		}

		return emailService.SendOperatorSetup(ctx, emailData)

	case JobTypeOperatorPasswordReset:
		var payload OperatorPasswordResetPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal operator password reset payload: %w", err)
		}

		emailData := email.OperatorPasswordResetEmail{
			Email:     payload.Email,
			Name:      payload.Name,
			ResetURL:  payload.ResetURL,
			ExpiresAt: payload.ExpiresAt,
		}

		return emailService.SendOperatorPasswordReset(ctx, emailData)

	case JobTypePlatformPaymentFailed:
		var payload PlatformPaymentFailedPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal platform payment failed payload: %w", err)
		}

		emailData := email.PlatformPaymentFailedEmail{
			Email:            payload.Email,
			Name:             payload.Name,
			UpdatePaymentURL: payload.UpdatePaymentURL,
		}

		return emailService.SendPlatformPaymentFailed(ctx, emailData)

	case JobTypePlatformSuspended:
		var payload PlatformSuspendedPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal platform suspended payload: %w", err)
		}

		emailData := email.PlatformSuspendedEmail{
			Email:            payload.Email,
			Name:             payload.Name,
			UpdatePaymentURL: payload.UpdatePaymentURL,
		}

		return emailService.SendPlatformSuspended(ctx, emailData)

	// Wholesale Application Email Jobs
	case JobTypeWholesaleApproved:
		var payload WholesaleApprovedPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal wholesale approved payload: %w", err)
		}

		emailData := email.WholesaleApprovedEmail{
			Email:        payload.Email,
			CustomerName: payload.CustomerName,
			LoginURL:     payload.LoginURL,
		}

		return emailService.SendWholesaleApproved(ctx, emailData)

	case JobTypeWholesaleRejected:
		var payload WholesaleRejectedPayload
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal wholesale rejected payload: %w", err)
		}

		emailData := email.WholesaleRejectedEmail{
			Email:           payload.Email,
			CustomerName:    payload.CustomerName,
			RejectionReason: payload.RejectionReason,
			ShopURL:         payload.ShopURL,
		}

		return emailService.SendWholesaleRejected(ctx, emailData)

	default:
		return fmt.Errorf("unknown job type: %s", job.JobType)
	}
}
