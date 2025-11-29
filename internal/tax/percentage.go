package tax

import (
	"context"
)

// PercentageCalculator calculates tax using a simple percentage rate.
type PercentageCalculator struct {
	defaultRate float64 // e.g., 0.08 for 8%
}

// NewPercentageCalculator creates a new percentage-based tax calculator.
func NewPercentageCalculator(rate float64) Calculator {
	panic("not implemented")
}

// CalculateTax computes tax on subtotal + shipping using the configured rate.
func (c *PercentageCalculator) CalculateTax(ctx context.Context, params TaxParams) (*TaxResult, error) {
	panic("not implemented")
}
