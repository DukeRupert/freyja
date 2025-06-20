// internal/subscriber/product.go
package subscriber

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/dukerupert/freyja/internal/shared/interfaces"
	"github.com/rs/zerolog"
	"github.com/stripe/stripe-go/v82"
	stripePrice "github.com/stripe/stripe-go/v82/price"
	stripeProduct "github.com/stripe/stripe-go/v82/product"
)

type ProductEventSubscriber struct {
	productService interfaces.ProductService
	variantService interfaces.VariantService
	events         interfaces.EventPublisher
	logger         zerolog.Logger
}

func NewProductEventSubscriber(
	productService interfaces.ProductService,
	variantService interfaces.VariantService,
	events interfaces.EventPublisher,
	logger zerolog.Logger,
) *ProductEventSubscriber {
	return &ProductEventSubscriber{
		productService: productService,
		variantService: variantService,
		events:         events,
		logger:         logger.With().Str("component", "ProductEventSubscriber").Logger(),
	}
}

// Start subscribes to product and variant events
func (s *ProductEventSubscriber) Start(ctx context.Context) error {
	// Subscribe to product created events (now syncs all variants)
	if err := s.events.Subscribe(ctx, interfaces.EventProductCreated, s.handleProductCreated); err != nil {
		return fmt.Errorf("failed to subscribe to product.created events: %w", err)
	}

	// Subscribe to product updated events (may affect variants)
	if err := s.events.Subscribe(ctx, interfaces.EventProductUpdated, s.handleProductUpdated); err != nil {
		return fmt.Errorf("failed to subscribe to product.updated events: %w", err)
	}

	// Subscribe to product deactivated events (deactivates all variants)
	if err := s.events.Subscribe(ctx, interfaces.EventProductDeactivated, s.handleProductDeactivated); err != nil {
		return fmt.Errorf("failed to subscribe to product.deactivated events: %w", err)
	}

	// Subscribe to variant-specific events
	if err := s.events.Subscribe(ctx, interfaces.EventVariantCreated, s.handleVariantCreated); err != nil {
		return fmt.Errorf("failed to subscribe to variant.created events: %w", err)
	}

	if err := s.events.Subscribe(ctx, interfaces.EventVariantUpdated, s.handleVariantUpdated); err != nil {
		return fmt.Errorf("failed to subscribe to variant.updated events: %w", err)
	}

	if err := s.events.Subscribe(ctx, interfaces.EventVariantDeactivated, s.handleVariantDeactivated); err != nil {
		return fmt.Errorf("failed to subscribe to variant.deactivated events: %w", err)
	}

	// Subscribe to product stripe sync events (now syncs all variants for product)
	if err := s.events.Subscribe(ctx, interfaces.EventProductStripeSync, s.handleProductStripeSyncRequested); err != nil {
		return fmt.Errorf("failed to subscribe to product.stripe_sync_requested events: %w", err)
	}

	log.Println("[OK] Product event subscriber started")
	return nil
}

func (s *ProductEventSubscriber) handleProductStripeSyncRequested(ctx context.Context, event interfaces.Event) error {
	log.Printf("Processing product.stripe_sync_requested event: %s", event.AggregateID)

	productID, err := interfaces.ExtractAggregateID(event.AggregateID)
	if err != nil {
		return fmt.Errorf("invalid product ID in event: %w", err)
	}

	// Verify the product exists and is valid
	product, err := s.productService.GetBasicProductByID(ctx, int(productID))
	if err != nil {
		return fmt.Errorf("failed to get product %d: %w", productID, err)
	}

	if !product.Active {
		log.Printf("Skipping Stripe sync for inactive product %d", productID)
		return nil
	}

	// Sync all variants for this product to Stripe
	if err := s.syncAllVariantsForProduct(ctx, int32(productID)); err != nil {
		return fmt.Errorf("failed to sync variants for product %d to Stripe: %w", productID, err)
	}

	log.Printf("[OK] Product %d synced to Stripe successfully (via sync request)", productID)
	return nil
}

