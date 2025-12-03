package tax_test

import (
	"context"
	"math"
	"testing"

	"github.com/dukerupert/freyja/internal/tax"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

// Test_PercentageCalculator_SpecificationExample validates the exact example from the spec:
// Subtotal $25 (2500 cents) + Shipping $5 (500 cents) * 8% (0.08) = $2.40 (240 cents)
func Test_PercentageCalculator_SpecificationExample(t *testing.T) {
	calc := tax.NewPercentageCalculator(0.08)

	params := tax.TaxParams{
		LineItems: []tax.LineItem{
			{
				ProductID:   pgtype.UUID{Valid: true},
				Description: "Test Product",
				Quantity:    1,
				UnitPrice:   2500,
				TotalPrice:  2500,
				TaxCategory: "general_merchandise",
			},
		},
		ShippingCents: 500,
	}

	result, err := calc.CalculateTax(context.Background(), params)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int32(240), result.TotalTaxCents, "(2500 + 500) * 0.08 = 240 cents")
	assert.Len(t, result.Breakdown, 1, "Should have exactly one breakdown entry")
	assert.Equal(t, "state", result.Breakdown[0].Jurisdiction)
	assert.Equal(t, "Default Sales Tax", result.Breakdown[0].Name)
	assert.Equal(t, 0.08, result.Breakdown[0].Rate)
	assert.Equal(t, int32(240), result.Breakdown[0].AmountCents)
	assert.Empty(t, result.ProviderTxID, "No external provider for percentage calculator")
	assert.False(t, result.IsEstimate, "Percentage calculator provides exact amounts")
}

// Test_PercentageCalculator_DifferentTaxRates validates calculation accuracy across various rates
func Test_PercentageCalculator_DifferentTaxRates(t *testing.T) {
	tests := []struct {
		name        string
		rate        float64
		subtotal    int32
		shipping    int32
		expectedTax int32
		explanation string
	}{
		{
			name:        "zero percent rate",
			rate:        0.0,
			subtotal:    10000,
			shipping:    500,
			expectedTax: 0,
			explanation: "(10000 + 500) * 0.00 = 0",
		},
		{
			name:        "five percent rate",
			rate:        0.05,
			subtotal:    10000,
			shipping:    0,
			expectedTax: 500,
			explanation: "10000 * 0.05 = 500",
		},
		{
			name:        "eight percent rate",
			rate:        0.08,
			subtotal:    5000,
			shipping:    1000,
			expectedTax: 480,
			explanation: "(5000 + 1000) * 0.08 = 480",
		},
		{
			name:        "eight point five percent rate",
			rate:        0.085,
			subtotal:    10000,
			shipping:    0,
			expectedTax: 850,
			explanation: "10000 * 0.085 = 850",
		},
		{
			name:        "ten percent rate",
			rate:        0.10,
			subtotal:    7500,
			shipping:    500,
			expectedTax: 800,
			explanation: "(7500 + 500) * 0.10 = 800",
		},
		{
			name:        "twelve point five percent rate",
			rate:        0.125,
			subtotal:    8000,
			shipping:    0,
			expectedTax: 1000,
			explanation: "8000 * 0.125 = 1000",
		},
		{
			name:        "very small rate",
			rate:        0.001,
			subtotal:    100000,
			shipping:    0,
			expectedTax: 100,
			explanation: "100000 * 0.001 = 100",
		},
		{
			name:        "one hundred percent rate edge case",
			rate:        1.0,
			subtotal:    5000,
			shipping:    0,
			expectedTax: 5000,
			explanation: "5000 * 1.0 = 5000 (edge case: tax equals subtotal)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calc := tax.NewPercentageCalculator(tt.rate)

			params := tax.TaxParams{
				LineItems: []tax.LineItem{
					{TotalPrice: tt.subtotal},
				},
				ShippingCents: tt.shipping,
			}

			result, err := calc.CalculateTax(context.Background(), params)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedTax, result.TotalTaxCents, tt.explanation)
			assert.Equal(t, tt.rate, result.Breakdown[0].Rate)
		})
	}
}

