// internal/subscriber/product.go
package subscriber

import (
	"context"
	"fmt"
	"log"

	"github.com/dukerupert/freyja/internal/shared/interfaces"
	"github.com/stripe/stripe-go/v82"
	stripePrice "github.com/stripe/stripe-go/v82/price"
	stripeProduct "github.com/stripe/stripe-go/v82/product"
)

type ProductEventSubscriber struct {
	productService interfaces.ProductService
	variantService interfaces.VariantService
	events         interfaces.EventPublisher
}

func NewProductEventSubscriber(
	productService interfaces.ProductService,
	variantService interfaces.VariantService,
	events interfaces.EventPublisher,
) *ProductEventSubscriber {
	return &ProductEventSubscriber{
		productService: productService,
		variantService: variantService,
		events:         events,
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

	log.Println("✅ Product event subscriber started")
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

	log.Printf("✅ Product %d synced to Stripe successfully (via sync request)", productID)
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

	log.Printf("✅ Product %d synced to Stripe successfully (%d variants)", productID, len(variants))
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

	log.Printf("✅ Product %d updated in Stripe successfully (%d/%d variants)", productID, successCount, len(variants))
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

	log.Printf("✅ Product %d deactivated in Stripe successfully (%d/%d variants)", productID, successCount, len(variants))
	return nil
}

// Helper methods for Stripe operations
// syncVariantToStripe syncs a single variant to Stripe (creates Stripe Product + Prices)
func (s *ProductEventSubscriber) syncVariantToStripe(ctx context.Context, variant *interfaces.ProductVariant) error {
	// Create Stripe Product if it doesn't exist
	if !variant.StripeProductID.Valid {
		log.Printf("Creating Stripe product for variant %d (%s)", variant.ID, variant.Name)
		
		stripeProduct, err := s.createStripeProductForVariant(variant)
		if err != nil {
			return fmt.Errorf("failed to create Stripe product for variant %d: %w", variant.ID, err)
		}

		// Update our variant with Stripe Product ID
		if err := s.variantService.UpdateStripeIDs(ctx, variant.ID, stripeProduct.ID, nil); err != nil {
			return fmt.Errorf("failed to update variant with Stripe product ID: %w", err)
		}

		// Update the variant object for price creation
		variant.StripeProductID.String = stripeProduct.ID
		variant.StripeProductID.Valid = true
		
		log.Printf("✅ Created Stripe product %s for variant %d", stripeProduct.ID, variant.ID)
	} else {
		log.Printf("Stripe product already exists for variant %d: %s", variant.ID, variant.StripeProductID.String)
	}

	// Create all Price objects for this variant
	log.Printf("Creating Stripe prices for variant %d", variant.ID)
	if err := s.createAllStripePricesForVariant(ctx, variant); err != nil {
		return fmt.Errorf("failed to create Stripe prices for variant %d: %w", variant.ID, err)
	}

	log.Printf("✅ Successfully synced variant %d to Stripe", variant.ID)
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
		log.Printf("✅ Synced variant %d (%s) to Stripe", variant.ID, variant.Name)
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
		log.Printf("✅ Successfully synced all %d variants for product %d", successCount, productID)
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

func (s *ProductEventSubscriber) createSingleStripePriceForVariant(variant *interfaces.ProductVariant, recurringDays *int) (*stripe.Price, error) {
	// Validate that variant has a Stripe Product ID
	if !variant.StripeProductID.Valid || variant.StripeProductID.String == "" {
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
		
		log.Printf("Creating subscription price for variant %d (%d day interval)", variant.ID, *recurringDays)
	} else {
		params.Metadata["type"] = "onetime"
		log.Printf("Creating one-time price for variant %d", variant.ID)
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
		return nil, fmt.Errorf("failed to create %s Stripe price for variant %d: %w", priceType, variant.ID, err)
	}

	// Log successful creation
	priceType := "one-time"
	if recurringDays != nil {
		priceType = fmt.Sprintf("subscription (%d day)", *recurringDays)
	}
	log.Printf("✅ Created %s Stripe price %s for variant %d", priceType, stripePrice.ID, variant.ID)

	return stripePrice, nil
}

func (s *ProductEventSubscriber) createAllStripePricesForVariant(ctx context.Context, variant *interfaces.ProductVariant) error {
	log.Printf("Creating all Stripe prices for variant %d (%s)", variant.ID, variant.Name)
	
	priceUpdates := make(map[string]string)

	// One-time purchase price
	if !variant.StripePriceOnetimeID.Valid {
		price, err := s.createSingleStripePriceForVariant(variant, nil) // Fixed method name
		if err != nil {
			return fmt.Errorf("failed to create one-time price: %w", err)
		}
		priceUpdates["onetime"] = price.ID
		log.Printf("Created one-time price %s for variant %d", price.ID, variant.ID)
	} else {
		log.Printf("One-time price already exists for variant %d", variant.ID)
	}

	// Subscription prices (only create if variant supports subscriptions)
	if variant.IsSubscription {
		log.Printf("Creating subscription prices for variant %d", variant.ID)
		
		intervals := map[string]int{"14day": 14, "21day": 21, "30day": 30, "60day": 60}
		currentPrices := map[string]bool{
			"14day": variant.StripePrice14dayID.Valid,
			"21day": variant.StripePrice21dayID.Valid,
			"30day": variant.StripePrice30dayID.Valid,
			"60day": variant.StripePrice60dayID.Valid,
		}

		createdCount := 0
		for interval, days := range intervals {
			if !currentPrices[interval] {
				price, err := s.createSingleStripePriceForVariant(variant, &days) // Fixed method name
				if err != nil {
					return fmt.Errorf("failed to create %s subscription price: %w", interval, err)
				}
				priceUpdates[interval] = price.ID
				createdCount++
			}
		}
		
		log.Printf("Created %d subscription prices for variant %d", createdCount, variant.ID)
	} else {
		log.Printf("Variant %d does not support subscriptions, skipping subscription prices", variant.ID)
	}

	// Update all price IDs in database using variant service
	if len(priceUpdates) > 0 {
		log.Printf("Updating database with %d new price IDs for variant %d", len(priceUpdates), variant.ID)
		
		if err := s.variantService.UpdateStripeIDs(ctx, variant.ID, variant.StripeProductID.String, priceUpdates); err != nil {
			return fmt.Errorf("failed to update variant with new Stripe price IDs: %w", err)
		}
		
		log.Printf("✅ Successfully updated variant %d with new Stripe price IDs", variant.ID)
	} else {
		log.Printf("No new prices needed for variant %d - all prices already exist", variant.ID)
	}

	log.Printf("✅ Completed price creation for variant %d", variant.ID)
	return nil
}
