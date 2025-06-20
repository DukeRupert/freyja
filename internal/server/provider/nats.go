// internal/server/provider/nats.go
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dukerupert/freyja/internal/shared/interfaces"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
)

// =============================================================================
// NATS Event Publisher Implementation
// =============================================================================

type NATSEventPublisher struct {
	nc     *nats.Conn
	js     nats.JetStreamContext
	logger zerolog.Logger
}

// NewNATSEventPublisher creates a new NATS-based event publisher
func NewNATSEventPublisher(natsURL string, logger zerolog.Logger) (interfaces.EventPublisher, error) {
	eventLogger := logger.With().Str("component", "NATSEventPublisher").Logger()

	eventLogger.Info().
		Str("nats_url", natsURL).
		Msg("Connecting to NATS")

	// Configure NATS connection options
	opts := []nats.Option{
		nats.Name("freyja-event-publisher"),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(10),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			eventLogger.Error().
				Err(err).
				Msg("NATS disconnected")
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			eventLogger.Info().
				Str("connected_url", nc.ConnectedUrl()).
				Msg("NATS reconnected")
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			eventLogger.Info().Msg("NATS connection closed")
		}),
	}

	nc, err := nats.Connect(natsURL, opts...)
	if err != nil {
		eventLogger.Error().
			Err(err).
			Str("nats_url", natsURL).
			Msg("Failed to connect to NATS")
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		eventLogger.Error().
			Err(err).
			Msg("Failed to create JetStream context")
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	publisher := &NATSEventPublisher{
		nc:     nc,
		js:     js,
		logger: eventLogger,
	}

	if err := publisher.initializeStreams(); err != nil {
		nc.Close()
		eventLogger.Error().
			Err(err).
			Msg("Failed to initialize streams")
		return nil, fmt.Errorf("failed to initialize streams: %w", err)
	}

	eventLogger.Info().Msg("[OK] NATS Event Publisher initialized")
	return publisher, nil
}

// initializeStreams creates the necessary NATS JetStream streams
func (n *NATSEventPublisher) initializeStreams() error {
	n.logger.Debug().Msg("Initializing NATS streams")

	streams := []struct {
		name     string
		subjects []string
		maxAge   time.Duration
	}{
		{
			name:     "ORDERS",
			subjects: []string{"order.*", "payment.*", "checkout.*"},
			maxAge:   24 * time.Hour * 365, // 1 year retention for business events
		},
		{
			name:     "CART",
			subjects: []string{"cart.*"},
			maxAge:   24 * time.Hour * 30, // 30 days for cart events
		},
		{
			name:     "INVENTORY",
			subjects: []string{"inventory.*", "product.*", "variant.*"},
			maxAge:   24 * time.Hour * 90, // 90 days for inventory events
		},
		{
			name:     "CUSTOMER",
			subjects: []string{"customer.*"},
			maxAge:   24 * time.Hour * 365, // 1 year for customer events
		},
	}

	for _, stream := range streams {
		streamConfig := &nats.StreamConfig{
			Name:     stream.name,
			Subjects: stream.subjects,
			MaxAge:   stream.maxAge,
			Storage:  nats.FileStorage, // Persistent storage
			Replicas: 1,                // Single replica for MVP
		}

		// Try to get existing stream
		_, err := n.js.StreamInfo(stream.name)
		if err != nil {
			// Stream doesn't exist, create it
			createdStream, err := n.js.AddStream(streamConfig)
			if err != nil {
				n.logger.Error().
					Err(err).
					Str("stream_name", stream.name).
					Strs("subjects", stream.subjects).
					Msg("Failed to create stream")
				return fmt.Errorf("failed to create stream %s: %w", stream.name, err)
			}
			n.logger.Info().
				Str("stream_name", stream.name).
				Strs("subjects", stream.subjects).
				Dur("max_age", stream.maxAge).
				Uint64("messages", createdStream.State.Msgs).
				Msg("[OK] Created NATS stream")
		} else {
			// Stream exists, update it
			updatedStream, err := n.js.UpdateStream(streamConfig)
			if err != nil {
				n.logger.Warn().
					Err(err).
					Str("stream_name", stream.name).
					Msg("Failed to update stream")
			} else {
				n.logger.Debug().
					Str("stream_name", stream.name).
					Uint64("messages", updatedStream.State.Msgs).
					Int("consumers", updatedStream.State.Consumers).
					Msg("Updated existing NATS stream")
			}
		}
	}

	n.logger.Info().
		Int("stream_count", len(streams)).
		Msg("[OK] NATS streams initialized")
	return nil
}