// Test_PercentageCalculator_RoundingBehavior validates math.Round behavior for edge cases
func Test_PercentageCalculator_RoundingBehavior(t *testing.T) {
	tests := []struct {
		name        string
		rate        float64
		subtotal    int32
		shipping    int32
		expectedTax int32
		explanation string
	}{
		{
			name:        "rounds up from 0.5",
			rate:        0.08,
			subtotal:    1050,
			shipping:    0,
			expectedTax: 84,
			explanation: "1050 * 0.08 = 84.0 exactly (no rounding needed)",
		},
		{
			name:        "rounds up above midpoint",
			rate:        0.08,
			subtotal:    1062,
			shipping:    0,
			expectedTax: 85,
			explanation: "1062 * 0.08 = 84.96, rounds to 85",
		},
		{
			name:        "rounds down below midpoint",
			rate:        0.08,
			subtotal:    1040,
			shipping:    0,
			expectedTax: 83,
			explanation: "1040 * 0.08 = 83.2, rounds to 83",
		},
		{
			name:        "exact cent amount no rounding",
			rate:        0.10,
			subtotal:    1000,
			shipping:    0,
			expectedTax: 100,
			explanation: "1000 * 0.10 = 100.0 exactly",
		},
		{
			name:        "midpoint rounding 0.5 cents",
			rate:        0.08,
			subtotal:    1056,
			shipping:    0,
			expectedTax: 84,
			explanation: "1056 * 0.08 = 84.48, rounds to 84",
		},
		{
			name:        "complex rounding with shipping",
			rate:        0.085,
			subtotal:    4723,
			shipping:    387,
			expectedTax: 434,
			explanation: "(4723 + 387) * 0.085 = 434.35, rounds to 434",
		},
		{
			name:        "fractional cents round to nearest",
			rate:        0.065,
			subtotal:    1537,
			shipping:    0,
			expectedTax: 100,
			explanation: "1537 * 0.065 = 99.905, rounds to 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calc := tax.NewPercentageCalculator(tt.rate)

			params := tax.TaxParams{
				LineItems: []tax.LineItem{
					{TotalPrice: tt.subtotal},
				},
				ShippingCents: tt.shipping,
			}

			result, err := calc.CalculateTax(context.Background(), params)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedTax, result.TotalTaxCents, tt.explanation)

			// Verify rounding was applied correctly using math.Round
			taxableAmount := float64(tt.subtotal + tt.shipping)
			expectedFloat := math.Round(taxableAmount * tt.rate)
			assert.Equal(t, int32(expectedFloat), result.TotalTaxCents, "Should match math.Round behavior")
		})
	}
}

// Test_PercentageCalculator_MultipleLineItems validates tax calculation with multiple products
func Test_PercentageCalculator_MultipleLineItems(t *testing.T) {
	calc := tax.NewPercentageCalculator(0.08)

	params := tax.TaxParams{
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
			{
				ProductID:   pgtype.UUID{Valid: true},
				Description: "Coffee Mug",
				Quantity:    1,
				UnitPrice:   1500,
				TotalPrice:  1500,
				TaxCategory: "general_merchandise",
			},
		},
		ShippingCents: 750,
		CustomerType:  "retail",
	}

	result, err := calc.CalculateTax(context.Background(), params)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Subtotal: 3600 + 2200 + 1500 = 7300
	// Taxable: 7300 + 750 = 8050
	// Tax: 8050 * 0.08 = 644
	expectedTax := int32(644)
	assert.Equal(t, expectedTax, result.TotalTaxCents, "(3600 + 2200 + 1500 + 750) * 0.08 = 644")
	assert.Len(t, result.Breakdown, 1)
	assert.Equal(t, expectedTax, result.Breakdown[0].AmountCents)
}

