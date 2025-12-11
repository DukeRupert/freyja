package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/dukerupert/hiri/internal/domain"
	"github.com/dukerupert/hiri/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Custom domain service errors
var (
	ErrInvalidDomain         = domain.Errorf(domain.EINVALID, "", "Invalid domain format")
	ErrApexDomainNotAllowed  = domain.Errorf(domain.EINVALID, "", "Apex domains not supported, use a subdomain (e.g., www.example.com or shop.example.com)")
	ErrDomainAlreadyInUse    = domain.Errorf(domain.ECONFLICT, "", "This domain is already in use by another store")
	ErrDomainNotConfigured   = domain.Errorf(domain.ENOTFOUND, "", "No custom domain configured")
	ErrDomainNotVerified     = domain.Errorf(domain.EINVALID, "", "Domain has not been verified yet")
	ErrDomainAlreadyActive   = domain.Errorf(domain.ECONFLICT, "", "Domain is already active")
	ErrCNAMENotConfigured    = domain.Errorf(domain.EINVALID, "", "CNAME record not found or does not point to custom.hiri.coffee")
	ErrTXTNotConfigured      = domain.Errorf(domain.EINVALID, "", "TXT verification record not found")
	ErrVerificationFailed    = domain.Errorf(domain.EINVALID, "", "Domain verification failed")
	ErrDomainHealthCheckFail = domain.Errorf(domain.EINVALID, "", "Domain health check failed")
)

// CustomDomainService provides business logic for custom domain management
type CustomDomainService interface {
	// InitiateVerification starts the custom domain setup process
	// Validates domain format, generates verification token, stores in database
	// Returns the raw verification token (to show in UI) and instructions
	//
	// Business logic:
	// - Validate domain is not an apex domain (e.g., reject "example.com", accept "shop.example.com")
	// - Validate domain format using net.ParseRequestURI
	// - Check domain is not already in use by another tenant (UNIQUE constraint)
	// - Generate cryptographically secure 32-byte verification token
	// - Store SHA-256 hash of token in database (not plaintext)
	// - Set domain status to 'pending'
	// - Return raw token + DNS instructions for UI display
	InitiateVerification(ctx context.Context, tenantID uuid.UUID, domain string) (*domain.CustomDomain, error)

	// CheckVerification performs DNS verification for a pending domain
	// Looks up CNAME and TXT records, validates they match expectations
	// If valid, marks domain as 'verified'
	//
	// Business logic:
	// - Ensure domain is in 'pending' or 'failed' status (can retry failed)
	// - Mark status as 'verifying' to prevent concurrent checks
	// - Perform CNAME lookup: domain → custom.hiri.coffee
	// - Perform TXT lookup: _hiri-verify.domain → hiri-verify=<token>
	// - If both valid: mark as 'verified', set verified_at timestamp
	// - If invalid: mark as 'failed', store error message
	// - Use 10-second timeout on DNS lookups (prevent hanging)
	// - Return verification result with details for UI display
	CheckVerification(ctx context.Context, tenantID uuid.UUID) (*domain.DomainVerification, error)

	// ActivateDomain activates a verified custom domain
	// After this, Caddy will provision SSL certificate and serve traffic
	//
	// Business logic:
	// - Ensure domain is in 'verified' status
	// - Set status to 'active'
	// - Set activated_at timestamp
	// - Set last_checked_at to NOW (health monitoring starts)
	// - Storefront routes will now redirect from subdomain to custom domain
	ActivateDomain(ctx context.Context, tenantID uuid.UUID) error

	// RemoveDomain removes a custom domain configuration
	// Can be called from any status (pending, verified, active, failed)
	//
	// Business logic:
	// - Set all custom_domain_* columns to NULL
	// - Set status to 'none'
	// - Storefront reverts to subdomain routing
	// - Caddy will stop serving on custom domain (certificate remains cached for 90 days)
	RemoveDomain(ctx context.Context, tenantID uuid.UUID) error

	// GetDomainStatus retrieves current custom domain status for a tenant
	// Returns domain.CustomDomain with all status fields
	//
	// Business logic:
	// - Query tenants table for custom_domain_* columns
	// - If status is 'none' or domain is NULL, return nil (no domain configured)
	// - Otherwise, populate CustomDomain struct with database values
	// - DO NOT include raw verification token (only hash is stored)
	GetDomainStatus(ctx context.Context, tenantID uuid.UUID) (*domain.CustomDomain, error)

	// ValidateDomainForCaddy validates a domain for Caddy's on-demand TLS
	// Called by Caddy's 'ask' endpoint before issuing a certificate
	// Returns true only if domain is active and belongs to a tenant
	//
	// Business logic:
	// - Query: SELECT EXISTS(...) WHERE custom_domain = $1 AND status = 'active'
	// - Return true/false (no error - Caddy expects 200 or 404)
	// - Must be fast (< 5ms) as this is in certificate issuance path
	ValidateDomainForCaddy(ctx context.Context, domain string) (bool, error)

	// PerformHealthCheck checks if an active domain's CNAME is still valid
	// Called by daily background job for all active domains
	//
	// Business logic:
	// - Lookup CNAME for domain
	// - If CNAME points to custom.hiri.coffee: update last_checked_at
	// - If CNAME missing/invalid: mark as 'failed', set error message
	// - Send email notification to tenant if domain fails health check
	// - Return error only if database update fails (not if DNS fails)
	PerformHealthCheck(ctx context.Context, tenantID uuid.UUID, domain string) error
}

