package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/dukerupert/freyja/internal/repository"
)

// Job type constants for invoice jobs
const (
	JobTypeGenerateConsolidatedInvoice = "invoice:generate_consolidated"
	JobTypeMarkOverdueInvoices         = "invoice:mark_overdue"
	JobTypeSendInvoiceReminder         = "invoice:send_reminder"
	JobTypeSyncInvoiceFromStripe       = "invoice:sync_stripe"
)

// Invoice job payloads (JSON-serializable)

// GenerateConsolidatedInvoicePayload represents the payload for generating a consolidated invoice
type GenerateConsolidatedInvoicePayload struct {
	UserID             uuid.UUID `json:"user_id"`
	BillingPeriodStart time.Time `json:"billing_period_start"`
	BillingPeriodEnd   time.Time `json:"billing_period_end"`
}

// MarkOverdueInvoicesPayload represents the payload for the nightly overdue check
// This job processes all overdue invoices for a tenant
type MarkOverdueInvoicesPayload struct {
	// Empty - processes all overdue invoices for tenant
}

// SendInvoiceReminderPayload represents the payload for sending an invoice reminder
type SendInvoiceReminderPayload struct {
	InvoiceID    uuid.UUID `json:"invoice_id"`
	ReminderType string    `json:"reminder_type"` // "approaching_due", "past_due"
	DaysBefore   int       `json:"days_before"`   // Days before due date (for approaching_due)
	DaysOverdue  int       `json:"days_overdue"`  // Days past due date (for past_due)
}

// SyncInvoiceFromStripePayload represents the payload for syncing an invoice from Stripe
type SyncInvoiceFromStripePayload struct {
	StripeInvoiceID string `json:"stripe_invoice_id"`
}

// EnqueueGenerateConsolidatedInvoice enqueues a job to generate a consolidated invoice for a customer
func EnqueueGenerateConsolidatedInvoice(ctx context.Context, q repository.Querier, tenantID uuid.UUID, payload GenerateConsolidatedInvoicePayload) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = q.EnqueueJob(ctx, repository.EnqueueJobParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		JobType:    JobTypeGenerateConsolidatedInvoice,
		Queue:      "invoicing",
		Payload:    payloadJSON,
		Priority:   100,
		MaxRetries: 3,
		ScheduledAt: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
		TimeoutSeconds: 60,
		Metadata:       []byte("{}"),
	})

	return err
}

// EnqueueMarkOverdueInvoices enqueues a job to mark overdue invoices
// Typically scheduled to run nightly
func EnqueueMarkOverdueInvoices(ctx context.Context, q repository.Querier, tenantID uuid.UUID, scheduledAt time.Time) error {
	payload := MarkOverdueInvoicesPayload{}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = q.EnqueueJob(ctx, repository.EnqueueJobParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		JobType:    JobTypeMarkOverdueInvoices,
		Queue:      "invoicing",
		Payload:    payloadJSON,
		Priority:   50, // Lower priority - can run in off-peak hours
		MaxRetries: 3,
		ScheduledAt: pgtype.Timestamptz{
			Time:  scheduledAt,
			Valid: true,
		},
		TimeoutSeconds: 120,
		Metadata:       []byte("{}"),
	})

	return err
}

// EnqueueSendInvoiceReminder enqueues a job to send an invoice reminder
func EnqueueSendInvoiceReminder(ctx context.Context, q repository.Querier, tenantID uuid.UUID, payload SendInvoiceReminderPayload) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = q.EnqueueJob(ctx, repository.EnqueueJobParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		JobType:    JobTypeSendInvoiceReminder,
		Queue:      "invoicing",
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

// EnqueueSyncInvoiceFromStripe enqueues a job to sync an invoice from Stripe
// Called when receiving Stripe invoice webhooks
func EnqueueSyncInvoiceFromStripe(ctx context.Context, q repository.Querier, tenantID uuid.UUID, payload SyncInvoiceFromStripePayload) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = q.EnqueueJob(ctx, repository.EnqueueJobParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		JobType:    JobTypeSyncInvoiceFromStripe,
		Queue:      "invoicing",
		Payload:    payloadJSON,
		Priority:   80, // Higher priority - webhook processing
		MaxRetries: 5,
		ScheduledAt: pgtype.Timestamptz{
			Time:  time.Now(),
			Valid: true,
		},
		TimeoutSeconds: 30,
		Metadata:       []byte("{}"),
	})

	return err
}