// handleProductCreated syncs new product's variants to Stripe
func (s *ProductEventSubscriber) handleProductCreated(ctx context.Context, event interfaces.Event) error {
	log.Printf("Processing product.created event: %s", event.AggregateID)

	productID, err := interfaces.ExtractAggregateID(event.AggregateID)
	if err != nil {
		return fmt.Errorf("invalid product ID in event: %w", err)
	}

	// Verify the product exists and is valid
	product, err := s.productService.GetBasicProductByID(ctx, int(productID))
	if err != nil {
		return fmt.Errorf("failed to get product %d: %w", productID, err)
	}

	if !product.Active {
		log.Printf("Skipping Stripe sync for inactive product %d", productID)
		return nil
	}

	// Check if product has any variants yet
	variants, err := s.variantService.GetActiveVariantsByProduct(ctx, int32(productID))
	if err != nil {
		return fmt.Errorf("failed to get variants for product %d: %w", productID, err)
	}

	if len(variants) == 0 {
		log.Printf("No variants found for newly created product %d - Stripe sync will happen when variants are created", productID)
		return nil
	}

	// Sync all existing variants for this product to Stripe
	if err := s.syncAllVariantsForProduct(ctx, int32(productID)); err != nil {
		return fmt.Errorf("failed to sync variants for product %d to Stripe: %w", productID, err)
	}

	log.Printf("[OK] Product %d synced to Stripe successfully (%d variants)", productID, len(variants))
	return nil
}

// handleProductUpdated handles product updates and updates related variants
func (s *ProductEventSubscriber) handleProductUpdated(ctx context.Context, event interfaces.Event) error {
	log.Printf("Processing product.updated event: %s", event.AggregateID)

	productID, err := interfaces.ExtractAggregateID(event.AggregateID)
	if err != nil {
		return fmt.Errorf("invalid product ID in event: %w", err)
	}

	// Get the product
	product, err := s.productService.GetBasicProductByID(ctx, int(productID))
	if err != nil {
		return fmt.Errorf("failed to get product %d: %w", productID, err)
	}

	// Get all variants for this product
	variants, err := s.variantService.GetVariantsByProduct(ctx, int32(productID))
	if err != nil {
		return fmt.Errorf("failed to get variants for product %d: %w", productID, err)
	}

	if len(variants) == 0 {
		log.Printf("No variants found for product %d - nothing to update in Stripe", productID)
		return nil
	}

	// Check if product was deactivated
	if statusChanged, exists := event.Data["status_changed"].(bool); exists && statusChanged && !product.Active {
		log.Printf("Product %d was deactivated, deactivating all variants in Stripe", productID)
		return s.deactivateAllVariantsForProduct(ctx, variants)
	}

	// Check if product was reactivated
	if statusChanged, exists := event.Data["status_changed"].(bool); exists && statusChanged && product.Active {
		log.Printf("Product %d was reactivated, reactivating variants in Stripe", productID)
		return s.reactivateAllVariantsForProduct(ctx, variants)
	}

	// For other product updates (name, description changes), update all variant Stripe products
	// Note: In the variant model, product name/description changes might affect variant display names
	log.Printf("Product %d metadata updated, updating %d variants in Stripe", productID, len(variants))

	successCount := 0
	var lastError error

	for _, variant := range variants {
		if variant.StripeProductID.Valid {
			if err := s.updateStripeProductForVariant(ctx, &variant, product); err != nil {
				log.Printf("Failed to update Stripe product for variant %d: %v", variant.ID, err)
				lastError = err
				continue
			}
			successCount++
		}
	}

	if successCount == 0 && lastError != nil {
		return fmt.Errorf("failed to update any variant Stripe products for product %d: %w", productID, lastError)
	}

	log.Printf("[OK] Product %d updated in Stripe successfully (%d/%d variants)", productID, successCount, len(variants))
	return nil
}

