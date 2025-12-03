package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/dukerupert/freyja/internal/jobs"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	// VerificationTokenLength is the number of random bytes in the verification token (32 bytes = 256 bits)
	VerificationTokenLength = 32

	// VerificationTokenExpiry is how long a verification token is valid (24 hours)
	VerificationTokenExpiry = 24 * time.Hour

	// VerificationRateLimitPerUser is max verification requests per user in the rate limit window
	VerificationRateLimitPerUser = 5

	// VerificationRateLimitPerIP is max verification requests per IP address in the rate limit window
	VerificationRateLimitPerIP = 10

	// VerificationRateLimitWindow is the time window for rate limiting (1 hour)
	VerificationRateLimitWindow = 1 * time.Hour
)

var (
	// ErrVerificationTokenInvalid indicates the verification token is invalid, expired, or already used
	ErrVerificationTokenInvalid = errors.New("invalid or expired verification token")

	// ErrVerificationRateLimitExceeded indicates too many verification requests
	ErrVerificationRateLimitExceeded = errors.New("too many verification requests, please try again later")

	// ErrEmailAlreadyVerified indicates the email is already verified
	ErrEmailAlreadyVerified = errors.New("email is already verified")
)

// EmailVerificationService handles email verification operations
type EmailVerificationService interface {
	// SendVerificationEmail initiates an email verification flow
	// Returns nil on success (or if user doesn't exist, to prevent enumeration)
	SendVerificationEmail(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, email, firstName, ipAddress, userAgent string) error

	// VerifyEmail completes the email verification using a valid token
	VerifyEmail(ctx context.Context, tenantID uuid.UUID, rawToken string) error

	// IsEmailVerified checks if a user's email is verified
	IsEmailVerified(ctx context.Context, userID uuid.UUID) (bool, error)
}

type emailVerificationService struct {
	repo repository.Querier
}

// NewEmailVerificationService creates a new email verification service
func NewEmailVerificationService(repo repository.Querier) EmailVerificationService {
	return &emailVerificationService{
		repo: repo,
	}
}

// SendVerificationEmail initiates an email verification flow
func (s *emailVerificationService) SendVerificationEmail(
	ctx context.Context,
	tenantID uuid.UUID,
	userID uuid.UUID,
	email string,
	firstName string,
	ipAddress string,
	userAgent string,
) error {
	// Check rate limit for this user
	rateLimitCutoff := time.Now().Add(-VerificationRateLimitWindow)
	userCount, err := s.repo.CountRecentVerificationRequestsByUser(ctx, repository.CountRecentVerificationRequestsByUserParams{
		UserID:    uuidToPgtype(userID),
		CreatedAt: pgtype.Timestamptz{Time: rateLimitCutoff, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("error checking user rate limit: %w", err)
	}
	if userCount >= VerificationRateLimitPerUser {
		return ErrVerificationRateLimitExceeded
	}

	// Check rate limit for IP address
	ipCount, err := s.repo.CountRecentVerificationRequestsByIP(ctx, repository.CountRecentVerificationRequestsByIPParams{
		IpAddress: pgtype.Text{String: ipAddress, Valid: true},
		CreatedAt: pgtype.Timestamptz{Time: rateLimitCutoff, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("error checking IP rate limit: %w", err)
	}
	if ipCount >= VerificationRateLimitPerIP {
		return ErrVerificationRateLimitExceeded
	}

	// Generate secure token
	rawToken, err := generateVerificationToken()
	if err != nil {
		return fmt.Errorf("error generating token: %w", err)
	}

	// Hash the token for storage
	hashedToken := hashVerificationToken(rawToken)

	// Store token in database
	expiresAt := time.Now().Add(VerificationTokenExpiry)
	_, err = s.repo.CreateEmailVerificationToken(ctx, repository.CreateEmailVerificationTokenParams{
		TenantID:  uuidToPgtype(tenantID),
		UserID:    uuidToPgtype(userID),
		TokenHash: hashedToken,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
		IpAddress: pgtype.Text{String: ipAddress, Valid: true},
		UserAgent: pgtype.Text{String: userAgent, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("error creating email verification token: %w", err)
	}

	// Queue email job
	verifyURL := fmt.Sprintf("/verify-email?token=%s", rawToken)
	emailPayload := jobs.EmailVerificationPayload{
		Email:     email,
		FirstName: firstName,
		VerifyURL: verifyURL,
		ExpiresAt: expiresAt,
	}

	err = jobs.EnqueueEmailVerification(ctx, s.repo, tenantID, emailPayload)
	if err != nil {
		return fmt.Errorf("error queueing verification email: %w", err)
	}

	return nil
}

// VerifyEmail completes the email verification using a valid token
func (s *emailVerificationService) VerifyEmail(
	ctx context.Context,
	tenantID uuid.UUID,
	rawToken string,
) error {
	// Hash the raw token
	hashedToken := hashVerificationToken(rawToken)

	// Query database for valid token (checks: not used, not expired)
	tokenRecord, err := s.repo.GetEmailVerificationToken(ctx, repository.GetEmailVerificationTokenParams{
		TenantID:  uuidToPgtype(tenantID),
		TokenHash: hashedToken,
	})
	if err != nil {
		return ErrVerificationTokenInvalid
	}

	// Check user account status
	if tokenRecord.UserStatus == "suspended" || tokenRecord.UserStatus == "closed" {
		return ErrVerificationTokenInvalid
	}

	// Check if already verified
	if tokenRecord.UserEmailVerified {
		return ErrEmailAlreadyVerified
	}

	// Convert pgtype.UUID to uuid.UUID
	userID, err := pgtypeToUUID(tokenRecord.UserID)
	if err != nil {
		return fmt.Errorf("invalid user ID in token: %w", err)
	}

	// Mark email as verified
	err = s.repo.VerifyUserEmail(ctx, uuidToPgtype(userID))
	if err != nil {
		return fmt.Errorf("failed to verify email: %w", err)
	}

	// Mark this token as used
	err = s.repo.MarkEmailVerificationTokenUsed(ctx, repository.MarkEmailVerificationTokenUsedParams{
		TenantID:  uuidToPgtype(tenantID),
		TokenHash: hashedToken,
	})
	if err != nil {
		return fmt.Errorf("failed to mark token as used: %w", err)
	}

	// Invalidate all other tokens for this user
	err = s.repo.InvalidateUserEmailVerificationTokens(ctx, repository.InvalidateUserEmailVerificationTokensParams{
		TenantID: uuidToPgtype(tenantID),
		UserID:   uuidToPgtype(userID),
	})
	if err != nil {
		return fmt.Errorf("failed to invalidate other tokens: %w", err)
	}

	return nil
}

// IsEmailVerified checks if a user's email is verified
func (s *emailVerificationService) IsEmailVerified(ctx context.Context, userID uuid.UUID) (bool, error) {
	user, err := s.repo.GetUserByID(ctx, uuidToPgtype(userID))
	if err != nil {
		return false, fmt.Errorf("failed to get user: %w", err)
	}
	return user.EmailVerified, nil
}

// generateVerificationToken creates a cryptographically secure random token
func generateVerificationToken() (string, error) {
	b := make([]byte, VerificationTokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// hashVerificationToken creates a SHA-256 hash of a token for storage
func hashVerificationToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
