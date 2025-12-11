package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/dukerupert/hiri/internal/domain"
	"github.com/dukerupert/hiri/internal/jobs"
	"github.com/dukerupert/hiri/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stripe/stripe-go/v83"
	checkoutsession "github.com/stripe/stripe-go/v83/checkout/session"
	portalsession "github.com/stripe/stripe-go/v83/billingportal/session"
	stripeinvoice "github.com/stripe/stripe-go/v83/invoice"
	stripesubscription "github.com/stripe/stripe-go/v83/subscription"
)

// Onboarding-specific errors
var (
	ErrCheckoutFailed         = domain.Errorf(domain.EINTERNAL, "", "Failed to create checkout session")
	ErrTenantNotFound         = domain.Errorf(domain.ENOTFOUND, "", "Tenant not found")
	ErrTenantAlreadyExists    = domain.Errorf(domain.ECONFLICT, "", "Tenant with this email already exists")
	ErrInvalidCheckoutSession = domain.Errorf(domain.EINVALID, "", "Invalid checkout session")
)

// OnboardingService handles SaaS customer onboarding flows
type OnboardingService interface {
	// CreateCheckoutSession creates a Stripe Checkout session for new tenant signup
	// Returns checkout session URL to redirect customer to
	CreateCheckoutSession(ctx context.Context, params CreateCheckoutParams) (string, error)

	// ProcessCheckoutCompleted handles the checkout.session.completed webhook
	// Creates tenant, operator, setup token, and queues welcome email
	// Returns tenant ID and operator ID
	ProcessCheckoutCompleted(ctx context.Context, session CheckoutSession) (tenantID, operatorID uuid.UUID, err error)

	// ProcessInvoicePaid handles the invoice.paid webhook
	// Clears grace period if tenant is past_due
	ProcessInvoicePaid(ctx context.Context, invoiceID string) error

	// ProcessInvoicePaymentFailed handles the invoice.payment_failed webhook
	// Starts grace period, queues payment failed email
	ProcessInvoicePaymentFailed(ctx context.Context, invoiceID string) error

	// ProcessSubscriptionUpdated handles the customer.subscription.updated webhook
	// Syncs subscription status changes
	ProcessSubscriptionUpdated(ctx context.Context, subscriptionID string) error

	// ProcessSubscriptionDeleted handles the customer.subscription.deleted webhook
	// Sets tenant status to 'cancelled'
	ProcessSubscriptionDeleted(ctx context.Context, subscriptionID string) error

	// CreateBillingPortalSession creates a Stripe Customer Portal session
	// Returns portal URL to redirect operator to
	CreateBillingPortalSession(ctx context.Context, tenantID uuid.UUID, returnURL string) (string, error)

	// ExpireGracePeriods suspends tenants whose grace period has expired
	// Called by background job hourly
	// Returns count of suspended tenants
	ExpireGracePeriods(ctx context.Context) (int, error)
}

// CreateCheckoutParams contains parameters for creating a checkout session
type CreateCheckoutParams struct {
	SuccessURL string // URL to redirect after successful payment
	CancelURL  string // URL to redirect if checkout is cancelled
}

// CheckoutSession represents data from Stripe checkout.session.completed event
type CheckoutSession struct {
	ID           string
	CustomerID   string
	Email        string
	BusinessName string // From custom field or customer name
	AmountTotal  int64
}

// OnboardingConfig holds configuration for onboarding service
type OnboardingConfig struct {
	// SaaSPriceID is the Stripe price ID for the SaaS subscription ($149/month)
	SaaSPriceID string

	// BaseURL is the application base URL for generating links
	BaseURL string

	// SetupTokenExpiry is how long setup tokens are valid (default: 48 hours)
	SetupTokenExpiry time.Duration
}

// onboardingService implements OnboardingService
type onboardingService struct {
	repo            repository.Querier
	operatorService OperatorService
	config          OnboardingConfig
	logger          *slog.Logger
}

