// internal/subscriber/product.go
package subscriber

import (
	"context"
	"fmt"
	"log"

	"github.com/dukerupert/freyja/internal/interfaces"
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

func (s *ProductEventSubscriber) createStripeProduct(product *interfaces.Product) (*stripe.Product, error) {
	params := &stripe.ProductParams{
		Name:        stripe.String(product.Name),
		Description: stripe.String(product.Description.String),
		Active:      stripe.Bool(product.Active),
		Metadata: map[string]string{
			"internal_product_id": fmt.Sprintf("%d", product.ID),
		},
	}

	return stripeProduct.New(params)
}

func (s *ProductEventSubscriber) createAllStripePrices(ctx context.Context, product *interfaces.Product) error {
	priceUpdates := make(map[string]string)

	// One-time purchase price
	if !product.StripePriceOnetimeID.Valid {
		price, err := s.createSingleStripePrice(product, nil)
		if err != nil {
			return err
		}
		priceUpdates["onetime"] = price.ID
	}

	// Subscription prices
	intervals := map[string]int{"14day": 14, "21day": 21, "30day": 30, "60day": 60}
	currentPrices := map[string]bool{
		"14day": product.StripePrice14dayID.Valid,
		"21day": product.StripePrice21dayID.Valid,
		"30day": product.StripePrice30dayID.Valid,
		"60day": product.StripePrice60dayID.Valid,
	}

	for interval, days := range intervals {
		if !currentPrices[interval] {
			price, err := s.createSingleStripePrice(product, &days)
			if err != nil {
				return err
			}
			priceUpdates[interval] = price.ID
		}
	}

	// Update all price IDs in database
	if len(priceUpdates) > 0 {
		return s.productService.UpdateStripePriceIDs(ctx, product.ID, priceUpdates)
	}

	return nil
}

func (s *ProductEventSubscriber) createSingleStripePrice(product *interfaces.Product, recurringDays *int) (*stripe.Price, error) {
	params := &stripe.PriceParams{
		Product:    stripe.String(product.StripeProductID.String),
		UnitAmount: stripe.Int64(int64(product.Price)),
		Currency:   stripe.String("usd"),
		Metadata: map[string]string{
			"internal_product_id": fmt.Sprintf("%d", product.ID),
		},
	}

	if recurringDays != nil {
		params.Recurring = &stripe.PriceRecurringParams{
			Interval:      stripe.String("day"),
			IntervalCount: stripe.Int64(int64(*recurringDays)),
		}
		params.Metadata["subscription_days"] = fmt.Sprintf("%d", *recurringDays)
	} else {
		params.Metadata["type"] = "onetime"
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