// handleVariantCreated syncs newly created variants to Stripe
func (s *ProductEventSubscriber) handleVariantCreated(ctx context.Context, event interfaces.Event) error {
	logger := s.logger.With().
		Str("event_id", event.ID).
		Str("event_type", event.Type).
		Str("aggregate_id", event.AggregateID).
		Str("handler", "handleVariantCreated").
		Logger()

	logger.Info().Msg("Processing variant.created event")

	// Enhanced aggregate ID extraction with better error handling
	variantID, err := interfaces.ExtractAggregateID(event.AggregateID)
	if err != nil {
		logger.Error().
			Err(err).
			Str("expected_format", "variant:123").
			Str("received_format", event.AggregateID).
			Msg("Invalid aggregate ID format - this event will be skipped to prevent loop")

		// CRITICAL: Return nil instead of error to prevent retry loop
		// This allows the event to be marked as processed and removed from queue
		return nil
	}

	logger = logger.With().Int32("variant_id", variantID).Logger()
	logger.Info().Msg("Successfully extracted variant ID from aggregate")

	// Get the variant with enhanced error logging
	variant, err := s.variantService.GetByID(ctx, int32(variantID))
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to get variant from database")

		// If variant doesn't exist, don't retry - return nil to prevent loop
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "no rows") {
			logger.Warn().Msg("Variant not found - event will be skipped")
			return nil
		}

		// For other database errors, we might want to retry, so return the error
		return fmt.Errorf("failed to get variant %d: %w", variantID, err)
	}

	logger.Info().
		Str("variant_name", variant.Name).
		Bool("active", variant.Active).
		Bool("is_subscription", variant.IsSubscription).
		Msg("Successfully retrieved variant details")

	// Only sync active variants
	if !variant.Active {
		logger.Info().Msg("Skipping Stripe sync for inactive variant")
		return nil
	}

	// Sync to Stripe with detailed logging
	logger.Info().Msg("Starting Stripe sync for variant")

	if err := s.syncVariantToStripe(ctx, variant); err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to sync variant to Stripe")

		// Log specific Stripe error details if available
		if stripeErr, ok := err.(*stripe.Error); ok {
			logger.Error().
				Str("stripe_error_code", string(stripeErr.Code)).
				Str("stripe_error_type", string(stripeErr.Type)).
				Str("stripe_error_message", stripeErr.Msg).
				Msg("Stripe API error details")
		}

		return fmt.Errorf("failed to sync variant %d to Stripe: %w", variantID, err)
	}

	logger.Info().Msg("[OK] Variant synced to Stripe successfully")
	return nil
}

// handleVariantUpdated handles variant updates and syncs changes to Stripe
func (s *ProductEventSubscriber) handleVariantUpdated(ctx context.Context, event interfaces.Event) error {
	log.Printf("Processing variant.updated event: %s", event.AggregateID)

	variantID, err := interfaces.ExtractAggregateID(event.AggregateID)
	if err != nil {
		return fmt.Errorf("invalid variant ID in event: %w", err)
	}

	// Get the variant
	variant, err := s.variantService.GetByID(ctx, int32(variantID))
	if err != nil {
		return fmt.Errorf("failed to get variant %d: %w", variantID, err)
	}

	// Check if price changed (requires new Stripe prices since they're immutable)
	if priceChanged, exists := event.Data["price_changed"].(bool); exists && priceChanged {
		log.Printf("Price changed for variant %d, creating new Stripe prices", variantID)

		// Create new prices (Stripe prices are immutable)
		if err := s.createAllStripePricesForVariant(ctx, variant); err != nil {
			return fmt.Errorf("failed to create new Stripe prices for variant %d: %w", variantID, err)
		}
	} else {
		// Update Stripe Product metadata (name, description, active status, etc.)
		if variant.StripeProductID.Valid {
			// Get parent product for context
			product, err := s.productService.GetBasicProductByID(ctx, int(variant.ProductID))
			if err != nil {
				log.Printf("Warning: failed to get parent product %d for variant %d: %v", variant.ProductID, variantID, err)
				// Continue with just variant data
				product = nil
			}

			if err := s.updateStripeProductForVariant(ctx, variant, product); err != nil {
				return fmt.Errorf("failed to update Stripe product for variant %d: %w", variantID, err)
			}
		}
	}

	log.Printf("[OK] Variant %d updated in Stripe successfully", variantID)
	return nil
}

// handleVariantDeactivated deactivates variant in Stripe
func (s *ProductEventSubscriber) handleVariantDeactivated(ctx context.Context, event interfaces.Event) error {
	log.Printf("Processing variant.deactivated event: %s", event.AggregateID)

	variantID, err := interfaces.ExtractAggregateID(event.AggregateID)
	if err != nil {
		return fmt.Errorf("invalid variant ID in event: %w", err)
	}

	// Get the variant
	variant, err := s.variantService.GetByID(ctx, int32(variantID))
	if err != nil {
		return fmt.Errorf("failed to get variant %d: %w", variantID, err)
	}

	// Deactivate in Stripe if it exists
	if variant.StripeProductID.Valid {
		if err := s.deactivateStripeProduct(ctx, variant.StripeProductID.String); err != nil {
			return fmt.Errorf("failed to deactivate Stripe product for variant %d: %w", variantID, err)
		}
	} else {
		log.Printf("Variant %d has no Stripe product to deactivate", variantID)
	}

	log.Printf("[OK] Variant %d deactivated in Stripe successfully", variantID)
	return nil
}