// NewOnboardingService creates a new OnboardingService instance
func NewOnboardingService(
	repo repository.Querier,
	operatorService OperatorService,
	config OnboardingConfig,
	logger *slog.Logger,
) OnboardingService {
	if logger == nil {
		logger = slog.Default()
	}

	// Set defaults
	if config.SetupTokenExpiry == 0 {
		config.SetupTokenExpiry = 48 * time.Hour
	}

	return &onboardingService{
		repo:            repo,
		operatorService: operatorService,
		config:          config,
		logger:          logger.With("service", "onboarding"),
	}
}

// CreateCheckoutSession creates a Stripe Checkout session for new tenant signup
func (s *onboardingService) CreateCheckoutSession(ctx context.Context, params CreateCheckoutParams) (string, error) {
	if params.SuccessURL == "" || params.CancelURL == "" {
		return "", fmt.Errorf("success_url and cancel_url are required")
	}

	if s.config.SaaSPriceID == "" {
		return "", fmt.Errorf("SaaS price ID not configured")
	}

	// Create Stripe Checkout Session
	checkoutParams := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(s.config.SaaSPriceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(params.SuccessURL + "?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(params.CancelURL),
		// Collect customer email
		CustomerCreation: stripe.String("always"),
		// Custom fields to collect business name
		CustomFields: []*stripe.CheckoutSessionCustomFieldParams{
			{
				Key: stripe.String("business_name"),
				Label: &stripe.CheckoutSessionCustomFieldLabelParams{
					Type:   stripe.String("custom"),
					Custom: stripe.String("Business Name"),
				},
				Type: stripe.String("text"),
			},
		},
		// Subscription data
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{
				"source": "hiri_saas_checkout",
			},
		},
		// Allow promotion codes
		AllowPromotionCodes: stripe.Bool(true),
		// Consent collection for terms
		ConsentCollection: &stripe.CheckoutSessionConsentCollectionParams{
			TermsOfService: stripe.String("required"),
		},
	}

	session, err := checkoutsession.New(checkoutParams)
	if err != nil {
		s.logger.Error("failed to create checkout session",
			"error", err)
		return "", fmt.Errorf("%w: %v", ErrCheckoutFailed, err)
	}

	s.logger.Info("checkout session created",
		"session_id", session.ID)

	return session.URL, nil
}

// ProcessCheckoutCompleted handles the checkout.session.completed webhook
func (s *onboardingService) ProcessCheckoutCompleted(ctx context.Context, session CheckoutSession) (uuid.UUID, uuid.UUID, error) {
	s.logger.Info("processing checkout completed",
		"session_id", session.ID,
		"email", session.Email)

	// Generate unique slug from business name
	slug := generateSlug(session.BusinessName)
	if slug == "" {
		slug = generateSlug(session.Email)
	}

	// Ensure slug is unique
	for i := 0; i < 10; i++ {
		exists, err := s.repo.TenantSlugExists(ctx, slug)
		if err != nil {
			return uuid.Nil, uuid.Nil, fmt.Errorf("failed to check slug: %w", err)
		}
		if !exists {
			break
		}
		// Append random suffix
		slug = fmt.Sprintf("%s-%d", slug, time.Now().UnixNano()%1000)
	}

	// Create tenant with pending status
	tenant, err := s.repo.CreateTenant(ctx, repository.CreateTenantParams{
		Name:   session.BusinessName,
		Slug:   slug,
		Email:  session.Email,
		Status: "pending",
	})
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	tenantID, err := pgUUIDToUUID(tenant.ID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("failed to convert tenant ID: %w", err)
	}

	// Update tenant with Stripe customer ID
	err = s.repo.UpdateTenantStripeCustomer(ctx, repository.UpdateTenantStripeCustomerParams{
		ID:               tenant.ID,
		StripeCustomerID: pgtype.Text{String: session.CustomerID, Valid: true},
	})
	if err != nil {
		s.logger.Error("failed to update tenant stripe customer",
			"tenant_id", tenantID,
			"error", err)
		// Continue - not critical
	}

	// Create owner operator with setup token
	operator, rawToken, err := s.operatorService.CreateOperator(ctx, tenantID, session.Email, session.BusinessName, "owner")
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("failed to create operator: %w", err)
	}

	operatorID, err := pgUUIDToUUID(operator.ID)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("failed to convert operator ID: %w", err)
	}

	// Queue welcome email with setup link
	setupURL := fmt.Sprintf("%s/setup?token=%s", s.config.BaseURL, rawToken)
	err = s.enqueueWelcomeEmail(ctx, tenantID, session.Email, session.BusinessName, setupURL)
	if err != nil {
		s.logger.Error("failed to queue welcome email",
			"tenant_id", tenantID,
			"error", err)
		// Continue - not critical, operator can request password reset
	}

	s.logger.Info("checkout processing complete",
		"tenant_id", tenantID,
		"operator_id", operatorID,
		"slug", slug)

	return tenantID, operatorID, nil
}

