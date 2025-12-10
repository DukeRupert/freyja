package tax

import (
	"context"
	"fmt"
	"math"

	"github.com/dukerupert/hiri/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// PercentageCalculator calculates tax using database-stored state tax rates.
type PercentageCalculator struct {
	repo     repository.Querier
	tenantID pgtype.UUID
}

// NewPercentageCalculator creates a new percentage-based tax calculator.
// For backwards compatibility, if rate is non-zero, it creates a fixed-rate calculator.
func NewPercentageCalculator(rate float64) Calculator {
	return &fixedRateCalculator{rate: rate}
}

// NewDatabasePercentageCalculator creates a percentage calculator that looks up rates from database.
func NewDatabasePercentageCalculator(repo repository.Querier, tenantID pgtype.UUID) Calculator {
	return &PercentageCalculator{
		repo:     repo,
		tenantID: tenantID,
	}
}

// CalculateTax computes tax by looking up the state rate from database.
func (c *PercentageCalculator) CalculateTax(ctx context.Context, params TaxParams) (*TaxResult, error) {
	// If tax exemption is provided, return zero tax
	if params.TaxExemptionID != "" {
		return &TaxResult{
			TotalTaxCents: 0,
			Breakdown:     []TaxBreakdown{},
			ProviderTxID:  "",
			IsEstimate:    false,
		}, nil
	}

	// Calculate subtotal
	subtotal := int32(0)
	for _, item := range params.LineItems {
		subtotal += item.TotalPrice
	}

	// Look up tax rate for the shipping state
	taxRate, err := c.repo.GetTaxRateByState(ctx, repository.GetTaxRateByStateParams{
		TenantID: c.tenantID,
		State:    params.ShippingAddress.State,
	})

	// If no rate found for state, return zero tax (no nexus in that state)
	if err != nil {
		if err == pgx.ErrNoRows {
			return &TaxResult{
				TotalTaxCents: 0,
				Breakdown:     []TaxBreakdown{},
				ProviderTxID:  "",
				IsEstimate:    false,
			}, nil
		}
		return nil, fmt.Errorf("failed to get tax rate: %w", err)
	}

	// Calculate taxable amount (subtotal + shipping if tax_shipping is true)
	taxableAmount := subtotal
	if taxRate.TaxShipping {
		taxableAmount += params.ShippingCents
	}

	// Convert DECIMAL rate to float64
	rateFloat, err := taxRate.Rate.Float64Value()
	if err != nil {
		return nil, fmt.Errorf("failed to convert tax rate: %w", err)
	}
	rate := rateFloat.Float64

	// Calculate tax amount
	taxAmount := int32(math.Round(float64(taxableAmount) * rate))

	// Build jurisdiction name
	jurisdictionName := params.ShippingAddress.State
	if taxRate.Name.Valid && taxRate.Name.String != "" {
		jurisdictionName = taxRate.Name.String
	}

	return &TaxResult{
		TotalTaxCents: taxAmount,
		Breakdown: []TaxBreakdown{
			{
				Jurisdiction: "state",
				Name:         jurisdictionName,
				Rate:         rate,
				AmountCents:  taxAmount,
			},
		},
		ProviderTxID: "",
		IsEstimate:   false,
	}, nil
}

// fixedRateCalculator is for backwards compatibility with existing tests.
type fixedRateCalculator struct {
	rate float64
}

func (c *fixedRateCalculator) CalculateTax(ctx context.Context, params TaxParams) (*TaxResult, error) {
	subtotal := int32(0)
	for _, item := range params.LineItems {
		subtotal += item.TotalPrice
	}

	taxableAmount := subtotal + params.ShippingCents
	taxAmount := int32(math.Round(float64(taxableAmount) * c.rate))

	return &TaxResult{
		TotalTaxCents: taxAmount,
		Breakdown: []TaxBreakdown{
			{
				Jurisdiction: "state",
				Name:         "Default Sales Tax",
				Rate:         c.rate,
				AmountCents:  taxAmount,
			},
		},
		ProviderTxID: "",
		IsEstimate:   false,
	}, nil
}
