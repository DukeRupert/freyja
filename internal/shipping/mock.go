package shipping

import (
	"context"
)

// MockProvider is a test implementation of Provider.
type MockProvider struct {
	GetRatesFunc        func(ctx context.Context, params RateParams) ([]Rate, error)
	CreateLabelFunc     func(ctx context.Context, params LabelParams) (*Label, error)
	VoidLabelFunc       func(ctx context.Context, params VoidLabelParams) error
	TrackShipmentFunc   func(ctx context.Context, trackingNumber string) (*TrackingInfo, error)
	ValidateAddressFunc func(ctx context.Context, params ValidateAddressParams) (*AddressValidation, error)
}

// NewMockProvider creates a new mock shipping provider for testing.
func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

// GetRates delegates to the configured function or returns an empty slice.
func (m *MockProvider) GetRates(ctx context.Context, params RateParams) ([]Rate, error) {
	if m.GetRatesFunc != nil {
		return m.GetRatesFunc(ctx, params)
	}
	return []Rate{}, nil
}

// CreateLabel delegates to the configured function or returns an error.
func (m *MockProvider) CreateLabel(ctx context.Context, params LabelParams) (*Label, error) {
	if m.CreateLabelFunc != nil {
		return m.CreateLabelFunc(ctx, params)
	}
	return nil, ErrNotImplemented
}

// VoidLabel delegates to the configured function or returns an error.
func (m *MockProvider) VoidLabel(ctx context.Context, params VoidLabelParams) error {
	if m.VoidLabelFunc != nil {
		return m.VoidLabelFunc(ctx, params)
	}
	return ErrNotImplemented
}

// TrackShipment delegates to the configured function or returns an error.
func (m *MockProvider) TrackShipment(ctx context.Context, trackingNumber string) (*TrackingInfo, error) {
	if m.TrackShipmentFunc != nil {
		return m.TrackShipmentFunc(ctx, trackingNumber)
	}
	return nil, ErrNotImplemented
}

// ValidateAddress delegates to the configured function or returns valid.
func (m *MockProvider) ValidateAddress(ctx context.Context, params ValidateAddressParams) (*AddressValidation, error) {
	if m.ValidateAddressFunc != nil {
		return m.ValidateAddressFunc(ctx, params)
	}
	return &AddressValidation{
		Status:          AddressValid,
		OriginalAddress: params.Address,
	}, nil
}