// ProcessInvoicePaid handles the invoice.paid webhook
func (s *onboardingService) ProcessInvoicePaid(ctx context.Context, invoiceID string) error {
	s.logger.Info("processing invoice paid",
		"invoice_id", invoiceID)

	// Get invoice from Stripe
	inv, err := stripeinvoice.Get(invoiceID, nil)
	if err != nil {
		return fmt.Errorf("failed to get invoice: %w", err)
	}

	// Check if this is a subscription invoice
	if inv.Parent == nil || inv.Parent.SubscriptionDetails == nil || inv.Parent.SubscriptionDetails.Subscription == nil {
		s.logger.Debug("invoice not for subscription, skipping",
			"invoice_id", invoiceID)
		return nil
	}

	subscriptionID := inv.Parent.SubscriptionDetails.Subscription.ID

	// Get tenant by subscription ID
	tenant, err := s.repo.GetTenantByStripeSubscriptionID(ctx, pgtype.Text{
		String: subscriptionID,
		Valid:  true,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.logger.Debug("no tenant found for subscription, skipping",
				"subscription_id", subscriptionID)
			return nil
		}
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	// If tenant is past_due, clear the grace period
	if tenant.Status == "past_due" {
		err = s.repo.ClearTenantGracePeriod(ctx, tenant.ID)
		if err != nil {
			return fmt.Errorf("failed to clear grace period: %w", err)
		}

		tenantID, _ := pgUUIDToUUID(tenant.ID)
		s.logger.Info("cleared tenant grace period",
			"tenant_id", tenantID)
	}

	return nil
}

// ProcessInvoicePaymentFailed handles the invoice.payment_failed webhook
func (s *onboardingService) ProcessInvoicePaymentFailed(ctx context.Context, invoiceID string) error {
	s.logger.Info("processing invoice payment failed",
		"invoice_id", invoiceID)

	// Get invoice from Stripe
	inv, err := stripeinvoice.Get(invoiceID, nil)
	if err != nil {
		return fmt.Errorf("failed to get invoice: %w", err)
	}

	// Check if this is a subscription invoice
	if inv.Parent == nil || inv.Parent.SubscriptionDetails == nil || inv.Parent.SubscriptionDetails.Subscription == nil {
		s.logger.Debug("invoice not for subscription, skipping",
			"invoice_id", invoiceID)
		return nil
	}

	subscriptionID := inv.Parent.SubscriptionDetails.Subscription.ID

	// Get tenant by subscription ID
	tenant, err := s.repo.GetTenantByStripeSubscriptionID(ctx, pgtype.Text{
		String: subscriptionID,
		Valid:  true,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.logger.Debug("no tenant found for subscription, skipping",
				"subscription_id", subscriptionID)
			return nil
		}
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	tenantID, _ := pgUUIDToUUID(tenant.ID)

	// Start grace period if not already in past_due status
	if tenant.Status != "past_due" {
		err = s.repo.StartTenantGracePeriod(ctx, tenant.ID)
		if err != nil {
			return fmt.Errorf("failed to start grace period: %w", err)
		}

		s.logger.Info("started tenant grace period",
			"tenant_id", tenantID)
	}

	// Queue payment failed email
	err = s.enqueuePaymentFailedEmail(ctx, tenantID, tenant.Email, tenant.Name)
	if err != nil {
		s.logger.Error("failed to queue payment failed email",
			"tenant_id", tenantID,
			"error", err)
		// Continue - not critical
	}

	return nil
}

// ProcessSubscriptionUpdated handles the customer.subscription.updated webhook
func (s *onboardingService) ProcessSubscriptionUpdated(ctx context.Context, subscriptionID string) error {
	s.logger.Info("processing subscription updated",
		"subscription_id", subscriptionID)

	// Get subscription from Stripe
	sub, err := stripesubscription.Get(subscriptionID, nil)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Get tenant by subscription ID
	tenant, err := s.repo.GetTenantByStripeSubscriptionID(ctx, pgtype.Text{
		String: subscriptionID,
		Valid:  true,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Try to find by customer ID (for new subscriptions)
			tenant, err = s.repo.GetTenantByStripeCustomerID(ctx, pgtype.Text{
				String: sub.Customer.ID,
				Valid:  true,
			})
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					s.logger.Debug("no tenant found for subscription, skipping",
						"subscription_id", subscriptionID)
					return nil
				}
				return fmt.Errorf("failed to get tenant by customer: %w", err)
			}

			// Update subscription ID
			err = s.repo.UpdateTenantStripeSubscription(ctx, repository.UpdateTenantStripeSubscriptionParams{
				ID:                   tenant.ID,
				StripeSubscriptionID: pgtype.Text{String: subscriptionID, Valid: true},
			})
			if err != nil {
				s.logger.Error("failed to update tenant subscription ID",
					"error", err)
			}
		} else {
			return fmt.Errorf("failed to get tenant: %w", err)
		}
	}

	tenantID, _ := pgUUIDToUUID(tenant.ID)

	// Map Stripe subscription status to tenant status
	var newStatus string
	switch sub.Status {
	case stripe.SubscriptionStatusActive:
		if tenant.Status == "past_due" || tenant.Status == "pending" {
			newStatus = "active"
		}
	case stripe.SubscriptionStatusPastDue:
		newStatus = "past_due"
	case stripe.SubscriptionStatusCanceled:
		newStatus = "cancelled"
	case stripe.SubscriptionStatusUnpaid:
		newStatus = "suspended"
	default:
		// Other statuses (incomplete, trialing, etc.) - no action
		s.logger.Debug("subscription status unchanged",
			"subscription_id", subscriptionID,
			"status", sub.Status)
		return nil
	}

	if newStatus != "" && newStatus != tenant.Status {
		err = s.repo.SetTenantStatus(ctx, repository.SetTenantStatusParams{
			ID:     tenant.ID,
			Status: newStatus,
		})
		if err != nil {
			return fmt.Errorf("failed to update tenant status: %w", err)
		}

		s.logger.Info("tenant status updated",
			"tenant_id", tenantID,
			"old_status", tenant.Status,
			"new_status", newStatus)
	}

	return nil
}

