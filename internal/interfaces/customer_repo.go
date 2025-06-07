// internal/interfaces/customer_repo.go
package interfaces

import (
	"context"

	"github.com/dukerupert/freyja/internal/database"
)

// Use database types directly for MVP simplicity
type Customer = database.Customers

type CustomerRepository interface {
	// Basic CRUD operations
	GetByID(ctx context.Context, id int) (*Customer, error)
	GetByEmail(ctx context.Context, email string) (*Customer, error)
	GetByStripeID(ctx context.Context, stripeCustomerID string) (*Customer, error)
	Create(ctx context.Context, customer *Customer) error
	Update(ctx context.Context, customer *Customer) error
	UpdateStripeID(ctx context.Context, customerID int, stripeCustomerID string) error
	Delete(ctx context.Context, id int) error

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