// Activate a Stripe product
func (s *ProductEventSubscriber) activateStripeProduct(ctx context.Context, stripeProductID string) error {
	params := &stripe.ProductParams{
		Active: stripe.Bool(true),
	}

	_, err := stripeProduct.Update(stripeProductID, params)
	if err != nil {
		return fmt.Errorf("failed to activate Stripe product %s: %w", stripeProductID, err)
	}

	return nil
}

// Deactivate a Stripe product (you already have this one)
func (s *ProductEventSubscriber) deactivateStripeProduct(ctx context.Context, stripeProductID string) error {
	params := &stripe.ProductParams{
		Active: stripe.Bool(false),
	}

	_, err := stripeProduct.Update(stripeProductID, params)
	if err != nil {
		return fmt.Errorf("failed to deactivate Stripe product %s: %w", stripeProductID, err)
	}

	return nil
}

// Helper method to deactivate all variants
func (s *ProductEventSubscriber) deactivateAllVariantsForProduct(ctx context.Context, variants []interfaces.ProductVariant) error {
	for _, variant := range variants {
		if variant.StripeProductID.Valid {
			if err := s.deactivateStripeProduct(ctx, variant.StripeProductID.String); err != nil {
				log.Printf("Failed to deactivate variant %d in Stripe: %v", variant.ID, err)
			}
		}
	}
	return nil
}

// Helper method to reactivate variants
func (s *ProductEventSubscriber) reactivateAllVariantsForProduct(ctx context.Context, variants []interfaces.ProductVariant) error {
	for _, variant := range variants {
		if variant.StripeProductID.Valid && variant.Active {
			if err := s.activateStripeProduct(ctx, variant.StripeProductID.String); err != nil {
				log.Printf("Failed to reactivate variant %d in Stripe: %v", variant.ID, err)
			}
		}
	}
	return nil
}

// Helper method to update Stripe product for a variant
func (s *ProductEventSubscriber) updateStripeProductForVariant(ctx context.Context, variant *interfaces.ProductVariant, product *interfaces.Product) error {
	if !variant.StripeProductID.Valid {
		return nil
	}

	// Create updated name including variant options
	productName := variant.Name
	if variant.OptionsDisplay.Valid && variant.OptionsDisplay.String != "" {
		productName = fmt.Sprintf("%s (%s)", variant.Name, variant.OptionsDisplay.String)
	}

	params := &stripe.ProductParams{
		Name:   stripe.String(productName),
		Active: stripe.Bool(variant.Active && product.Active),
	}

	// Add description if available
	if description := s.getVariantDescription(variant); description != "" {
		params.Description = stripe.String(description)
	}

	_, err := stripeProduct.Update(variant.StripeProductID.String, params)
	return err
}

// handleProductDeactivated deactivates all variants for a product in Stripe
func (s *ProductEventSubscriber) handleProductDeactivated(ctx context.Context, event interfaces.Event) error {
	log.Printf("Processing product.deactivated event: %s", event.AggregateID)

	productID, err := interfaces.ExtractAggregateID(event.AggregateID)
	if err != nil {
		return fmt.Errorf("invalid product ID in event: %w", err)
	}

	// Get the product to verify it exists
	_, err = s.productService.GetBasicProductByID(ctx, int(productID))
	if err != nil {
		return fmt.Errorf("failed to get product %d: %w", productID, err)
	}

	// Get all variants for this product (including inactive ones)
	variants, err := s.variantService.GetVariantsByProduct(ctx, int32(productID))
	if err != nil {
		return fmt.Errorf("failed to get variants for product %d: %w", productID, err)
	}

	if len(variants) == 0 {
		log.Printf("No variants found for product %d - nothing to deactivate in Stripe", productID)
		return nil
	}

	// Deactivate all variants in Stripe
	successCount := 0
	var lastError error

	for _, variant := range variants {
		if variant.StripeProductID.Valid {
			if err := s.deactivateStripeProduct(ctx, variant.StripeProductID.String); err != nil {
				log.Printf("Failed to deactivate variant %d in Stripe: %v", variant.ID, err)
				lastError = err
				continue
			}
			successCount++
			log.Printf("Deactivated variant %d (%s) in Stripe", variant.ID, variant.Name)
		}
	}

	if successCount == 0 && lastError != nil {
		return fmt.Errorf("failed to deactivate any variants for product %d in Stripe: %w", productID, lastError)
	}

	log.Printf("[OK] Product %d deactivated in Stripe successfully (%d/%d variants)", productID, successCount, len(variants))
	return nil
}