// ProcessSubscriptionDeleted handles the customer.subscription.deleted webhook
func (s *onboardingService) ProcessSubscriptionDeleted(ctx context.Context, subscriptionID string) error {
	s.logger.Info("processing subscription deleted",
		"subscription_id", subscriptionID)

	// Get tenant by subscription ID
	tenant, err := s.repo.GetTenantByStripeSubscriptionID(ctx, pgtype.Text{
		String: subscriptionID,
		Valid:  true,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.logger.Debug("no tenant found for subscription, skipping",
				"subscription_id", subscriptionID)
			return nil
		}
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	tenantID, _ := pgUUIDToUUID(tenant.ID)

	// Cancel the tenant
	err = s.repo.CancelTenant(ctx, tenant.ID)
	if err != nil {
		return fmt.Errorf("failed to cancel tenant: %w", err)
	}

	s.logger.Info("tenant cancelled",
		"tenant_id", tenantID)

	// Queue cancellation email
	err = s.enqueueCancellationEmail(ctx, tenantID, tenant.Email, tenant.Name)
	if err != nil {
		s.logger.Error("failed to queue cancellation email",
			"tenant_id", tenantID,
			"error", err)
	}

	return nil
}

// CreateBillingPortalSession creates a Stripe Customer Portal session
func (s *onboardingService) CreateBillingPortalSession(ctx context.Context, tenantID uuid.UUID, returnURL string) (string, error) {
	// Get tenant
	tenant, err := s.repo.GetTenantByID(ctx, pgtype.UUID{Bytes: tenantID, Valid: true})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrTenantNotFound
		}
		return "", fmt.Errorf("failed to get tenant: %w", err)
	}

	if !tenant.StripeCustomerID.Valid {
		return "", fmt.Errorf("tenant has no Stripe customer ID")
	}

	// Create portal session
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(tenant.StripeCustomerID.String),
		ReturnURL: stripe.String(returnURL),
	}

	session, err := portalsession.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create portal session: %w", err)
	}

	s.logger.Info("billing portal session created",
		"tenant_id", tenantID)

	return session.URL, nil
}

