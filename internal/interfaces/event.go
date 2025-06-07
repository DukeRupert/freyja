// internal/interfaces/events.go
package interfaces

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// =============================================================================
// Event Publisher Interface
// =============================================================================

type EventPublisher interface {
	// Publish a single event
	PublishEvent(ctx context.Context, event Event) error

	// Publish multiple events in a batch
	PublishEvents(ctx context.Context, events []Event) error

	// Subscribe to events (for consumers)
	Subscribe(ctx context.Context, eventType string, handler EventHandler) error

	// Close the publisher
	Close() error
}

// Event represents a domain event in our system
type Event struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	AggregateID string                 `json:"aggregate_id"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time              `json:"timestamp"`
	Version     int                    `json:"version,omitempty"`
}

// EventHandler is a function that processes events
type EventHandler func(ctx context.Context, event Event) error

// =============================================================================
// Common Event Types (for reference)
// =============================================================================

const (
	// Cart events
	EventCartItemAdded   = "cart.item_added"
	EventCartItemUpdated = "cart.item_updated"
	EventCartItemRemoved = "cart.item_removed"
	EventCartCleared     = "cart.cleared"

	// Checkout events
	EventCheckoutInitiated      = "checkout.initiated"
	EventCheckoutSessionCreated = "checkout.session_created"

	// Payment events
	EventPaymentProcessing = "payment.processing"
	EventPaymentConfirmed  = "payment.confirmed"
	EventPaymentFailed     = "payment.failed"

	// Order events
	EventOrderCreated   = "order.created"
	EventOrderConfirmed = "order.confirmed"
	EventOrderCancelled = "order.cancelled"
	EventOrderShipped   = "order.shipped"
	EventOrderDelivered = "order.delivered"

	// Inventory events
	EventInventoryReserved  = "inventory.reserved"
	EventInventoryCommitted = "inventory.committed"
	EventInventoryReleased  = "inventory.released"

	// Customer events
	EventCustomerCreated       = "customer.created"
	EventCustomerUpdated       = "customer.updated"
	EventCustomerStripeEnsured = "customer.stripe_ensured"

	// Product events
	EventProductCreated     = "product.created"
	EventProductUpdated     = "product.updated"
	EventProductDeactivated = "product.deactivated"
	EventProductStockLow    = "product.stock_low"
	EventProductStripeSync  = "product.stripe_sync_requested"
)

// =============================================================================
// Event Data Structures (for type safety)
// =============================================================================

type CartItemAddedData struct {
	CartID    int32 `json:"cart_id"`
	ProductID int32 `json:"product_id"`
	Quantity  int32 `json:"quantity"`
	Price     int32 `json:"price"`
}

type PaymentConfirmedData struct {
	PaymentIntentID string `json:"payment_intent_id"`
	Amount          int32  `json:"amount"`
	CustomerID      int32  `json:"customer_id"`
	OrderID         *int32 `json:"order_id,omitempty"`
}

type OrderCreatedData struct {
	OrderID    int32  `json:"order_id"`
	CustomerID int32  `json:"customer_id"`
	Total      int32  `json:"total"`
	ItemCount  int    `json:"item_count"`
	PaymentID  string `json:"payment_id,omitempty"`
}

type InventoryReservedData struct {
	ProductID     int32     `json:"product_id"`
	Quantity      int32     `json:"quantity"`
	ReservationID string    `json:"reservation_id"`
	ExpiresAt     time.Time `json:"expires_at"`
}

// =============================================================================
// Event Builder Helpers
// =============================================================================

// BuildEvent creates a new event with generated ID and timestamp
func BuildEvent(eventType, aggregateID string, data map[string]interface{}) Event {
	return Event{
		ID:          generateEventID(),
		Type:        eventType,
		AggregateID: aggregateID,
		Data:        data,
		Timestamp:   time.Now(),
		Version:     1,
	}
}

// BuildCartEvent creates a cart-related event
func BuildCartEvent(eventType string, cartID int32, data map[string]interface{}) Event {
	return BuildEvent(eventType, fmt.Sprintf("cart:%d", cartID), data)
}

// BuildOrderEvent creates an order-related event
func BuildOrderEvent(eventType string, orderID int32, data map[string]interface{}) Event {
	return BuildEvent(eventType, fmt.Sprintf("order:%d", orderID), data)
}

// BuildCustomerEvent creates a customer-related event
func BuildCustomerEvent(eventType string, customerID int32, data map[string]interface{}) Event {
	return BuildEvent(eventType, fmt.Sprintf("customer:%d", customerID), data)
}

// BuildProductEvent creates a product-related event
func BuildProductEvent(eventType string, productID int32, data map[string]interface{}) Event {
	return BuildEvent(eventType, fmt.Sprintf("product:%d", productID), data)
}

// Helper function to generate event IDs
func generateEventID() string {
	// For MVP, use timestamp-based ID
	// In production, you might want to use UUID
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}

// =============================================================================
// Event Validation
// =============================================================================

// ValidateEvent checks if an event has required fields
func ValidateEvent(event Event) error {
	if event.ID == "" {
		return fmt.Errorf("event ID is required")
	}
	if event.Type == "" {
		return fmt.Errorf("event type is required")
	}
	if event.AggregateID == "" {
		return fmt.Errorf("aggregate ID is required")
	}
	if event.Timestamp.IsZero() {
		return fmt.Errorf("event timestamp is required")
	}
	return nil
}

// IsEventType checks if an event is of a specific type
func IsEventType(event Event, eventType string) bool {
	return event.Type == eventType
}

// IsAggregateType checks if an event belongs to a specific aggregate type
func IsAggregateType(event Event, aggregateType string) bool {
	return strings.HasPrefix(event.AggregateID, aggregateType+":")
}

// ExtractAggregateID extracts the numeric ID from an aggregate ID
func ExtractAggregateID(aggregateID string) (int32, error) {
	parts := strings.Split(aggregateID, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid aggregate ID format: %s", aggregateID)
	}

	id, err := strconv.ParseInt(parts[1], 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid aggregate ID: %w", err)
	}

	return int32(id), nil
}
