// internal/provider/nats.go
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/dukerupert/freyja/internal/shared/interfaces"
	"github.com/nats-io/nats.go"
)

type NATSEventPublisher struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

func NewNATSEventPublisher(natsURL string) (interfaces.EventPublisher, error) {
	// Connect to NATS
	nc, err := nats.Connect(natsURL,
		nats.ReconnectWait(2*time.Second),
		nats.MaxReconnects(-1), // Unlimited reconnects
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Printf("NATS disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("NATS reconnected to %v", nc.ConnectedUrl())
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	publisher := &NATSEventPublisher{
		nc: nc,
		js: js,
	}

	// Initialize streams
	if err := publisher.initializeStreams(); err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to initialize streams: %w", err)
	}

	return publisher, nil
}

// initializeStreams creates the necessary JetStream streams
func (n *NATSEventPublisher) initializeStreams() error {
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
			_, err = n.js.AddStream(streamConfig)
			if err != nil {
				return fmt.Errorf("failed to create stream %s: %w", stream.name, err)
			}
			log.Printf("Created NATS stream: %s", stream.name)
		} else {
			// Stream exists, update it
			_, err = n.js.UpdateStream(streamConfig)
			if err != nil {
				log.Printf("Warning: failed to update stream %s: %v", stream.name, err)
			}
		}
	}

	return nil
}

// PublishEvent publishes a single event to NATS JetStream
func (n *NATSEventPublisher) PublishEvent(ctx context.Context, event interfaces.Event) error {
	// Validate event
	if err := interfaces.ValidateEvent(event); err != nil {
		return fmt.Errorf("invalid event: %w", err)
	}

	// Serialize event to JSON
	eventData, err := json.Marshal(event)
	if err != nil {
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

	// Publish the event
	ack, err := n.js.PublishAsync(subject, eventData, pubOpts...)
	if err != nil {
		return fmt.Errorf("failed to publish event %s: %w", event.ID, err)
	}

	// Wait for acknowledgment with context
	select {
	case <-ack.Ok():
		log.Printf("Event published: %s -> %s", event.ID, subject)
		return nil
	case err := <-ack.Err():
		return fmt.Errorf("event publish failed: %w", err)
	case <-publishCtx.Done():
		return fmt.Errorf("event publish timeout: %w", publishCtx.Err())
	}
}

// PublishEvents publishes multiple events in a batch
func (n *NATSEventPublisher) PublishEvents(ctx context.Context, events []interfaces.Event) error {
	if len(events) == 0 {
		return nil
	}

	// For MVP, publish events sequentially
	// In production, you might want to use batch publishing
	for _, event := range events {
		if err := n.PublishEvent(ctx, event); err != nil {
			return fmt.Errorf("failed to publish event %s in batch: %w", event.ID, err)
		}
	}

	return nil
}

// Subscribe creates a subscription to events of a specific type
func (n *NATSEventPublisher) Subscribe(ctx context.Context, eventType string, handler interfaces.EventHandler) error {
	subject := convertEventTypeToSubject(eventType)

	// Create a durable consumer for this event type
	consumerName := fmt.Sprintf("consumer_%s", sanitizeConsumerName(eventType))

	// Subscribe with manual acknowledgment
	sub, err := n.js.Subscribe(subject, func(msg *nats.Msg) {
		// Parse the event
		var event interfaces.Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("Failed to unmarshal event: %v", err)
			msg.Nak() // Negative acknowledgment
			return
		}

		// Handle the event
		if err := handler(ctx, event); err != nil {
			log.Printf("Event handler failed for %s: %v", event.ID, err)
			msg.Nak() // Negative acknowledgment - will be retried
			return
		}

		// Acknowledge successful processing
		msg.Ack()
		log.Printf("Event processed: %s", event.ID)
	}, nats.Durable(consumerName), nats.ManualAck())

	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", subject, err)
	}

	log.Printf("Subscribed to events: %s", subject)

	// Keep subscription alive until context is done
	go func() {
		<-ctx.Done()
		sub.Unsubscribe()
		log.Printf("Unsubscribed from events: %s", subject)
	}()

	return nil
}

// Close closes the NATS connection
func (n *NATSEventPublisher) Close() error {
	if n.nc != nil {
		n.nc.Close()
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
// Event Handler Examples (for reference)
// =============================================================================

// Example event handlers that could be registered

func OrderCreatedHandler(ctx context.Context, event interfaces.Event) error {
	log.Printf("Processing order created event: %s", event.AggregateID)

	// Extract order data
	orderID, err := interfaces.ExtractAggregateID(event.AggregateID)
	if err != nil {
		return fmt.Errorf("invalid order ID: %w", err)
	}

	// Business logic for order created
	// - Send confirmation email
	// - Reserve inventory
	// - Create fulfillment task
	// - Update analytics

	log.Printf("Order created processing completed for order %d", orderID)
	return nil
}

func PaymentConfirmedHandler(ctx context.Context, event interfaces.Event) error {
	log.Printf("Processing payment confirmed event: %s", event.Data)

	// Business logic for payment confirmed
	// - Update order status
	// - Start fulfillment process
	// - Send receipt email
	// - Update customer loyalty points

	return nil
}

func InventoryReservedHandler(ctx context.Context, event interfaces.Event) error {
	log.Printf("Processing inventory reserved event: %s", event.Data)

	// Business logic for inventory reservation
	// - Update stock levels
	// - Set reservation expiry
	// - Check for low stock alerts

	return nil
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
func NewNATSEventPublisherWithConfig(config NATSConfig) (interfaces.EventPublisher, error) {
	opts := []nats.Option{
		nats.Name(config.ConnectionName),
		nats.ReconnectWait(config.ReconnectWait),
		nats.MaxReconnects(config.MaxReconnects),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Printf("NATS disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("NATS reconnected to %v", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Printf("NATS connection closed")
		}),
	}

	nc, err := nats.Connect(config.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	publisher := &NATSEventPublisher{
		nc: nc,
		js: js,
	}

	if err := publisher.initializeStreams(); err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to initialize streams: %w", err)
	}

	return publisher, nil
}
