package billing

import (
	"context"

	"github.com/dukerupert/freyja/internal/tax"
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

// CalculateTax returns an estimate for display purposes.
//
// The actual tax calculation is performed by Stripe during payment intent creation.
// This method returns an estimate based on estimateRate for frontend preview.
//
// The TaxResult.IsEstimate field is set to true to indicate this is not final.
// The TaxResult.ProviderTxID is empty until Stripe calculates actual tax.
func (c *StripeTaxCalculator) CalculateTax(ctx context.Context, params tax.TaxParams) (*tax.TaxResult, error) {
	// TODO: Implementation
	//
	// Steps:
	// 1. Calculate subtotal from params.LineItems
	// 2. Add params.ShippingCents
	// 3. Calculate estimate: (subtotal + shipping) * c.estimateRate
	// 4. Return TaxResult:
	//    - TotalTaxCents: estimated amount
	//    - Breakdown: single entry with "Estimated Tax"
	//    - ProviderTxID: empty (will be filled when Stripe calculates)
	//    - IsEstimate: true
	//
	// Note: Checkout service should check IsEstimate and enable Stripe Tax
	// in CreatePaymentIntent call.
	panic("not implemented")
}
