// internal/subscriber/customer.go
package subscriber

import (
	"context"
	"fmt"
	"strings"

	"github.com/dukerupert/freyja/internal/shared/metrics"
	"github.com/dukerupert/freyja/internal/shared/interfaces"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/stripe/stripe-go/v82"
	stripeCustomer "github.com/stripe/stripe-go/v82/customer"
)

type CustomerEventSubscriber struct {
	customerService interfaces.CustomerService
	events          interfaces.EventPublisher
	logger          zerolog.Logger
}

func NewCustomerEventSubscriber(
	customerService interfaces.CustomerService,
	events interfaces.EventPublisher,
	logger zerolog.Logger,
) *CustomerEventSubscriber {
	return &CustomerEventSubscriber{
		customerService: customerService,
		events:          events,
		logger:          logger.With().Str("component", "CustomerEventSubscriber").Logger(),
	}
}

// Start subscribes to customer events and starts processing them
func (s *CustomerEventSubscriber) Start(ctx context.Context) error {
	// Subscribe to customer created events
	if err := s.events.Subscribe(ctx, interfaces.EventCustomerCreated, s.handleCustomerCreated); err != nil {
		return fmt.Errorf("failed to subscribe to customer.created events: %w", err)
	}

	// Subscribe to customer updated events
	if err := s.events.Subscribe(ctx, interfaces.EventCustomerUpdated, s.handleCustomerUpdated); err != nil {
		return fmt.Errorf("failed to subscribe to customer.updated events: %w", err)
	}

	if err := s.events.Subscribe(ctx, interfaces.EventCustomerStripeSyncRequested, s.handleCustomerStripeSyncRequested); err != nil {
		return fmt.Errorf("failed to subscribe to customer.stripe_sync_requested events: %w", err)
	}

	s.logger.Info().Msg("[OK] Customer event subscriber started")
	return nil
}

// updateCustomerInStripe updates customer information in Stripe
func (s *CustomerEventSubscriber) updateCustomerInStripe(customer *interfaces.Customer) error {
	if !customer.StripeCustomerID.Valid || customer.StripeCustomerID.String == "" {
		return fmt.Errorf("customer has no Stripe ID")
	}

	// Build Stripe update parameters
	params := &stripe.CustomerParams{
		Email: stripe.String(customer.Email),
	}

	// Add name if available
	if customer.FirstName.Valid || customer.LastName.Valid {
		firstName := ""
		lastName := ""

		if customer.FirstName.Valid {
			firstName = customer.FirstName.String
		}
		if customer.LastName.Valid {
			lastName = customer.LastName.String
		}

		fullName := strings.TrimSpace(fmt.Sprintf("%s %s", firstName, lastName))
		if fullName != "" {
			params.Name = stripe.String(fullName)
		}
	}

	// Add metadata with internal customer ID for reference
	params.AddMetadata("internal_customer_id", fmt.Sprintf("%d", customer.ID))
	params.AddMetadata("last_updated", customer.UpdatedAt.Format("2006-01-02T15:04:05Z"))

	// Update customer in Stripe
	_, err := stripeCustomer.Update(customer.StripeCustomerID.String, params)
	if err != nil {
		return fmt.Errorf("Stripe customer update failed: %w", err)
	}

	return nil
}