type customDomainService struct {
	repo   repository.Querier
	logger *slog.Logger
}

// NewCustomDomainService creates a new CustomDomainService instance
func NewCustomDomainService(repo repository.Querier, logger *slog.Logger) CustomDomainService {
	if logger == nil {
		logger = slog.Default()
	}
	return &customDomainService{
		repo:   repo,
		logger: logger,
	}
}

// InitiateVerification starts the custom domain setup process
func (s *customDomainService) InitiateVerification(ctx context.Context, tenantID uuid.UUID, domainName string) (*domain.CustomDomain, error) {
	domainName = strings.TrimSpace(strings.ToLower(domainName))

	if err := validateDomainFormat(domainName); err != nil {
		return nil, err
	}

	_, tokenHash, err := generateDomainVerificationToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate verification token: %w", err)
	}

	var tenantPgUUID pgtype.UUID
	if err := tenantPgUUID.Scan(tenantID.String()); err != nil {
		return nil, fmt.Errorf("failed to convert tenant ID: %w", err)
	}

	var domainText pgtype.Text
	domainText.String = domainName
	domainText.Valid = true

	var tokenText pgtype.Text
	tokenText.String = tokenHash
	tokenText.Valid = true

	err = s.repo.SetCustomDomain(ctx, repository.SetCustomDomainParams{
		ID:                            tenantPgUUID,
		CustomDomain:                  domainText,
		CustomDomainVerificationToken: tokenText,
	})
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return nil, ErrDomainAlreadyInUse
		}
		return nil, fmt.Errorf("failed to set custom domain: %w", err)
	}

	s.logger.Info("custom domain initiated", "tenant_id", tenantID, "domain", domainName)

	customDomain := &domain.CustomDomain{
		TenantID:          tenantID,
		Domain:            domainName,
		Status:            domain.CustomDomainStatusPending,
		VerificationToken: tokenHash,
	}
	customDomain.DNSInstructions = customDomain.GetDNSInstructions()

	return customDomain, nil
}

