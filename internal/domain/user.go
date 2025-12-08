package domain

import (
	"context"

	"github.com/dukerupert/freyja/internal/repository"
)

// UserService provides business logic for customer user operations.
// Implementations should be tenant-scoped.
type UserService interface {
	// Register creates a new user account.
	Register(ctx context.Context, email, password, firstName, lastName string) (*repository.User, error)

	// Authenticate verifies email/password and returns the user if valid.
	Authenticate(ctx context.Context, email, password string) (*repository.User, error)

	// CreateSession creates a new session for a user.
	CreateSession(ctx context.Context, userID string) (string, error)

	// GetUserBySessionToken retrieves a user from a session token.
	GetUserBySessionToken(ctx context.Context, token string) (*repository.User, error)

	// DeleteSession logs out a user by deleting their session.
	DeleteSession(ctx context.Context, token string) error

	// GetUserByID retrieves a user by ID.
	GetUserByID(ctx context.Context, userID string) (*repository.User, error)
}

// SessionData represents the data stored in a session.
type SessionData struct {
	UserID string `json:"user_id"`
}