// ExpireGracePeriods suspends tenants whose grace period has expired
func (s *onboardingService) ExpireGracePeriods(ctx context.Context) (int, error) {
	s.logger.Info("checking for expired grace periods")

	tenants, err := s.repo.GetTenantsWithExpiredGracePeriod(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get tenants with expired grace period: %w", err)
	}

	suspended := 0
	for _, tenant := range tenants {
		err := s.repo.SuspendTenant(ctx, tenant.ID)
		if err != nil {
			tenantID, _ := pgUUIDToUUID(tenant.ID)
			s.logger.Error("failed to suspend tenant",
				"tenant_id", tenantID,
				"error", err)
			continue
		}

		tenantID, _ := pgUUIDToUUID(tenant.ID)
		s.logger.Info("tenant suspended due to expired grace period",
			"tenant_id", tenantID)

		// Queue suspension email
		err = s.enqueueSuspensionEmail(ctx, tenantID, tenant.Email, tenant.Name)
		if err != nil {
			s.logger.Error("failed to queue suspension email",
				"tenant_id", tenantID,
				"error", err)
		}

		suspended++
	}

	s.logger.Info("grace period expiration complete",
		"suspended_count", suspended)

	return suspended, nil
}

// Helper functions

// generateSlug creates a URL-safe slug from a string
func generateSlug(input string) string {
	if input == "" {
		return ""
	}

	// Convert to lowercase
	slug := strings.ToLower(input)

	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")

	// Remove special characters except hyphens
	reg := regexp.MustCompile(`[^a-z0-9-]`)
	slug = reg.ReplaceAllString(slug, "")

	// Remove duplicate hyphens
	reg = regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Trim hyphens from edges
	slug = strings.Trim(slug, "-")

	// Limit length
	if len(slug) > 50 {
		slug = slug[:50]
	}

	return slug
}

// pgUUIDToUUID converts pgtype.UUID to uuid.UUID
func pgUUIDToUUID(pg pgtype.UUID) (uuid.UUID, error) {
	if !pg.Valid {
		return uuid.Nil, fmt.Errorf("invalid pgtype.UUID")
	}
	return uuid.UUID(pg.Bytes), nil
}

// Email job payloads for SaaS onboarding

// SaaSWelcomePayload represents the payload for a SaaS welcome email job
type SaaSWelcomePayload struct {
	Email        string `json:"email"`
	BusinessName string `json:"business_name"`
	SetupURL     string `json:"setup_url"`
}

// SaaSPaymentFailedPayload represents the payload for a SaaS payment failed email job
type SaaSPaymentFailedPayload struct {
	Email        string    `json:"email"`
	BusinessName string    `json:"business_name"`
	FailedDate   time.Time `json:"failed_date"`
	GraceEndDate time.Time `json:"grace_end_date"`
	BillingURL   string    `json:"billing_url"`
}

