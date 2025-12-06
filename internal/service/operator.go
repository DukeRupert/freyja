package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"time"

	"github.com/dukerupert/freyja/internal/auth"
	"github.com/dukerupert/freyja/internal/domain"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Operator service constants
const (
	// OperatorSetupTokenExpiry is how long setup tokens are valid (48 hours)
	OperatorSetupTokenExpiry = 48 * time.Hour

	// OperatorResetTokenExpiry is how long reset tokens are valid (1 hour)
	OperatorResetTokenExpiry = 1 * time.Hour

	// OperatorSessionExpiry is how long sessions are valid (7 days)
	OperatorSessionExpiry = 7 * 24 * time.Hour

	// OperatorTokenLength is the number of random bytes in tokens (32 bytes = 256 bits)
	OperatorTokenLength = 32
)

// Operator service errors
var (
	ErrOperatorNotFound     = errors.New("operator not found")
	ErrOperatorSuspended    = errors.New("operator account is suspended")
	ErrOperatorPending      = errors.New("operator account setup not complete")
	ErrOperatorExists       = errors.New("operator with this email already exists")
	ErrOperatorInvalidToken = errors.New("invalid or expired token")
	ErrWeakPassword         = errors.New("password must be at least 8 characters")
)

// OperatorService provides business logic for tenant operator operations
type OperatorService interface {
	// Authentication methods

	// Authenticate verifies email/password and returns the operator if valid
	Authenticate(ctx context.Context, email, password string) (*repository.TenantOperator, error)

	// CreateSession creates a new session for an operator
	// Returns the raw session token (not hashed) to be set as cookie
	CreateSession(ctx context.Context, operatorID uuid.UUID, userAgent, ipAddress string) (string, error)

	// GetOperatorBySessionToken retrieves an operator from a session token
	GetOperatorBySessionToken(ctx context.Context, rawToken string) (*repository.TenantOperator, error)

	// DeleteSession logs out an operator by deleting their session
	DeleteSession(ctx context.Context, rawToken string) error

	// DeleteAllSessions logs out an operator from all devices
	DeleteAllSessions(ctx context.Context, operatorID uuid.UUID) error

	// Password management methods

	// CreateOperator creates a new operator with a setup token
	// Returns the operator and the raw setup token (to be sent via email)
	CreateOperator(ctx context.Context, tenantID uuid.UUID, email, name, role string) (*repository.TenantOperator, string, error)

	// ValidateSetupToken validates a setup token and returns the associated operator
	ValidateSetupToken(ctx context.Context, rawToken string) (*repository.TenantOperator, error)

	// SetPassword sets the operator's password and activates their account
	SetPassword(ctx context.Context, operatorID uuid.UUID, password string) error

	// RequestPasswordReset initiates a password reset flow
	// Returns the raw reset token (to be sent via email)
	RequestPasswordReset(ctx context.Context, email string) (string, *repository.TenantOperator, error)

	// ValidateResetToken validates a reset token and returns the associated operator
	ValidateResetToken(ctx context.Context, rawToken string) (*repository.TenantOperator, error)

	// ResetPassword completes password reset using a valid token
	ResetPassword(ctx context.Context, rawToken, newPassword string) error

	// ResendSetupToken regenerates and returns a new setup token for a pending operator
	ResendSetupToken(ctx context.Context, email string) (string, *repository.TenantOperator, error)

	// Operator retrieval methods

	// GetOperatorByID retrieves an operator by ID
	GetOperatorByID(ctx context.Context, operatorID uuid.UUID) (*repository.TenantOperator, error)

	// GetOperatorByEmail retrieves an operator by email
	GetOperatorByEmail(ctx context.Context, email string) (*repository.TenantOperator, error)

	// UpdateLastLogin updates the last login timestamp
	UpdateLastLogin(ctx context.Context, operatorID uuid.UUID) error
}

type operatorService struct {
	repo   repository.Querier
	logger *slog.Logger
}

// NewOperatorService creates a new OperatorService instance
func NewOperatorService(repo repository.Querier, logger *slog.Logger) OperatorService {
	if logger == nil {
		logger = slog.Default()
	}
	return &operatorService{
		repo:   repo,
		logger: logger,
	}
}

