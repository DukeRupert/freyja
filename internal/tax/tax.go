package tax

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// Calculator defines the interface for tax calculation.
// Implementations: PercentageCalculator, StripeTaxCalculator, NoTaxCalculator
type Calculator interface {
	// CalculateTax computes tax for order line items and shipping.
	// Returns tax amount in cents.
	CalculateTax(ctx context.Context, params TaxParams) (*TaxResult, error)
}

// TaxParams contains all information needed for tax calculation.
type TaxParams struct {
	ShippingAddress Address
	LineItems       []LineItem
	ShippingCents   int32
	CustomerType    string // "retail" or "wholesale"
	TaxExemptionID  string // Optional exemption certificate
}

// Address represents a physical address for tax purposes.
type Address struct {
	Line1      string
	Line2      string
	City       string
	State      string
	PostalCode string
	Country    string
}

// LineItem represents a single item being taxed.
type LineItem struct {
	ProductID   pgtype.UUID
	Description string
	Quantity    int32
	UnitPrice   int32
	TotalPrice  int32
	TaxCategory string // "food", "general_merchandise", etc.
}

// TaxResult contains the calculated tax amount and breakdown.
type TaxResult struct {
	TotalTaxCents int32
	Breakdown     []TaxBreakdown
	ProviderTxID  string // For audit trail
	IsEstimate    bool
}

// TaxBreakdown represents tax for a single jurisdiction.
type TaxBreakdown struct {
	Jurisdiction string  // "state", "county", "city"
	Name         string  // e.g., "Washington State"
	Rate         float64 // e.g., 0.065 for 6.5%
	AmountCents  int32
}