// SaaSCancellationPayload represents the payload for a SaaS cancellation email job
type SaaSCancellationPayload struct {
	Email        string    `json:"email"`
	BusinessName string    `json:"business_name"`
	CancelDate   time.Time `json:"cancel_date"`
}

// SaaSSuspensionPayload represents the payload for a SaaS suspension email job
type SaaSSuspensionPayload struct {
	Email        string    `json:"email"`
	BusinessName string    `json:"business_name"`
	SuspendDate  time.Time `json:"suspend_date"`
	BillingURL   string    `json:"billing_url"`
}

// Job type constants for SaaS email jobs
const (
	JobTypeSaaSWelcome       = "email:saas_welcome"
	JobTypeSaaSPaymentFailed = "email:saas_payment_failed"
	JobTypeSaaSCancellation  = "email:saas_cancellation"
	JobTypeSaaSSuspension    = "email:saas_suspension"
)

// enqueueWelcomeEmail queues a welcome email for new tenant signup
func (s *onboardingService) enqueueWelcomeEmail(ctx context.Context, tenantID uuid.UUID, email, businessName, setupURL string) error {
	payload := SaaSWelcomePayload{
		Email:        email,
		BusinessName: businessName,
		SetupURL:     setupURL,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = s.repo.EnqueueJob(ctx, repository.EnqueueJobParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		JobType:    JobTypeSaaSWelcome,
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

// enqueuePaymentFailedEmail queues a payment failed email
func (s *onboardingService) enqueuePaymentFailedEmail(ctx context.Context, tenantID uuid.UUID, email, businessName string) error {
	billingURL := fmt.Sprintf("%s/admin/billing", s.config.BaseURL)
	payload := SaaSPaymentFailedPayload{
		Email:        email,
		BusinessName: businessName,
		FailedDate:   time.Now(),
		GraceEndDate: time.Now().Add(7 * 24 * time.Hour), // 7 day grace period
		BillingURL:   billingURL,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = s.repo.EnqueueJob(ctx, repository.EnqueueJobParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		JobType:    JobTypeSaaSPaymentFailed,
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

// enqueueCancellationEmail queues a cancellation email
func (s *onboardingService) enqueueCancellationEmail(ctx context.Context, tenantID uuid.UUID, email, businessName string) error {
	payload := SaaSCancellationPayload{
		Email:        email,
		BusinessName: businessName,
		CancelDate:   time.Now(),
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = s.repo.EnqueueJob(ctx, repository.EnqueueJobParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		JobType:    JobTypeSaaSCancellation,
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

// enqueueSuspensionEmail queues a suspension email
func (s *onboardingService) enqueueSuspensionEmail(ctx context.Context, tenantID uuid.UUID, email, businessName string) error {
	billingURL := fmt.Sprintf("%s/admin/billing", s.config.BaseURL)
	payload := SaaSSuspensionPayload{
		Email:        email,
		BusinessName: businessName,
		SuspendDate:  time.Now(),
		BillingURL:   billingURL,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = s.repo.EnqueueJob(ctx, repository.EnqueueJobParams{
		TenantID:   pgtype.UUID{Bytes: tenantID, Valid: true},
		JobType:    JobTypeSaaSSuspension,
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

// Add job processing to the email worker (note: implementation depends on existing worker structure)
// The following function types should be registered with the email worker:
//
// ProcessSaaSWelcome(ctx context.Context, payload SaaSWelcomePayload) error
// ProcessSaaSPaymentFailed(ctx context.Context, payload SaaSPaymentFailedPayload) error
// ProcessSaaSCancellation(ctx context.Context, payload SaaSCancellationPayload) error
// ProcessSaaSSuspension(ctx context.Context, payload SaaSSuspensionPayload) error

// Ensure jobs package import is used
var _ = jobs.JobTypePasswordReset