// Helper methods for Stripe operations
// syncVariantToStripe syncs a single variant to Stripe (creates Stripe Product + Prices)
func (s *ProductEventSubscriber) syncVariantToStripe(ctx context.Context, variant *interfaces.ProductVariant) error {
	// Create context-specific logger
	logger := s.logger.With().
		Int32("variant_id", variant.ID).
		Str("variant_name", variant.Name).
		Bool("is_subscription", variant.IsSubscription).
		Str("function", "syncVariantToStripe").
		Logger()

	// Create Stripe Product if it doesn't exist
	if !variant.StripeProductID.Valid {
		logger.Info().Msg("Creating Stripe product for variant")
		
		stripeProduct, err := s.createStripeProductForVariant(variant)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to create Stripe product")
			return fmt.Errorf("failed to create Stripe product for variant %d: %w", variant.ID, err)
		}

		// Update our variant with Stripe Product ID
		if err := s.variantService.UpdateStripeIDs(ctx, variant.ID, stripeProduct.ID, nil); err != nil {
			logger.Error().Err(err).Str("stripe_product_id", stripeProduct.ID).Msg("Failed to update variant with Stripe product ID")
			return fmt.Errorf("failed to update variant with Stripe product ID: %w", err)
		}

		// Update the variant object for price creation
		variant.StripeProductID.String = stripeProduct.ID
		variant.StripeProductID.Valid = true
		
		logger.Info().Str("stripe_product_id", stripeProduct.ID).Msg("[OK] Created and linked Stripe product")
	} else {
		logger.Info().Str("stripe_product_id", variant.StripeProductID.String).Msg("Stripe product already exists")
	}

	// Create all Price objects for this variant
	logger.Info().Msg("Creating Stripe prices for variant")
	if err := s.createAllStripePricesForVariant(ctx, variant); err != nil {
		logger.Error().Err(err).Msg("Failed to create Stripe prices")
		return fmt.Errorf("failed to create Stripe prices for variant %d: %w", variant.ID, err)
	}

	logger.Info().Msg("[OK] Successfully synced variant to Stripe")
	return nil
}

// syncAllVariantsForProduct syncs all variants of a product to Stripe
func (s *ProductEventSubscriber) syncAllVariantsForProduct(ctx context.Context, productID int32) error {
	log.Printf("Starting Stripe sync for all variants of product %d", productID)

	// Get all active variants for this product
	variants, err := s.variantService.GetActiveVariantsByProduct(ctx, productID)
	if err != nil {
		return fmt.Errorf("failed to get variants for product %d: %w", productID, err)
	}

	if len(variants) == 0 {
		log.Printf("No active variants found for product %d - sync complete", productID)
		return nil
	}

	log.Printf("Found %d active variants for product %d", len(variants), productID)

	// Sync each variant to Stripe
	successCount := 0
	errorCount := 0
	var errors []error

	for _, variant := range variants {
		if err := s.syncVariantToStripe(ctx, &variant); err != nil {
			log.Printf("Failed to sync variant %d (%s) to Stripe: %v", variant.ID, variant.Name, err)
			errors = append(errors, fmt.Errorf("variant %d: %w", variant.ID, err))
			errorCount++
			continue
		}
		successCount++
		log.Printf("[OK] Synced variant %d (%s) to Stripe", variant.ID, variant.Name)
	}

	// Report results
	if successCount == 0 && errorCount > 0 {
		return fmt.Errorf("failed to sync any variants for product %d (%d errors): %v", productID, errorCount, errors[0])
	}

	if errorCount > 0 {
		log.Printf("⚠️ Partial success: %d/%d variants synced for product %d (%d errors)",
			successCount, len(variants), productID, errorCount)
		// You might want to return an error here if partial failures should fail the operation
		// For now, we'll continue with partial success
	} else {
		log.Printf("[OK] Successfully synced all %d variants for product %d", successCount, productID)
	}

	return nil
}