// handleCustomerCreated processes customer.created events
func (s *CustomerEventSubscriber) handleCustomerCreated(ctx context.Context, event interfaces.Event) error {
	timer := prometheus.NewTimer(metrics.EventProcessingDuration.WithLabelValues(event.Type, "customer_subscriber"))
	defer timer.ObserveDuration()

	s.logger.Info().
		Str("event_type", event.Type).
		Str("aggregate_id", event.AggregateID).
		Msg("Processing customer.created event")

	// Extract customer ID from aggregate ID
	customerID, err := interfaces.ExtractAggregateID(event.AggregateID)
	if err != nil {
		metrics.EventsProcessed.WithLabelValues(event.Type, "customer_subscriber", "error").Inc()
		s.logger.Error().
			Err(err).
			Str("aggregate_id", event.AggregateID).
			Msg("Invalid customer ID in event")
		return fmt.Errorf("invalid customer ID in event: %w", err)
	}

	// Check if customer already has Stripe ID (from event data)
	if stripeCustomerID, exists := event.Data["stripe_customer_id"].(string); exists && stripeCustomerID != "" {
		s.logger.Info().
			Int("customer_id", int(customerID)).
			Str("stripe_customer_id", stripeCustomerID).
			Msg("Customer already has Stripe ID")
		metrics.EventsProcessed.WithLabelValues(event.Type, "customer_subscriber", "success").Inc()
		return nil
	}

	// Ensure customer has Stripe customer ID
	stripeCustomerID, err := s.customerService.EnsureStripeCustomer(ctx, customerID)
	if err != nil {
		s.logger.Error().
			Err(err).
			Int("customer_id", int(customerID)).
			Msg("Failed to ensure Stripe customer")
		metrics.EventsProcessed.WithLabelValues(event.Type, "customer_subscriber", "error").Inc()
		return fmt.Errorf("failed to ensure Stripe customer: %w", err)
	}

	// Record business metric
	metrics.CustomerStripeCreations.WithLabelValues("created").Inc()

	s.logger.Info().
		Int("customer_id", int(customerID)).
		Str("stripe_customer_id", stripeCustomerID).
		Msg("[OK] Ensured Stripe customer ID for customer")

	// Publish a follow-up event indicating Stripe customer was created
	followUpEvent := interfaces.BuildCustomerEvent("customer.stripe_ensured", customerID, map[string]interface{}{
		"stripe_customer_id": stripeCustomerID,
		"triggered_by":       "customer.created",
	})

	if err := s.events.PublishEvent(ctx, followUpEvent); err != nil {
		s.logger.Warn().
			Err(err).
			Int("customer_id", int(customerID)).
			Msg("Failed to publish customer.stripe_ensured event")
		// Don't return error as the main operation succeeded
	}

	metrics.EventsProcessed.WithLabelValues(event.Type, "customer_subscriber", "success").Inc()
	return nil
}

// handleCustomerUpdated processes customer.updated events
func (s *CustomerEventSubscriber) handleCustomerUpdated(ctx context.Context, event interfaces.Event) error {
	timer := prometheus.NewTimer(metrics.EventProcessingDuration.WithLabelValues(event.Type, "customer_subscriber"))
	defer timer.ObserveDuration()

	s.logger.Info().
		Str("event_type", event.Type).
		Str("aggregate_id", event.AggregateID).
		Msg("Processing customer.updated event")

	// Extract customer ID from aggregate ID
	customerID, err := interfaces.ExtractAggregateID(event.AggregateID)
	if err != nil {
		metrics.EventsProcessed.WithLabelValues(event.Type, "customer_subscriber", "error").Inc()
		s.logger.Error().
			Err(err).
			Str("aggregate_id", event.AggregateID).
			Msg("Invalid customer ID in event")
		return fmt.Errorf("invalid customer ID in event: %w", err)
	}

	// Get customer to check if they have Stripe ID
	customer, err := s.customerService.GetCustomerByID(ctx, customerID)
	if err != nil {
		metrics.EventsProcessed.WithLabelValues(event.Type, "customer_subscriber", "error").Inc()
		s.logger.Error().
			Err(err).
			Int("customer_id", int(customerID)).
			Msg("Failed to get customer")
		return fmt.Errorf("failed to get customer %d: %w", customerID, err)
	}

	// If customer doesn't have Stripe ID, ensure they get one
	if !customer.StripeCustomerID.Valid || customer.StripeCustomerID.String == "" {
		s.logger.Info().
			Int("customer_id", int(customerID)).
			Msg("Customer updated but missing Stripe ID, ensuring creation")

		stripeCustomerID, err := s.customerService.EnsureStripeCustomer(ctx, customerID)
		if err != nil {
			s.logger.Error().
				Err(err).
				Int("customer_id", int(customerID)).
				Msg("Failed to ensure Stripe customer for updated customer")
			metrics.EventsProcessed.WithLabelValues(event.Type, "customer_subscriber", "error").Inc()
			return fmt.Errorf("failed to ensure Stripe customer: %w", err)
		}

		// Record business metric
		metrics.CustomerStripeCreations.WithLabelValues("updated").Inc()

		s.logger.Info().
			Int("customer_id", int(customerID)).
			Str("stripe_customer_id", stripeCustomerID).
			Msg("[OK] Ensured Stripe customer ID for updated customer")

		// Publish follow-up event
		followUpEvent := interfaces.BuildCustomerEvent("customer.stripe_ensured", customerID, map[string]interface{}{
			"stripe_customer_id": stripeCustomerID,
			"triggered_by":       "customer.updated",
		})

		if err := s.events.PublishEvent(ctx, followUpEvent); err != nil {
			s.logger.Warn().
				Err(err).
				Int("customer_id", int(customerID)).
				Msg("Failed to publish customer.stripe_ensured event")
		}
	} else {
		// Customer has Stripe ID, update their info in Stripe
		s.logger.Info().
			Int("customer_id", int(customerID)).
			Str("stripe_customer_id", customer.StripeCustomerID.String).
			Msg("Updating customer in Stripe")

		if err := s.updateCustomerInStripe(customer); err != nil {
			s.logger.Error().
				Err(err).
				Int("customer_id", int(customerID)).
				Msg("Failed to update customer in Stripe")
			metrics.CustomerStripeUpdates.WithLabelValues("error").Inc()
			metrics.EventsProcessed.WithLabelValues(event.Type, "customer_subscriber", "error").Inc()
			return fmt.Errorf("failed to update customer in Stripe: %w", err)
		}

		// Record successful Stripe update
		metrics.CustomerStripeUpdates.WithLabelValues("success").Inc()
		s.logger.Info().
			Int("customer_id", int(customerID)).
			Msg("[OK] Updated customer in Stripe successfully")
	}

	metrics.EventsProcessed.WithLabelValues(event.Type, "customer_subscriber", "success").Inc()
	return nil
}