// Test_PercentageCalculator_SingleLineItem validates basic single item calculation
func Test_PercentageCalculator_SingleLineItem(t *testing.T) {
	calc := tax.NewPercentageCalculator(0.08)

	params := tax.TaxParams{
		LineItems: []tax.LineItem{
			{
				ProductID:   pgtype.UUID{Valid: true},
				Description: "House Blend - 1lb",
				Quantity:    1,
				UnitPrice:   1600,
				TotalPrice:  1600,
				TaxCategory: "food",
			},
		},
		ShippingCents: 0,
	}

	result, err := calc.CalculateTax(context.Background(), params)

	assert.NoError(t, err)
	assert.Equal(t, int32(128), result.TotalTaxCents, "1600 * 0.08 = 128")
}

// Test_PercentageCalculator_ShippingScenarios validates different shipping cost scenarios
func Test_PercentageCalculator_ShippingScenarios(t *testing.T) {
	tests := []struct {
		name        string
		subtotal    int32
		shipping    int32
		rate        float64
		expectedTax int32
		explanation string
	}{
		{
			name:        "with shipping cost",
			subtotal:    5000,
			shipping:    1000,
			rate:        0.08,
			expectedTax: 480,
			explanation: "(5000 + 1000) * 0.08 = 480",
		},
		{
			name:        "zero shipping cost",
			subtotal:    5000,
			shipping:    0,
			rate:        0.08,
			expectedTax: 400,
			explanation: "5000 * 0.08 = 400",
		},
		{
			name:        "tax on shipping only (no items)",
			subtotal:    0,
			shipping:    1000,
			rate:        0.08,
			expectedTax: 80,
			explanation: "1000 * 0.08 = 80 (shipping only)",
		},
		{
			name:        "high shipping relative to subtotal",
			subtotal:    2000,
			shipping:    5000,
			rate:        0.10,
			expectedTax: 700,
			explanation: "(2000 + 5000) * 0.10 = 700",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calc := tax.NewPercentageCalculator(tt.rate)

			var lineItems []tax.LineItem
			if tt.subtotal > 0 {
				lineItems = []tax.LineItem{
					{TotalPrice: tt.subtotal},
				}
			}

			params := tax.TaxParams{
				LineItems:     lineItems,
				ShippingCents: tt.shipping,
			}

			result, err := calc.CalculateTax(context.Background(), params)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedTax, result.TotalTaxCents, tt.explanation)
		})
	}
}

// Test_PercentageCalculator_EdgeCases validates boundary conditions and edge cases
func Test_PercentageCalculator_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		rate        float64
		lineItems   []tax.LineItem
		shipping    int32
		expectedTax int32
		description string
	}{
		{
			name:        "empty line items zero shipping",
			rate:        0.08,
			lineItems:   []tax.LineItem{},
			shipping:    0,
			expectedTax: 0,
			description: "No items, no shipping = zero tax",
		},
		{
			name:        "zero subtotal with shipping",
			rate:        0.08,
			lineItems:   []tax.LineItem{},
			shipping:    500,
			expectedTax: 40,
			description: "Tax applies to shipping even with no items",
		},
		{
			name: "zero shipping with items",
			rate: 0.08,
			lineItems: []tax.LineItem{
				{TotalPrice: 5000},
			},
			shipping:    0,
			expectedTax: 400,
			description: "Tax applies to items even with no shipping",
		},
		{
			name: "very large order amount",
			rate: 0.08,
			lineItems: []tax.LineItem{
				{TotalPrice: 1000000}, // $10,000
			},
			shipping:    5000,
			expectedTax: 80400,
			description: "(1000000 + 5000) * 0.08 = 80400",
		},
		{
			name: "very small order amount",
			rate: 0.08,
			lineItems: []tax.LineItem{
				{TotalPrice: 10}, // 10 cents
			},
			shipping:    0,
			expectedTax: 1,
			description: "10 * 0.08 = 0.8, rounds to 1",
		},
		{
			name: "multiple line items with zero prices",
			rate: 0.08,
			lineItems: []tax.LineItem{
				{TotalPrice: 0},
				{TotalPrice: 0},
				{TotalPrice: 1000},
			},
			shipping:    0,
			expectedTax: 80,
			description: "Only non-zero items contribute to tax",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calc := tax.NewPercentageCalculator(tt.rate)

			params := tax.TaxParams{
				LineItems:     tt.lineItems,
				ShippingCents: tt.shipping,
			}

			result, err := calc.CalculateTax(context.Background(), params)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedTax, result.TotalTaxCents, tt.description)
		})
	}
}

