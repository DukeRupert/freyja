package tax

import (
	"context"
)

// MockCalculator is a test implementation of Calculator.
type MockCalculator struct {
	CalculateTaxFunc func(ctx context.Context, params TaxParams) (*TaxResult, error)
}

// NewMockCalculator creates a new mock tax calculator for testing.
func NewMockCalculator() *MockCalculator {
	panic("not implemented")
}

// CalculateTax delegates to the configured function or returns a default result.
func (m *MockCalculator) CalculateTax(ctx context.Context, params TaxParams) (*TaxResult, error) {
	panic("not implemented")
}
