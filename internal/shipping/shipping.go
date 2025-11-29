package shipping

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNotImplemented is returned when a method is not yet implemented.
	ErrNotImplemented = errors.New("not implemented")
)

// Provider defines the interface for shipping operations.
// Implementations can integrate with carriers like FedEx, UPS, USPS, etc.
type Provider interface {
	// GetRates returns available shipping options for a shipment.
	GetRates(ctx context.Context, params RateParams) ([]Rate, error)

	// CreateLabel generates a shipping label (post-MVP).
	CreateLabel(ctx context.Context, params LabelParams) (*Label, error)

	// VoidLabel cancels a shipping label (post-MVP).
	VoidLabel(ctx context.Context, labelID string) error

	// TrackShipment gets tracking information (post-MVP).
	TrackShipment(ctx context.Context, trackingNumber string) (*TrackingInfo, error)
}

// RateParams contains parameters for calculating shipping rates.
type RateParams struct {
	OriginAddress      ShippingAddress
	DestinationAddress ShippingAddress
	Packages           []Package
	ServiceTypes       []string // Optional filter for specific service types
}

// ShippingAddress represents a complete shipping address.
type ShippingAddress struct {
	Name       string
	Company    string
	Line1      string
	Line2      string
	City       string
	State      string
	PostalCode string
	Country    string
	Phone      string
	Email      string
}

// Package represents a physical package to be shipped.
type Package struct {
	WeightGrams int32
	LengthCm    int32
	WidthCm     int32
	HeightCm    int32
}

// Rate represents a shipping rate option.
type Rate struct {
	RateID                string
	Carrier               string
	ServiceName           string
	ServiceCode           string
	CostCents             int32
	EstimatedDaysMin      int
	EstimatedDaysMax      int
	EstimatedDeliveryDate time.Time
}

// Label represents a shipping label (post-MVP).
type Label struct {
	LabelID        string
	TrackingNumber string
	LabelURL       string
	CreatedAt      time.Time
}

// LabelParams contains parameters for creating a shipping label (post-MVP).
type LabelParams struct {
	RateID             string
	OriginAddress      ShippingAddress
	DestinationAddress ShippingAddress
	Package            Package
}

// TrackingInfo contains shipment tracking information (post-MVP).
type TrackingInfo struct {
	TrackingNumber string
	Status         string
	Events         []TrackingEvent
	EstimatedDeliveryDate time.Time
}

// TrackingEvent represents a single tracking event (post-MVP).
type TrackingEvent struct {
	Timestamp   time.Time
	Status      string
	Location    string
	Description string
}