func (s *ProductEventSubscriber) createStripeProductForVariant(variant *interfaces.ProductVariant) (*stripe.Product, error) {
	// Create a descriptive name that includes variant information
	productName := variant.Name
	if variant.OptionsDisplay.Valid && variant.OptionsDisplay.String != "" {
		productName = fmt.Sprintf("%s (%s)", variant.Name, variant.OptionsDisplay.String)
	}

	params := &stripe.ProductParams{
		Name:   stripe.String(productName),
		Active: stripe.Bool(variant.Active),
		Metadata: map[string]string{
			"internal_variant_id": fmt.Sprintf("%d", variant.ID),
			"internal_product_id": fmt.Sprintf("%d", variant.ProductID),
		},
	}

	// Add description if available
	if description := s.getVariantDescription(variant); description != "" {
		params.Description = stripe.String(description)
	}

	// Add variant-specific metadata for better tracking
	if variant.IsSubscription {
		params.Metadata["supports_subscription"] = "true"
	} else {
		params.Metadata["supports_subscription"] = "false"
	}

	// Add options metadata if available
	if variant.OptionsDisplay.Valid && variant.OptionsDisplay.String != "" {
		params.Metadata["variant_options"] = variant.OptionsDisplay.String
	}

	stripeProduct, err := stripeProduct.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create Stripe product for variant %d: %w", variant.ID, err)
	}

	log.Printf("Created Stripe product %s for variant %d (%s)", stripeProduct.ID, variant.ID, productName)
	return stripeProduct, nil
}

func (s *ProductEventSubscriber) getVariantDescription(variant *interfaces.ProductVariant) string {
	// You might want to fetch the parent product description
	// or create a variant-specific description
	// This depends on your business requirements

	if variant.IsSubscription {
		return fmt.Sprintf("Subscription variant of %s", variant.Name)
	}
	return fmt.Sprintf("One-time purchase variant of %s", variant.Name)
}

// createSingleStripePriceForVariant creates a single price (one-time or subscription)
func (s *ProductEventSubscriber) createSingleStripePriceForVariant(variant *interfaces.ProductVariant, recurringDays *int) (*stripe.Price, error) {
	logger := s.logger.With().
		Int32("variant_id", variant.ID).
		Str("function", "createSingleStripePriceForVariant").
		Logger()

	// Validate that variant has a Stripe Product ID
	if !variant.StripeProductID.Valid || variant.StripeProductID.String == "" {
		logger.Error().Msg("Variant does not have a valid Stripe Product ID")
		return nil, fmt.Errorf("variant %d does not have a valid Stripe Product ID", variant.ID)
	}

	params := &stripe.PriceParams{
		Product:    stripe.String(variant.StripeProductID.String),
		UnitAmount: stripe.Int64(int64(variant.Price)),
		Currency:   stripe.String("usd"),
		Metadata: map[string]string{
			"internal_variant_id": fmt.Sprintf("%d", variant.ID),
			"internal_product_id": fmt.Sprintf("%d", variant.ProductID),
		},
	}

	// Configure pricing type and metadata
	if recurringDays != nil {
		params.Recurring = &stripe.PriceRecurringParams{
			Interval:      stripe.String("day"),
			IntervalCount: stripe.Int64(int64(*recurringDays)),
		}
		params.Metadata["subscription_days"] = fmt.Sprintf("%d", *recurringDays)
		params.Metadata["type"] = "subscription"
		
		logger = logger.With().Int("recurring_days", *recurringDays).Str("price_type", "subscription").Logger()
		logger.Info().Msg("Creating subscription price")
	} else {
		params.Metadata["type"] = "onetime"
		logger = logger.With().Str("price_type", "one_time").Logger()
		logger.Info().Msg("Creating one-time price")
	}

	// Add variant options to metadata for both types
	if variant.OptionsDisplay.Valid && variant.OptionsDisplay.String != "" {
		params.Metadata["variant_options"] = variant.OptionsDisplay.String
	}

	// Add subscription capability flag
	params.Metadata["supports_subscription"] = fmt.Sprintf("%t", variant.IsSubscription)

	// Add variant name for easier identification in Stripe dashboard
	params.Metadata["variant_name"] = variant.Name

	// Create the price
	stripePrice, err := stripePrice.New(params)
	if err != nil {
		priceType := "one-time"
		if recurringDays != nil {
			priceType = fmt.Sprintf("subscription (%d days)", *recurringDays)
		}
		logger.Error().Err(err).Str("price_type", priceType).Msg("Failed to create Stripe price")
		return nil, fmt.Errorf("failed to create %s Stripe price for variant %d: %w", priceType, variant.ID, err)
	}

	// Log successful creation
	priceType := "one-time"
	if recurringDays != nil {
		priceType = fmt.Sprintf("subscription (%d day)", *recurringDays)
	}
	logger.Info().Str("stripe_price_id", stripePrice.ID).Str("price_type", priceType).Msg("[OK] Successfully created Stripe price")

	return stripePrice, nil
}