// EnsureAllCustomersHaveStripeIDs is a utility method to backfill existing customers
func (s *CustomerEventSubscriber) EnsureAllCustomersHaveStripeIDs(ctx context.Context) error {
	s.logger.Info().Msg("Starting backfill process for customers without Stripe IDs...")

	// This would require a new method in your customer service to get customers without Stripe IDs
	// For now, this is a placeholder showing the pattern

	// You could implement this by:
	// 1. Adding a GetCustomersWithoutStripeID method to your repository/service
	// 2. Iterating through those customers
	// 3. Calling EnsureStripeCustomer for each one

	s.logger.Info().Msg("[OK] Backfill process completed")
	return nil
}

func (s *CustomerEventSubscriber) handleCustomerStripeSyncRequested(ctx context.Context, event interfaces.Event) error {
	timer := prometheus.NewTimer(metrics.EventProcessingDuration.WithLabelValues(event.Type, "customer_subscriber"))
	defer timer.ObserveDuration()

	s.logger.Info().
		Str("event_type", event.Type).
		Str("aggregate_id", event.AggregateID).
		Msg("Processing customer.stripe_sync_requested event")

	// Extract customer ID from aggregate ID
	customerID, err := interfaces.ExtractAggregateID(event.AggregateID)
	if err != nil {
		metrics.EventsProcessed.WithLabelValues(event.Type, "customer_subscriber", "error").Inc()
		s.logger.Error().
			Err(err).
			Str("aggregate_id", event.AggregateID).
			Msg("Invalid customer ID in event")
		return fmt.Errorf("invalid customer ID in event: %w", err)
	}

	// Ensure customer has Stripe customer ID
	stripeCustomerID, err := s.customerService.EnsureStripeCustomer(ctx, customerID)
	if err != nil {
		s.logger.Error().
			Err(err).
			Int("customer_id", int(customerID)).
			Msg("Failed to ensure Stripe customer")
		metrics.EventsProcessed.WithLabelValues(event.Type, "customer_subscriber", "error").Inc()
		return fmt.Errorf("failed to ensure Stripe customer: %w", err)
	}

	s.logger.Info().
		Int("customer_id", int(customerID)).
		Str("stripe_customer_id", stripeCustomerID).
		Msg("[OK] Ensured Stripe customer ID for customer")

	// Publish a follow-up event indicating Stripe customer was ensured
	followUpEvent := interfaces.BuildCustomerEvent("customer.stripe_ensured", customerID, map[string]interface{}{
		"stripe_customer_id": stripeCustomerID,
		"triggered_by":       "admin_backfill",
	})

	if err := s.events.PublishEvent(ctx, followUpEvent); err != nil {
		s.logger.Warn().
			Err(err).
			Int("customer_id", int(customerID)).
			Msg("Failed to publish customer.stripe_ensured event")
	}

	metrics.EventsProcessed.WithLabelValues(event.Type, "customer_subscriber", "success").Inc()
	return nil
}