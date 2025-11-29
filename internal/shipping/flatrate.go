package shipping

import (
	"context"
)

// FlatRateProvider returns predefined flat-rate shipping options.
// Used for MVP when real carrier integration is not needed.
type FlatRateProvider struct {
	rates []FlatRate
}

// FlatRate defines a single flat-rate shipping option.
type FlatRate struct {
	ServiceName string
	ServiceCode string
	CostCents   int32
	DaysMin     int
	DaysMax     int
}

// NewFlatRateProvider creates a new flat-rate shipping provider.
func NewFlatRateProvider(rates []FlatRate) Provider {
	panic("not implemented")
}

// GetRates converts flat rates to Rate objects.
func (p *FlatRateProvider) GetRates(ctx context.Context, params RateParams) ([]Rate, error) {
	panic("not implemented")
}

// CreateLabel is not supported for flat-rate provider.
func (p *FlatRateProvider) CreateLabel(ctx context.Context, params LabelParams) (*Label, error) {
	panic("not implemented")
}

// VoidLabel is not supported for flat-rate provider.
func (p *FlatRateProvider) VoidLabel(ctx context.Context, labelID string) error {
	panic("not implemented")
}

// TrackShipment is not supported for flat-rate provider.
func (p *FlatRateProvider) TrackShipment(ctx context.Context, trackingNumber string) (*TrackingInfo, error) {
	panic("not implemented")
}
