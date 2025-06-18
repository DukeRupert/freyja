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
	events         interfaces.EventPublisher
}

func NewProductEventSubscriber(
	productService interfaces.ProductService,
	events interfaces.EventPublisher,
) *ProductEventSubscriber {
	return &ProductEventSubscriber{
		productService: productService,
		events:         events,
	}
}

// Start subscribes to product events
func (s *ProductEventSubscriber) Start(ctx context.Context) error {
	// Subscribe to product created events
	if err := s.events.Subscribe(ctx, interfaces.EventProductCreated, s.handleProductCreated); err != nil {
		return fmt.Errorf("failed to subscribe to product.created events: %w", err)
	}

	// Subscribe to product updated events
	if err := s.events.Subscribe(ctx, interfaces.EventProductUpdated, s.handleProductUpdated); err != nil {
		return fmt.Errorf("failed to subscribe to product.updated events: %w", err)
	}

	// Subscribe to product deactivated events
	if err := s.events.Subscribe(ctx, interfaces.EventProductDeactivated, s.handleProductDeactivated); err != nil {
		return fmt.Errorf("failed to subscribe to product.deactivated events: %w", err)
	}

	// *** FIX: Subscribe to product stripe sync events ***
	if err := s.events.Subscribe(ctx, interfaces.EventProductStripeSync, s.handleProductStripeSyncRequested); err != nil {
		return fmt.Errorf("failed to subscribe to product.stripe_sync_requested events: %w", err)
	}

	log.Println("✅ Product event subscriber started")
	return nil
}

// *** FIX: Add missing handler for Stripe sync requests ***
func (s *ProductEventSubscriber) handleProductStripeSyncRequested(ctx context.Context, event interfaces.Event) error {
	log.Printf("Processing product.stripe_sync_requested event: %s", event.AggregateID)

	productID, err := interfaces.ExtractAggregateID(event.AggregateID)
	if err != nil {
		return fmt.Errorf("invalid product ID in event: %w", err)
	}

	// Get the product
	product, err := s.productService.GetByID(ctx, int(productID))
	if err != nil {
		return fmt.Errorf("failed to get product %d: %w", productID, err)
	}

	// Sync to Stripe (this will create Product + all Price objects)
	if err := s.syncProductToStripe(ctx, product); err != nil {
		return fmt.Errorf("failed to sync product to Stripe: %w", err)
	}

	log.Printf("✅ Product %d synced to Stripe successfully (via sync request)", productID)
	return nil
}

// handleProductCreated syncs new products to Stripe
func (s *ProductEventSubscriber) handleProductCreated(ctx context.Context, event interfaces.Event) error {
	log.Printf("Processing product.created event: %s", event.AggregateID)

	productID, err := interfaces.ExtractAggregateID(event.AggregateID)
	if err != nil {
		return fmt.Errorf("invalid product ID in event: %w", err)
	}

	// Get the product
	product, err := s.productService.GetByID(ctx, int(productID))
	if err != nil {
		return fmt.Errorf("failed to get product %d: %w", productID, err)
	}

	// Sync to Stripe
	if err := s.syncProductToStripe(ctx, product); err != nil {
		return fmt.Errorf("failed to sync product to Stripe: %w", err)
	}

	log.Printf("✅ Product %d synced to Stripe successfully", productID)
	return nil
}

// handleProductUpdated handles product updates
func (s *ProductEventSubscriber) handleProductUpdated(ctx context.Context, event interfaces.Event) error {
	log.Printf("Processing product.updated event: %s", event.AggregateID)

	productID, err := interfaces.ExtractAggregateID(event.AggregateID)
	if err != nil {
		return fmt.Errorf("invalid product ID in event: %w", err)
	}

	// Get the product
	product, err := s.productService.GetByID(ctx, int(productID))
	if err != nil {
		return fmt.Errorf("failed to get product %d: %w", productID, err)
	}

	// If price changed, we need to create new Price objects in Stripe
	if priceChanged, exists := event.Data["price_changed"].(bool); exists && priceChanged {
		log.Printf("Price changed for product %d, creating new Stripe prices", productID)
		if err := s.recreateAllStripePricesforProduct(ctx, product); err != nil {
			return fmt.Errorf("failed to create new Stripe prices: %w", err)
		}
	} else {
		// Just update the Stripe Product (name, description, etc.)
		if err := s.updateStripeProduct(ctx, product); err != nil {
			return fmt.Errorf("failed to update Stripe product: %w", err)
		}
	}

	log.Printf("✅ Product %d updated in Stripe successfully", productID)
	return nil
}

// handleProductDeactivated deactivates products in Stripe
func (s *ProductEventSubscriber) handleProductDeactivated(ctx context.Context, event interfaces.Event) error {
	log.Printf("Processing product.deactivated event: %s", event.AggregateID)

	productID, err := interfaces.ExtractAggregateID(event.AggregateID)
	if err != nil {
		return fmt.Errorf("invalid product ID in event: %w", err)
	}

	// Get the product
	product, err := s.productService.GetByID(ctx, int(productID))
	if err != nil {
		return fmt.Errorf("failed to get product %d: %w", productID, err)
	}

	// Deactivate in Stripe if it exists
	if product.StripeProductID.Valid {
		if err := s.deactivateStripeProduct(ctx, product.StripeProductID.String); err != nil {
			return fmt.Errorf("failed to deactivate Stripe product: %w", err)
		}
	}

	log.Printf("✅ Product %d deactivated in Stripe successfully", productID)
	return nil
}

