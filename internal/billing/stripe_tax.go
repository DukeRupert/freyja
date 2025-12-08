package billing

import (
	"context"
	"fmt"

	"github.com/dukerupert/freyja/internal/tax"
	"github.com/stripe/stripe-go/v83"
	"github.com/stripe/stripe-go/v83/tax/calculation"
)

// StripeTaxCalculator delegates tax calculation to Stripe Tax.
//
// This is a special tax calculator that signals to the billing provider
// that Stripe should calculate tax during Payment Intent creation.
//
// Unlike other tax calculators (PercentageCalculator, NoTaxCalculator),
// this doesn't calculate tax itself - it returns a placeholder result
// and the actual tax is calculated by Stripe when creating the payment intent.
//
// Usage in checkout service:
//  1. Calculate subtotal and shipping
//  2. Call StripeTaxCalculator.CalculateTax() - returns estimate/placeholder
//  3. Pass EnableStripeTax=true to CreatePaymentIntent
//  4. Stripe calculates actual tax based on address and line items
//  5. Tax amount included in PaymentIntent response
//
// Note: Requires Stripe Tax to be enabled in Stripe dashboard.
type StripeTaxCalculator struct {
	// estimateRate is used for frontend preview before payment intent creation
	// Actual tax calculated by Stripe may differ
	estimateRate float64
}

// NewStripeTaxCalculator creates a tax calculator that delegates to Stripe Tax.
//
// The estimateRate is used for displaying estimated tax to customer before
// they proceed to payment. The actual tax is calculated by Stripe.
//
// If estimateRate is 0, returns 0 estimate (no preview shown).
func NewStripeTaxCalculator(estimateRate float64) tax.Calculator {
	return &StripeTaxCalculator{
		estimateRate: estimateRate,
	}
}

// CalculateTax calls the Stripe Tax Calculation API to calculate tax.
//
// This implementation calls Stripe's Tax Calculation API to get accurate
// tax calculations based on customer address and line items. Unlike the
// old placeholder implementation, this returns actual tax amounts.
//
// The tax calculation ID is returned in TaxResult.ProviderTxID for
// later attachment to the payment intent.
func (c *StripeTaxCalculator) CalculateTax(ctx context.Context, params tax.TaxParams) (*tax.TaxResult, error) {
	// Build tax calculation request
	calcParams := &stripe.TaxCalculationParams{
		Currency: stripe.String("usd"),
		CustomerDetails: &stripe.TaxCalculationCustomerDetailsParams{
			Address: &stripe.AddressParams{
				Line1:      stripe.String(params.ShippingAddress.Line1),
				Line2:      stripe.String(params.ShippingAddress.Line2),
				City:       stripe.String(params.ShippingAddress.City),
				State:      stripe.String(params.ShippingAddress.State),
				PostalCode: stripe.String(params.ShippingAddress.PostalCode),
				Country:    stripe.String(params.ShippingAddress.Country),
			},
			AddressSource: stripe.String("shipping"),
		},
		LineItems: buildStripeTaxLineItems(params),
	}

	// Add tax exemption if provided
	if params.TaxExemptionID != "" {
		calcParams.CustomerDetails.TaxIDs = []*stripe.TaxCalculationCustomerDetailsTaxIDParams{
			{
				Type:  stripe.String("us_ein"),
				Value: stripe.String(params.TaxExemptionID),
			},
		}
	}

	calc, err := calculation.New(calcParams)
	if err != nil {
		return nil, wrapStripeError(err)
	}

	// Parse the response
	totalTaxCents := int32(calc.TaxAmountExclusive)

	// Build tax breakdown by jurisdiction
	breakdown := buildTaxBreakdown(calc)

	return &tax.TaxResult{
		TotalTaxCents: totalTaxCents,
		Breakdown:     breakdown,
		ProviderTxID:  calc.ID, // Stripe tax calculation ID (txcd_...)
		IsEstimate:    false,   // This is an actual calculation
	}, nil
}

// buildStripeTaxLineItems converts our line items to Stripe's format
func buildStripeTaxLineItems(params tax.TaxParams) []*stripe.TaxCalculationLineItemParams {
	lineItems := make([]*stripe.TaxCalculationLineItemParams, 0, len(params.LineItems)+1)

	// Add product line items
	for _, item := range params.LineItems {
		taxCode := "txcd_99999999" // general merchandise
		if item.TaxCategory == "food" {
			taxCode = "txcd_30011000" // food/beverages
		}

		// Convert UUID to string for reference
		productIDStr := fmt.Sprintf("%x-%x-%x-%x-%x",
			item.ProductID.Bytes[0:4],
			item.ProductID.Bytes[4:6],
			item.ProductID.Bytes[6:8],
			item.ProductID.Bytes[8:10],
			item.ProductID.Bytes[10:16])

		lineItems = append(lineItems, &stripe.TaxCalculationLineItemParams{
			Amount:    stripe.Int64(int64(item.TotalPrice)),
			Reference: stripe.String(productIDStr),
			TaxCode:   stripe.String(taxCode),
		})
	}

	// Add shipping as a line item if present
	if params.ShippingCents > 0 {
		lineItems = append(lineItems, &stripe.TaxCalculationLineItemParams{
			Amount:    stripe.Int64(int64(params.ShippingCents)),
			Reference: stripe.String("shipping"),
			TaxCode:   stripe.String("txcd_92010001"), // shipping tax code
		})
	}

	return lineItems
}

// buildTaxBreakdown extracts tax breakdown by jurisdiction from Stripe response
func buildTaxBreakdown(calc *stripe.TaxCalculation) []tax.TaxBreakdown {
	// Stripe v83 TaxBreakdown structure is simpler
	// It contains Amount, TaxRateDetails which has State, Country, PercentageDecimal, etc.
	jurisdictionMap := make(map[string]*tax.TaxBreakdown)

	if calc.TaxBreakdown != nil {
		for _, item := range calc.TaxBreakdown {
			if item.TaxRateDetails == nil {
				continue
			}

			// Build jurisdiction key from state and country
			state := item.TaxRateDetails.State
			country := item.TaxRateDetails.Country
			taxType := string(item.TaxRateDetails.TaxType)

			// Determine jurisdiction name and level
			var jurisdictionName, jurisdictionLevel string
			if state != "" {
				jurisdictionName = state
				jurisdictionLevel = "state"
			} else if country != "" {
				jurisdictionName = country
				jurisdictionLevel = "country"
			} else {
				continue // Skip if we don't have location info
			}

			key := fmt.Sprintf("%s|%s|%s", jurisdictionLevel, jurisdictionName, taxType)

			// Parse percentage decimal (e.g., "8.5" -> 0.085)
			var rate float64
			if item.TaxRateDetails.PercentageDecimal != "" {
				_, _ = fmt.Sscanf(item.TaxRateDetails.PercentageDecimal, "%f", &rate)
				rate = rate / 100.0 // Convert percentage to decimal
			}

			if existing, ok := jurisdictionMap[key]; ok {
				// Aggregate amounts for same jurisdiction
				existing.AmountCents += int32(item.Amount)
			} else {
				// New jurisdiction
				jurisdictionMap[key] = &tax.TaxBreakdown{
					Jurisdiction: jurisdictionLevel,
					Name:         jurisdictionName,
					Rate:         rate,
					AmountCents:  int32(item.Amount),
				}
			}
		}
	}

	// Convert map to slice
	breakdown := make([]tax.TaxBreakdown, 0, len(jurisdictionMap))
	for _, item := range jurisdictionMap {
		breakdown = append(breakdown, *item)
	}

	return breakdown
}