// CheckVerification performs DNS verification for a pending domain
func (s *customDomainService) CheckVerification(ctx context.Context, tenantID uuid.UUID) (*domain.DomainVerification, error) {
	customDomain, err := s.GetDomainStatus(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if customDomain == nil {
		return nil, ErrDomainNotConfigured
	}
	if !customDomain.CanVerify() {
		return nil, fmt.Errorf("domain cannot be verified in current status: %s", customDomain.Status)
	}

	tenantPgUUID, err := convertToPgUUID(tenantID)
	if err != nil {
		return nil, err
	}

	err = s.repo.MarkDomainVerifying(ctx, tenantPgUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to mark domain as verifying: %w", err)
	}

	verification := &domain.DomainVerification{
		Domain:     customDomain.Domain,
		VerifiedAt: time.Now(),
	}

	cnameTarget, err := lookupCNAME(customDomain.Domain, domain.CustomDomainDNSTimeout)
	if err != nil {
		verification.ErrorMessage = fmt.Sprintf("CNAME lookup failed: %v", err)
	} else {
		verification.CNAMETarget = cnameTarget
		verification.CNAMEValid = strings.HasSuffix(cnameTarget, domain.CustomDomainCNAMETarget)
	}

	status, err := s.repo.GetCustomDomainStatus(ctx, tenantPgUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get custom domain status: %w", err)
	}

	txtRecords, err := lookupTXT(domain.CustomDomainTXTPrefix+"."+customDomain.Domain, domain.CustomDomainDNSTimeout)
	if err != nil {
		if verification.ErrorMessage != "" {
			verification.ErrorMessage += "; "
		}
		verification.ErrorMessage += fmt.Sprintf("TXT lookup failed: %v", err)
	} else {
		var tokenHash string
		if status.CustomDomainVerificationToken.Valid {
			tokenHash = status.CustomDomainVerificationToken.String
		}
		if tokenHash != "" {
			hashLen := len(tokenHash)
			prefixLen := 64
			if hashLen < prefixLen {
				prefixLen = hashLen
			}
			expectedPrefix := domain.CustomDomainTXTValuePrefix + tokenHash[:prefixLen]
			for _, record := range txtRecords {
				if strings.HasPrefix(record, expectedPrefix) {
					verification.TXTValid = true
					verification.TXTValue = record
					break
				}
			}
		}
	}

	verification.Verified = verification.CNAMEValid && verification.TXTValid

	if verification.Verified {
		err = s.repo.MarkDomainVerified(ctx, tenantPgUUID)
		if err != nil {
			return nil, fmt.Errorf("failed to mark domain as verified: %w", err)
		}
		s.logger.Info("domain verification succeeded", "tenant_id", tenantID, "domain", customDomain.Domain)
	} else {
		if verification.ErrorMessage == "" {
			if !verification.CNAMEValid {
				verification.ErrorMessage = "CNAME record not found or does not point to custom.hiri.coffee"
			}
			if !verification.TXTValid {
				if verification.ErrorMessage != "" {
					verification.ErrorMessage += "; "
				}
				verification.ErrorMessage += "TXT verification record not found"
			}
		}
		var errorMsg pgtype.Text
		errorMsg.String = verification.ErrorMessage
		errorMsg.Valid = true

		err = s.repo.MarkDomainVerificationFailed(ctx, repository.MarkDomainVerificationFailedParams{
			ID:                         tenantPgUUID,
			CustomDomainErrorMessage:   errorMsg,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to mark verification as failed: %w", err)
		}
		s.logger.Info("domain verification failed", "tenant_id", tenantID, "domain", customDomain.Domain, "error", verification.ErrorMessage)
	}

	return verification, nil
}

// ActivateDomain activates a verified custom domain
func (s *customDomainService) ActivateDomain(ctx context.Context, tenantID uuid.UUID) error {
	customDomain, err := s.GetDomainStatus(ctx, tenantID)
	if err != nil {
		return err
	}
	if customDomain == nil {
		return ErrDomainNotConfigured
	}
	if !customDomain.CanActivate() {
		if customDomain.Status == domain.CustomDomainStatusActive {
			return ErrDomainAlreadyActive
		}
		return ErrDomainNotVerified
	}

	tenantPgUUID, err := convertToPgUUID(tenantID)
	if err != nil {
		return err
	}

	err = s.repo.ActivateCustomDomain(ctx, tenantPgUUID)
	if err != nil {
		return fmt.Errorf("failed to activate custom domain: %w", err)
	}

	s.logger.Info("custom domain activated", "tenant_id", tenantID, "domain", customDomain.Domain)

	return nil
}

// RemoveDomain removes a custom domain configuration
func (s *customDomainService) RemoveDomain(ctx context.Context, tenantID uuid.UUID) error {
	tenantPgUUID, err := convertToPgUUID(tenantID)
	if err != nil {
		return err
	}

	err = s.repo.DeactivateCustomDomain(ctx, tenantPgUUID)
	if err != nil {
		return fmt.Errorf("failed to deactivate custom domain: %w", err)
	}

	s.logger.Info("custom domain removed", "tenant_id", tenantID)

	return nil
}

// GetDomainStatus retrieves current custom domain status for a tenant
func (s *customDomainService) GetDomainStatus(ctx context.Context, tenantID uuid.UUID) (*domain.CustomDomain, error) {
	tenantPgUUID, err := convertToPgUUID(tenantID)
	if err != nil {
		return nil, err
	}

	status, err := s.repo.GetCustomDomainStatus(ctx, tenantPgUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get custom domain status: %w", err)
	}

	if !status.CustomDomain.Valid || status.CustomDomain.String == "" || status.CustomDomainStatus == "none" {
		return nil, nil
	}

	customDomain := &domain.CustomDomain{
		TenantID: tenantID,
		Domain:   status.CustomDomain.String,
		Status:   domain.CustomDomainStatus(status.CustomDomainStatus),
	}

	if status.CustomDomainVerificationToken.Valid {
		customDomain.VerificationToken = status.CustomDomainVerificationToken.String
	}

	if status.CustomDomainVerifiedAt.Valid {
		t := status.CustomDomainVerifiedAt.Time
		customDomain.VerifiedAt = &t
	}

	if status.CustomDomainActivatedAt.Valid {
		t := status.CustomDomainActivatedAt.Time
		customDomain.ActivatedAt = &t
	}

	if status.CustomDomainLastCheckedAt.Valid {
		t := status.CustomDomainLastCheckedAt.Time
		customDomain.LastCheckedAt = &t
	}

	if status.CustomDomainErrorMessage.Valid && status.CustomDomainErrorMessage.String != "" {
		msg := status.CustomDomainErrorMessage.String
		customDomain.ErrorMessage = &msg
	}

	customDomain.DNSInstructions = customDomain.GetDNSInstructions()

	return customDomain, nil
}

// ValidateDomainForCaddy validates a domain for Caddy's on-demand TLS
func (s *customDomainService) ValidateDomainForCaddy(ctx context.Context, domainName string) (bool, error) {
	var domainText pgtype.Text
	domainText.String = domainName
	domainText.Valid = true

	result, err := s.repo.ValidateDomainForCaddy(ctx, domainText)
	if err != nil {
		return false, nil
	}

	return result, nil
}

// PerformHealthCheck checks if an active domain's CNAME is still valid
func (s *customDomainService) PerformHealthCheck(ctx context.Context, tenantID uuid.UUID, domainName string) error {
	cnameTarget, err := lookupCNAME(domainName, domain.CustomDomainDNSTimeout)

	isHealthy := false
	var errorMessage string

	if err != nil {
		errorMessage = fmt.Sprintf("CNAME lookup failed: %v", err)
	} else {
		isHealthy = strings.HasSuffix(cnameTarget, domain.CustomDomainCNAMETarget)
		if !isHealthy {
			errorMessage = fmt.Sprintf("CNAME record does not point to %s (found: %s)", domain.CustomDomainCNAMETarget, cnameTarget)
		}
	}

	tenantPgUUID, err := convertToPgUUID(tenantID)
	if err != nil {
		return err
	}

	var errorMsg pgtype.Text
	errorMsg.String = errorMessage
	errorMsg.Valid = errorMessage != ""

	err = s.repo.UpdateCustomDomainHealthCheck(ctx, repository.UpdateCustomDomainHealthCheckParams{
		ID:                       tenantPgUUID,
		Column2:                  isHealthy,
		CustomDomainErrorMessage: errorMsg,
	})
	if err != nil {
		return fmt.Errorf("failed to update health check status: %w", err)
	}

	s.logger.Info("domain health check completed", "tenant_id", tenantID, "domain", domainName, "healthy", isHealthy)

	return nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// convertToPgUUID converts a uuid.UUID to pgtype.UUID
func convertToPgUUID(id uuid.UUID) (pgtype.UUID, error) {
	var pgUUID pgtype.UUID
	if err := pgUUID.Scan(id.String()); err != nil {
		return pgUUID, fmt.Errorf("failed to convert UUID: %w", err)
	}
	return pgUUID, nil
}

// validateDomainFormat validates the domain format
func validateDomainFormat(domainName string) error {
	if domainName == "" {
		return ErrInvalidDomain
	}

	if strings.Contains(domainName, "://") {
		return ErrInvalidDomain
	}

	if strings.Contains(domainName, "/") {
		return ErrInvalidDomain
	}

	if strings.Count(domainName, ".") < 1 {
		return ErrInvalidDomain
	}

	if strings.Count(domainName, ".") == 1 {
		return ErrApexDomainNotAllowed
	}

	return nil
}

// generateDomainVerificationToken generates a cryptographically secure verification token
func generateDomainVerificationToken() (rawToken string, tokenHash string, err error) {
	bytes := make([]byte, domain.CustomDomainVerificationTokenLength)
	_, err = rand.Read(bytes)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	rawToken = hex.EncodeToString(bytes)

	hash := sha256.Sum256([]byte(rawToken))
	tokenHash = hex.EncodeToString(hash[:])

	return rawToken, tokenHash, nil
}

// lookupCNAME performs a CNAME DNS lookup with timeout
func lookupCNAME(domainName string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resolver := net.DefaultResolver
	cname, err := resolver.LookupCNAME(ctx, domainName)
	if err != nil {
		return "", err
	}

	cname = strings.TrimSuffix(cname, ".")

	return cname, nil
}

// lookupTXT performs a TXT DNS lookup with timeout
func lookupTXT(domainName string, timeout time.Duration) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resolver := net.DefaultResolver
	records, err := resolver.LookupTXT(ctx, domainName)
	if err != nil {
		return nil, err
	}

	return records, nil
}
