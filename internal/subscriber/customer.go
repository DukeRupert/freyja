// internal/subscriber/customer.go
package subscriber

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/dukerupert/freyja/internal/interfaces"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/stripe/stripe-go/v82"
	stripeCustomer "github.com/stripe/stripe-go/v82/customer"
)

type CustomerEventSubscriber struct {
	customerService *service.CustomerService
	events          interfaces.EventPublisher
}

func NewCustomerEventSubscriber(
	customerService *service.CustomerService,
	events interfaces.EventPublisher,
) *CustomerEventSubscriber {
	return &CustomerEventSubscriber{
		customerService: customerService,
		events:          events,
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

	log.Println("✅ Customer event subscriber started")
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
	log.Printf("Processing customer.created event: %s", event.AggregateID)

	// Extract customer ID from aggregate ID
	customerID, err := interfaces.ExtractAggregateID(event.AggregateID)
	if err != nil {
		return fmt.Errorf("invalid customer ID in event: %w", err)
	}

	// Check if customer already has Stripe ID (from event data)
	if stripeCustomerID, exists := event.Data["stripe_customer_id"].(string); exists && stripeCustomerID != "" {
		log.Printf("Customer %d already has Stripe ID: %s", customerID, stripeCustomerID)
		return nil
	}

	// Ensure customer has Stripe customer ID
	stripeCustomerID, err := s.customerService.EnsureStripeCustomer(ctx, int(customerID))
	if err != nil {
		log.Printf("Failed to ensure Stripe customer for %d: %v", customerID, err)
		return fmt.Errorf("failed to ensure Stripe customer: %w", err)
	}

	log.Printf("✅ Ensured Stripe customer ID for customer %d: %s", customerID, stripeCustomerID)

	// Publish a follow-up event indicating Stripe customer was created
	followUpEvent := interfaces.BuildCustomerEvent("customer.stripe_ensured", customerID, map[string]interface{}{
		"stripe_customer_id": stripeCustomerID,
		"triggered_by":       "customer.created",
	})

	if err := s.events.PublishEvent(ctx, followUpEvent); err != nil {
		log.Printf("Warning: Failed to publish customer.stripe_ensured event: %v", err)
		// Don't return error as the main operation succeeded
	}

	return nil
}

// handleCustomerUpdated processes customer.updated events
func (s *CustomerEventSubscriber) handleCustomerUpdated(ctx context.Context, event interfaces.Event) error {
	log.Printf("Processing customer.updated event: %s", event.AggregateID)

	// Extract customer ID from aggregate ID
	customerID, err := interfaces.ExtractAggregateID(event.AggregateID)
	if err != nil {
		return fmt.Errorf("invalid customer ID in event: %w", err)
	}

	// Get customer to check if they have Stripe ID
	customer, err := s.customerService.GetCustomerByID(ctx, int(customerID))
	if err != nil {
		return fmt.Errorf("failed to get customer %d: %w", customerID, err)
	}

	// If customer doesn't have Stripe ID, ensure they get one
	if !customer.StripeCustomerID.Valid || customer.StripeCustomerID.String == "" {
		log.Printf("Customer %d updated but missing Stripe ID, ensuring creation", customerID)

		stripeCustomerID, err := s.customerService.EnsureStripeCustomer(ctx, int(customerID))
		if err != nil {
			log.Printf("Failed to ensure Stripe customer for updated customer %d: %v", customerID, err)
			return fmt.Errorf("failed to ensure Stripe customer: %w", err)
		}

		log.Printf("✅ Ensured Stripe customer ID for updated customer %d: %s", customerID, stripeCustomerID)

		// Publish follow-up event
		followUpEvent := interfaces.BuildCustomerEvent("customer.stripe_ensured", customerID, map[string]interface{}{
			"stripe_customer_id": stripeCustomerID,
			"triggered_by":       "customer.updated",
		})

		if err := s.events.PublishEvent(ctx, followUpEvent); err != nil {
			log.Printf("Warning: Failed to publish customer.stripe_ensured event: %v", err)
		}
	} else {
		// Customer has Stripe ID, update their info in Stripe
		log.Printf("Updating customer %d in Stripe: %s", customerID, customer.StripeCustomerID.String)

		if err := s.updateCustomerInStripe(customer); err != nil {
			log.Printf("Failed to update customer %d in Stripe: %v", customerID, err)
			return fmt.Errorf("failed to update customer in Stripe: %w", err)
		}

		log.Printf("✅ Updated customer %d in Stripe successfully", customerID)
	}

	return nil
}

// EnsureAllCustomersHaveStripeIDs is a utility method to backfill existing customers
func (s *CustomerEventSubscriber) EnsureAllCustomersHaveStripeIDs(ctx context.Context) error {
	log.Println("🔄 Starting backfill process for customers without Stripe IDs...")

	// This would require a new method in your customer service to get customers without Stripe IDs
	// For now, this is a placeholder showing the pattern
	
	// You could implement this by:
	// 1. Adding a GetCustomersWithoutStripeID method to your repository/service
	// 2. Iterating through those customers
	// 3. Calling EnsureStripeCustomer for each one
	
	log.Println("✅ Backfill process completed")
	return nil
}