// PublishEvent publishes a single event to NATS JetStream
func (n *NATSEventPublisher) PublishEvent(ctx context.Context, event interfaces.Event) error {
	logger := n.logger.With().
		Str("event_id", event.ID).
		Str("event_type", event.Type).
		Str("aggregate_id", event.AggregateID).
		Logger()

	// Validate event
	if err := interfaces.ValidateEvent(event); err != nil {
		logger.Error().
			Err(err).
			Msg("Invalid event validation failed")
		return fmt.Errorf("invalid event: %w", err)
	}

	// Serialize event to JSON
	eventData, err := json.Marshal(event)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to marshal event to JSON")
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Convert event type to NATS subject
	subject := convertEventTypeToSubject(event.Type)

	// Publish with context
	pubOpts := []nats.PubOpt{
		nats.MsgId(event.ID), // Deduplication
	}

	// Set a timeout for publishing
	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	logger.Debug().
		Str("subject", subject).
		Int("data_size_bytes", len(eventData)).
		Msg("Publishing event to NATS")

	// Publish the event
	ack, err := n.js.PublishAsync(subject, eventData, pubOpts...)
	if err != nil {
		logger.Error().
			Err(err).
			Str("subject", subject).
			Msg("Failed to publish event")
		return fmt.Errorf("failed to publish event %s: %w", event.ID, err)
	}

	// Wait for acknowledgment with context
	select {
	case <-ack.Ok():
		logger.Info().
			Str("subject", subject).
			Msg("[OK] Event published successfully")
		return nil
	case err := <-ack.Err():
		logger.Error().
			Err(err).
			Str("subject", subject).
			Msg("Event publish acknowledgment failed")
		return fmt.Errorf("event publish failed: %w", err)
	case <-publishCtx.Done():
		logger.Error().
			Err(publishCtx.Err()).
			Str("subject", subject).
			Msg("Event publish timeout")
		return fmt.Errorf("event publish timeout: %w", publishCtx.Err())
	}
}

// PublishEvents publishes multiple events in a batch
func (n *NATSEventPublisher) PublishEvents(ctx context.Context, events []interfaces.Event) error {
	if len(events) == 0 {
		n.logger.Debug().Msg("No events to publish in batch")
		return nil
	}

	logger := n.logger.With().
		Int("batch_size", len(events)).
		Logger()

	logger.Info().Msg("Publishing event batch")

	// For MVP, publish events sequentially
	// In production, you might want to use batch publishing
	successCount := 0
	var lastError error

	for i, event := range events {
		if err := n.PublishEvent(ctx, event); err != nil {
			logger.Error().
				Err(err).
				Str("event_id", event.ID).
				Str("event_type", event.Type).
				Int("batch_index", i).
				Msg("Failed to publish event in batch")
			lastError = err
			continue
		}
		successCount++
	}

	if successCount == 0 && lastError != nil {
		logger.Error().
			Err(lastError).
			Int("failed_count", len(events)).
			Msg("Failed to publish entire event batch")
		return fmt.Errorf("failed to publish any events in batch: %w", lastError)
	}

	if successCount < len(events) {
		logger.Warn().
			Int("success_count", successCount).
			Int("total_count", len(events)).
			Int("failed_count", len(events)-successCount).
			Msg("Partial success publishing event batch")
	} else {
		logger.Info().
			Int("success_count", successCount).
			Msg("[OK] Event batch published successfully")
	}

	return lastError // Return last error for partial failures
}

