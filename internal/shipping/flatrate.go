package shipping

import (
	"context"
	"time"
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
	return &FlatRateProvider{rates: rates}
}

// GetRates converts flat rates to Rate objects.
func (p *FlatRateProvider) GetRates(ctx context.Context, params RateParams) ([]Rate, error) {
	result := make([]Rate, len(p.rates))
	for i, fr := range p.rates {
		result[i] = Rate{
			RateID:                fr.ServiceCode,
			Carrier:               "Flat Rate",
			ServiceName:           fr.ServiceName,
			ServiceCode:           fr.ServiceCode,
			CostCents:             fr.CostCents,
			EstimatedDaysMin:      fr.DaysMin,
			EstimatedDaysMax:      fr.DaysMax,
			EstimatedDeliveryDate: time.Now().AddDate(0, 0, fr.DaysMax),
		}
	}
	return result, nil
}

// CreateLabel is not supported for flat-rate provider.
func (p *FlatRateProvider) CreateLabel(ctx context.Context, params LabelParams) (*Label, error) {
	return nil, ErrNotImplemented
}

// VoidLabel is not supported for flat-rate provider.
func (p *FlatRateProvider) VoidLabel(ctx context.Context, labelID string) error {
	return ErrNotImplemented
}

// TrackShipment is not supported for flat-rate provider.
func (p *FlatRateProvider) TrackShipment(ctx context.Context, trackingNumber string) (*TrackingInfo, error) {
	return nil, ErrNotImplemented
}
