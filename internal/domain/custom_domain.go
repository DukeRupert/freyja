package domain

import (
	"time"

	"github.com/google/uuid"
)

// CustomDomainStatus represents the state of a tenant's custom domain
type CustomDomainStatus string

const (
	// CustomDomainStatusNone indicates no custom domain is configured
	CustomDomainStatusNone CustomDomainStatus = "none"

	// CustomDomainStatusPending indicates domain has been entered, waiting for DNS records
	CustomDomainStatusPending CustomDomainStatus = "pending"

	// CustomDomainStatusVerifying indicates DNS verification is in progress
	CustomDomainStatusVerifying CustomDomainStatus = "verifying"

	// CustomDomainStatusVerified indicates DNS verification succeeded, ready to activate
	CustomDomainStatusVerified CustomDomainStatus = "verified"

	// CustomDomainStatusActive indicates domain is live and serving traffic
	CustomDomainStatusActive CustomDomainStatus = "active"

	// CustomDomainStatusFailed indicates verification or health check failed
	CustomDomainStatusFailed CustomDomainStatus = "failed"
)

// DomainVerification represents the DNS verification result
type DomainVerification struct {
	// Domain being verified
	Domain string

	// Whether CNAME record points to custom.freyja.app
	CNAMEValid bool
	CNAMETarget string

	// Whether TXT record contains the verification token
	TXTValid bool
	TXTValue string

	// Overall verification status
	Verified bool

	// Error message if verification failed
	ErrorMessage string

	// Timestamp of verification attempt
	VerifiedAt time.Time
}

// CustomDomain represents a tenant's custom domain configuration
// This is a view model combining data from tenants table + computed DNS status
type CustomDomain struct {
	TenantID       uuid.UUID
	Domain         string
	Status         CustomDomainStatus
	VerifiedAt     *time.Time
	ActivatedAt    *time.Time
	LastCheckedAt  *time.Time
	ErrorMessage   *string

	// Computed fields (not stored in DB)
	VerificationToken string // Raw token (only available during setup)
	DNSInstructions   DNSInstructions
}

// DNSInstructions provides the DNS records a tenant needs to add
type DNSInstructions struct {
	// CNAME record for traffic routing
	CNAME DNSRecord

	// TXT record for verification
	TXT DNSRecord
}

// DNSRecord represents a single DNS record instruction
type DNSRecord struct {
	Type  string // "CNAME" or "TXT"
	Name  string // e.g., "shop.example.com" or "_freyja-verify.shop.example.com"
	Value string // e.g., "custom.freyja.app" or "freyja-verify=abc123..."
	TTL   int    // Recommended TTL in seconds (3600 = 1 hour)
}

// CustomDomainConstants defines configuration constants
const (
	// CustomDomainCNAMETarget is the target for CNAME records
	CustomDomainCNAMETarget = "custom.freyja.app"

	// CustomDomainTXTPrefix is the subdomain prefix for TXT verification records
	CustomDomainTXTPrefix = "_freyja-verify"

	// CustomDomainTXTValuePrefix is the prefix for TXT record values
	CustomDomainTXTValuePrefix = "freyja-verify="

	// CustomDomainVerificationTokenLength is the number of random bytes for tokens
	CustomDomainVerificationTokenLength = 32 // 64 hex chars

	// CustomDomainDNSTimeout is the timeout for DNS lookups
	CustomDomainDNSTimeout = 10 * time.Second

	// CustomDomainHealthCheckInterval is how often to check active domains
	CustomDomainHealthCheckInterval = 24 * time.Hour

	// CustomDomainPendingExpiryDays is how long pending domains remain before cleanup
	CustomDomainPendingExpiryDays = 30
)

// IsActive returns true if the custom domain is active and serving traffic
func (cd CustomDomain) IsActive() bool {
	return cd.Status == CustomDomainStatusActive
}

// IsVerified returns true if the domain has been verified (but not necessarily active)
func (cd CustomDomain) IsVerified() bool {
	return cd.Status == CustomDomainStatusVerified || cd.Status == CustomDomainStatusActive
}

// IsPending returns true if the domain is waiting for DNS configuration
func (cd CustomDomain) IsPending() bool {
	return cd.Status == CustomDomainStatusPending
}

// IsFailed returns true if verification or health check failed
func (cd CustomDomain) IsFailed() bool {
	return cd.Status == CustomDomainStatusFailed
}

// CanActivate returns true if the domain can be activated
func (cd CustomDomain) CanActivate() bool {
	return cd.Status == CustomDomainStatusVerified
}

// CanVerify returns true if the domain can be verified
func (cd CustomDomain) CanVerify() bool {
	return cd.Status == CustomDomainStatusPending || cd.Status == CustomDomainStatusFailed
}

// GetCNAMERecord returns the CNAME DNS record instruction
func (cd CustomDomain) GetCNAMERecord() DNSRecord {
	return DNSRecord{
		Type:  "CNAME",
		Name:  cd.Domain,
		Value: CustomDomainCNAMETarget,
		TTL:   3600,
	}
}

// GetTXTRecord returns the TXT DNS record instruction
func (cd CustomDomain) GetTXTRecord() DNSRecord {
	tokenValue := cd.VerificationToken
	if len(tokenValue) > 64 {
		tokenValue = tokenValue[:64]
	}
	return DNSRecord{
		Type:  "TXT",
		Name:  CustomDomainTXTPrefix + "." + cd.Domain,
		Value: CustomDomainTXTValuePrefix + tokenValue,
		TTL:   3600,
	}
}

// GetDNSInstructions returns both CNAME and TXT record instructions
func (cd CustomDomain) GetDNSInstructions() DNSInstructions {
	return DNSInstructions{
		CNAME: cd.GetCNAMERecord(),
		TXT:   cd.GetTXTRecord(),
	}
}
