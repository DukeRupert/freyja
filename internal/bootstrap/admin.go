// Package bootstrap handles one-time initialization tasks for the application.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/dukerupert/freyja/internal/auth"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// AdminConfig contains configuration for the initial admin user.
type AdminConfig struct {
	Email     string
	Password  string
	FirstName string
	LastName  string
}

// Validate checks that the admin configuration is valid.
func (c *AdminConfig) Validate() error {
	if c.Email == "" {
		return errors.New("admin email is required")
	}
	if c.Password == "" {
		return errors.New("admin password is required")
	}
	if len(c.Password) < 12 {
		return errors.New("admin password must be at least 12 characters")
	}
	return nil
}

// EnsureMasterAdmin creates the initial admin user if it doesn't exist.
// This function is idempotent - safe to call on every startup.
//
// If the admin user already exists (by email), it returns without error.
// If AdminConfig is nil or has empty Email/Password, it logs a warning and skips.
//
// Returns error if:
// - Password is too short (< 12 characters)
// - Password hashing fails
// - Database operation fails (other than conflict)
func EnsureMasterAdmin(
	ctx context.Context,
	repo *repository.Queries,
	tenantID pgtype.UUID,
	cfg *AdminConfig,
	logger *slog.Logger,
) error {
	// If no config provided, skip admin creation (allows running without admin in dev)
	if cfg == nil || cfg.Email == "" || cfg.Password == "" {
		logger.Warn("bootstrap: skipping admin creation - FREYJA_ADMIN_EMAIL or FREYJA_ADMIN_PASSWORD not set",
			"hint", "Set these environment variables to create an admin user on first startup",
		)
		return nil
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid admin configuration: %w", err)
	}

	// Check if admin already exists
	existing, err := repo.GetUserByEmail(ctx, repository.GetUserByEmailParams{
		TenantID: tenantID,
		Email:    cfg.Email,
	})
	if err == nil && existing.ID.Valid {
		logger.Info("bootstrap: admin user already exists",
			"email", cfg.Email,
		)
		return nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to check for existing admin: %w", err)
	}

	// Hash the password
	passwordHash, err := auth.HashPassword(cfg.Password)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	// Set default names if not provided
	firstName := cfg.FirstName
	if firstName == "" {
		firstName = "Admin"
	}
	lastName := cfg.LastName
	if lastName == "" {
		lastName = "User"
	}

	// Create the admin user
	user, err := repo.CreateAdminUser(ctx, repository.CreateAdminUserParams{
		TenantID:     tenantID,
		Email:        cfg.Email,
		PasswordHash: pgtype.Text{String: passwordHash, Valid: true},
		FirstName:    pgtype.Text{String: firstName, Valid: true},
		LastName:     pgtype.Text{String: lastName, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	// User might be nil if ON CONFLICT DO NOTHING triggered (race condition)
	if !user.ID.Valid {
		logger.Info("bootstrap: admin user already exists (concurrent creation)",
			"email", cfg.Email,
		)
		return nil
	}

	logger.Info("bootstrap: admin user created successfully",
		"email", cfg.Email,
		"user_id", user.ID.Bytes,
	)

	return nil
}