// Authenticate verifies email/password and returns the operator if valid
func (s *operatorService) Authenticate(ctx context.Context, email, password string) (*repository.TenantOperator, error) {
	// Get operator by email (global lookup, not tenant-scoped)
	operator, err := s.repo.GetTenantOperatorByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOperatorNotFound
		}
		return nil, fmt.Errorf("failed to get operator: %w", err)
	}

	// Check operator status
	switch operator.Status {
	case "suspended":
		return nil, ErrOperatorSuspended
	case "pending":
		return nil, ErrOperatorPending
	}

	// Verify password
	if !operator.PasswordHash.Valid {
		return nil, ErrInvalidPassword
	}

	if err := auth.VerifyPassword(password, operator.PasswordHash.String); err != nil {
		if errors.Is(err, auth.ErrPasswordMismatch) {
			return nil, ErrInvalidPassword
		}
		return nil, fmt.Errorf("failed to verify password: %w", err)
	}

	return &operator, nil
}

// CreateSession creates a new session for an operator
func (s *operatorService) CreateSession(ctx context.Context, operatorID uuid.UUID, userAgent, ipAddress string) (string, error) {
	// Generate session token
	rawToken, err := generateSecureToken(OperatorTokenLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate session token: %w", err)
	}

	// Hash token for storage
	tokenHash := hashOperatorToken(rawToken)

	// Parse IP address
	var ipAddr *netip.Addr
	if ipAddress != "" {
		if addr, err := netip.ParseAddr(ipAddress); err == nil {
			ipAddr = &addr
		}
	}

	// Create session
	expiresAt := pgtype.Timestamptz{}
	_ = expiresAt.Scan(time.Now().Add(OperatorSessionExpiry))

	_, err = s.repo.CreateOperatorSession(ctx, repository.CreateOperatorSessionParams{
		OperatorID: uuidToPgtype(operatorID),
		TokenHash:  tokenHash,
		UserAgent:  pgtype.Text{String: userAgent, Valid: userAgent != ""},
		IpAddress:  ipAddr,
		ExpiresAt:  expiresAt,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}

	return rawToken, nil
}

// GetOperatorBySessionToken retrieves an operator from a session token
func (s *operatorService) GetOperatorBySessionToken(ctx context.Context, rawToken string) (*repository.TenantOperator, error) {
	// Hash the token to look up
	tokenHash := hashOperatorToken(rawToken)

	// Get session by token hash
	session, err := s.repo.GetOperatorSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionExpired
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Session expiry is checked in the query, but double-check
	if session.ExpiresAt.Valid && session.ExpiresAt.Time.Before(time.Now()) {
		return nil, ErrSessionExpired
	}

	// Get operator
	operatorUUID, err := pgtypeToUUID(session.OperatorID)
	if err != nil {
		return nil, fmt.Errorf("invalid operator ID in session: %w", err)
	}

	operator, err := s.repo.GetTenantOperatorByID(ctx, session.OperatorID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOperatorNotFound
		}
		return nil, fmt.Errorf("failed to get operator: %w", err)
	}

	// Check operator status
	switch operator.Status {
	case "suspended":
		s.logger.Warn("suspended operator attempted session access",
			"operator_id", operatorUUID,
			"email", operator.Email)
		return nil, ErrOperatorSuspended
	case "pending":
		return nil, ErrOperatorPending
	}

	return &operator, nil
}

