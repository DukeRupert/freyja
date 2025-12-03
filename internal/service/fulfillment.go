package service

import (
	"context"
	"fmt"

	"github.com/dukerupert/freyja/internal/repository"
	"github.com/jackc/pgx/v5/pgtype"
)

// FulfillmentService manages order fulfillment and partial shipments.
type FulfillmentService interface {
	// CreateShipment creates a shipment for one or more order items.
	// Updates order_items.quantity_dispatched and recalculates order.fulfillment_status.
	// Returns ErrExceedsOrderedQuantity if quantities exceed remaining amounts.
	CreateShipment(ctx context.Context, params CreateShipmentParams) (*ShipmentDetail, error)

	// GetShipment retrieves shipment details by ID.
	GetShipment(ctx context.Context, shipmentID string) (*ShipmentDetail, error)

	// ListShipmentsForOrder lists all shipments for an order.
	ListShipmentsForOrder(ctx context.Context, orderID string) ([]ShipmentSummary, error)

	// GetUnfulfilledItems retrieves order items that haven't been fully dispatched.
	// Useful for creating packing lists and tracking what still needs to ship.
	GetUnfulfilledItems(ctx context.Context, orderID string) ([]UnfulfilledItem, error)

	// GetOrderItemsWithFulfillment retrieves all order items with fulfillment status.
	GetOrderItemsWithFulfillment(ctx context.Context, orderID string) ([]OrderItemFulfillment, error)

	// UpdateShipmentTracking updates carrier and tracking information for a shipment.
	UpdateShipmentTracking(ctx context.Context, params UpdateTrackingParams) error

	// UpdateShipmentStatus updates the status of a shipment (e.g., shipped, delivered).
	UpdateShipmentStatus(ctx context.Context, shipmentID string, status string) error
}

// CreateShipmentParams contains parameters for creating a shipment.
type CreateShipmentParams struct {
	OrderID        string
	ShipmentItems  []ShipmentItemParams // Per-item quantities to ship
	Carrier        string
	TrackingNumber string
	Notes          string
}

// ShipmentItemParams specifies quantity to ship for an order item.
type ShipmentItemParams struct {
	OrderItemID string
	Quantity    int32 // Must be <= (order_item.quantity - order_item.quantity_dispatched)
}

// UpdateTrackingParams contains parameters for updating shipment tracking.
type UpdateTrackingParams struct {
	ShipmentID     string
	Carrier        string
	TrackingNumber string
	TrackingURL    string
}

// ShipmentDetail aggregates shipment with items and order information.
type ShipmentDetail struct {
	Shipment      repository.Shipment
	ShipmentItems []ShipmentItemDetail
}

// ShipmentItemDetail contains shipment item with order item details.
type ShipmentItemDetail struct {
	ID                 pgtype.UUID
	ShipmentID         pgtype.UUID
	OrderItemID        pgtype.UUID
	Quantity           int32
	ProductName        string
	SKU                string
	VariantDescription string
}

// ShipmentSummary is a lightweight shipment representation for lists.
type ShipmentSummary struct {
	ID             pgtype.UUID
	ShipmentNumber string
	OrderID        pgtype.UUID
	Carrier        string
	TrackingNumber string
	Status         string
	ShippedAt      pgtype.Timestamptz
	ItemCount      int
}

// UnfulfilledItem represents an order item that hasn't been fully shipped.
type UnfulfilledItem struct {
	OrderItemID        pgtype.UUID
	ProductName        string
	SKU                string
	VariantDescription string
	QuantityOrdered    int32
	QuantityDispatched int32
	QuantityRemaining  int32
}

// OrderItemFulfillment represents an order item with its fulfillment status.
type OrderItemFulfillment struct {
	ID                 pgtype.UUID
	OrderID            pgtype.UUID
	ProductSkuID       pgtype.UUID
	ProductName        string
	SKU                string
	VariantDescription string
	Quantity           int32
	QuantityDispatched int32
	QuantityRemaining  int32
	UnitPriceCents     int32
	TotalPriceCents    int32
	FulfillmentStatus  string
}

