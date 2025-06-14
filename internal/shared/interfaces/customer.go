// internal/interfaces/customer_repo.go
package interfaces

import (
	"context"

	"github.com/dukerupert/freyja/internal/database"
)

// Use database types directly for MVP simplicity
type Customer = database.Customers

type CustomerRepository interface {
	// Basic CRUD operations - Fixed to use int32 consistently
	GetByID(ctx context.Context, id int32) (*Customer, error)
	GetByEmail(ctx context.Context, email string) (*Customer, error)
	GetByStripeID(ctx context.Context, stripeCustomerID string) (*Customer, error)
	Create(ctx context.Context, customer *Customer) error
	Update(ctx context.Context, customer *Customer) error
	UpdateStripeID(ctx context.Context, customerID int32, stripeCustomerID string) error
	Delete(ctx context.Context, id int32) error

	// Query operations
	GetCount(ctx context.Context) (int64, error)
	GetCountWithStripeID(ctx context.Context) (int64, error)
	GetWithoutStripeID(ctx context.Context, limit, offset int) ([]Customer, error)
	Search(ctx context.Context, query string, limit, offset int) ([]Customer, error)
	GetRecent(ctx context.Context, limit int) ([]Customer, error)
	GetAll(ctx context.Context, limit, offset int) ([]Customer, error)

	// Analytics support
	GetCustomersWithOrderStats(ctx context.Context, limit int) ([]CustomerOrderSummary, error)
}

type CustomerService interface {
	// Customer management
	CreateCustomer(ctx context.Context, req CreateCustomerRequest) (*Customer, error)
	GetCustomerByID(ctx context.Context, customerID int32) (*Customer, error)
	GetCustomerByEmail(ctx context.Context, email string) (*Customer, error)
	GetCustomerByStripeID(ctx context.Context, stripeCustomerID string) (*Customer, error)
	UpdateCustomer(ctx context.Context, customerID int32, req UpdateCustomerRequest) (*Customer, error)
	UpdateStripeID(ctx context.Context, customerID int32, stripeCustomerID string) error
	DeleteCustomer(ctx context.Context, customerID int32) error

	// Stripe integration
	EnsureStripeCustomer(ctx context.Context, customerID int32) (string, error)
	UpdateCustomerInStripe(ctx context.Context, customerID int32) error
	CreateCustomerFromStripe(ctx context.Context, stripeCustomerID, email string) (*Customer, error)

	// Customer queries and analytics
	GetCustomerCount(ctx context.Context) (int64, error)
	GetCustomersWithStripeCount(ctx context.Context) (int64, error)
	GetCustomersWithoutStripeIDs(ctx context.Context, limit, offset int) ([]Customer, error)
	GetCustomerStats(ctx context.Context) (*CustomerStats, error)
	SearchCustomers(ctx context.Context, query string, limit, offset int) ([]Customer, error)
	GetRecentCustomers(ctx context.Context, limit int) ([]Customer, error)

	// Validation and utilities - Fixed to use int32
	ValidateCustomer(customer *Customer) error
	IsEmailTaken(ctx context.Context, email string, excludeCustomerID *int32) (bool, error)
}

// Request/Response types for Customer Service
type CreateCustomerRequest struct {
	Email        string `json:"email" validate:"required,email"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	PasswordHash string `json:"-"` // Never include in JSON
}

type UpdateCustomerRequest struct {
	Email     string  `json:"email,omitempty" validate:"omitempty,email"`
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
}

// Customer statistics for admin dashboard
type CustomerStats struct {
	TotalCustomers         int64                  `json:"total_customers"`
	CustomersWithStripe    int64                  `json:"customers_with_stripe"`
	CustomersWithoutStripe int64                  `json:"customers_without_stripe"`
	StripeSyncPercentage   float64                `json:"stripe_sync_percentage"`
	RecentCustomers        []Customer             `json:"recent_customers"`
	TopCustomersByOrders   []CustomerOrderSummary `json:"top_customers_by_orders,omitempty"`
}

// Customer order summary for analytics
type CustomerOrderSummary struct {
	CustomerID  int32  `json:"customer_id"`
	Email       string `json:"email"`
	FirstName   string `json:"first_name,omitempty"`
	LastName    string `json:"last_name,omitempty"`
	OrderCount  int64  `json:"order_count"`
	TotalSpent  int64  `json:"total_spent"` // in cents
	LastOrderAt string `json:"last_order_at,omitempty"`
}

// Customer search filters
type CustomerSearchFilters struct {
	Query       string `json:"query,omitempty"`
	HasStripeID *bool  `json:"has_stripe_id,omitempty"`
	Active      *bool  `json:"active,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Offset      int    `json:"offset,omitempty"`
}