// DeleteSession logs out an operator by deleting their session
func (s *operatorService) DeleteSession(ctx context.Context, rawToken string) error {
	tokenHash := hashOperatorToken(rawToken)
	if err := s.repo.DeleteOperatorSession(ctx, tokenHash); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// DeleteAllSessions logs out an operator from all devices
func (s *operatorService) DeleteAllSessions(ctx context.Context, operatorID uuid.UUID) error {
	if err := s.repo.DeleteOperatorSessionsByOperatorID(ctx, uuidToPgtype(operatorID)); err != nil {
		return fmt.Errorf("failed to delete all sessions: %w", err)
	}
	return nil
}

// CreateOperator creates a new operator with a setup token
func (s *operatorService) CreateOperator(ctx context.Context, tenantID uuid.UUID, email, name, role string) (*repository.TenantOperator, string, error) {
	// Check if operator already exists
	existing, err := s.repo.GetTenantOperatorByEmailAndTenant(ctx, repository.GetTenantOperatorByEmailAndTenantParams{
		TenantID: uuidToPgtype(tenantID),
		Email:    email,
	})
	if err == nil && existing.ID.Valid {
		return nil, "", ErrOperatorExists
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, "", fmt.Errorf("failed to check existing operator: %w", err)
	}

	// Generate setup token
	rawToken, err := generateSecureToken(OperatorTokenLength)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate setup token: %w", err)
	}
	tokenHash := hashOperatorToken(rawToken)

	// Set default role
	if role == "" {
		role = string(domain.OperatorRoleOwner)
	}

	// Create operator
	expiresAt := pgtype.Timestamptz{}
	_ = expiresAt.Scan(time.Now().Add(OperatorSetupTokenExpiry))

	operator, err := s.repo.CreateTenantOperator(ctx, repository.CreateTenantOperatorParams{
		TenantID:            uuidToPgtype(tenantID),
		Email:               email,
		Name:                pgtype.Text{String: name, Valid: name != ""},
		Role:                role,
		SetupTokenHash:      pgtype.Text{String: tokenHash, Valid: true},
		SetupTokenExpiresAt: expiresAt,
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to create operator: %w", err)
	}

	s.logger.Info("operator created",
		"operator_id", operator.ID,
		"tenant_id", tenantID,
		"email", email)

	return &operator, rawToken, nil
}

// ValidateSetupToken validates a setup token and returns the associated operator
func (s *operatorService) ValidateSetupToken(ctx context.Context, rawToken string) (*repository.TenantOperator, error) {
	tokenHash := hashOperatorToken(rawToken)

	operator, err := s.repo.GetTenantOperatorBySetupToken(ctx, pgtype.Text{String: tokenHash, Valid: true})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOperatorInvalidToken
		}
		return nil, fmt.Errorf("failed to get operator by setup token: %w", err)
	}

	return &operator, nil
}

// SetPassword sets the operator's password and activates their account
func (s *operatorService) SetPassword(ctx context.Context, operatorID uuid.UUID, password string) error {
	// Validate password
	if len(password) < 8 {
		return ErrWeakPassword
	}
	if len(password) > 72 {
		return fmt.Errorf("password must be less than 72 characters")
	}

	// Hash password
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Set password and activate
	err = s.repo.SetOperatorPassword(ctx, repository.SetOperatorPasswordParams{
		ID:           uuidToPgtype(operatorID),
		PasswordHash: pgtype.Text{String: passwordHash, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to set password: %w", err)
	}

	s.logger.Info("operator password set and account activated",
		"operator_id", operatorID)

	return nil
}

// RequestPasswordReset initiates a password reset flow
func (s *operatorService) RequestPasswordReset(ctx context.Context, email string) (string, *repository.TenantOperator, error) {
	// Get operator by email
	operator, err := s.repo.GetTenantOperatorByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Don't reveal whether email exists
			return "", nil, nil
		}
		return "", nil, fmt.Errorf("failed to get operator: %w", err)
	}

	// Only allow reset for active operators
	if operator.Status != "active" {
		return "", nil, nil
	}

	// Generate reset token
	rawToken, err := generateSecureToken(OperatorTokenLength)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate reset token: %w", err)
	}
	tokenHash := hashOperatorToken(rawToken)

	// Set reset token
	expiresAt := pgtype.Timestamptz{}
	_ = expiresAt.Scan(time.Now().Add(OperatorResetTokenExpiry))

	err = s.repo.SetOperatorResetToken(ctx, repository.SetOperatorResetTokenParams{
		ID:                  operator.ID,
		ResetTokenHash:      pgtype.Text{String: tokenHash, Valid: true},
		ResetTokenExpiresAt: expiresAt,
	})
	if err != nil {
		return "", nil, fmt.Errorf("failed to set reset token: %w", err)
	}

	s.logger.Info("operator password reset requested",
		"operator_id", operator.ID,
		"email", email)

	return rawToken, &operator, nil
}

