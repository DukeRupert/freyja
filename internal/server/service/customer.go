// internal/service/customer.go
package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"

	"github.com/dukerupert/freyja/internal/server/provider"
	"github.com/dukerupert/freyja/internal/shared/interfaces"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stripe/stripe-go/v82"
	stripeCustomer "github.com/stripe/stripe-go/v82/customer"
)

type CustomerService struct {
	repo           interfaces.CustomerRepository
	stripeProvider *provider.StripeProvider
	events         interfaces.EventPublisher
}

func NewCustomerService(
	repo interfaces.CustomerRepository,
	stripeProvider *provider.StripeProvider,
	events interfaces.EventPublisher,
) interfaces.CustomerService {
	return &CustomerService{
		repo:           repo,
		stripeProvider: stripeProvider,
		events:         events,
	}
}

// CreateCustomer creates a customer in both our database and Stripe
func (s *CustomerService) CreateCustomer(ctx context.Context, req interfaces.CreateCustomerRequest) (*interfaces.Customer, error) {
	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if customer already exists by email
	existingCustomer, err := s.repo.GetByEmail(ctx, req.Email)
	if err == nil && existingCustomer != nil {
		return nil, fmt.Errorf("customer with email %s already exists", req.Email)
	}

	// Create customer in our database first
	customer := &interfaces.Customer{
		Email:        req.Email,
		FirstName:    pgtype.Text{String: req.FirstName, Valid: req.FirstName != ""},
		LastName:     pgtype.Text{String: req.LastName, Valid: req.LastName != ""},
		PasswordHash: req.PasswordHash, // Should be hashed before calling this service
	}

	if err := s.repo.Create(ctx, customer); err != nil {
		return nil, fmt.Errorf("failed to create customer in database: %w", err)
	}

	// Create customer in Stripe
	stripeParams := &stripe.CustomerParams{
		Email: stripe.String(req.Email),
	}

	if req.FirstName != "" || req.LastName != "" {
		stripeParams.Name = stripe.String(fmt.Sprintf("%s %s", req.FirstName, req.LastName))
	}

	stripeCustomer, err := stripeCustomer.New(stripeParams)
	if err != nil {
		// If Stripe creation fails, we should consider rolling back the database creation
		// For now, we'll log the error and continue
		fmt.Printf("Warning: Failed to create Stripe customer for %s: %v\n", req.Email, err)
	} else {
		// Update our customer record with the Stripe customer ID
		if err := s.repo.UpdateStripeID(ctx, customer.ID, stripeCustomer.ID); err != nil {
			fmt.Printf("Warning: Failed to update Stripe customer ID: %v\n", err)
		} else {
			customer.StripeCustomerID = pgtype.Text{String: stripeCustomer.ID, Valid: true}
		}
	}

	// Publish customer created event
	if err := s.publishCustomerEvent(ctx, interfaces.EventCustomerCreated, customer.ID, map[string]interface{}{
		"email":              customer.Email,
		"stripe_customer_id": customer.StripeCustomerID.String,
	}); err != nil {
		fmt.Printf("Warning: Failed to publish customer created event: %v\n", err)
	}

	return customer, nil
}

// UpdateStripeCustomerID updates the Stripe customer ID for a customer
func (s *CustomerService) UpdateStripeID(ctx context.Context, customerID int32, stripeCustomerID string) error {
	return s.repo.UpdateStripeID(ctx, customerID, stripeCustomerID)
}

// CreateCustomerFromStripe creates a customer record from a Stripe customer (for guest checkouts)
func (s *CustomerService) CreateCustomerFromStripe(ctx context.Context, stripeCustomerID, email string) (*interfaces.Customer, error) {
	// Validate email
	if email == "" {
		return nil, fmt.Errorf("Stripe customer missing email")
	}

	// Check if customer already exists with this email
	existingCustomer, err := s.GetCustomerByEmail(ctx, email)
	if err == nil && existingCustomer != nil {
		// Customer exists, just link the Stripe ID
		log.Printf("Customer with email %s already exists (ID: %d), linking Stripe ID %s",
			email, existingCustomer.ID, stripeCustomerID)

		if err := s.UpdateStripeID(ctx, existingCustomer.ID, stripeCustomerID); err != nil {
			return nil, fmt.Errorf("failed to link existing customer to Stripe: %w", err)
		}

		return existingCustomer, nil
	}

	// Create new customer for guest checkout
	// Generate a placeholder password since they checked out as guest
	placeholderPassword := generateGuestPassword() // You'll need to implement this

	req := interfaces.CreateCustomerRequest{
		Email:        email,
		PasswordHash: placeholderPassword,
		FirstName:    "",
		LastName:     "",
	}

	// Create the customer
	customer, err := s.CreateCustomer(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}

	// Update with Stripe customer ID
	if err := s.UpdateStripeID(ctx, customer.ID, stripeCustomerID); err != nil {
		log.Printf("Warning: Failed to link new customer %d to Stripe ID %s: %v",
			customer.ID, stripeCustomerID, err)
	}

	// Publish event for guest conversion
	if err := s.publishCustomerEvent(ctx, interfaces.EventGuestCustomerCreated, customer.ID, map[string]interface{}{
		"email":              customer.Email,
		"stripe_customer_id": stripeCustomerID,
		"converted_from":     "guest_checkout",
	}); err != nil {
		log.Printf("Warning: Failed to publish guest customer created event: %v", err)
	}

	return customer, nil
}

