// internal/server/subscriber/materialized_view.go
package subscriber

import (
	"context"
	"fmt"

	"github.com/dukerupert/freyja/internal/shared/interfaces"
	"github.com/rs/zerolog"
)

type MaterializedViewSubscriber struct {
	productRepo interfaces.ProductRepository
	events      interfaces.EventPublisher
	logger      zerolog.Logger
}

func NewMaterializedViewSubscriber(
	productRepo interfaces.ProductRepository,
	events interfaces.EventPublisher,
	logger zerolog.Logger,
) *MaterializedViewSubscriber {
	return &MaterializedViewSubscriber{
		productRepo: productRepo,
		events:      events,
		logger:      logger.With().Str("component", "MaterializedViewSubscriber").Logger(),
	}
}

// Start subscribes to all events that should trigger materialized view refresh
func (s *MaterializedViewSubscriber) Start(ctx context.Context) error {
	// Product events that affect the materialized view
	productEvents := []string{
		interfaces.EventProductCreated,
		interfaces.EventProductActivated,
		interfaces.EventProductDeleted,
	}

	// Variant events that affect the materialized view
	variantEvents := []string{
		interfaces.EventVariantActivated,
		interfaces.EventVariantDeleted,
		interfaces.EventVariantStockUpdated,
		interfaces.EventVariantPriceUpdated,
	}

	// Subscribe to all product events
	for _, eventType := range productEvents {
		if err := s.events.Subscribe(ctx, eventType, s.handleProductEvent); err != nil {
			return fmt.Errorf("failed to subscribe to %s events: %w", eventType, err)
		}
		s.logger.Debug().
			Str("event_type", eventType).
			Msg("Subscribed to product event")
	}

	// Subscribe to all variant events
	for _, eventType := range variantEvents {
		if err := s.events.Subscribe(ctx, eventType, s.handleVariantEvent); err != nil {
			return fmt.Errorf("failed to subscribe to %s events: %w", eventType, err)
		}
		s.logger.Debug().
			Str("event_type", eventType).
			Msg("Subscribed to variant event")
	}

	s.logger.Info().Msg("[OK] Materialized view subscriber started")
	return nil
}

// handleProductEvent refreshes the materialized view when product changes occur
func (s *MaterializedViewSubscriber) handleProductEvent(ctx context.Context, event interfaces.Event) error {
	s.logger.Info().
		Str("event_type", event.Type).
		Str("aggregate_id", event.AggregateID).
		Msg("Refreshing product_stock_summary after product event")
	
	if err := s.refreshMaterializedView(ctx, event.Type); err != nil {
		s.logger.Error().
			Err(err).
			Str("event_type", event.Type).
			Str("aggregate_id", event.AggregateID).
			Msg("Failed to refresh materialized view after product event")
		return fmt.Errorf("failed to refresh materialized view after product event %s: %w", event.Type, err)
	}

	return nil
}

// handleVariantEvent refreshes the materialized view when variant changes occur
func (s *MaterializedViewSubscriber) handleVariantEvent(ctx context.Context, event interfaces.Event) error {
	s.logger.Info().
		Str("event_type", event.Type).
		Str("aggregate_id", event.AggregateID).
		Msg("Refreshing product_stock_summary after variant event")
	
	if err := s.refreshMaterializedView(ctx, event.Type); err != nil {
		s.logger.Error().
			Err(err).
			Str("event_type", event.Type).
			Str("aggregate_id", event.AggregateID).
			Msg("Failed to refresh materialized view after variant event")
		return fmt.Errorf("failed to refresh materialized view after variant event %s: %w", event.Type, err)
	}

	return nil
}

// refreshMaterializedView performs the actual refresh operation
func (s *MaterializedViewSubscriber) refreshMaterializedView(ctx context.Context, eventType string) error {
	s.logger.Debug().
		Str("triggered_by", eventType).
		Msg("Starting materialized view refresh")
	
	err := s.productRepo.RefreshProductStockSummary(ctx)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("triggered_by", eventType).
			Msg("Failed to refresh product_stock_summary")
		return err
	}

	s.logger.Info().
		Str("triggered_by", eventType).
		Msg("[OK] Successfully refreshed product_stock_summary")
	return nil
}