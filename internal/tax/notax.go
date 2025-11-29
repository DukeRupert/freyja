package tax

import "context"

// NoTaxCalculator returns zero tax for all calculations.
// Used for tax-exempt customers or wholesale accounts.
type NoTaxCalculator struct{}

// NewNoTaxCalculator creates a new no-tax calculator.
func NewNoTaxCalculator() Calculator {
	panic("not implemented")
}

// CalculateTax always returns zero tax.
func (c *NoTaxCalculator) CalculateTax(ctx context.Context, params TaxParams) (*TaxResult, error) {
	panic("not implemented")
}