// Helper function to generate a secure placeholder password for guest accounts
func generateGuestPassword() string {
	// Generate a random password that the user doesn't know
	// They'll need to use "forgot password" if they want to access the account
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return fmt.Sprintf("guest_%x", bytes)
}

// GetCustomerByID retrieves a customer by ID
func (s *CustomerService) GetCustomerByID(ctx context.Context, customerID int32) (*interfaces.Customer, error) {
	if customerID <= 0 {
		return nil, fmt.Errorf("invalid customer ID: %d", customerID)
	}

	customer, err := s.repo.GetByID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	return customer, nil
}

// GetCustomerByEmail retrieves a customer by email
func (s *CustomerService) GetCustomerByEmail(ctx context.Context, email string) (*interfaces.Customer, error) {
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}

	customer, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer by email: %w", err)
	}

	return customer, nil
}

// UpdateCustomer updates customer information
func (s *CustomerService) UpdateCustomer(ctx context.Context, customerID int32, req interfaces.UpdateCustomerRequest) (*interfaces.Customer, error) {
	// Get existing customer
	customer, err := s.repo.GetByID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	// Update fields
	if req.Email != "" {
		customer.Email = req.Email
	}
	if req.FirstName != nil {
		customer.FirstName = pgtype.Text{String: *req.FirstName, Valid: *req.FirstName != ""}
	}
	if req.LastName != nil {
		customer.LastName = pgtype.Text{String: *req.LastName, Valid: *req.LastName != ""}
	}

	// Update in database
	if err := s.repo.Update(ctx, customer); err != nil {
		return nil, fmt.Errorf("failed to update customer: %w", err)
	}

	// Update in Stripe if we have a Stripe customer ID
	if customer.StripeCustomerID.Valid {
		stripeParams := &stripe.CustomerParams{
			Email: stripe.String(customer.Email),
		}

		if customer.FirstName.Valid || customer.LastName.Valid {
			name := fmt.Sprintf("%s %s", customer.FirstName.String, customer.LastName.String)
			stripeParams.Name = stripe.String(name)
		}

		_, err := stripeCustomer.Update(customer.StripeCustomerID.String, stripeParams)
		if err != nil {
			fmt.Printf("Warning: Failed to update Stripe customer: %v\n", err)
		}
	}

	// Publish customer updated event
	if err := s.publishCustomerEvent(ctx, interfaces.EventCustomerUpdated, customer.ID, map[string]interface{}{
		"email": customer.Email,
	}); err != nil {
		fmt.Printf("Warning: Failed to publish customer updated event: %v\n", err)
	}

	return customer, nil
}

// EnsureStripeCustomer ensures a customer has a Stripe customer ID
func (s *CustomerService) EnsureStripeCustomer(ctx context.Context, customerID int32) (string, error) {
	customer, err := s.repo.GetByID(ctx, customerID)
	if err != nil {
		return "", fmt.Errorf("failed to get customer: %w", err)
	}

	// If customer already has Stripe ID, return it
	if customer.StripeCustomerID.Valid {
		return customer.StripeCustomerID.String, nil
	}

	// Create Stripe customer
	stripeParams := &stripe.CustomerParams{
		Email: stripe.String(customer.Email),
	}

	if customer.FirstName.Valid || customer.LastName.Valid {
		name := fmt.Sprintf("%s %s", customer.FirstName.String, customer.LastName.String)
		stripeParams.Name = stripe.String(name)
	}

	stripeCustomer, err := stripeCustomer.New(stripeParams)
	if err != nil {
		return "", fmt.Errorf("failed to create Stripe customer: %w", err)
	}

	// Update our customer record
	if err := s.repo.UpdateStripeID(ctx, customerID, stripeCustomer.ID); err != nil {
		return "", fmt.Errorf("failed to update customer Stripe ID: %w", err)
	}

	return stripeCustomer.ID, nil
}

