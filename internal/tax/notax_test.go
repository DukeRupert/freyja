package tax_test

import (
	"context"
	"testing"

	"github.com/dukerupert/freyja/internal/tax"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

func TestNoTaxCalculator_CalculateTax_ReturnsZeroTax(t *testing.T) {
	calc := tax.NewNoTaxCalculator()

	params := tax.TaxParams{
		ShippingAddress: tax.Address{
			Line1:      "123 Main St",
			City:       "Seattle",
			State:      "WA",
			PostalCode: "98101",
			Country:    "US",
		},
		LineItems: []tax.LineItem{
			{
				ProductID:   pgtype.UUID{Valid: true},
				Description: "Ethiopian Yirgacheffe - 12oz",
				Quantity:    2,
				UnitPrice:   1800,
				TotalPrice:  3600,
				TaxCategory: "food",
			},
			{
				ProductID:   pgtype.UUID{Valid: true},
				Description: "Colombian Supremo - 1lb",
				Quantity:    1,
				UnitPrice:   2200,
				TotalPrice:  2200,
				TaxCategory: "food",
			},
		},
		ShippingCents:  500,
		CustomerType:   "wholesale",
		TaxExemptionID: "EX-12345",
	}

	result, err := calc.CalculateTax(context.Background(), params)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int32(0), result.TotalTaxCents, "NoTaxCalculator should always return zero tax")
	assert.Empty(t, result.Breakdown, "NoTaxCalculator should return empty breakdown")
	assert.Empty(t, result.ProviderTxID, "NoTaxCalculator should not have a provider transaction ID")
	assert.False(t, result.IsEstimate, "NoTaxCalculator result should not be marked as estimate")
}

func TestNoTaxCalculator_CalculateTax_EmptyLineItems(t *testing.T) {
	calc := tax.NewNoTaxCalculator()

	params := tax.TaxParams{
		ShippingAddress: tax.Address{
			Line1:      "456 Oak Ave",
			City:       "Portland",
			State:      "OR",
			PostalCode: "97201",
			Country:    "US",
		},
		LineItems:      []tax.LineItem{},
		ShippingCents:  0,
		CustomerType:   "retail",
		TaxExemptionID: "",
	}

	result, err := calc.CalculateTax(context.Background(), params)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int32(0), result.TotalTaxCents, "Should return zero tax even with no items")
}

func TestNoTaxCalculator_CalculateTax_VariousInputs(t *testing.T) {
	calc := tax.NewNoTaxCalculator()

	tests := []struct {
		name          string
		params        tax.TaxParams
		expectedTax   int32
		expectedBreak int
	}{
		{
			name: "retail customer with tax exemption",
			params: tax.TaxParams{
				LineItems: []tax.LineItem{
					{TotalPrice: 5000},
				},
				ShippingCents:  1000,
				CustomerType:   "retail",
				TaxExemptionID: "EXEMPT-001",
			},
			expectedTax:   0,
			expectedBreak: 0,
		},
		{
			name: "wholesale customer no exemption",
			params: tax.TaxParams{
				LineItems: []tax.LineItem{
					{TotalPrice: 10000},
					{TotalPrice: 15000},
				},
				ShippingCents:  2000,
				CustomerType:   "wholesale",
				TaxExemptionID: "",
			},
			expectedTax:   0,
			expectedBreak: 0,
		},
		{
			name: "large order with multiple items",
			params: tax.TaxParams{
				LineItems: []tax.LineItem{
					{TotalPrice: 50000},
					{TotalPrice: 75000},
					{TotalPrice: 100000},
				},
				ShippingCents:  5000,
				CustomerType:   "wholesale",
				TaxExemptionID: "CORP-EXEMPT",
			},
			expectedTax:   0,
			expectedBreak: 0,
		},
		{
			name: "zero shipping cost",
			params: tax.TaxParams{
				LineItems: []tax.LineItem{
					{TotalPrice: 3000},
				},
				ShippingCents:  0,
				CustomerType:   "retail",
				TaxExemptionID: "TAX-FREE",
			},
			expectedTax:   0,
			expectedBreak: 0,
		},
		{
			name: "international address",
			params: tax.TaxParams{
				ShippingAddress: tax.Address{
					Line1:      "10 Downing Street",
					City:       "London",
					State:      "",
					PostalCode: "SW1A 2AA",
					Country:    "GB",
				},
				LineItems: []tax.LineItem{
					{TotalPrice: 8000},
				},
				ShippingCents: 3000,
				CustomerType:  "retail",
			},
			expectedTax:   0,
			expectedBreak: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := calc.CalculateTax(context.Background(), tt.params)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedTax, result.TotalTaxCents)
			assert.Len(t, result.Breakdown, tt.expectedBreak)
			assert.Empty(t, result.ProviderTxID)
			assert.False(t, result.IsEstimate)
		})
	}
}

func TestNoTaxCalculator_CalculateTax_NilContext(t *testing.T) {
	calc := tax.NewNoTaxCalculator()

	params := tax.TaxParams{
		LineItems: []tax.LineItem{
			{TotalPrice: 1000},
		},
		ShippingCents: 500,
	}

	// Use context.TODO() as per Go best practices (context is not used in calculation)
	result, err := calc.CalculateTax(context.TODO(), params)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int32(0), result.TotalTaxCents)
}

func TestNoTaxCalculator_CalculateTax_CanceledContext(t *testing.T) {
	calc := tax.NewNoTaxCalculator()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	params := tax.TaxParams{
		LineItems: []tax.LineItem{
			{TotalPrice: 2000},
		},
		ShippingCents: 300,
	}

	// NoTaxCalculator doesn't use context, so should still work
	result, err := calc.CalculateTax(ctx, params)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int32(0), result.TotalTaxCents)
}

func TestNoTaxCalculator_NewConstructor(t *testing.T) {
	calc := tax.NewNoTaxCalculator()

	assert.NotNil(t, calc, "NewNoTaxCalculator should return non-nil calculator")

	// Verify it implements the Calculator interface
	var _ tax.Calculator = calc
}

func TestNoTaxCalculator_Idempotency(t *testing.T) {
	calc := tax.NewNoTaxCalculator()

	params := tax.TaxParams{
		LineItems: []tax.LineItem{
			{TotalPrice: 5000},
		},
		ShippingCents: 750,
	}

	// Call multiple times with same params
	result1, err1 := calc.CalculateTax(context.Background(), params)
	result2, err2 := calc.CalculateTax(context.Background(), params)
	result3, err3 := calc.CalculateTax(context.Background(), params)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)

	// All results should be identical
	assert.Equal(t, result1.TotalTaxCents, result2.TotalTaxCents)
	assert.Equal(t, result1.TotalTaxCents, result3.TotalTaxCents)
	assert.Equal(t, int32(0), result1.TotalTaxCents)
}