// ValidateResetToken validates a reset token and returns the associated operator
func (s *operatorService) ValidateResetToken(ctx context.Context, rawToken string) (*repository.TenantOperator, error) {
	tokenHash := hashOperatorToken(rawToken)

	operator, err := s.repo.GetTenantOperatorByResetToken(ctx, pgtype.Text{String: tokenHash, Valid: true})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOperatorInvalidToken
		}
		return nil, fmt.Errorf("failed to get operator by reset token: %w", err)
	}

	return &operator, nil
}

// ResetPassword completes password reset using a valid token
func (s *operatorService) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	// Validate token first
	operator, err := s.ValidateResetToken(ctx, rawToken)
	if err != nil {
		return err
	}

	// Validate password
	if len(newPassword) < 8 {
		return ErrWeakPassword
	}
	if len(newPassword) > 72 {
		return fmt.Errorf("password must be less than 72 characters")
	}

	// Hash password
	passwordHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password and clear reset token
	err = s.repo.UpdateOperatorPassword(ctx, repository.UpdateOperatorPasswordParams{
		ID:           operator.ID,
		PasswordHash: pgtype.Text{String: passwordHash, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Delete all existing sessions (force re-login)
	operatorUUID, _ := pgtypeToUUID(operator.ID)
	_ = s.DeleteAllSessions(ctx, operatorUUID)

	s.logger.Info("operator password reset completed",
		"operator_id", operator.ID)

	return nil
}

// ResendSetupToken regenerates and returns a new setup token for a pending operator
func (s *operatorService) ResendSetupToken(ctx context.Context, email string) (string, *repository.TenantOperator, error) {
	// Get operator by email
	operator, err := s.repo.GetTenantOperatorByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Don't reveal whether email exists
			return "", nil, nil
		}
		return "", nil, fmt.Errorf("failed to get operator: %w", err)
	}

	// Only allow resend for pending operators
	if operator.Status != "pending" {
		return "", nil, nil
	}

	// Generate new setup token
	rawToken, err := generateSecureToken(OperatorTokenLength)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate setup token: %w", err)
	}
	tokenHash := hashOperatorToken(rawToken)

	// Update setup token
	expiresAt := pgtype.Timestamptz{}
	_ = expiresAt.Scan(time.Now().Add(OperatorSetupTokenExpiry))

	err = s.repo.SetOperatorSetupToken(ctx, repository.SetOperatorSetupTokenParams{
		ID:                  operator.ID,
		SetupTokenHash:      pgtype.Text{String: tokenHash, Valid: true},
		SetupTokenExpiresAt: expiresAt,
	})
	if err != nil {
		return "", nil, fmt.Errorf("failed to update setup token: %w", err)
	}

	s.logger.Info("operator setup token regenerated",
		"operator_id", operator.ID,
		"email", email)

	return rawToken, &operator, nil
}

// GetOperatorByID retrieves an operator by ID
func (s *operatorService) GetOperatorByID(ctx context.Context, operatorID uuid.UUID) (*repository.TenantOperator, error) {
	operator, err := s.repo.GetTenantOperatorByID(ctx, uuidToPgtype(operatorID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOperatorNotFound
		}
		return nil, fmt.Errorf("failed to get operator: %w", err)
	}
	return &operator, nil
}

// GetOperatorByEmail retrieves an operator by email
func (s *operatorService) GetOperatorByEmail(ctx context.Context, email string) (*repository.TenantOperator, error) {
	operator, err := s.repo.GetTenantOperatorByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOperatorNotFound
		}
		return nil, fmt.Errorf("failed to get operator: %w", err)
	}
	return &operator, nil
}

// UpdateLastLogin updates the last login timestamp
func (s *operatorService) UpdateLastLogin(ctx context.Context, operatorID uuid.UUID) error {
	if err := s.repo.UpdateOperatorLastLogin(ctx, uuidToPgtype(operatorID)); err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}
	return nil
}

// Helper functions

// generateSecureToken creates a cryptographically secure random token
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// hashOperatorToken creates SHA-256 hash of token for storage
func hashOperatorToken(rawToken string) string {
	hash := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(hash[:])
}