// GetCustomerByStripeID retrieves a customer by Stripe customer ID
func (s *CustomerService) GetCustomerByStripeID(ctx context.Context, stripeCustomerID string) (*interfaces.Customer, error) {
	if stripeCustomerID == "" {
		return nil, fmt.Errorf("Stripe customer ID is required")
	}

	return s.repo.GetByStripeID(ctx, stripeCustomerID)
}

// DeleteCustomer soft deletes a customer
func (s *CustomerService) DeleteCustomer(ctx context.Context, customerID int32) error {
	if customerID <= 0 {
		return fmt.Errorf("invalid customer ID: %d", customerID)
	}

	return s.repo.Delete(ctx, customerID)
}

// UpdateCustomerInStripe updates customer info in Stripe
func (s *CustomerService) UpdateCustomerInStripe(ctx context.Context, customerID int32) error {
	customer, err := s.repo.GetByID(ctx, customerID)
	if err != nil {
		return fmt.Errorf("failed to get customer: %w", err)
	}

	if !customer.StripeCustomerID.Valid {
		return fmt.Errorf("customer has no Stripe ID")
	}

	// Update logic would go here using stripeProvider
	return nil
}

// GetCustomerCount returns total customer count
func (s *CustomerService) GetCustomerCount(ctx context.Context) (int64, error) {
	return s.repo.GetCount(ctx)
}

// GetCustomersWithStripeCount returns count of customers with Stripe IDs
func (s *CustomerService) GetCustomersWithStripeCount(ctx context.Context) (int64, error) {
	return s.repo.GetCountWithStripeID(ctx)
}

// GetCustomersWithoutStripeIDs returns customers missing Stripe IDs
func (s *CustomerService) GetCustomersWithoutStripeIDs(ctx context.Context, limit, offset int) ([]interfaces.Customer, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	return s.repo.GetWithoutStripeID(ctx, limit, offset)
}

// GetCustomerStats returns customer analytics
func (s *CustomerService) GetCustomerStats(ctx context.Context) (*interfaces.CustomerStats, error) {
	stats := &interfaces.CustomerStats{}

	// Get total customers
	total, err := s.GetCustomerCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer count: %w", err)
	}
	stats.TotalCustomers = total

	// Get customers with Stripe
	withStripe, err := s.GetCustomersWithStripeCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get customers with Stripe count: %w", err)
	}
	stats.CustomersWithStripe = withStripe
	stats.CustomersWithoutStripe = total - withStripe

	// Calculate percentage
	if total > 0 {
		stats.StripeSyncPercentage = float64(withStripe) / float64(total) * 100
	}

	// Get recent customers
	recent, err := s.GetRecentCustomers(ctx, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent customers: %w", err)
	}
	stats.RecentCustomers = recent

	return stats, nil
}

// SearchCustomers searches customers by email/name
func (s *CustomerService) SearchCustomers(ctx context.Context, query string, limit, offset int) ([]interfaces.Customer, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	return s.repo.Search(ctx, query, limit, offset)
}

// GetRecentCustomers returns recently created customers
func (s *CustomerService) GetRecentCustomers(ctx context.Context, limit int) ([]interfaces.Customer, error) {
	if limit <= 0 {
		limit = 10
	}

	return s.repo.GetRecent(ctx, limit)
}

// ValidateCustomer validates customer data
func (s *CustomerService) ValidateCustomer(customer *interfaces.Customer) error {
	if customer == nil {
		return fmt.Errorf("customer cannot be nil")
	}

	if customer.Email == "" {
		return fmt.Errorf("customer email is required")
	}

	// Add more validation as needed
	return nil
}

// IsEmailTaken checks if email is already in use
func (s *CustomerService) IsEmailTaken(ctx context.Context, email string, excludeCustomerID *int32) (bool, error) {
	customer, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if err.Error() == "customer not found" {
			return false, nil
		}
		return false, err
	}

	// If excluding a specific customer ID (for updates)
	if excludeCustomerID != nil && customer.ID == *excludeCustomerID {
		return false, nil
	}

	return true, nil
}

// Helper methods
func (s *CustomerService) validateCreateRequest(req interfaces.CreateCustomerRequest) error {
	if req.Email == "" {
		return fmt.Errorf("email is required")
	}
	if req.PasswordHash == "" {
		return fmt.Errorf("password hash is required")
	}
	return nil
}

func (s *CustomerService) publishCustomerEvent(ctx context.Context, eventType string, customerID int32, data map[string]interface{}) error {
	if s.events == nil {
		return nil // Events publisher not configured
	}

	event := interfaces.BuildCustomerEvent(eventType, customerID, data)
	return s.events.PublishEvent(ctx, event)
}
