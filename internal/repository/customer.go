// internal/repository/customer.go
package repository

import (
	"context"
	"fmt"
	"time"

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

// Basic CRUD operations

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

func (r *PostgresCustomerRepository) GetByStripeID(ctx context.Context, stripeCustomerID string) (*interfaces.Customer, error) {
	customer, err := r.db.Queries.GetCustomerByStripeID(ctx, pgtype.Text{
		String: stripeCustomerID,
		Valid:  true,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("customer not found")
		}
		return nil, fmt.Errorf("failed to get customer by Stripe ID: %w", err)
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
	// Use soft delete (archive) instead of hard delete
	_, err := r.db.Queries.ArchiveCustomer(ctx, int32(id))
	if err != nil {
		return fmt.Errorf("failed to archive customer: %w", err)
	}
	return nil
}

// Query operations

func (r *PostgresCustomerRepository) GetCount(ctx context.Context) (int64, error) {
	count, err := r.db.Queries.GetCustomerCount(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get customer count: %w", err)
	}
	return count, nil
}

func (r *PostgresCustomerRepository) GetCountWithStripeID(ctx context.Context) (int64, error) {
	count, err := r.db.Queries.GetCustomerCountWithStripeID(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get customer count with Stripe ID: %w", err)
	}
	return count, nil
}

func (r *PostgresCustomerRepository) GetWithoutStripeID(ctx context.Context, limit, offset int) ([]interfaces.Customer, error) {
	customers, err := r.db.Queries.GetCustomersWithoutStripeID(ctx, database.GetCustomersWithoutStripeIDParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get customers without Stripe ID: %w", err)
	}

	return customers, nil
}

func (r *PostgresCustomerRepository) Search(ctx context.Context, query string, limit, offset int) ([]interfaces.Customer, error) {
	searchTerm := "%" + query + "%"

	customers, err := r.db.Queries.SearchCustomers(ctx, database.SearchCustomersParams{
		Email:  searchTerm,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search customers: %w", err)
	}

	// Convert to []interfaces.Customer
	result := make([]interfaces.Customer, len(customers))
	for i, customer := range customers {
		result[i] = interfaces.Customer{
			ID:               customer.ID,
			Email:            customer.Email,
			FirstName:        customer.FirstName,
			LastName:         customer.LastName,
			PasswordHash:     customer.PasswordHash,
			StripeCustomerID: customer.StripeCustomerID,
			CreatedAt:        customer.CreatedAt,
			UpdatedAt:        customer.UpdatedAt,
		}
	}

	return result, nil
}

func (r *PostgresCustomerRepository) GetRecent(ctx context.Context, limit int) ([]interfaces.Customer, error) {
	customers, err := r.db.Queries.GetRecentCustomers(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("failed to get recent customers: %w", err)
	}

	// Convert to []interfaces.Customer
	result := make([]interfaces.Customer, len(customers))
	for i, customer := range customers {
		result[i] = interfaces.Customer{
			ID:               customer.ID,
			Email:            customer.Email,
			FirstName:        customer.FirstName,
			LastName:         customer.LastName,
			PasswordHash:     customer.PasswordHash,
			StripeCustomerID: customer.StripeCustomerID,
			CreatedAt:        customer.CreatedAt,
			UpdatedAt:        customer.UpdatedAt,
		}
	}

	return result, nil
}

func (r *PostgresCustomerRepository) GetAll(ctx context.Context, limit, offset int) ([]interfaces.Customer, error) {
	customers, err := r.db.Queries.ListCustomers(ctx, database.ListCustomersParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get customers: %w", err)
	}

	// Convert to []interfaces.Customer
	result := make([]interfaces.Customer, len(customers))
	for i, customer := range customers {
		result[i] = interfaces.Customer{
			ID:               customer.ID,
			Email:            customer.Email,
			FirstName:        customer.FirstName,
			LastName:         customer.LastName,
			PasswordHash:     customer.PasswordHash,
			StripeCustomerID: customer.StripeCustomerID,
			CreatedAt:        customer.CreatedAt,
			UpdatedAt:        customer.UpdatedAt,
		}
	}

	return result, nil
}

// Analytics support

func (r *PostgresCustomerRepository) GetCustomersWithOrderStats(ctx context.Context, limit int) ([]interfaces.CustomerOrderSummary, error) {
	results, err := r.db.Queries.GetCustomersWithOrderStats(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("failed to get customers with order stats: %w", err)
	}

	summaries := make([]interfaces.CustomerOrderSummary, len(results))
	for i, result := range results {
		summaries[i] = interfaces.CustomerOrderSummary{
			CustomerID: result.CustomerID,
			Email:      result.Email,
			OrderCount: result.OrderCount,
			TotalSpent: result.TotalSpent,
		}

		// Handle optional fields
		if result.FirstName.Valid {
			summaries[i].FirstName = result.FirstName.String
		}
		if result.LastName.Valid {
			summaries[i].LastName = result.LastName.String
		}

		// Handle LastOrderAt - it's an interface{} that could be time.Time or nil
		if result.LastOrderAt != nil {
			if lastOrderTime, ok := result.LastOrderAt.(time.Time); ok {
				summaries[i].LastOrderAt = lastOrderTime.Format("2006-01-02T15:04:05Z")
			}
		}
	}

	return summaries, nil
}
