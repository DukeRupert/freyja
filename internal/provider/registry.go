package provider

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dukerupert/freyja/internal/billing"
	"github.com/dukerupert/freyja/internal/crypto"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/tax"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/sync/singleflight"
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

	// loadGroup ensures only one goroutine loads a provider for a given cache key at a time
	// This prevents duplicate provider instantiation during concurrent requests
	loadGroup singleflight.Group
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
	if repo == nil {
		panic("repo cannot be nil")
	}
	if factory == nil {
		panic("factory cannot be nil")
	}
	if encryptor == nil {
		panic("encryptor cannot be nil")
	}

	if cacheTTL <= 0 {
		cacheTTL = 1 * time.Hour
	}

	return &DefaultRegistry{
		repo:      repo,
		factory:   factory,
		encryptor: encryptor,
		cacheTTL:  cacheTTL,
		cache:     sync.Map{},
	}
}

// GetTaxCalculator returns the tax calculator for the tenant.
func (r *DefaultRegistry) GetTaxCalculator(ctx context.Context, tenantID pgtype.UUID) (tax.Calculator, error) {
	key := makeCacheKey(tenantID, ProviderTypeTax)
	keyString := fmt.Sprintf("%s:%s", key.tenantID, key.providerType)

	// Check cache first
	if cached, ok := r.cache.Load(key); ok {
		entry := cached.(cacheEntry)
		if entry.expiresAt.After(time.Now()) {
			return entry.provider.(tax.Calculator), nil
		}
	}

	// Use singleflight to ensure only one goroutine loads the provider for this key
	result, err, _ := r.loadGroup.Do(keyString, func() (interface{}, error) {
		// Double-check cache inside singleflight to handle race between cache check and Do
		if cached, ok := r.cache.Load(key); ok {
			entry := cached.(cacheEntry)
			if entry.expiresAt.After(time.Now()) {
				return entry.provider.(tax.Calculator), nil
			}
		}

		calculator, err := r.loadTaxCalculator(ctx, tenantID)
		if err != nil {
			return nil, err
		}

		r.cache.Store(key, cacheEntry{
			provider:  calculator,
			expiresAt: time.Now().Add(r.cacheTTL),
		})

		return calculator, nil
	})

	if err != nil {
		return nil, err
	}

	return result.(tax.Calculator), nil
}

// GetBillingProvider returns the billing provider for the tenant.
func (r *DefaultRegistry) GetBillingProvider(ctx context.Context, tenantID pgtype.UUID) (billing.Provider, error) {
	key := makeCacheKey(tenantID, ProviderTypeBilling)
	keyString := fmt.Sprintf("%s:%s", key.tenantID, key.providerType)

	// Check cache first
	if cached, ok := r.cache.Load(key); ok {
		entry := cached.(cacheEntry)
		if entry.expiresAt.After(time.Now()) {
			return entry.provider.(billing.Provider), nil
		}
	}

	// Use singleflight to ensure only one goroutine loads the provider for this key
	result, err, _ := r.loadGroup.Do(keyString, func() (interface{}, error) {
		// Double-check cache inside singleflight to handle race between cache check and Do
		if cached, ok := r.cache.Load(key); ok {
			entry := cached.(cacheEntry)
			if entry.expiresAt.After(time.Now()) {
				return entry.provider.(billing.Provider), nil
			}
		}

		provider, err := r.loadBillingProvider(ctx, tenantID)
		if err != nil {
			return nil, err
		}

		r.cache.Store(key, cacheEntry{
			provider:  provider,
			expiresAt: time.Now().Add(r.cacheTTL),
		})

		return provider, nil
	})

	if err != nil {
		return nil, err
	}

	return result.(billing.Provider), nil
}

// InvalidateCache removes cached provider instances for the given tenant and type.
func (r *DefaultRegistry) InvalidateCache(tenantID pgtype.UUID, providerType ProviderType) {
	key := makeCacheKey(tenantID, providerType)
	r.cache.Delete(key)
}

// InvalidateAllCache clears all cached provider instances.
func (r *DefaultRegistry) InvalidateAllCache() {
	r.cache = sync.Map{}
}

