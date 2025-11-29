package tax

import (
	"context"
	"math"
)

// PercentageCalculator calculates tax using a simple percentage rate.
type PercentageCalculator struct {
	defaultRate float64 // e.g., 0.08 for 8%
}

// NewPercentageCalculator creates a new percentage-based tax calculator.
func NewPercentageCalculator(rate float64) Calculator {
	return &PercentageCalculator{defaultRate: rate}
}

// CalculateTax computes tax on subtotal + shipping using the configured rate.
func (c *PercentageCalculator) CalculateTax(ctx context.Context, params TaxParams) (*TaxResult, error) {
	subtotal := int32(0)
	for _, item := range params.LineItems {
		subtotal += item.TotalPrice
	}

	taxableAmount := subtotal + params.ShippingCents
	taxAmount := int32(math.Round(float64(taxableAmount) * c.defaultRate))

	return &TaxResult{
		TotalTaxCents: taxAmount,
		Breakdown: []TaxBreakdown{
			{
				Jurisdiction: "state",
				Name:         "Default Sales Tax",
				Rate:         c.defaultRate,
				AmountCents:  taxAmount,
			},
		},
		ProviderTxID: "",
		IsEstimate:   false,
	}, nil
}
