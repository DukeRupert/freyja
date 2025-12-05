# Tenant Provider Configuration System

## Overview

This document describes the architecture for allowing tenants (coffee roasters) to choose and configure their preferred external service providers for tax calculation, shipping rates, payment processing, and email delivery.

## Problem Statement

Freyja needs a flexible multi-tenant provider configuration system that:
- Allows each tenant to choose between different providers (free vs. paid integrations)
- Securely stores API credentials (encrypted at rest)
- Validates provider connectivity before enabling
- Resolves the correct provider implementation at runtime

## Provider Options

| Type | Free Options | Paid Integrations |
|------|--------------|-------------------|
| **Tax** | none (0%), percentage (state rates) | stripe_tax (automatic) |
| **Shipping** | flat_rate (tenant-configured) | easypost (real carrier rates) |
| **Billing** | — | stripe (required) |
| **Email** | smtp (tenant provides server) | postmark (dedicated delivery) |

## Architecture Decision

**Chosen Approach:** Database-Backed Provider Registry with Lazy Loading

### Why This Approach

1. **Simplicity:** Map-based cache with RWMutex is easy to understand and debug
2. **Security:** Credentials only decrypted on cache miss (not all loaded at startup)
3. **Performance:** O(1) lookup after cache warmup, 1-hour TTL
4. **Reversibility:** Cache strategy isolated to factory service, easy to change

### Tradeoffs Accepted

- Cache invalidation complexity (solved with explicit invalidation on config save)
- Up to 1-hour stale config window (mitigated by explicit cache invalidation)
- Memory growth with tenant count (~4MB for 100 tenants, negligible)

## Database Schema

### tenant_provider_configs

Stores provider selection and encrypted credentials per tenant.

```sql
CREATE TABLE tenant_provider_configs (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    type VARCHAR(20) NOT NULL,      -- 'tax', 'shipping', 'billing', 'email'
    name VARCHAR(50) NOT NULL,      -- provider name (e.g., 'stripe', 'easypost')
    is_enabled BOOLEAN DEFAULT FALSE,
    config JSONB DEFAULT '{}',      -- non-sensitive settings
    secrets_encrypted TEXT,         -- AES-256-GCM encrypted credentials
    last_tested_at TIMESTAMPTZ,
    test_status VARCHAR(20),        -- 'success', 'failed', 'pending'
    test_message TEXT,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ,
    UNIQUE (tenant_id, type)
);
```

### tenant_shipping_rates

Flat-rate shipping options per tenant (used when shipping provider = 'flat_rate').

```sql
CREATE TABLE tenant_shipping_rates (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    service_name VARCHAR(100) NOT NULL,
    service_code VARCHAR(50) NOT NULL,
    cost_cents INT NOT NULL,
    estimated_days_min INT NOT NULL,
    estimated_days_max INT NOT NULL,
    is_enabled BOOLEAN DEFAULT TRUE,
    sort_order INT DEFAULT 0,
    UNIQUE (tenant_id, service_code)
);
```

## Core Components

### Package Structure

```
internal/
├── provider/
│   ├── types.go       # ProviderType, ProviderName constants
│   ├── config.go      # TenantProviderConfig struct
│   ├── registry.go    # ProviderRegistry interface + DefaultRegistry
│   ├── factory.go     # ProviderFactory interface + DefaultFactory
│   └── validator.go   # ProviderValidator for testing credentials
├── crypto/
│   └── encrypt.go     # AESEncryptor for credential encryption
```

### Key Interfaces

```go
// ProviderRegistry manages provider instances for all tenants
type ProviderRegistry interface {
    GetTaxCalculator(ctx, tenantID) (tax.Calculator, error)
    GetShippingProvider(ctx, tenantID) (shipping.Provider, error)
    GetBillingProvider(ctx, tenantID) (billing.Provider, error)
    GetEmailSender(ctx, tenantID) (email.Sender, error)
    InvalidateCache(tenantID, providerType)
}

// ProviderFactory creates provider instances from configuration
type ProviderFactory interface {
    CreateTaxCalculator(ctx, config, tenantID) (tax.Calculator, error)
    CreateShippingProvider(ctx, config, tenantID) (shipping.Provider, error)
    CreateBillingProvider(ctx, config, tenantID) (billing.Provider, error)
    CreateEmailSender(ctx, config, tenantID) (email.Sender, error)
}

// Encryptor handles encryption/decryption of provider secrets
type Encryptor interface {
    Encrypt(plaintext map[string]string) (string, error)
    Decrypt(ciphertext string) (map[string]string, error)
}
```

## Security

### Encryption

- Algorithm: AES-256-GCM (authenticated encryption)
- Master key: 32-byte key from `ENCRYPTION_MASTER_KEY` environment variable
- Storage: Base64-encoded ciphertext in `secrets_encrypted` column
- Decryption: Only when creating provider instances (lazy loading)

### Access Control

- Only admin users can view/edit provider settings
- Secrets never returned in API responses
- All config changes logged for audit

### Validation Flow

1. User enters API key in admin UI
2. "Test Connection" button calls validation endpoint
3. Backend creates temporary provider, makes test API call
4. Success/failure returned to frontend
5. User clicks "Save" → credentials encrypted and stored
6. Provider cache invalidated

## Admin UI Structure

```
Settings > Integrations
├── Tax Calculation
│   ├── ○ No Tax (0% on all orders)
│   ├── ○ Percentage Rates → [Configure by state]
│   └── ○ Stripe Tax (automatic nexus calculation)
│
├── Shipping Rates
│   ├── ○ Flat Rate → [Configure custom rates]
│   └── ○ EasyPost → [API Key] [Test Connection]
│
├── Payment Processing
│   └── ● Stripe → [API Keys] [Webhook Secret] [Test]
│
└── Email Delivery
    ├── ○ SMTP → [Host, Port, Username, Password]
    └── ○ Postmark → [API Key] [Send Test Email]
```

## Implementation Phases

### Phase 1: Infrastructure
- [ ] Database migration for `tenant_provider_configs` and `tenant_shipping_rates`
- [ ] Encryption package (`internal/crypto/encrypt.go`)
- [ ] Provider package skeleton (`internal/provider/`)
- [ ] sqlc queries for provider configs

### Phase 2: Registry Implementation
- [ ] DefaultRegistry with caching
- [ ] DefaultFactory with all provider types
- [ ] ProviderValidator for each provider type
- [ ] Integration with existing services

### Phase 3: Admin UI
- [ ] Settings handler (`internal/handler/admin/settings.go`)
- [ ] Validation endpoints
- [ ] Settings templates
- [ ] Cache invalidation on save

### Phase 4: Migration
- [ ] Seed default configs for existing tenants
- [ ] Migrate .env credentials to encrypted storage
- [ ] Update service initialization

## Environment Variables

```bash
# Required for credential encryption
ENCRYPTION_MASTER_KEY=your-32-byte-key-here

# Legacy (will be migrated to database)
STRIPE_API_KEY=sk_...
STRIPE_WEBHOOK_SECRET=whsec_...
EASYPOST_API_KEY=EZAK...
POSTMARK_API_KEY=...
```

## Open Questions (Resolved)

1. **Cache warming:** Lazy-load only (better security, acceptable latency)
2. **Provider lifecycle:** One instance per tenant (cached), not per request
3. **Webhook rotation:** Support checking both old and new secrets for 24h
4. **Provider fallback:** No automatic fallback for MVP (manual switch in admin)

## Future Considerations

- Plan-based feature gating (e.g., EasyPost only on Pro plan)
- Automatic provider health monitoring
- Multi-region encryption key synchronization
- Provider usage analytics per tenant