// loadTaxCalculator loads tax calculator configuration from database and creates instance.
func (r *DefaultRegistry) loadTaxCalculator(ctx context.Context, tenantID pgtype.UUID) (tax.Calculator, error) {
	config, err := r.loadConfig(ctx, tenantID, ProviderTypeTax)
	if err != nil {
		return nil, fmt.Errorf("failed to load tax provider config: %w", err)
	}

	if config == nil {
		return nil, fmt.Errorf("no tax provider configured for tenant")
	}

	calculator, err := r.factory.CreateTaxCalculator(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create tax calculator: %w", err)
	}

	return calculator, nil
}

// loadBillingProvider loads billing provider configuration from database and creates instance.
func (r *DefaultRegistry) loadBillingProvider(ctx context.Context, tenantID pgtype.UUID) (billing.Provider, error) {
	config, err := r.loadConfig(ctx, tenantID, ProviderTypeBilling)
	if err != nil {
		return nil, fmt.Errorf("failed to load billing provider config: %w", err)
	}

	if config == nil {
		return nil, fmt.Errorf("no billing provider configured for tenant")
	}

	provider, err := r.factory.CreateBillingProvider(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create billing provider: %w", err)
	}

	return provider, nil
}

// loadConfig loads and decrypts provider configuration from database.
// Returns the default/highest priority active config for the given tenant and type.
func (r *DefaultRegistry) loadConfig(ctx context.Context, tenantID pgtype.UUID, providerType ProviderType) (*TenantProviderConfig, error) {
	// NOTE: Repository query method for provider configs does not exist yet.
	// This is a stub implementation that should be replaced once the database
	// queries are implemented via sqlc.
	//
	// Expected query signature:
	// GetActiveProviderConfigs(ctx context.Context, tenantID pgtype.UUID, providerType string) ([]ProviderConfig, error)
	//
	// For now, return nil to indicate no provider is configured.
	// When the database layer is ready, uncomment and implement the following:
	//
	// configs, err := r.repo.GetActiveProviderConfigs(ctx, tenantID, string(providerType))
	// if err != nil {
	//     return nil, fmt.Errorf("failed to query provider configs: %w", err)
	// }
	//
	// if len(configs) == 0 {
	//     return nil, nil
	// }
	//
	// var selectedConfig *repository.ProviderConfig
	// for i := range configs {
	//     if configs[i].IsDefault {
	//         selectedConfig = &configs[i]
	//         break
	//     }
	// }
	//
	// if selectedConfig == nil {
	//     selectedConfig = &configs[0]
	//     for i := range configs {
	//         if configs[i].Priority < selectedConfig.Priority {
	//             selectedConfig = &configs[i]
	//         }
	//     }
	// }
	//
	// decrypted, err := r.encryptor.Decrypt(selectedConfig.ConfigJSON)
	// if err != nil {
	//     return nil, fmt.Errorf("failed to decrypt config: %w", err)
	// }
	//
	// var configMap map[string]interface{}
	// if err := encoding/json.Unmarshal(decrypted, &configMap); err != nil {
	//     return nil, fmt.Errorf("failed to unmarshal config JSON: %w", err)
	// }
	//
	// return &TenantProviderConfig{
	//     ID:         selectedConfig.ID,
	//     TenantID:   selectedConfig.TenantID,
	//     Type:       ProviderType(selectedConfig.Type),
	//     Name:       ProviderName(selectedConfig.Name),
	//     IsActive:   selectedConfig.IsActive,
	//     IsDefault:  selectedConfig.IsDefault,
	//     Priority:   selectedConfig.Priority,
	//     Config:     configMap,
	//     ConfigJSON: selectedConfig.ConfigJSON,
	//     CreatedAt:  selectedConfig.CreatedAt.Time,
	//     UpdatedAt:  selectedConfig.UpdatedAt.Time,
	// }, nil

	return nil, nil
}

// makeCacheKey creates a cache key from tenant ID and provider type.
func makeCacheKey(tenantID pgtype.UUID, providerType ProviderType) cacheKey {
	var tenantIDString string
	if tenantID.Valid {
		tenantIDString = fmt.Sprintf("%x-%x-%x-%x-%x",
			tenantID.Bytes[0:4],
			tenantID.Bytes[4:6],
			tenantID.Bytes[6:8],
			tenantID.Bytes[8:10],
			tenantID.Bytes[10:16])
	}

	return cacheKey{
		tenantID:     tenantIDString,
		providerType: providerType,
	}
}
