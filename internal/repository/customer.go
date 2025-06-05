// internal/repository/customer.go
package repository

import (
	"context"
	"fmt"

	"github.com/dukerupert/freyja/internal/database"
	"github.com/dukerupert/freyja/internal/interfaces"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type PostgresCustomerRepository struct {
	db *database.DB
}

func NewPostgresCustomerRepository(db *database.DB) interfaces.CustomerRepository {
	return &PostgresCustomerRepository{
		db: db,
	}
}

func (r *PostgresCustomerRepository) GetByID(ctx context.Context, id int) (*interfaces.Customer, error) {
	customer, err := r.db.Queries.GetCustomer(ctx, int32(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("customer not found")
		}
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}
	return &customer, nil
}

func (r *PostgresCustomerRepository) GetByEmail(ctx context.Context, email string) (*interfaces.Customer, error) {
	customer, err := r.db.Queries.GetCustomerByEmail(ctx, email)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("customer not found")
		}
		return nil, fmt.Errorf("failed to get customer by email: %w", err)
	}
	return &customer, nil
}

func (r *PostgresCustomerRepository) Create(ctx context.Context, customer *interfaces.Customer) error {
	firstName := pgtype.Text{String: customer.FirstName.String, Valid: customer.FirstName.Valid}
	lastName := pgtype.Text{String: customer.LastName.String, Valid: customer.LastName.Valid}

	created, err := r.db.Queries.CreateCustomer(ctx, database.CreateCustomerParams{
		Email:        customer.Email,
		FirstName:    firstName,
		LastName:     lastName,
		PasswordHash: customer.PasswordHash,
	})
	if err != nil {
		return fmt.Errorf("failed to create customer: %w", err)
	}

	// Update the customer with the generated ID and timestamps
	customer.ID = created.ID
	customer.CreatedAt = created.CreatedAt
	customer.UpdatedAt = created.UpdatedAt

	return nil
}

func (r *PostgresCustomerRepository) Update(ctx context.Context, customer *interfaces.Customer) error {
	updated, err := r.db.Queries.UpdateCustomer(ctx, database.UpdateCustomerParams{
		ID:        customer.ID,
		Email:     customer.Email,
		FirstName: customer.FirstName,
		LastName:  customer.LastName,
	})
	if err != nil {
		return fmt.Errorf("failed to update customer: %w", err)
	}

	// Update the customer with fresh data
	customer.Email = updated.Email
	customer.FirstName = updated.FirstName
	customer.LastName = updated.LastName
	customer.UpdatedAt = updated.UpdatedAt

	return nil
}

func (r *PostgresCustomerRepository) UpdateStripeID(ctx context.Context, customerID int, stripeCustomerID string) error {
	stripeID := pgtype.Text{String: stripeCustomerID, Valid: true}
	
	_, err := r.db.Queries.UpdateCustomerStripeID(ctx, database.UpdateCustomerStripeIDParams{
		ID:               int32(customerID),
		StripeCustomerID: stripeID,
	})
	if err != nil {
		return fmt.Errorf("failed to update customer Stripe ID: %w", err)
	}

	return nil
}

func (r *PostgresCustomerRepository) Delete(ctx context.Context, id int) error {
	err := r.db.Queries.DeleteCustomer(ctx, int32(id))
	if err != nil {
		return fmt.Errorf("failed to delete customer: %w", err)
	}
	return nil
}