// Helper methods for Stripe operations
func (s *ProductEventSubscriber) syncProductToStripe(ctx context.Context, product *interfaces.Product) error {
	// Create Stripe Product if it doesn't exist
	if !product.StripeProductID.Valid {
		stripeProduct, err := s.createStripeProduct(product)
		if err != nil {
			return err
		}

		// Update our product with Stripe Product ID
		if err := s.productService.UpdateStripeProductID(ctx, product.ID, stripeProduct.ID); err != nil {
			return err
		}

		product.StripeProductID.String = stripeProduct.ID
		product.StripeProductID.Valid = true
	}

	// Create all Price objects
	return s.createAllStripePrices(ctx, product)
}

func (s *ProductEventSubscriber) createStripeProduct(variant *interfaces.ProductVariant) (*stripe.Product, error) {
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

	// Add description if available (you might need to fetch the parent product description)
	if description := s.getVariantDescription(variant); description != "" {
		params.Description = stripe.String(description)
	}

	return stripeProduct.New(params)
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

func (s *ProductEventSubscriber) createAllStripePricesForVariant(ctx context.Context, variant *interfaces.ProductVariant) error {
	priceUpdates := make(map[string]string)

	// One-time purchase price
	if !variant.StripePriceOnetimeID.Valid {
		price, err := s.createSingleStripePrice(variant, nil)
		if err != nil {
			return err
		}
		priceUpdates["onetime"] = price.ID
	}

	// Subscription prices (only create if variant supports subscriptions)
	if variant.IsSubscription {
		intervals := map[string]int{"14day": 14, "21day": 21, "30day": 30, "60day": 60}
		currentPrices := map[string]bool{
			"14day": variant.StripePrice14dayID.Valid,
			"21day": variant.StripePrice21dayID.Valid,
			"30day": variant.StripePrice30dayID.Valid,
			"60day": variant.StripePrice60dayID.Valid,
		}

		for interval, days := range intervals {
			if !currentPrices[interval] {
				price, err := s.createSingleStripePrice(variant, &days)
				if err != nil {
					return err
				}
				priceUpdates[interval] = price.ID
			}
		}
	}

	// Update all price IDs in database using variant repository
	if len(priceUpdates) > 0 {
		return s.variantService.UpdateStripeIDs(ctx, variant.ID, variant.StripeProductID.String, priceUpdates)
	}

	return nil
}
func (s *ProductEventSubscriber) createSingleStripePrice(variant *interfaces.ProductVariant, recurringDays *int) (*stripe.Price, error) {
	params := &stripe.PriceParams{
		Product:    stripe.String(variant.StripeProductID.String),
		UnitAmount: stripe.Int64(int64(variant.Price)),
		Currency:   stripe.String("usd"),
		Metadata: map[string]string{
			"internal_variant_id": fmt.Sprintf("%d", variant.ID),
			"internal_product_id": fmt.Sprintf("%d", variant.ProductID),
		},
	}

	if recurringDays != nil {
		params.Recurring = &stripe.PriceRecurringParams{
			Interval:      stripe.String("day"),
			IntervalCount: stripe.Int64(int64(*recurringDays)),
		}
		params.Metadata["subscription_days"] = fmt.Sprintf("%d", *recurringDays)
		params.Metadata["type"] = "subscription"
		
		// Add variant options to subscription metadata for clarity
		if variant.OptionsDisplay.Valid && variant.OptionsDisplay.String != "" {
			params.Metadata["variant_options"] = variant.OptionsDisplay.String
		}
	} else {
		params.Metadata["type"] = "onetime"
		
		// Add variant options to one-time purchase metadata
		if variant.OptionsDisplay.Valid && variant.OptionsDisplay.String != "" {
			params.Metadata["variant_options"] = variant.OptionsDisplay.String
		}
	}

	// Add subscription capability flag
	if variant.IsSubscription {
		params.Metadata["supports_subscription"] = "true"
	} else {
		params.Metadata["supports_subscription"] = "false"
	}

	return stripePrice.New(params)
}

func (s *ProductEventSubscriber) recreateAllStripePricesforProduct(ctx context.Context, product *interfaces.Product) error {
	// When price changes, create new Price objects (Stripe Prices are immutable)
	priceUpdates := make(map[string]string)

	// Create new one-time price
	price, err := s.createSingleStripePrice(product, nil)
	if err != nil {
		return err
	}
	priceUpdates["onetime"] = price.ID

	// Create new subscription prices
	intervals := map[string]int{"14day": 14, "21day": 21, "30day": 30, "60day": 60}
	for interval, days := range intervals {
		price, err := s.createSingleStripePrice(product, &days)
		if err != nil {
			return err
		}
		priceUpdates[interval] = price.ID
	}

	return s.productService.UpdateStripePriceIDs(ctx, product.ID, priceUpdates)
}

func (s *ProductEventSubscriber) updateStripeProduct(ctx context.Context, product *interfaces.Product) error {
	if !product.StripeProductID.Valid {
		return nil // Nothing to update
	}

	params := &stripe.ProductParams{
		Name:        stripe.String(product.Name),
		Description: stripe.String(product.Description.String),
		Active:      stripe.Bool(product.Active),
	}

	_, err := stripeProduct.Update(product.StripeProductID.String, params)
	return err
}

func (s *ProductEventSubscriber) deactivateStripeProduct(ctx context.Context, stripeProductID string) error {
	params := &stripe.ProductParams{
		Active: stripe.Bool(false),
	}

	_, err := stripeProduct.Update(stripeProductID, params)
	return err
}