// Subscribe creates a subscription to events of a specific type
func (n *NATSEventPublisher) Subscribe(ctx context.Context, eventType string, handler interfaces.EventHandler) error {
	subject := convertEventTypeToSubject(eventType)
	consumerName := fmt.Sprintf("consumer_%s", sanitizeConsumerName(eventType))

	logger := n.logger.With().
		Str("event_type", eventType).
		Str("subject", subject).
		Str("consumer_name", consumerName).
		Logger()

	logger.Info().Msg("Creating NATS subscription")

	// Subscribe with manual acknowledgment
	sub, err := n.js.Subscribe(subject, func(msg *nats.Msg) {
		msgLogger := logger.With().
			Str("msg_subject", msg.Subject).
			Str("msg_reply", msg.Reply).
			Int("data_size", len(msg.Data)).
			Logger()

		msgLogger.Debug().Msg("Received NATS message")

		// Parse the event
		var event interfaces.Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			msgLogger.Error().
				Err(err).
				Str("raw_data", string(msg.Data)).
				Msg("Failed to unmarshal event - sending NAK")
			msg.Nak() // Negative acknowledgment
			return
		}

		eventLogger := msgLogger.With().
			Str("event_id", event.ID).
			Str("event_type", event.Type).
			Str("aggregate_id", event.AggregateID).
			Time("event_timestamp", event.Timestamp).
			Logger()

		eventLogger.Debug().Msg("Processing event")

		// Handle the event
		if err := handler(ctx, event); err != nil {
			eventLogger.Error().
				Err(err).
				Msg("Event handler failed - sending NAK for retry")
			msg.Nak() // Negative acknowledgment - will be retried
			return
		}

		// Acknowledge successful processing
		msg.Ack()
		eventLogger.Info().Msg("[OK] Event processed successfully")
	}, nats.Durable(consumerName), nats.ManualAck())

	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to create NATS subscription")
		return fmt.Errorf("failed to subscribe to %s: %w", subject, err)
	}

	logger.Info().Msg("[OK] NATS subscription created")

	// Keep subscription alive until context is done
	go func() {
		<-ctx.Done()
		sub.Unsubscribe()
		logger.Info().Msg("NATS subscription closed")
	}()

	return nil
}

// Close closes the NATS connection
func (n *NATSEventPublisher) Close() error {
	if n.nc != nil {
		n.logger.Info().Msg("Closing NATS connection")
		n.nc.Close()
		n.logger.Info().Msg("[OK] NATS connection closed")
	}
	return nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// convertEventTypeToSubject converts event types to NATS subjects
func convertEventTypeToSubject(eventType string) string {
	// Convert dots to NATS subject hierarchy
	// e.g., "order.created" -> "order.created"
	// e.g., "cart.item_added" -> "cart.item_added"
	return eventType
}

// sanitizeConsumerName creates a valid NATS consumer name
func sanitizeConsumerName(eventType string) string {
	// Replace dots and special characters for consumer names
	name := eventType
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, "-", "_")
	return name
}

// =============================================================================
// NATS Configuration Helper
// =============================================================================

// NATSConfig holds NATS configuration
type NATSConfig struct {
	URL            string
	MaxReconnects  int
	ReconnectWait  time.Duration
	ConnectionName string
}

// NewNATSEventPublisherWithConfig creates a NATS publisher with custom config
func NewNATSEventPublisherWithConfig(config NATSConfig, logger zerolog.Logger) (interfaces.EventPublisher, error) {
	eventLogger := logger.With().Str("component", "NATSEventPublisher").Logger()

	opts := []nats.Option{
		nats.Name(config.ConnectionName),
		nats.ReconnectWait(config.ReconnectWait),
		nats.MaxReconnects(config.MaxReconnects),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			eventLogger.Error().
				Err(err).
				Msg("NATS disconnected")
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			eventLogger.Info().
				Str("connected_url", nc.ConnectedUrl()).
				Msg("NATS reconnected")
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			eventLogger.Info().Msg("NATS connection closed")
		}),
	}

	nc, err := nats.Connect(config.URL, opts...)
	if err != nil {
		eventLogger.Error().
			Err(err).
			Str("nats_url", config.URL).
			Msg("Failed to connect to NATS")
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		eventLogger.Error().
			Err(err).
			Msg("Failed to create JetStream context")
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	publisher := &NATSEventPublisher{
		nc:     nc,
		js:     js,
		logger: eventLogger,
	}

	if err := publisher.initializeStreams(); err != nil {
		nc.Close()
		eventLogger.Error().
			Err(err).
			Msg("Failed to initialize streams")
		return nil, fmt.Errorf("failed to initialize streams: %w", err)
	}

	eventLogger.Info().Msg("[OK] NATS Event Publisher initialized with custom config")
	return publisher, nil
}