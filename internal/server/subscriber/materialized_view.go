// internal/server/subscriber/materialized_view.go
package subscriber

import (
	"context"
	"fmt"
	"log"

	"github.com/dukerupert/freyja/internal/shared/interfaces"
)

type MaterializedViewSubscriber struct {
	productRepo interfaces.ProductRepository
	events      interfaces.EventPublisher
}

func NewMaterializedViewSubscriber(
	productRepo interfaces.ProductRepository,
	events interfaces.EventPublisher,
) *MaterializedViewSubscriber {
	return &MaterializedViewSubscriber{
		productRepo: productRepo,
		events:      events,
	}
}

// Start subscribes to all events that should trigger materialized view refresh
func (s *MaterializedViewSubscriber) Start(ctx context.Context) error {
	// Product events that affect the materialized view
	productEvents := []string{
		interfaces.EventProductCreated,
		interfaces.EventProductUpdated,
		interfaces.EventProductActivated,
		interfaces.EventProductDeactivated,
		interfaces.EventProductDeleted,
	}

	// Variant events that affect the materialized view
	variantEvents := []string{
		interfaces.EventVariantCreated,
		interfaces.EventVariantUpdated,
		interfaces.EventVariantActivated,
		interfaces.EventVariantDeactivated,
		interfaces.EventVariantDeleted,
		interfaces.EventVariantStockUpdated,
		interfaces.EventVariantPriceUpdated,
	}

	// Subscribe to all product events
	for _, eventType := range productEvents {
		if err := s.events.Subscribe(ctx, eventType, s.handleProductEvent); err != nil {
			return fmt.Errorf("failed to subscribe to %s events: %w", eventType, err)
		}
	}

	// Subscribe to all variant events
	for _, eventType := range variantEvents {
		if err := s.events.Subscribe(ctx, eventType, s.handleVariantEvent); err != nil {
			return fmt.Errorf("failed to subscribe to %s events: %w", eventType, err)
		}
	}

	log.Println("✅ Materialized view subscriber started")
	return nil
}

// handleProductEvent refreshes the materialized view when product changes occur
func (s *MaterializedViewSubscriber) handleProductEvent(ctx context.Context, event interfaces.Event) error {
	log.Printf("Refreshing product_stock_summary after product event: %s (ID: %s)", event.Type, event.AggregateID)
	
	if err := s.refreshMaterializedView(ctx, event.Type); err != nil {
		return fmt.Errorf("failed to refresh materialized view after product event %s: %w", event.Type, err)
	}

	return nil
}

// handleVariantEvent refreshes the materialized view when variant changes occur
func (s *MaterializedViewSubscriber) handleVariantEvent(ctx context.Context, event interfaces.Event) error {
	log.Printf("Refreshing product_stock_summary after variant event: %s (ID: %s)", event.Type, event.AggregateID)
	
	if err := s.refreshMaterializedView(ctx, event.Type); err != nil {
		return fmt.Errorf("failed to refresh materialized view after variant event %s: %w", event.Type, err)
	}

	return nil
}

// refreshMaterializedView performs the actual refresh operation
func (s *MaterializedViewSubscriber) refreshMaterializedView(ctx context.Context, eventType string) error {
	log.Printf("Starting materialized view refresh triggered by: %s", eventType)
	
	err := s.productRepo.RefreshProductStockSummary(ctx)
	if err != nil {
		log.Printf("❌ Failed to refresh product_stock_summary: %v", err)
		return err
	}

	log.Printf("✅ Successfully refreshed product_stock_summary after %s event", eventType)
	return nil
}