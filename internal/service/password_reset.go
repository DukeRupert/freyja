package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/dukerupert/freyja/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	// TokenLength is the number of random bytes in the reset token (32 bytes = 256 bits)
	TokenLength = 32

	// TokenExpiry is how long a reset token is valid (1 hour)
	TokenExpiry = 1 * time.Hour

	// RateLimitPerEmail is max reset requests per email in the rate limit window
	RateLimitPerEmail = 3

	// RateLimitPerIP is max reset requests per IP address in the rate limit window
	RateLimitPerIP = 5

	// RateLimitWindow is the time window for rate limiting (15 minutes)
	RateLimitWindow = 15 * time.Minute
)

var (
	// ErrInvalidToken indicates the reset token is invalid, expired, or already used
	ErrInvalidToken = errors.New("invalid or expired reset token")

	// ErrRateLimitExceeded indicates too many reset requests
	ErrRateLimitExceeded = errors.New("too many password reset requests, please try again later")
)

// PasswordResetService handles password reset operations
type PasswordResetService interface {
	// RequestPasswordReset initiates a password reset flow
	// Returns the raw token (to be sent via email) and any error
	RequestPasswordReset(ctx context.Context, tenantID uuid.UUID, email, ipAddress, userAgent string) (string, error)

	// ValidateResetToken verifies a reset token and returns the associated user ID
	ValidateResetToken(ctx context.Context, tenantID uuid.UUID, rawToken string) (uuid.UUID, error)

	// ResetPassword completes the password reset using a valid token
	ResetPassword(ctx context.Context, tenantID uuid.UUID, rawToken, newPassword string) error
}

type passwordResetService struct {
	repo repository.Querier
}

// NewPasswordResetService creates a new password reset service
func NewPasswordResetService(repo repository.Querier) PasswordResetService {
	return &passwordResetService{
		repo: repo,
	}
}

// RequestPasswordReset initiates a password reset flow
func (s *passwordResetService) RequestPasswordReset(
	ctx context.Context,
	tenantID uuid.UUID,
	email string,
	ipAddress string,
	userAgent string,
) (string, error) {
	// TODO: Implement password reset request logic
	// 1. Validate email and get user by email (GetUserByEmail)
	// 2. Check rate limits for email (CountRecentResetRequestsByEmail)
	// 3. Check rate limits for IP address (CountRecentResetRequestsByIP)
	// 4. Generate secure token (generateToken)
	// 5. Hash the token (hashToken)
	// 6. Store token in database (CreatePasswordResetToken)
	// 7. Return raw token to be sent via email
	return "", fmt.Errorf("not implemented")
}

// ValidateResetToken verifies a reset token and returns the associated user ID
func (s *passwordResetService) ValidateResetToken(
	ctx context.Context,
	tenantID uuid.UUID,
	rawToken string,
) (uuid.UUID, error) {
	// TODO: Implement token validation logic
	// 1. Hash the raw token
	// 2. Query database for valid token (GetPasswordResetToken)
	// 3. Check if token exists, is not used, and not expired (handled by query)
	// 4. Check user account status
	// 5. Return user ID if valid, or ErrInvalidToken
	return uuid.Nil, fmt.Errorf("not implemented")
}

// ResetPassword completes the password reset using a valid token
func (s *passwordResetService) ResetPassword(
	ctx context.Context,
	tenantID uuid.UUID,
	rawToken string,
	newPassword string,
) error {
	// TODO: Implement password reset logic
	// 1. Validate token and get user ID (ValidateResetToken)
	// 2. Hash new password (auth.HashPassword)
	// 3. Update user password (UpdateUserPassword)
	// 4. Mark token as used (MarkPasswordResetTokenUsed)
	// 5. Invalidate all other tokens for user (InvalidateUserPasswordResetTokens)
	// 6. Consider: Delete user sessions to force re-login (optional security measure)
	return fmt.Errorf("not implemented")
}

// generateToken creates a cryptographically secure random token
func generateToken() (string, error) {
	// TODO: Generate TokenLength bytes of random data
	// Use crypto/rand.Read for cryptographic security
	// Encode as hex string for URL-safe transmission
	b := make([]byte, TokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// hashToken creates a SHA-256 hash of a token for storage
func hashToken(token string) string {
	// TODO: Hash the token using SHA-256
	// Return hex-encoded hash string
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// Helper functions for converting between uuid.UUID and pgtype.UUID
func uuidToPgtype(u uuid.UUID) pgtype.UUID {
	var pgtypeUUID pgtype.UUID
	_ = pgtypeUUID.Scan(u.String())
	return pgtypeUUID
}

func pgtypeToUUID(u pgtype.UUID) (uuid.UUID, error) {
	if !u.Valid {
		return uuid.Nil, errors.New("invalid UUID")
	}
	return uuid.FromBytes(u.Bytes[:])
}