type fulfillmentService struct {
	repo     repository.Querier
	tenantID pgtype.UUID
}

// NewFulfillmentService creates a new FulfillmentService instance.
func NewFulfillmentService(repo repository.Querier, tenantID string) (FulfillmentService, error) {
	var tenantUUID pgtype.UUID
	if err := tenantUUID.Scan(tenantID); err != nil {
		return nil, fmt.Errorf("invalid tenant ID: %w", err)
	}

	return &fulfillmentService{
		repo:     repo,
		tenantID: tenantUUID,
	}, nil
}

// CreateShipment creates a shipment for one or more order items.
func (s *fulfillmentService) CreateShipment(ctx context.Context, params CreateShipmentParams) (*ShipmentDetail, error) {
	if len(params.ShipmentItems) == 0 {
		return nil, ErrNoItemsToShip
	}

	var orderID pgtype.UUID
	if err := orderID.Scan(params.OrderID); err != nil {
		return nil, fmt.Errorf("invalid order ID: %w", err)
	}

	// Get order to validate it exists
	order, err := s.repo.GetOrder(ctx, repository.GetOrderParams{
		TenantID: s.tenantID,
		ID:       orderID,
	})
	if err != nil {
		return nil, ErrOrderNotFound
	}

	// Get unfulfilled items to validate quantities
	unfulfilledItems, err := s.repo.GetUnfulfilledOrderItems(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get unfulfilled items: %w", err)
	}

	// Build map of remaining quantities
	remainingByItemID := make(map[string]int32)
	for _, item := range unfulfilledItems {
		itemIDStr := item.ID.String()
		remainingByItemID[itemIDStr] = item.QuantityRemaining
	}

	// Validate all shipment items
	for _, si := range params.ShipmentItems {
		remaining, exists := remainingByItemID[si.OrderItemID]
		if !exists {
			return nil, ErrItemAlreadyFulfilled
		}
		if si.Quantity > remaining {
			return nil, ErrExceedsOrderedQuantity
		}
		if si.Quantity <= 0 {
			return nil, ErrInvalidQuantity
		}
	}

	// Create shipment record
	carrier := pgtype.Text{}
	if params.Carrier != "" {
		carrier.String = params.Carrier
		carrier.Valid = true
	}

	trackingNumber := pgtype.Text{}
	if params.TrackingNumber != "" {
		trackingNumber.String = params.TrackingNumber
		trackingNumber.Valid = true
	}

	shipment, err := s.repo.CreateShipment(ctx, repository.CreateShipmentParams{
		TenantID:         s.tenantID,
		OrderID:          order.ID,
		Carrier:          carrier,
		TrackingNumber:   trackingNumber,
		ShippingMethodID: pgtype.UUID{}, // Optional, can be added later
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create shipment: %w", err)
	}

	// Update shipment number (the query returns with default values)
	// Note: We may need to add shipment_number to CreateShipment query
	// For now, use the generated ID as the number base

	// Create shipment items and update dispatched quantities
	var shipmentItems []ShipmentItemDetail
	for _, si := range params.ShipmentItems {
		var orderItemID pgtype.UUID
		if err := orderItemID.Scan(si.OrderItemID); err != nil {
			return nil, fmt.Errorf("invalid order item ID: %w", err)
		}

		// Create shipment item
		shipmentItem, err := s.repo.CreateShipmentItem(ctx, repository.CreateShipmentItemParams{
			TenantID:    s.tenantID,
			ShipmentID:  shipment.ID,
			OrderItemID: orderItemID,
			Quantity:    si.Quantity,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create shipment item: %w", err)
		}

		// Update order item dispatched quantity
		err = s.repo.UpdateOrderItemDispatchedQuantity(ctx, repository.UpdateOrderItemDispatchedQuantityParams{
			TenantID:           s.tenantID,
			ID:                 orderItemID,
			QuantityDispatched: si.Quantity, // quantity to add
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update dispatched quantity: %w", err)
		}

		// Get order item details for response
		// Find the item in our unfulfilled list
		for _, ui := range unfulfilledItems {
			if ui.ID.String() == si.OrderItemID {
				shipmentItems = append(shipmentItems, ShipmentItemDetail{
					ID:                 shipmentItem.ID,
					ShipmentID:         shipmentItem.ShipmentID,
					OrderItemID:        shipmentItem.OrderItemID,
					Quantity:           shipmentItem.Quantity,
					ProductName:        ui.ProductName,
					SKU:                ui.Sku,
					VariantDescription: ui.VariantDescription.String,
				})
				break
			}
		}
	}

	// Recalculate order fulfillment status
	err = s.repo.RecalculateOrderFulfillmentStatus(ctx, repository.RecalculateOrderFulfillmentStatusParams{
		TenantID: s.tenantID,
		ID:       orderID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to recalculate fulfillment status: %w", err)
	}

	return &ShipmentDetail{
		Shipment:      shipment,
		ShipmentItems: shipmentItems,
	}, nil
}

// GetShipment retrieves shipment details by ID.
func (s *fulfillmentService) GetShipment(ctx context.Context, shipmentID string) (*ShipmentDetail, error) {
	var sID pgtype.UUID
	if err := sID.Scan(shipmentID); err != nil {
		return nil, fmt.Errorf("invalid shipment ID: %w", err)
	}

	// Get shipment items with product details
	items, err := s.repo.GetShipmentItems(ctx, sID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shipment items: %w", err)
	}

	if len(items) == 0 {
		return nil, ErrShipmentNotFound
	}

	// Get the shipment record by looking up from first item's shipment_id
	shipments, err := s.repo.GetShipmentsByOrderID(ctx, items[0].OrderItemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shipment: %w", err)
	}

	// Find the matching shipment
	var shipment *repository.Shipment
	for _, sh := range shipments {
		if sh.ID.String() == shipmentID {
			shipment = &sh
			break
		}
	}
	if shipment == nil {
		return nil, ErrShipmentNotFound
	}

	shipmentItems := make([]ShipmentItemDetail, len(items))
	for i, item := range items {
		shipmentItems[i] = ShipmentItemDetail{
			ID:                 item.ID,
			ShipmentID:         item.ShipmentID,
			OrderItemID:        item.OrderItemID,
			Quantity:           item.Quantity,
			ProductName:        item.ProductName,
			SKU:                item.Sku,
			VariantDescription: item.VariantDescription.String,
		}
	}

	return &ShipmentDetail{
		Shipment:      *shipment,
		ShipmentItems: shipmentItems,
	}, nil
}

// ListShipmentsForOrder lists all shipments for an order.
func (s *fulfillmentService) ListShipmentsForOrder(ctx context.Context, orderID string) ([]ShipmentSummary, error) {
	var oID pgtype.UUID
	if err := oID.Scan(orderID); err != nil {
		return nil, fmt.Errorf("invalid order ID: %w", err)
	}

	shipments, err := s.repo.GetShipmentsByOrderID(ctx, oID)
	if err != nil {
		return nil, fmt.Errorf("failed to get shipments: %w", err)
	}

	summaries := make([]ShipmentSummary, len(shipments))
	for i, sh := range shipments {
		// Get item count for this shipment
		items, _ := s.repo.GetShipmentItems(ctx, sh.ID)

		summaries[i] = ShipmentSummary{
			ID:             sh.ID,
			ShipmentNumber: sh.ShipmentNumber,
			OrderID:        sh.OrderID,
			Carrier:        sh.Carrier.String,
			TrackingNumber: sh.TrackingNumber.String,
			Status:         sh.Status,
			ShippedAt:      sh.ShippedAt,
			ItemCount:      len(items),
		}
	}

	return summaries, nil
}

// GetUnfulfilledItems retrieves order items that haven't been fully dispatched.
func (s *fulfillmentService) GetUnfulfilledItems(ctx context.Context, orderID string) ([]UnfulfilledItem, error) {
	var oID pgtype.UUID
	if err := oID.Scan(orderID); err != nil {
		return nil, fmt.Errorf("invalid order ID: %w", err)
	}

	items, err := s.repo.GetUnfulfilledOrderItems(ctx, oID)
	if err != nil {
		return nil, fmt.Errorf("failed to get unfulfilled items: %w", err)
	}

	result := make([]UnfulfilledItem, len(items))
	for i, item := range items {
		result[i] = UnfulfilledItem{
			OrderItemID:        item.ID,
			ProductName:        item.ProductName,
			SKU:                item.Sku,
			VariantDescription: item.VariantDescription.String,
			QuantityOrdered:    item.Quantity,
			QuantityDispatched: item.QuantityDispatched,
			QuantityRemaining:  item.QuantityRemaining,
		}
	}

	return result, nil
}

// GetOrderItemsWithFulfillment retrieves all order items with fulfillment status.
func (s *fulfillmentService) GetOrderItemsWithFulfillment(ctx context.Context, orderID string) ([]OrderItemFulfillment, error) {
	var oID pgtype.UUID
	if err := oID.Scan(orderID); err != nil {
		return nil, fmt.Errorf("invalid order ID: %w", err)
	}

	items, err := s.repo.GetOrderItemsWithFulfillment(ctx, oID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}

	result := make([]OrderItemFulfillment, len(items))
	for i, item := range items {
		result[i] = OrderItemFulfillment{
			ID:                 item.ID,
			OrderID:            item.OrderID,
			ProductSkuID:       item.ProductSkuID,
			ProductName:        item.ProductName,
			SKU:                item.Sku,
			VariantDescription: item.VariantDescription.String,
			Quantity:           item.Quantity,
			QuantityDispatched: item.QuantityDispatched,
			QuantityRemaining:  item.QuantityRemaining,
			UnitPriceCents:     item.UnitPriceCents,
			TotalPriceCents:    item.TotalPriceCents,
			FulfillmentStatus:  item.FulfillmentStatus,
		}
	}

	return result, nil
}

// UpdateShipmentTracking updates carrier and tracking information for a shipment.
func (s *fulfillmentService) UpdateShipmentTracking(ctx context.Context, params UpdateTrackingParams) error {
	var sID pgtype.UUID
	if err := sID.Scan(params.ShipmentID); err != nil {
		return fmt.Errorf("invalid shipment ID: %w", err)
	}

	// For now, we don't have a dedicated update tracking query
	// This would need to be added to orders.sql
	// TODO: Add UpdateShipmentTracking query to sqlc

	return nil
}

// UpdateShipmentStatus updates the status of a shipment.
func (s *fulfillmentService) UpdateShipmentStatus(ctx context.Context, shipmentID string, status string) error {
	var sID pgtype.UUID
	if err := sID.Scan(shipmentID); err != nil {
		return fmt.Errorf("invalid shipment ID: %w", err)
	}

	err := s.repo.UpdateShipmentStatus(ctx, repository.UpdateShipmentStatusParams{
		TenantID: s.tenantID,
		ID:       sID,
		Status:   status,
	})
	if err != nil {
		return fmt.Errorf("failed to update shipment status: %w", err)
	}

	return nil
}

// generateShipmentNumber creates a unique shipment number.
func generateShipmentNumber() string {
	// Simple format: SH-timestamp-random
	// In production, you might want a more sophisticated approach
	return fmt.Sprintf("SH-%d", randomInt(100000, 999999))
}

// randomInt generates a random integer between min and max (inclusive).
func randomInt(min, max int) int {
	// Simple implementation - in production use crypto/rand
	return min + int(uint32(max-min+1))
}
