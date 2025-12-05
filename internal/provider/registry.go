package provider

import (
	"context"
	"sync"
	"time"

	"github.com/dukerupert/freyja/internal/billing"
	"github.com/dukerupert/freyja/internal/crypto"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/tax"
	"github.com/jackc/pgx/v5/pgtype"
)

// ProviderRegistry manages provider instances for all tenants.
// It provides caching and lazy loading of provider implementations based on
// tenant-specific configurations stored in the database.
//
// Registry responsibilities:
// - Load provider configs from database
// - Decrypt sensitive configuration data
// - Create provider instances via factory
// - Cache provider instances to avoid repeated database queries
// - Handle cache invalidation when configs change
type ProviderRegistry interface {
	// GetTaxCalculator returns the active tax calculator for the given tenant.
	// Returns cached instance if available and not expired, otherwise loads from database.
	// If multiple tax providers are configured, returns the one marked as default or highest priority.
	GetTaxCalculator(ctx context.Context, tenantID pgtype.UUID) (tax.Calculator, error)

	// GetBillingProvider returns the active billing provider for the given tenant.
	// Returns cached instance if available and not expired, otherwise loads from database.
	GetBillingProvider(ctx context.Context, tenantID pgtype.UUID) (billing.Provider, error)

	// TODO: Add GetShippingProvider(ctx context.Context, tenantID pgtype.UUID) (shipping.Provider, error)
	// TODO: Add GetEmailProvider(ctx context.Context, tenantID pgtype.UUID) (email.Provider, error)

	// InvalidateCache removes cached provider instances for the given tenant.
	// Call this when tenant provider configuration is updated.
	InvalidateCache(tenantID pgtype.UUID, providerType ProviderType)

	// InvalidateAllCache clears all cached provider instances.
	// Useful for testing or when encryption keys are rotated.
	InvalidateAllCache()
}

// DefaultRegistry implements ProviderRegistry with in-memory caching.
type DefaultRegistry struct {
	repo      repository.Querier
	factory   ProviderFactory
	encryptor crypto.Encryptor
	cacheTTL  time.Duration

	// cache stores provider instances keyed by "tenantID:providerType"
	// Values are cacheEntry structs containing the provider and expiration time
	cache sync.Map
}

// cacheKey generates a unique cache key for a tenant and provider type.
type cacheKey struct {
	tenantID     string
	providerType ProviderType
}

// cacheEntry holds a cached provider instance with expiration metadata.
type cacheEntry struct {
	provider  interface{} // Can be tax.Calculator, billing.Provider, etc.
	expiresAt time.Time
}

// NewDefaultRegistry creates a provider registry with caching.
// If cacheTTL is zero or negative, defaults to 1 hour.
func NewDefaultRegistry(
	repo repository.Querier,
	factory ProviderFactory,
	encryptor crypto.Encryptor,
	cacheTTL time.Duration,
) *DefaultRegistry {
	// TODO: Initialize DefaultRegistry with provided dependencies
	// TODO: Validate that repo, factory, and encryptor are not nil
	// TODO: Set default cacheTTL to 1 hour if not provided or <= 0
	// TODO: Initialize cache as sync.Map
	// TODO: Return initialized registry
	return nil
}

// GetTaxCalculator returns the tax calculator for the tenant.
func (r *DefaultRegistry) GetTaxCalculator(ctx context.Context, tenantID pgtype.UUID) (tax.Calculator, error) {
	// TODO: Generate cache key using makeCacheKey(tenantID, ProviderTypeTax)
	// TODO: Check cache using r.cache.Load(key)
	// TODO: If cached and entry.expiresAt is after time.Now(), return cached instance
	// TODO: If cache miss or expired, call r.loadTaxCalculator(ctx, tenantID)
	// TODO: Store newly loaded instance in cache with TTL: r.cache.Store(key, cacheEntry{...})
	// TODO: Return instance
	return nil, nil
}

// GetBillingProvider returns the billing provider for the tenant.
func (r *DefaultRegistry) GetBillingProvider(ctx context.Context, tenantID pgtype.UUID) (billing.Provider, error) {
	// TODO: Generate cache key using makeCacheKey(tenantID, ProviderTypeBilling)
	// TODO: Check cache using r.cache.Load(key)
	// TODO: If cached and entry.expiresAt is after time.Now(), return cached instance
	// TODO: If cache miss or expired, call r.loadBillingProvider(ctx, tenantID)
	// TODO: Store newly loaded instance in cache with TTL: r.cache.Store(key, cacheEntry{...})
	// TODO: Return instance
	return nil, nil
}

// InvalidateCache removes cached provider instances for the given tenant and type.
func (r *DefaultRegistry) InvalidateCache(tenantID pgtype.UUID, providerType ProviderType) {
	// TODO: Generate cache key using makeCacheKey(tenantID, providerType)
	// TODO: Delete from cache using r.cache.Delete(key)
}

// InvalidateAllCache clears all cached provider instances.
func (r *DefaultRegistry) InvalidateAllCache() {
	// TODO: Replace r.cache with a new sync.Map
	// This is simpler than iterating and deleting each key
}

// loadTaxCalculator loads tax calculator configuration from database and creates instance.
func (r *DefaultRegistry) loadTaxCalculator(ctx context.Context, tenantID pgtype.UUID) (tax.Calculator, error) {
	// TODO: Call r.loadConfig(ctx, tenantID, ProviderTypeTax) to get config from database
	// TODO: If config is nil (no active provider configured), return error "no tax provider configured"
	// TODO: Call r.factory.CreateTaxCalculator(config) to create instance
	// TODO: Return instance or error
	return nil, nil
}

// loadBillingProvider loads billing provider configuration from database and creates instance.
func (r *DefaultRegistry) loadBillingProvider(ctx context.Context, tenantID pgtype.UUID) (billing.Provider, error) {
	// TODO: Call r.loadConfig(ctx, tenantID, ProviderTypeBilling) to get config from database
	// TODO: If config is nil (no active provider configured), return error "no billing provider configured"
	// TODO: Call r.factory.CreateBillingProvider(config) to create instance
	// TODO: Return instance or error
	return nil, nil
}

// loadConfig loads and decrypts provider configuration from database.
// Returns the default/highest priority active config for the given tenant and type.
func (r *DefaultRegistry) loadConfig(ctx context.Context, tenantID pgtype.UUID, providerType ProviderType) (*TenantProviderConfig, error) {
	// TODO: Call repository method to get active configs for tenant and type
	//       Example: configs, err := r.repo.GetActiveProviderConfigs(ctx, tenantID, string(providerType))
	// TODO: Handle database errors
	// TODO: If no configs found, return nil, nil (not an error, just no provider configured)
	// TODO: Find the config marked as default (is_default = true)
	// TODO: If no default, find the config with highest priority (lowest priority number)
	// TODO: Decrypt config.ConfigJSON using r.encryptor.Decrypt(config.ConfigJSON)
	// TODO: Unmarshal decrypted JSON into config.Config map[string]interface{}
	// TODO: Return config
	return nil, nil
}

// makeCacheKey creates a cache key from tenant ID and provider type.
func makeCacheKey(tenantID pgtype.UUID, providerType ProviderType) cacheKey {
	// TODO: Convert tenantID to string (handle pgtype.UUID properly)
	// TODO: Return cacheKey{tenantID: tenantIDString, providerType: providerType}
	return cacheKey{}
}