// Test_PercentageCalculator_TaxResultStructure validates the structure of returned TaxResult
func Test_PercentageCalculator_TaxResultStructure(t *testing.T) {
	rate := 0.085
	calc := tax.NewPercentageCalculator(rate)

	params := tax.TaxParams{
		LineItems: []tax.LineItem{
			{TotalPrice: 5000},
		},
		ShippingCents: 1000,
	}

	result, err := calc.CalculateTax(context.Background(), params)

	assert.NoError(t, err)
	assert.NotNil(t, result, "Result should not be nil")

	// Verify TotalTaxCents
	expectedTax := int32(510) // (5000 + 1000) * 0.085 = 510
	assert.Equal(t, expectedTax, result.TotalTaxCents)

	// Verify Breakdown has exactly 1 entry
	assert.Len(t, result.Breakdown, 1, "Should have exactly one breakdown entry")

	// Verify Breakdown fields
	breakdown := result.Breakdown[0]
	assert.Equal(t, "state", breakdown.Jurisdiction, "Jurisdiction should be 'state'")
	assert.Equal(t, "Default Sales Tax", breakdown.Name, "Name should be 'Default Sales Tax'")
	assert.Equal(t, rate, breakdown.Rate, "Rate should match configured rate")
	assert.Equal(t, expectedTax, breakdown.AmountCents, "Breakdown amount should match total tax")

	// Verify ProviderTxID is empty
	assert.Empty(t, result.ProviderTxID, "ProviderTxID should be empty (no external provider)")

	// Verify IsEstimate is false
	assert.False(t, result.IsEstimate, "IsEstimate should be false (exact calculation)")
}

// Test_PercentageCalculator_ContextHandling validates behavior with different context states
func Test_PercentageCalculator_ContextHandling(t *testing.T) {
	calc := tax.NewPercentageCalculator(0.08)

	params := tax.TaxParams{
		LineItems: []tax.LineItem{
			{TotalPrice: 1000},
		},
		ShippingCents: 500,
	}

	t.Run("nil context", func(t *testing.T) {
		// Should work even with nil context (context not used in calculation)
		result, err := calc.CalculateTax(nil, params)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int32(120), result.TotalTaxCents)
	})

	t.Run("canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Should still work (context not used in percentage calculation)
		result, err := calc.CalculateTax(ctx, params)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int32(120), result.TotalTaxCents)
	})

	t.Run("background context", func(t *testing.T) {
		result, err := calc.CalculateTax(context.Background(), params)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int32(120), result.TotalTaxCents)
	})
}

// Test_PercentageCalculator_NewConstructor validates the constructor function
func Test_PercentageCalculator_NewConstructor(t *testing.T) {
	tests := []struct {
		name string
		rate float64
	}{
		{"zero rate", 0.0},
		{"standard rate", 0.08},
		{"high rate", 0.125},
		{"fractional rate", 0.0875},
		{"very small rate", 0.001},
		{"one hundred percent", 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calc := tax.NewPercentageCalculator(tt.rate)

			assert.NotNil(t, calc, "NewPercentageCalculator should return non-nil calculator")

			// Verify it implements the Calculator interface
			var _ tax.Calculator = calc

			// Verify the rate is used correctly
			params := tax.TaxParams{
				LineItems:     []tax.LineItem{{TotalPrice: 1000}},
				ShippingCents: 0,
			}
			result, err := calc.CalculateTax(context.Background(), params)

			assert.NoError(t, err)
			assert.Equal(t, tt.rate, result.Breakdown[0].Rate, "Rate should be preserved in result")
		})
	}
}

