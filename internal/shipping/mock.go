package shipping

import (
	"context"
)

// MockProvider is a test implementation of Provider.
type MockProvider struct {
	GetRatesFunc      func(ctx context.Context, params RateParams) ([]Rate, error)
	CreateLabelFunc   func(ctx context.Context, params LabelParams) (*Label, error)
	VoidLabelFunc     func(ctx context.Context, labelID string) error
	TrackShipmentFunc func(ctx context.Context, trackingNumber string) (*TrackingInfo, error)
}

// NewMockProvider creates a new mock shipping provider for testing.
func NewMockProvider() *MockProvider {
	panic("not implemented")
}

// GetRates delegates to the configured function or returns a default result.
func (m *MockProvider) GetRates(ctx context.Context, params RateParams) ([]Rate, error) {
	panic("not implemented")
}

// CreateLabel delegates to the configured function or returns an error.
func (m *MockProvider) CreateLabel(ctx context.Context, params LabelParams) (*Label, error) {
	panic("not implemented")
}

// VoidLabel delegates to the configured function or returns an error.
func (m *MockProvider) VoidLabel(ctx context.Context, labelID string) error {
	panic("not implemented")
}

// TrackShipment delegates to the configured function or returns an error.
func (m *MockProvider) TrackShipment(ctx context.Context, trackingNumber string) (*TrackingInfo, error) {
	panic("not implemented")
}
