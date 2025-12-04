package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dukerupert/freyja/internal/auth"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrUserExists       = errors.New("user with this email already exists")
	ErrInvalidEmail     = errors.New("invalid email address")
	ErrUserNotFound     = errors.New("user not found")
	ErrInvalidPassword  = errors.New("invalid password")
	ErrAccountSuspended = errors.New("account is suspended")
	ErrAccountPending   = errors.New("account is pending approval")
	ErrEmailNotVerified = errors.New("email has not been verified")
)

// SessionData represents the data stored in a session
type SessionData struct {
	UserID string `json:"user_id"`
}

// UserService provides business logic for user operations
type UserService interface {
	// Register creates a new user account
	Register(ctx context.Context, email, password, firstName, lastName string) (*repository.User, error)

	// Authenticate verifies email/password and returns the user if valid
	Authenticate(ctx context.Context, email, password string) (*repository.User, error)

	// CreateSession creates a new session for a user
	CreateSession(ctx context.Context, userID string) (string, error)

	// GetUserBySessionToken retrieves a user from a session token
	GetUserBySessionToken(ctx context.Context, token string) (*repository.User, error)

	// DeleteSession logs out a user by deleting their session
	DeleteSession(ctx context.Context, token string) error

	// GetUserByID retrieves a user by ID
	GetUserByID(ctx context.Context, userID string) (*repository.User, error)
}

type userService struct {
	repo     repository.Querier
	tenantID pgtype.UUID
}

// NewUserService creates a new UserService instance
func NewUserService(repo repository.Querier, tenantID string) (UserService, error) {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		return nil, fmt.Errorf("invalid tenant ID: %w", err)
	}

	return &userService{
		repo:     repo,
		tenantID: tenantUUID,
	}, nil
}

// Register creates a new user account
func (s *userService) Register(ctx context.Context, email, password, firstName, lastName string) (*repository.User, error) {
	// Validate email (basic check)
	if email == "" || len(email) < 3 {
		return nil, ErrInvalidEmail
	}

	// Check if user already exists
	existingUser, err := s.repo.GetUserByEmail(ctx, repository.GetUserByEmailParams{
		TenantID: s.tenantID,
		Email:    email,
	})
	if err == nil && existingUser.ID.Valid {
		return nil, ErrUserExists
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	// Hash password
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	var firstNameText, lastNameText, passwordHashText pgtype.Text
	if firstName != "" {
		firstNameText = pgtype.Text{String: firstName, Valid: true}
	}
	if lastName != "" {
		lastNameText = pgtype.Text{String: lastName, Valid: true}
	}
	passwordHashText = pgtype.Text{String: passwordHash, Valid: true}

	user, err := s.repo.CreateUser(ctx, repository.CreateUserParams{
		TenantID:     s.tenantID,
		Email:        email,
		PasswordHash: passwordHashText,
		FirstName:    firstNameText,
		LastName:     lastNameText,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

// Authenticate verifies email/password and returns the user if valid
func (s *userService) Authenticate(ctx context.Context, email, password string) (*repository.User, error) {
	// Get user by email
	user, err := s.repo.GetUserByEmail(ctx, repository.GetUserByEmailParams{
		TenantID: s.tenantID,
		Email:    email,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check account status
	switch user.Status {
	case "suspended":
		return nil, ErrAccountSuspended
	case "pending":
		return nil, ErrAccountPending
	case "closed":
		return nil, ErrUserNotFound
	}

	// Verify password
	if !user.PasswordHash.Valid {
		return nil, ErrInvalidPassword
	}

	if err := auth.VerifyPassword(password, user.PasswordHash.String); err != nil {
		if errors.Is(err, auth.ErrPasswordMismatch) {
			return nil, ErrInvalidPassword
		}
		return nil, fmt.Errorf("failed to verify password: %w", err)
	}

	// Check if email is verified
	if !user.EmailVerified {
		return nil, ErrEmailNotVerified
	}

	return &user, nil
}

// CreateSession creates a new session for a user
func (s *userService) CreateSession(ctx context.Context, userID string) (string, error) {
	// Generate session token
	token, err := GenerateSessionID()
	if err != nil {
		return "", fmt.Errorf("failed to generate session token: %w", err)
	}

	// Store user ID in session data
	sessionData := SessionData{UserID: userID}
	dataJSON, err := json.Marshal(sessionData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal session data: %w", err)
	}

	// Create session (expires in 30 days)
	expiresAt := pgtype.Timestamptz{}
	expiresAt.Scan(time.Now().Add(30 * 24 * time.Hour))

	_, err = s.repo.CreateSession(ctx, repository.CreateSessionParams{
		Token:     token,
		Data:      dataJSON,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return token, nil
}

// GetUserBySessionToken retrieves a user from a session token
func (s *userService) GetUserBySessionToken(ctx context.Context, token string) (*repository.User, error) {
	// Get session
	session, err := s.repo.GetSessionByToken(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Parse session data
	var sessionData SessionData
	if err := json.Unmarshal(session.Data, &sessionData); err != nil {
		return nil, fmt.Errorf("failed to parse session data: %w", err)
	}

	// Get user
	var userUUID pgtype.UUID
	if err := userUUID.Scan(sessionData.UserID); err != nil {
		return nil, fmt.Errorf("invalid user ID in session: %w", err)
	}

	// SECURITY: Use tenant-scoped query to prevent cross-tenant access
	user, err := s.repo.GetUserByIDAndTenant(ctx, repository.GetUserByIDAndTenantParams{
		ID:       userUUID,
		TenantID: s.tenantID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// DeleteSession logs out a user by deleting their session
func (s *userService) DeleteSession(ctx context.Context, token string) error {
	if err := s.repo.DeleteSession(ctx, token); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// GetUserByID retrieves a user by ID
func (s *userService) GetUserByID(ctx context.Context, userID string) (*repository.User, error) {
	var userUUID pgtype.UUID
	if err := userUUID.Scan(userID); err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	user, err := s.repo.GetUserByID(ctx, userUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}