// Test_PercentageCalculator_Idempotency validates that repeated calls produce identical results
func Test_PercentageCalculator_Idempotency(t *testing.T) {
	calc := tax.NewPercentageCalculator(0.08)

	params := tax.TaxParams{
		LineItems: []tax.LineItem{
			{TotalPrice: 5000},
			{TotalPrice: 3000},
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
	assert.Equal(t, int32(700), result1.TotalTaxCents, "(5000 + 3000 + 750) * 0.08 = 700")

	// Breakdown should be identical
	assert.Equal(t, result1.Breakdown[0].AmountCents, result2.Breakdown[0].AmountCents)
	assert.Equal(t, result1.Breakdown[0].AmountCents, result3.Breakdown[0].AmountCents)

	// Metadata should be identical
	assert.Equal(t, result1.IsEstimate, result2.IsEstimate)
	assert.Equal(t, result1.IsEstimate, result3.IsEstimate)
	assert.Equal(t, result1.ProviderTxID, result2.ProviderTxID)
	assert.Equal(t, result1.ProviderTxID, result3.ProviderTxID)
}

// Test_PercentageCalculator_RealWorldScenarios validates realistic coffee shop order scenarios
func Test_PercentageCalculator_RealWorldScenarios(t *testing.T) {
	tests := []struct {
		name        string
		rate        float64
		lineItems   []tax.LineItem
		shipping    int32
		expectedTax int32
		description string
	}{
		{
			name: "typical retail coffee order",
			rate: 0.08,
			lineItems: []tax.LineItem{
				{
					Description: "Ethiopian Yirgacheffe - 12oz",
					Quantity:    1,
					UnitPrice:   1800,
					TotalPrice:  1800,
				},
			},
			shipping:    500,
			expectedTax: 184,
			description: "Single bag with standard shipping",
		},
		{
			name: "multi-item subscription order",
			rate: 0.085,
			lineItems: []tax.LineItem{
				{
					Description: "House Blend - 1lb",
					Quantity:    2,
					UnitPrice:   1600,
					TotalPrice:  3200,
				},
				{
					Description: "Decaf Colombia - 12oz",
					Quantity:    1,
					UnitPrice:   1900,
					TotalPrice:  1900,
				},
			},
			shipping:    0,
			expectedTax: 434,
			description: "Subscription with free shipping",
		},
		{
			name: "bulk order with merchandise",
			rate: 0.08,
			lineItems: []tax.LineItem{
				{
					Description: "Kenya AA - 1lb",
					Quantity:    5,
					UnitPrice:   2200,
					TotalPrice:  11000,
				},
				{
					Description: "Travel Mug",
					Quantity:    1,
					UnitPrice:   2500,
					TotalPrice:  2500,
				},
			},
			shipping:    1200,
			expectedTax: 1176,
			description: "Large order with taxable merchandise",
		},
		{
			name: "small gift order",
			rate: 0.075,
			lineItems: []tax.LineItem{
				{
					Description: "Gift Card",
					Quantity:    1,
					UnitPrice:   2500,
					TotalPrice:  2500,
				},
			},
			shipping:    0,
			expectedTax: 188,
			description: "Gift card purchase (2500 * 0.075 = 187.5, rounds to 188)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calc := tax.NewPercentageCalculator(tt.rate)

			params := tax.TaxParams{
				LineItems:     tt.lineItems,
				ShippingCents: tt.shipping,
				CustomerType:  "retail",
			}

			result, err := calc.CalculateTax(context.Background(), params)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedTax, result.TotalTaxCents, tt.description)
			assert.Len(t, result.Breakdown, 1)
			assert.Equal(t, tt.expectedTax, result.Breakdown[0].AmountCents)
		})
	}
}
