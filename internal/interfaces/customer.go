// internal/interfaces/customer.go
package interfaces

import (
	"context"

	"github.com/dukerupert/freyja/internal/database"
)

// Use database types directly for MVP simplicity
type Customer = database.Customers

type CustomerRepository interface {
	GetByID(ctx context.Context, id int) (*Customer, error)
	GetByEmail(ctx context.Context, email string) (*Customer, error)
	Create(ctx context.Context, customer *Customer) error
	Update(ctx context.Context, customer *Customer) error
	UpdateStripeID(ctx context.Context, customerID int, stripeCustomerID string) error
	Delete(ctx context.Context, id int) error
}