package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/dukerupert/freyja/internal/domain"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/google/uuid"
)

// Custom domain service errors
var (
	ErrInvalidDomain         = errors.New("invalid domain format")
	ErrApexDomainNotAllowed  = errors.New("apex domains not supported, use a subdomain (e.g., www.example.com or shop.example.com)")
	ErrDomainAlreadyInUse    = errors.New("this domain is already in use by another store")
	ErrDomainNotConfigured   = errors.New("no custom domain configured")
	ErrDomainNotVerified     = errors.New("domain has not been verified yet")
	ErrDomainAlreadyActive   = errors.New("domain is already active")
	ErrCNAMENotConfigured    = errors.New("CNAME record not found or does not point to custom.freyja.app")
	ErrTXTNotConfigured      = errors.New("TXT verification record not found")
	ErrVerificationFailed    = errors.New("domain verification failed")
	ErrDomainHealthCheckFail = errors.New("domain health check failed")
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
	// - Perform CNAME lookup: domain → custom.freyja.app
	// - Perform TXT lookup: _freyja-verify.domain → freyja-verify=<token>
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
	// - If CNAME points to custom.freyja.app: update last_checked_at
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
func (s *customDomainService) InitiateVerification(ctx context.Context, tenantID uuid.UUID, domain string) (*domain.CustomDomain, error) {
	// TODO: Implement InitiateVerification
	//
	// Implementation steps:
	// 1. Validate domain format:
	//    - Call validateDomainFormat(domain) helper
	//    - Check it's not an apex domain (must have subdomain)
	//    - Return ErrInvalidDomain or ErrApexDomainNotAllowed if invalid
	//
	// 2. Generate verification token:
	//    - rawToken, tokenHash := generateVerificationToken()
	//    - rawToken: 64-char hex string (show in UI)
	//    - tokenHash: SHA-256 hash (store in DB)
	//
	// 3. Store in database:
	//    - Call repo.SetCustomDomain(ctx, tenantID, domain, tokenHash)
	//    - Handle UNIQUE constraint violation → ErrDomainAlreadyInUse
	//
	// 4. Build and return CustomDomain struct:
	//    - Status: pending
	//    - VerificationToken: rawToken (only time it's available)
	//    - DNSInstructions: GetDNSInstructions()
	//
	// 5. Log event:
	//    - logger.Info("custom domain initiated", "tenant_id", tenantID, "domain", domain)

	return nil, errors.New("not implemented")
}

// CheckVerification performs DNS verification for a pending domain
func (s *customDomainService) CheckVerification(ctx context.Context, tenantID uuid.UUID) (*domain.DomainVerification, error) {
	// TODO: Implement CheckVerification
	//
	// Implementation steps:
	// 1. Get current domain status:
	//    - customDomain := GetDomainStatus(ctx, tenantID)
	//    - Ensure status is 'pending' or 'failed' (can retry)
	//    - Return ErrDomainNotConfigured if no domain
	//
	// 2. Mark as 'verifying':
	//    - repo.MarkDomainVerifying(ctx, tenantID)
	//    - Prevents concurrent verification attempts
	//
	// 3. Verify CNAME record:
	//    - cnameTarget, err := lookupCNAME(customDomain.Domain, 10*time.Second)
	//    - Check if cnameTarget ends with "custom.freyja.app"
	//    - If not, record error "CNAME not found or incorrect"
	//
	// 4. Verify TXT record:
	//    - txtRecords, err := lookupTXT("_freyja-verify."+domain, 10*time.Second)
	//    - expectedValue := "freyja-verify=" + rawToken (need to retrieve from hash somehow?)
	//    - NOTE: Problem - we store hash, not raw token. Need to rethink this.
	//    - Alternative: Store raw token temporarily in session or require user to copy it
	//    - Check if any txtRecord matches expectedValue
	//
	// 5. Update database based on result:
	//    - If both valid: repo.MarkDomainVerified(ctx, tenantID)
	//    - If invalid: repo.MarkDomainVerificationFailed(ctx, tenantID, errorMsg)
	//
	// 6. Return verification result:
	//    - DomainVerification struct with CNAMEValid, TXTValid, ErrorMessage
	//
	// 7. Log result:
	//    - logger.Info("domain verification completed", "tenant_id", tenantID, "verified", verified)

	return nil, errors.New("not implemented")
}

// ActivateDomain activates a verified custom domain
func (s *customDomainService) ActivateDomain(ctx context.Context, tenantID uuid.UUID) error {
	// TODO: Implement ActivateDomain
	//
	// Implementation steps:
	// 1. Get current domain status:
	//    - customDomain := GetDomainStatus(ctx, tenantID)
	//    - Ensure status is 'verified'
	//    - Return ErrDomainNotVerified if not
	//
	// 2. Activate domain:
	//    - repo.ActivateCustomDomain(ctx, tenantID)
	//    - Sets status = 'active', activated_at = NOW, last_checked_at = NOW
	//
	// 3. Log activation:
	//    - logger.Info("custom domain activated", "tenant_id", tenantID, "domain", customDomain.Domain)
	//
	// 4. Trigger Caddy reload (optional, Caddy polls):
	//    - On-demand TLS will provision certificate on first request
	//    - No manual reload needed, but could signal Caddy for faster provisioning

	return errors.New("not implemented")
}

// RemoveDomain removes a custom domain configuration
func (s *customDomainService) RemoveDomain(ctx context.Context, tenantID uuid.UUID) error {
	// TODO: Implement RemoveDomain
	//
	// Implementation steps:
	// 1. Deactivate domain:
	//    - repo.DeactivateCustomDomain(ctx, tenantID)
	//    - Sets all custom_domain_* columns to NULL, status = 'none'
	//
	// 2. Log removal:
	//    - logger.Info("custom domain removed", "tenant_id", tenantID)
	//
	// 3. Note about certificates:
	//    - Caddy will keep certificate cached for 90 days (Let's Encrypt validity)
	//    - Certificate won't renew, will expire naturally
	//    - No manual cleanup needed

	return errors.New("not implemented")
}

// GetDomainStatus retrieves current custom domain status for a tenant
func (s *customDomainService) GetDomainStatus(ctx context.Context, tenantID uuid.UUID) (*domain.CustomDomain, error) {
	// TODO: Implement GetDomainStatus
	//
	// Implementation steps:
	// 1. Query database:
	//    - status, err := repo.GetCustomDomainStatus(ctx, tenantID)
	//    - Handle sql.ErrNoRows → tenant not found
	//
	// 2. Check if domain is configured:
	//    - If status.custom_domain is NULL or empty, return nil (no domain)
	//    - If status.custom_domain_status == 'none', return nil
	//
	// 3. Build CustomDomain struct:
	//    - Map database columns to domain.CustomDomain
	//    - VerificationToken: empty string (we don't store raw token)
	//    - DNSInstructions: can be computed from domain
	//
	// 4. Return CustomDomain:
	//    - Includes status, timestamps, error message if any

	return nil, errors.New("not implemented")
}

// ValidateDomainForCaddy validates a domain for Caddy's on-demand TLS
func (s *customDomainService) ValidateDomainForCaddy(ctx context.Context, domain string) (bool, error) {
	// TODO: Implement ValidateDomainForCaddy
	//
	// Implementation steps:
	// 1. Query database:
	//    - isValid, err := repo.ValidateDomainForCaddy(ctx, domain)
	//    - Returns boolean from EXISTS query
	//
	// 2. Return result:
	//    - Return (isValid, nil) in all cases
	//    - Caddy expects 200 (true) or 404 (false), no errors
	//
	// 3. Performance note:
	//    - This is CRITICAL PATH for certificate issuance
	//    - Must complete in < 5ms
	//    - Uses idx_tenants_custom_domain_active index
	//
	// 4. Security note:
	//    - Prevents certificate issuance for unauthorized domains
	//    - Only returns true for domains with status = 'active'

	return false, errors.New("not implemented")
}

// PerformHealthCheck checks if an active domain's CNAME is still valid
func (s *customDomainService) PerformHealthCheck(ctx context.Context, tenantID uuid.UUID, domain string) error {
	// TODO: Implement PerformHealthCheck
	//
	// Implementation steps:
	// 1. Lookup CNAME:
	//    - cnameTarget, err := lookupCNAME(domain, 10*time.Second)
	//    - If err != nil, treat as failed check
	//
	// 2. Validate CNAME points to custom.freyja.app:
	//    - isValid := strings.HasSuffix(cnameTarget, "custom.freyja.app")
	//
	// 3. Update database:
	//    - If valid: repo.UpdateCustomDomainHealthCheck(ctx, tenantID, true, nil)
	//    - If invalid: repo.UpdateCustomDomainHealthCheck(ctx, tenantID, false, "CNAME not found...")
	//
	// 4. Send email notification if health check fails:
	//    - if !isValid: emailService.SendCustomDomainFailureEmail(tenant)
	//    - Email template: "Your custom domain configuration has an issue"
	//
	// 5. Log result:
	//    - logger.Info("domain health check", "tenant_id", tenantID, "domain", domain, "valid", isValid)
	//
	// 6. Return error only if database update fails:
	//    - DNS lookup failures are expected (domain might be temporarily down)
	//    - Return nil if health check completes (even if DNS invalid)

	return errors.New("not implemented")
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// validateDomainFormat validates the domain format
func validateDomainFormat(domain string) error {
	// TODO: Implement validateDomainFormat
	//
	// Validation rules:
	// 1. Domain must not be empty
	// 2. Domain must not contain protocol (reject "https://example.com")
	// 3. Domain must not contain path (reject "example.com/path")
	// 4. Domain must be a valid hostname
	// 5. Domain must be a subdomain, not apex (reject "example.com", accept "shop.example.com")
	//
	// Implementation:
	// - Use net.ParseRequestURI to validate format
	// - Check for single dot: if strings.Count(domain, ".") == 1, it's apex
	// - Return ErrApexDomainNotAllowed if apex
	// - Return ErrInvalidDomain if other format issues

	return errors.New("not implemented")
}

// generateVerificationToken generates a cryptographically secure verification token
func generateVerificationToken() (rawToken string, tokenHash string, error error) {
	// TODO: Implement generateVerificationToken
	//
	// Implementation steps:
	// 1. Generate 32 random bytes:
	//    - bytes := make([]byte, domain.CustomDomainVerificationTokenLength)
	//    - _, err := rand.Read(bytes)
	//    - Return error if random generation fails
	//
	// 2. Encode to hex string:
	//    - rawToken := hex.EncodeToString(bytes)
	//    - This is 64 characters (32 bytes * 2 hex chars per byte)
	//
	// 3. Hash token with SHA-256:
	//    - hash := sha256.Sum256([]byte(rawToken))
	//    - tokenHash := hex.EncodeToString(hash[:])
	//
	// 4. Return both:
	//    - rawToken: shown in UI (not stored in DB)
	//    - tokenHash: stored in DB
	//
	// Security rationale:
	// - Raw token never stored in database
	// - Database compromise doesn't expose valid tokens
	// - Same pattern as password reset tokens, email verification tokens

	return "", "", errors.New("not implemented")
}

// lookupCNAME performs a CNAME DNS lookup with timeout
func lookupCNAME(domain string, timeout time.Duration) (string, error) {
	// TODO: Implement lookupCNAME
	//
	// Implementation steps:
	// 1. Create context with timeout:
	//    - ctx, cancel := context.WithTimeout(context.Background(), timeout)
	//    - defer cancel()
	//
	// 2. Perform CNAME lookup:
	//    - resolver := net.DefaultResolver
	//    - cname, err := resolver.LookupCNAME(ctx, domain)
	//    - Return error if lookup fails
	//
	// 3. Normalize result:
	//    - CNAME records end with a dot (e.g., "custom.freyja.app.")
	//    - Strip trailing dot: strings.TrimSuffix(cname, ".")
	//
	// 4. Return normalized CNAME target

	return "", errors.New("not implemented")
}

// lookupTXT performs a TXT DNS lookup with timeout
func lookupTXT(domain string, timeout time.Duration) ([]string, error) {
	// TODO: Implement lookupTXT
	//
	// Implementation steps:
	// 1. Create context with timeout:
	//    - ctx, cancel := context.WithTimeout(context.Background(), timeout)
	//    - defer cancel()
	//
	// 2. Perform TXT lookup:
	//    - resolver := net.DefaultResolver
	//    - records, err := resolver.LookupTXT(ctx, domain)
	//    - Return error if lookup fails
	//
	// 3. Return TXT records:
	//    - Multiple TXT records may exist (e.g., SPF, DKIM)
	//    - Caller will search for the one matching "freyja-verify=..."

	return nil, errors.New("not implemented")
}
