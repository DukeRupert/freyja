package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/dukerupert/hiri/internal/auth"
	"github.com/dukerupert/hiri/internal/domain"
	"github.com/dukerupert/hiri/internal/jobs"
	"github.com/dukerupert/hiri/internal/repository"
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
	ErrInvalidToken = domain.Errorf(domain.EINVALID, "", "Invalid or expired reset token")

	// ErrRateLimitExceeded indicates too many reset requests
	ErrRateLimitExceeded = domain.Errorf(domain.ERATELIMIT, "", "Too many password reset requests, please try again later")
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
	// Get user by email
	user, err := s.repo.GetUserByEmail(ctx, repository.GetUserByEmailParams{
		TenantID: uuidToPgtype(tenantID),
		Email:    email,
	})
	if err != nil {
		// Always return nil to prevent user enumeration
		// Log the error internally but don't expose to caller
		fmt.Printf("password reset request for non-existent email: %s\n", email)
		return "", nil
	}

	// Check rate limit for this email
	rateLimitCutoff := time.Now().Add(-RateLimitWindow)
	emailCount, err := s.repo.CountRecentResetRequestsByEmail(ctx, repository.CountRecentResetRequestsByEmailParams{
		UserID:    user.ID,
		CreatedAt: pgtype.Timestamptz{Time: rateLimitCutoff, Valid: true},
	})
	if err != nil {
		fmt.Printf("error checking email rate limit: %v\n", err)
		return "", nil
	}
	if emailCount >= RateLimitPerEmail {
		// Log rate limit exceeded but still return nil to prevent enumeration
		fmt.Printf("rate limit exceeded for email: %s\n", email)
		return "", nil
	}

	// Check rate limit for IP address
	ipCount, err := s.repo.CountRecentResetRequestsByIP(ctx, repository.CountRecentResetRequestsByIPParams{
		IpAddress: pgtype.Text{String: ipAddress, Valid: true},
		CreatedAt: pgtype.Timestamptz{Time: rateLimitCutoff, Valid: true},
	})
	if err != nil {
		fmt.Printf("error checking IP rate limit: %v\n", err)
		return "", nil
	}
	if ipCount >= RateLimitPerIP {
		// Log rate limit exceeded but still return nil to prevent enumeration
		fmt.Printf("rate limit exceeded for IP: %s\n", ipAddress)
		return "", nil
	}

	// Generate secure token
	rawToken, err := generateToken()
	if err != nil {
		fmt.Printf("error generating token: %v\n", err)
		return "", nil
	}

	// Hash the token for storage
	hashedToken := hashToken(rawToken)

	// Store token in database
	expiresAt := time.Now().Add(TokenExpiry)
	_, err = s.repo.CreatePasswordResetToken(ctx, repository.CreatePasswordResetTokenParams{
		TenantID:  uuidToPgtype(tenantID),
		UserID:    user.ID,
		TokenHash: hashedToken,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
		IpAddress: pgtype.Text{String: ipAddress, Valid: true},
		UserAgent: pgtype.Text{String: userAgent, Valid: true},
	})
	if err != nil {
		fmt.Printf("error creating password reset token: %v\n", err)
		return "", nil
	}

	// Queue email job
	resetURL := fmt.Sprintf("/reset-password?token=%s", rawToken)
	emailPayload := jobs.PasswordResetPayload{
		Email:     email,
		FirstName: user.FirstName.String,
		ResetURL:  resetURL,
		ExpiresAt: expiresAt,
	}

	err = jobs.EnqueuePasswordResetEmail(ctx, s.repo, tenantID, emailPayload)
	if err != nil {
		fmt.Printf("error queueing password reset email: %v\n", err)
	}

	// Return nil to prevent user enumeration (caller should always show success)
	return "", nil
}

// ValidateResetToken verifies a reset token and returns the associated user ID
func (s *passwordResetService) ValidateResetToken(
	ctx context.Context,
	tenantID uuid.UUID,
	rawToken string,
) (uuid.UUID, error) {
	// Hash the raw token
	hashedToken := hashToken(rawToken)

	// Query database for valid token (checks: not used, not expired)
	tokenRecord, err := s.repo.GetPasswordResetToken(ctx, repository.GetPasswordResetTokenParams{
		TenantID:  uuidToPgtype(tenantID),
		TokenHash: hashedToken,
	})
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}

	// Check user account status
	if tokenRecord.UserStatus == "suspended" || tokenRecord.UserStatus == "closed" {
		return uuid.Nil, ErrInvalidToken
	}

	// Convert pgtype.UUID to uuid.UUID
	userID, err := pgtypeToUUID(tokenRecord.UserID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user ID in token: %w", err)
	}

	return userID, nil
}

// ResetPassword completes the password reset using a valid token
func (s *passwordResetService) ResetPassword(
	ctx context.Context,
	tenantID uuid.UUID,
	rawToken string,
	newPassword string,
) error {
	// Validate token and get user ID
	userID, err := s.ValidateResetToken(ctx, tenantID, rawToken)
	if err != nil {
		return err
	}

	// Hash new password
	passwordHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update user password
	err = s.repo.UpdateUserPassword(ctx, repository.UpdateUserPasswordParams{
		ID:           uuidToPgtype(userID),
		PasswordHash: pgtype.Text{String: passwordHash, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Mark this token as used
	hashedToken := hashToken(rawToken)
	err = s.repo.MarkPasswordResetTokenUsed(ctx, repository.MarkPasswordResetTokenUsedParams{
		TenantID:  uuidToPgtype(tenantID),
		TokenHash: hashedToken,
	})
	if err != nil {
		return fmt.Errorf("failed to mark token as used: %w", err)
	}

	// Invalidate all other tokens for this user
	err = s.repo.InvalidateUserPasswordResetTokens(ctx, repository.InvalidateUserPasswordResetTokensParams{
		TenantID: uuidToPgtype(tenantID),
		UserID:   uuidToPgtype(userID),
	})
	if err != nil {
		return fmt.Errorf("failed to invalidate other tokens: %w", err)
	}

	return nil
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