// createAllStripePricesForVariant creates all necessary price objects for a variant
func (s *ProductEventSubscriber) createAllStripePricesForVariant(ctx context.Context, variant *interfaces.ProductVariant) error {
	logger := s.logger.With().
		Int32("variant_id", variant.ID).
		Str("variant_name", variant.Name).
		Bool("is_subscription", variant.IsSubscription).
		Str("function", "createAllStripePricesForVariant").
		Logger()

	logger.Info().Msg("Starting price creation process")
	
	priceUpdates := make(map[string]string)
	totalPricesCreated := 0

	// One-time purchase price
	if !variant.StripePriceOnetimeID.Valid {
		logger.Info().Msg("Creating one-time purchase price")
		price, err := s.createSingleStripePriceForVariant(variant, nil)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to create one-time price")
			return fmt.Errorf("failed to create one-time price: %w", err)
		}
		priceUpdates["onetime"] = price.ID
		totalPricesCreated++
		logger.Info().Str("stripe_price_id", price.ID).Msg("[OK] Created one-time price")
	} else {
		logger.Info().Str("stripe_price_id", variant.StripePriceOnetimeID.String).Msg("One-time price already exists")
	}

	// Subscription prices (only create if variant supports subscriptions)
	if variant.IsSubscription {
		logger.Info().Msg("Creating subscription prices")
		
		intervals := map[string]int{"14day": 14, "21day": 21, "30day": 30, "60day": 60}
		currentPrices := map[string]bool{
			"14day": variant.StripePrice14dayID.Valid,
			"21day": variant.StripePrice21dayID.Valid,
			"30day": variant.StripePrice30dayID.Valid,
			"60day": variant.StripePrice60dayID.Valid,
		}

		subscriptionPricesCreated := 0
		for interval, days := range intervals {
			intervalLogger := logger.With().Str("interval", interval).Int("days", days).Logger()
			
			if !currentPrices[interval] {
				intervalLogger.Info().Msg("Creating subscription price")
				price, err := s.createSingleStripePriceForVariant(variant, &days)
				if err != nil {
					intervalLogger.Error().Err(err).Msg("Failed to create subscription price")
					return fmt.Errorf("failed to create %s subscription price: %w", interval, err)
				}
				priceUpdates[interval] = price.ID
				subscriptionPricesCreated++
				totalPricesCreated++
				intervalLogger.Info().Str("stripe_price_id", price.ID).Msg("[OK] Created subscription price")
			} else {
				intervalLogger.Info().Msg("Subscription price already exists")
			}
		}
		
		logger.Info().Int("created_count", subscriptionPricesCreated).Msg("Completed subscription price creation")
	} else {
		logger.Info().Msg("Variant does not support subscriptions, skipping subscription prices")
	}

	// Update all price IDs in database using variant service
	if len(priceUpdates) > 0 {
		logger.Info().Int("price_count", len(priceUpdates)).Msg("Updating database with new price IDs")
		
		if err := s.variantService.UpdateStripeIDs(ctx, variant.ID, variant.StripeProductID.String, priceUpdates); err != nil {
			logger.Error().Err(err).Int("price_count", len(priceUpdates)).Msg("Failed to update variant with price IDs")
			return fmt.Errorf("failed to update variant with new Stripe price IDs: %w", err)
		}
		
		logger.Info().Int("price_count", len(priceUpdates)).Msg("[OK] Successfully updated variant with new Stripe price IDs")
	} else {
		logger.Info().Msg("No new prices needed - all prices already exist")
	}

	logger.Info().Int("total_created", totalPricesCreated).Msg("[OK] Completed price creation process")
	return nil
}
