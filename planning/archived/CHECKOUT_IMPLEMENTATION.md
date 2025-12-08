# Checkout Implementation Guide

**Status:** Planning
**Target:** Complete B2C checkout flow for MVP
**Estimated Effort:** 10-15 days
**Architecture Reference:** Designed by freyja-architect (2025-11-28)

---

## Table of Contents

1. [Overview](#overview)
2. [Architecture Decisions](#architecture-decisions)
3. [Phase 1: Interfaces & Simple Providers](#phase-1-interfaces--simple-providers)
4. [Phase 2: Checkout Service](#phase-2-checkout-service)
5. [Phase 3: HTTP Handlers](#phase-3-http-handlers)
6. [Phase 4: Database](#phase-4-database)
7. [Phase 5: Testing & Polish](#phase-5-testing--polish)
8. [Implementation Checklist](#implementation-checklist)

---

## Overview

### Goal

Implement a complete checkout flow that converts a shopping cart into a paid order, integrating with external services (Stripe for payments, optional tax calculation, shipping providers) through clean, testable abstractions.

### User Flow

```
Cart Review → Shipping Address → Shipping Method → Payment → Order Confirmation
```

### Core Principles

- **Interface-based abstractions** - All external services (tax, shipping, address validation) behind interfaces
- **Idempotent operations** - Prevent duplicate orders via Payment Intent ID
- **Transaction safety** - Atomic cart→order conversion
- **Flexibility** - Support multiple tax strategies (provider-calculated, Stripe Tax, or skip)
- **MVP simplicity** - Start with flat-rate shipping, basic validation, optional tax

---

## Architecture Decisions

### 1. State Management: Ephemeral

**Decision:** Use existing `carts` table + form POST data. No `checkout_sessions` table for MVP.

**Rationale:**
- Simpler implementation - fewer tables to maintain
- Cart already has `last_activity_at` for abandonment tracking
- Can use cart.metadata JSONB for partial progress if needed later

**Future:** Can add dedicated `checkout_sessions` table post-MVP for detailed analytics.

### 2. Tax Calculation Flexibility

**Decision:** Tax calculation is **optional** via strategy pattern.

**Three supported strategies:**

```go
// Strategy 1: Application calculates tax
taxCalc := tax.NewPercentageCalculator(0.08)
checkoutSvc := NewCheckoutService(..., taxCalc, ...)

// Strategy 2: Stripe calculates tax
taxCalc := tax.NewStripeTaxCalculator() // delegates to Stripe Tax
checkoutSvc := NewCheckoutService(..., taxCalc, ...)

// Strategy 3: No tax calculation (wholesale, tax-exempt)
taxCalc := tax.NewNoTaxCalculator()
checkoutSvc := NewCheckoutService(..., taxCalc, ...)
```

**Implementation note:** The `tax.Calculator` interface remains the same. Different implementations provide different strategies.

**Stripe Tax integration:**
- Tax calculated during Payment Intent creation
- Stripe returns tax amount, we record it in order
- No separate tax API call needed

### 3. Payment Flow: Stripe Payment Intents

**Workflow:**
1. Calculate order total → Create Payment Intent with total
2. Frontend uses Stripe.js to collect payment → Customer enters card
3. Stripe confirms payment → Returns success to frontend
4. Frontend calls `POST /checkout/complete` → Backend creates order

**Idempotency:** Payment Intent ID used as idempotency key for order creation.

### 4. Plugin Architecture

All external services use constructor injection:

```go
// main.go
taxCalc := tax.NewPercentageCalculator(0.08)
shippingProv := shipping.NewFlatRateProvider(config)
addrValidator := address.NewBasicValidator()
billingProv := billing.NewStripeProvider(stripeKey)

checkoutSvc := service.NewCheckoutService(
    repo,
    cartSvc,
    billingProv,
    shippingProv,
    taxCalc,      // Pluggable tax strategy
    addrValidator,
    tenantID,
)
```

**Testing:** Mock implementations for each interface.

---

## Phase 1: Interfaces & Simple Providers

**Duration:** 2-3 days
**Goal:** Define plugin interfaces and implement simple MVP providers

### 1.1 Tax Package

**Files:**
- `internal/tax/tax.go` - Interface + types
- `internal/tax/percentage.go` - Simple percentage calculator (MVP)
- `internal/tax/notax.go` - No-op calculator (wholesale/exempt)
- `internal/tax/stripe.go` - Stripe Tax integration (future)
- `internal/tax/mock.go` - Mock for testing

#### Interface Definition

```go
// internal/tax/tax.go
package tax

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// Calculator defines tax calculation interface.
// Implementations: PercentageCalculator, StripeTaxCalculator, NoTaxCalculator
type Calculator interface {
	// CalculateTax computes tax for order line items and shipping.
	// Returns tax amount in cents.
	CalculateTax(ctx context.Context, params TaxParams) (*TaxResult, error)
}

type TaxParams struct {
	ShippingAddress Address
	LineItems       []LineItem
	ShippingCents   int32
	CustomerType    string // "retail" or "wholesale"
	TaxExemptionID  string // Optional exemption certificate
}

type Address struct {
	Line1      string
	Line2      string
	City       string
	State      string
	PostalCode string
	Country    string
}

type LineItem struct {
	ProductID   pgtype.UUID
	Description string
	Quantity    int32
	UnitPrice   int32
	TotalPrice  int32
	TaxCategory string // "food", "general_merchandise", etc.
}

type TaxResult struct {
	TotalTaxCents int32
	Breakdown     []TaxBreakdown
	ProviderTxID  string // For audit trail
	IsEstimate    bool
}

type TaxBreakdown struct {
	Jurisdiction string  // "state", "county", "city"
	Name         string  // e.g., "Washington State"
	Rate         float64 // e.g., 0.065 for 6.5%
	AmountCents  int32
}
```

#### Implementation: PercentageCalculator

```go
// internal/tax/percentage.go
package tax

import (
	"context"
	"math"
)

type PercentageCalculator struct {
	defaultRate float64 // e.g., 0.08 for 8%
}

func NewPercentageCalculator(rate float64) Calculator {
	return &PercentageCalculator{defaultRate: rate}
}

func (c *PercentageCalculator) CalculateTax(ctx context.Context, params TaxParams) (*TaxResult, error) {
	// Calculate tax on subtotal + shipping
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
		ProviderTxID: "", // No external provider
		IsEstimate:   false,
	}, nil
}
```

#### Implementation: NoTaxCalculator

```go
// internal/tax/notax.go
package tax

import "context"

type NoTaxCalculator struct{}

func NewNoTaxCalculator() Calculator {
	return &NoTaxCalculator{}
}

func (c *NoTaxCalculator) CalculateTax(ctx context.Context, params TaxParams) (*TaxResult, error) {
	return &TaxResult{
		TotalTaxCents: 0,
		Breakdown:     []TaxBreakdown{},
		ProviderTxID:  "",
		IsEstimate:    false,
	}, nil
}
```

#### Tests

```go
// internal/tax/percentage_test.go
package tax_test

import (
	"context"
	"testing"

	"github.com/dukerupert/freyja/internal/tax"
	"github.com/stretchr/testify/assert"
)

func TestPercentageCalculator_CalculateTax(t *testing.T) {
	calc := tax.NewPercentageCalculator(0.08) // 8%

	params := tax.TaxParams{
		LineItems: []tax.LineItem{
			{TotalPrice: 1000}, // $10.00
			{TotalPrice: 1500}, // $15.00
		},
		ShippingCents: 500, // $5.00
	}

	result, err := calc.CalculateTax(context.Background(), params)

	assert.NoError(t, err)
	assert.Equal(t, int32(240), result.TotalTaxCents) // (25 + 5) * 0.08 = 2.40
	assert.Len(t, result.Breakdown, 1)
}
```

**Deliverables:**
- [ ] `internal/tax/tax.go` - Interface definition
- [ ] `internal/tax/percentage.go` - Percentage calculator
- [ ] `internal/tax/notax.go` - No-op calculator
- [ ] `internal/tax/mock.go` - Mock implementation
- [ ] `internal/tax/percentage_test.go` - Unit tests

---

### 1.2 Shipping Package

**Files:**
- `internal/shipping/shipping.go` - Interface + types
- `internal/shipping/flatrate.go` - Flat-rate provider (MVP)
- `internal/shipping/mock.go` - Mock for testing

#### Interface Definition

```go
// internal/shipping/shipping.go
package shipping

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type Provider interface {
	// GetRates returns available shipping options for a shipment
	GetRates(ctx context.Context, params RateParams) ([]Rate, error)

	// CreateLabel generates shipping label (post-MVP)
	CreateLabel(ctx context.Context, params LabelParams) (*Label, error)

	// VoidLabel cancels a label (post-MVP)
	VoidLabel(ctx context.Context, labelID string) error

	// TrackShipment gets tracking info (post-MVP)
	TrackShipment(ctx context.Context, trackingNumber string) (*TrackingInfo, error)
}

type RateParams struct {
	OriginAddress      ShippingAddress
	DestinationAddress ShippingAddress
	Packages           []Package
	ServiceTypes       []string // Optional filter
}

type ShippingAddress struct {
	Name       string
	Company    string
	Line1      string
	Line2      string
	City       string
	State      string
	PostalCode string
	Country    string
	Phone      string
	Email      string
}

type Package struct {
	WeightGrams int32
	LengthCm    int32
	WidthCm     int32
	HeightCm    int32
}

type Rate struct {
	RateID               string
	Carrier              string
	ServiceName          string
	ServiceCode          string
	CostCents            int32
	EstimatedDaysMin     int
	EstimatedDaysMax     int
	EstimatedDeliveryDate time.Time
}

// Label, LabelParams, TrackingInfo - defined but not implemented until post-MVP
type Label struct{}
type LabelParams struct{}
type TrackingInfo struct{}
```

#### Implementation: FlatRateProvider

```go
// internal/shipping/flatrate.go
package shipping

import (
	"context"
	"time"
)

type FlatRateProvider struct {
	rates []FlatRate
}

type FlatRate struct {
	ServiceName string
	ServiceCode string
	CostCents   int32
	DaysMin     int
	DaysMax     int
}

func NewFlatRateProvider(rates []FlatRate) Provider {
	return &FlatRateProvider{rates: rates}
}

func (p *FlatRateProvider) GetRates(ctx context.Context, params RateParams) ([]Rate, error) {
	// Convert flat rates to Rate objects
	result := make([]Rate, len(p.rates))
	for i, fr := range p.rates {
		result[i] = Rate{
			RateID:           fr.ServiceCode,
			Carrier:          "Flat Rate",
			ServiceName:      fr.ServiceName,
			ServiceCode:      fr.ServiceCode,
			CostCents:        fr.CostCents,
			EstimatedDaysMin: fr.DaysMin,
			EstimatedDaysMax: fr.DaysMax,
			EstimatedDeliveryDate: time.Now().AddDate(0, 0, fr.DaysMax),
		}
	}

	return result, nil
}

// Stub implementations (not used in MVP)
func (p *FlatRateProvider) CreateLabel(ctx context.Context, params LabelParams) (*Label, error) {
	return nil, ErrNotImplemented
}

func (p *FlatRateProvider) VoidLabel(ctx context.Context, labelID string) error {
	return ErrNotImplemented
}

func (p *FlatRateProvider) TrackShipment(ctx context.Context, trackingNumber string) (*TrackingInfo, error) {
	return nil, ErrNotImplemented
}
```

**Deliverables:**
- [ ] `internal/shipping/shipping.go` - Interface definition
- [ ] `internal/shipping/flatrate.go` - Flat rate provider
- [ ] `internal/shipping/mock.go` - Mock implementation
- [ ] `internal/shipping/flatrate_test.go` - Unit tests

---

### 1.3 Address Package

**Files:**
- `internal/address/address.go` - Interface + types
- `internal/address/basic.go` - Basic format validator (MVP)
- `internal/address/mock.go` - Mock for testing

#### Interface Definition

```go
// internal/address/address.go
package address

import "context"

type Validator interface {
	// Validate checks if address is valid and deliverable
	Validate(ctx context.Context, addr Address) (*ValidationResult, error)
}

type Address struct {
	Line1      string
	Line2      string
	City       string
	State      string
	PostalCode string
	Country    string
}

type ValidationResult struct {
	IsValid           bool
	IsNormalized      bool
	NormalizedAddress Address
	Messages          []ValidationMessage
}

type ValidationMessage struct {
	Severity string // "error", "warning", "info"
	Code     string
	Message  string
	Field    string
}
```

#### Implementation: BasicValidator

```go
// internal/address/basic.go
package address

import (
	"context"
	"regexp"
	"strings"
)

type BasicValidator struct{}

func NewBasicValidator() Validator {
	return &BasicValidator{}
}

var zipRegex = regexp.MustCompile(`^\d{5}(-\d{4})?$`)

func (v *BasicValidator) Validate(ctx context.Context, addr Address) (*ValidationResult, error) {
	result := &ValidationResult{
		IsValid:      true,
		IsNormalized: false,
		NormalizedAddress: addr,
		Messages:     []ValidationMessage{},
	}

	// Required fields
	if strings.TrimSpace(addr.Line1) == "" {
		result.IsValid = false
		result.Messages = append(result.Messages, ValidationMessage{
			Severity: "error",
			Code:     "MISSING_LINE1",
			Message:  "Street address is required",
			Field:    "line1",
		})
	}

	if strings.TrimSpace(addr.City) == "" {
		result.IsValid = false
		result.Messages = append(result.Messages, ValidationMessage{
			Severity: "error",
			Code:     "MISSING_CITY",
			Message:  "City is required",
			Field:    "city",
		})
	}

	if strings.TrimSpace(addr.State) == "" {
		result.IsValid = false
		result.Messages = append(result.Messages, ValidationMessage{
			Severity: "error",
			Code:     "MISSING_STATE",
			Message:  "State is required",
			Field:    "state",
		})
	}

	// Validate ZIP code format (US only for now)
	if addr.Country == "US" && !zipRegex.MatchString(addr.PostalCode) {
		result.IsValid = false
		result.Messages = append(result.Messages, ValidationMessage{
			Severity: "error",
			Code:     "INVALID_POSTAL_CODE",
			Message:  "ZIP code must be 5 digits or 9 digits (12345 or 12345-6789)",
			Field:    "postal_code",
		})
	}

	return result, nil
}
```

**Deliverables:**
- [ ] `internal/address/address.go` - Interface definition
- [ ] `internal/address/basic.go` - Basic validator
- [ ] `internal/address/mock.go` - Mock implementation
- [ ] `internal/address/basic_test.go` - Unit tests

---

## Phase 2: Checkout Service

**Duration:** 3-4 days
**Goal:** Implement checkout orchestration logic and complete Stripe integration

### 2.1 CheckoutService Interface

**File:** `internal/service/checkout.go`

```go
package service

import (
	"context"

	"github.com/dukerupert/freyja/internal/address"
	"github.com/dukerupert/freyja/internal/billing"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/shipping"
	"github.com/dukerupert/freyja/internal/tax"
	"github.com/jackc/pgx/v5/pgtype"
)

type CheckoutService interface {
	// ValidateAndNormalizeAddress validates shipping/billing address
	ValidateAndNormalizeAddress(ctx context.Context, addr address.Address) (*address.ValidationResult, error)

	// GetShippingRates calculates available shipping for cart
	GetShippingRates(ctx context.Context, cartID string, shippingAddr address.Address) ([]shipping.Rate, error)

	// CalculateOrderTotal computes full order total including tax and shipping
	CalculateOrderTotal(ctx context.Context, params OrderTotalParams) (*OrderTotal, error)

	// CreatePaymentIntent initiates Stripe Payment Intent
	CreatePaymentIntent(ctx context.Context, params PaymentIntentParams) (*billing.PaymentIntent, error)

	// CompleteCheckout converts cart to order (idempotent)
	CompleteCheckout(ctx context.Context, params CompleteCheckoutParams) (*Order, error)
}

type OrderTotalParams struct {
	CartID              string
	ShippingAddress     address.Address
	BillingAddress      address.Address
	SelectedShippingRate shipping.Rate
	DiscountCode        string
}

type OrderTotal struct {
	SubtotalCents int32
	ShippingCents int32
	TaxCents      int32
	DiscountCents int32
	TotalCents    int32

	TaxBreakdown      []tax.TaxBreakdown
	ShippingRateID    string
	DiscountCodeID    pgtype.UUID
	DiscountCodeValue string
}

type PaymentIntentParams struct {
	CartID          string
	OrderTotal      *OrderTotal
	ShippingAddress address.Address
	BillingAddress  address.Address
	CustomerEmail   string
	IdempotencyKey  string
}

type CompleteCheckoutParams struct {
	CartID          string
	PaymentIntentID string
	ShippingAddress address.Address
	BillingAddress  address.Address
	ShippingRateID  string
	DiscountCodeID  pgtype.UUID
	CustomerNotes   string
	IdempotencyKey  string
}

type Order struct {
	ID            pgtype.UUID
	OrderNumber   string
	Status        string
	SubtotalCents int32
	TaxCents      int32
	ShippingCents int32
	DiscountCents int32
	TotalCents    int32
	Currency      string
	CreatedAt     pgtype.Timestamptz

	PaymentID         pgtype.UUID
	ShippingAddressID pgtype.UUID
	BillingAddressID  pgtype.UUID
}
```

### 2.2 Key Method: CompleteCheckout

**Idempotency pattern:**

```go
func (s *checkoutService) CompleteCheckout(ctx context.Context, params CompleteCheckoutParams) (*Order, error) {
	// Step 1: Check if order already exists (idempotency)
	existingOrder, err := s.repo.GetOrderByPaymentIntentID(ctx, params.PaymentIntentID)
	if err == nil {
		// Order already created, return it
		return existingOrder, nil
	}

	// Step 2: Verify payment with Stripe
	intent, err := s.billingProvider.GetPaymentIntent(ctx, params.PaymentIntentID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify payment: %w", err)
	}

	if intent.Status != "succeeded" {
		return nil, ErrPaymentNotSucceeded
	}

	// Step 3: Start database transaction
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Step 4: Get cart and validate
	cart, err := s.cartService.GetCart(ctx, params.CartID)
	if err != nil {
		return nil, err
	}

	if cart.Status != "active" {
		return nil, ErrCartNotActive
	}

	// Step 5: Check inventory availability
	for _, item := range cart.Items {
		available, err := tx.GetSKUStock(ctx, item.SKUID)
		if err != nil {
			return nil, err
		}
		if available < item.Quantity {
			return nil, ErrInsufficientInventory{SKU: item.SKUID, Available: available}
		}
	}

	// Step 6: Create addresses
	shippingAddrID, err := tx.CreateAddress(ctx, params.ShippingAddress)
	if err != nil {
		return nil, err
	}

	billingAddrID, err := tx.CreateAddress(ctx, params.BillingAddress)
	if err != nil {
		return nil, err
	}

	// Step 7: Create order
	order, err := tx.CreateOrder(ctx, CreateOrderParams{
		TenantID:          s.tenantID,
		UserID:            cart.UserID,
		ShippingAddressID: shippingAddrID,
		BillingAddressID:  billingAddrID,
		SubtotalCents:     intent.Amount - intent.ShippingCents - intent.TaxCents,
		ShippingCents:     intent.ShippingCents,
		TaxCents:          intent.TaxCents,
		TotalCents:        intent.Amount,
		Currency:          intent.Currency,
		CustomerNotes:     params.CustomerNotes,
	})
	if err != nil {
		return nil, err
	}

	// Step 8: Create order items
	for _, item := range cart.Items {
		_, err := tx.CreateOrderItem(ctx, CreateOrderItemParams{
			OrderID:     order.ID,
			SKUID:       item.SKUID,
			Quantity:    item.Quantity,
			UnitPrice:   item.PriceCents,
			TotalPrice:  item.TotalCents,
		})
		if err != nil {
			return nil, err
		}
	}

	// Step 9: Create payment record (links payment_intent_id)
	_, err = tx.CreatePayment(ctx, CreatePaymentParams{
		OrderID:            order.ID,
		Provider:           "stripe",
		ProviderPaymentID:  params.PaymentIntentID,
		Amount:             intent.Amount,
		Currency:           intent.Currency,
		Status:             "succeeded",
	})
	if err != nil {
		return nil, err
	}

	// Step 10: Decrement inventory
	for _, item := range cart.Items {
		err := tx.DecrementSKUStock(ctx, item.SKUID, item.Quantity)
		if err != nil {
			return nil, err
		}
	}

	// Step 11: Mark cart as converted
	err = tx.UpdateCartStatus(ctx, cart.ID, "converted", order.ID)
	if err != nil {
		return nil, err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return order, nil
}
```

### 2.3 Complete Stripe Integration

**File:** `internal/billing/stripe.go`

Currently all methods panic. Need to implement:
- `CreatePaymentIntent()` - Create payment intent with amount
- `GetPaymentIntent()` - Verify payment status
- `CreateCustomer()` - Create Stripe customer (future)
- `UpdatePaymentIntent()` - Update amount if cart changes

**Example implementation:**

```go
// internal/billing/stripe.go
package billing

import (
	"context"
	"fmt"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/paymentintent"
)

type stripeProvider struct {
	apiKey string
}

func NewStripeProvider(apiKey string) Provider {
	stripe.Key = apiKey
	return &stripeProvider{apiKey: apiKey}
}

func (p *stripeProvider) CreatePaymentIntent(ctx context.Context, params CreatePaymentIntentParams) (*PaymentIntent, error) {
	// Stripe amounts are in smallest currency unit (cents for USD)
	intentParams := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(int64(params.AmountCents)),
		Currency: stripe.String(params.Currency),
		Metadata: map[string]string{
			"cart_id":    params.CartID,
			"tenant_id":  params.TenantID,
			"order_type": "retail",
		},
	}

	// Set idempotency key to prevent duplicates
	intentParams.IdempotencyKey = stripe.String(params.IdempotencyKey)

	// Optional: Automatic tax calculation via Stripe Tax
	if params.EnableStripeTax {
		intentParams.AutomaticPaymentMethods = &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		}
		// Configure tax calculation
		// (requires Stripe Tax setup in dashboard)
	}

	intent, err := paymentintent.New(intentParams)
	if err != nil {
		return nil, fmt.Errorf("stripe: failed to create payment intent: %w", err)
	}

	return &PaymentIntent{
		ID:           intent.ID,
		ClientSecret: intent.ClientSecret,
		Amount:       int32(intent.Amount),
		Currency:     string(intent.Currency),
		Status:       string(intent.Status),
	}, nil
}

func (p *stripeProvider) GetPaymentIntent(ctx context.Context, id string) (*PaymentIntent, error) {
	intent, err := paymentintent.Get(id, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe: failed to get payment intent: %w", err)
	}

	return &PaymentIntent{
		ID:           intent.ID,
		ClientSecret: intent.ClientSecret,
		Amount:       int32(intent.Amount),
		Currency:     string(intent.Currency),
		Status:       string(intent.Status),
	}, nil
}
```

**Deliverables:**
- [ ] `internal/service/checkout.go` - CheckoutService implementation
- [ ] `internal/billing/stripe.go` - Complete Stripe integration
- [ ] `internal/service/checkout_test.go` - Unit tests with mocks
- [ ] Integration test with test database

---

## Phase 3: HTTP Handlers

**Duration:** 2-3 days
**Goal:** Build checkout flow handlers and templates

### 3.1 Checkout Handlers

**Files:**
- `internal/handler/storefront/checkout.go` - Main checkout handlers
- `internal/handler/storefront/checkout_test.go` - Handler tests

#### Route Structure

```
GET  /checkout                 → Cart review
POST /checkout/shipping        → Submit shipping address
POST /checkout/shipping-method → Select shipping method
POST /checkout/payment         → Create payment intent
POST /checkout/complete        → Complete order
GET  /checkout/confirmation/:orderID → Order confirmation
```

#### Handler: Cart Review

```go
// GET /checkout
func (h *CheckoutHandler) CartReview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get cart from session
	cartID := getCartIDFromSession(r)
	cart, err := h.cartService.GetCart(ctx, cartID)
	if err != nil {
		http.Error(w, "Cart not found", http.StatusNotFound)
		return
	}

	if len(cart.Items) == 0 {
		http.Redirect(w, r, "/cart", http.StatusSeeOther)
		return
	}

	data := BaseTemplateData(r)
	data["Cart"] = cart

	h.renderer.Render(w, "checkout/cart_review.html", data)
}
```

#### Handler: Shipping Address

```go
// POST /checkout/shipping
func (h *CheckoutHandler) SubmitShippingAddress(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse form
	r.ParseForm()
	addr := address.Address{
		Line1:      r.FormValue("address_line1"),
		Line2:      r.FormValue("address_line2"),
		City:       r.FormValue("city"),
		State:      r.FormValue("state"),
		PostalCode: r.FormValue("postal_code"),
		Country:    r.FormValue("country"),
	}

	// Validate address
	result, err := h.checkoutService.ValidateAndNormalizeAddress(ctx, addr)
	if err != nil {
		http.Error(w, "Address validation failed", http.StatusInternalServerError)
		return
	}

	if !result.IsValid {
		// Show validation errors
		data := BaseTemplateData(r)
		data["Address"] = addr
		data["ValidationErrors"] = result.Messages
		h.renderer.Render(w, "checkout/shipping_address.html", data)
		return
	}

	// Address valid, store in session and get shipping rates
	storeAddressInSession(r, result.NormalizedAddress)

	cartID := getCartIDFromSession(r)
	rates, err := h.checkoutService.GetShippingRates(ctx, cartID, result.NormalizedAddress)
	if err != nil {
		http.Error(w, "Failed to get shipping rates", http.StatusInternalServerError)
		return
	}

	data := BaseTemplateData(r)
	data["ShippingAddress"] = result.NormalizedAddress
	data["ShippingRates"] = rates

	h.renderer.Render(w, "checkout/shipping_method.html", data)
}
```

#### Handler: Complete Checkout

```go
// POST /checkout/complete
func (h *CheckoutHandler) CompleteCheckout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get payment intent ID from form
	paymentIntentID := r.FormValue("payment_intent_id")
	if paymentIntentID == "" {
		http.Error(w, "Missing payment intent", http.StatusBadRequest)
		return
	}

	// Get checkout data from session
	cartID := getCartIDFromSession(r)
	shippingAddr := getShippingAddressFromSession(r)
	billingAddr := getBillingAddressFromSession(r)
	shippingRateID := getShippingRateIDFromSession(r)

	// Complete checkout (idempotent)
	order, err := h.checkoutService.CompleteCheckout(ctx, service.CompleteCheckoutParams{
		CartID:          cartID,
		PaymentIntentID: paymentIntentID,
		ShippingAddress: shippingAddr,
		BillingAddress:  billingAddr,
		ShippingRateID:  shippingRateID,
		IdempotencyKey:  paymentIntentID, // Use payment intent ID as idempotency key
	})
	if err != nil {
		// Handle specific errors
		if errors.Is(err, service.ErrPaymentNotSucceeded) {
			http.Error(w, "Payment not completed", http.StatusBadRequest)
			return
		}
		if errors.Is(err, service.ErrInsufficientInventory) {
			http.Error(w, "Some items are out of stock", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to create order", http.StatusInternalServerError)
		return
	}

	// Clear cart session
	clearCartSession(w, r)

	// Redirect to confirmation
	http.Redirect(w, r, fmt.Sprintf("/checkout/confirmation/%s", order.ID), http.StatusSeeOther)
}
```

**Deliverables:**
- [ ] `internal/handler/storefront/checkout.go` - All checkout handlers
- [ ] Session helpers for storing checkout state
- [ ] Error handling for all edge cases
- [ ] Handler tests

---

### 3.2 Templates

**Files:**
- `web/templates/checkout/cart_review.html`
- `web/templates/checkout/shipping_address.html`
- `web/templates/checkout/shipping_method.html`
- `web/templates/checkout/payment.html`
- `web/templates/checkout/confirmation.html`

#### Template: Payment with Stripe.js

```html
{{define "title"}}Payment - Freyja Coffee{{end}}

{{define "content"}}
<div class="max-w-2xl mx-auto px-4 py-8">
  <h1 class="text-2xl font-semibold mb-8">Payment</h1>

  <!-- Order Summary -->
  <div class="bg-neutral-50 p-6 rounded-lg mb-8">
    <h2 class="font-medium mb-4">Order Summary</h2>
    <div class="space-y-2">
      <div class="flex justify-between">
        <span>Subtotal</span>
        <span>${{.OrderTotal.SubtotalCents | formatCents}}</span>
      </div>
      <div class="flex justify-between">
        <span>Shipping</span>
        <span>${{.OrderTotal.ShippingCents | formatCents}}</span>
      </div>
      <div class="flex justify-between">
        <span>Tax</span>
        <span>${{.OrderTotal.TaxCents | formatCents}}</span>
      </div>
      <div class="flex justify-between font-semibold text-lg border-t pt-2 mt-2">
        <span>Total</span>
        <span>${{.OrderTotal.TotalCents | formatCents}}</span>
      </div>
    </div>
  </div>

  <!-- Payment Form -->
  <form id="payment-form">
    <div id="card-element" class="p-4 border border-neutral-300 rounded-lg mb-4">
      <!-- Stripe.js injects card input here -->
    </div>

    <div id="card-errors" class="text-red-600 text-sm mb-4"></div>

    <button
      id="submit-button"
      type="submit"
      class="w-full px-6 py-3 bg-teal-700 text-white font-medium rounded-lg hover:bg-teal-800"
    >
      Pay ${{.OrderTotal.TotalCents | formatCents}}
    </button>
  </form>
</div>

<script src="https://js.stripe.com/v3/"></script>
<script>
  const stripe = Stripe('{{.StripePublishableKey}}');
  const elements = stripe.elements();
  const cardElement = elements.create('card');
  cardElement.mount('#card-element');

  const form = document.getElementById('payment-form');
  const clientSecret = '{{.ClientSecret}}';

  form.addEventListener('submit', async (e) => {
    e.preventDefault();

    const {error, paymentIntent} = await stripe.confirmCardPayment(clientSecret, {
      payment_method: {
        card: cardElement,
      }
    });

    if (error) {
      document.getElementById('card-errors').textContent = error.message;
    } else if (paymentIntent.status === 'succeeded') {
      // Submit to complete checkout
      const completeForm = document.createElement('form');
      completeForm.method = 'POST';
      completeForm.action = '/checkout/complete';

      const input = document.createElement('input');
      input.type = 'hidden';
      input.name = 'payment_intent_id';
      input.value = paymentIntent.id;

      completeForm.appendChild(input);
      document.body.appendChild(completeForm);
      completeForm.submit();
    }
  });
</script>
{{end}}
```

**Deliverables:**
- [ ] All checkout templates
- [ ] Stripe.js integration
- [ ] Loading states and error handling
- [ ] Mobile responsive design

---

## Phase 4: Database

**Duration:** 1-2 days
**Goal:** Add necessary queries and migrations

### 4.1 Migrations

#### Migration 00016: Payment Idempotency

```sql
-- +goose Up
-- Prevent duplicate orders from same payment intent
ALTER TABLE payments
  ADD CONSTRAINT unique_provider_payment_id
  UNIQUE (tenant_id, provider_payment_id);

-- +goose Down
ALTER TABLE payments
  DROP CONSTRAINT unique_provider_payment_id;
```

#### Migration 00017: Checkout Indexes

```sql
-- +goose Up
-- Fast lookup for idempotency check
CREATE INDEX idx_payments_provider_payment_id
  ON payments(tenant_id, provider_payment_id)
  WHERE provider_payment_id IS NOT NULL;

-- Fast lookup of converted carts
CREATE INDEX idx_carts_converted_to_order_id
  ON carts(tenant_id, converted_to_order_id)
  WHERE converted_to_order_id IS NOT NULL;

-- Cart status filtering
CREATE INDEX idx_carts_status
  ON carts(tenant_id, status);

-- +goose Down
DROP INDEX idx_payments_provider_payment_id;
DROP INDEX idx_carts_converted_to_order_id;
DROP INDEX idx_carts_status;
```

### 4.2 sqlc Queries

#### Orders

```sql
-- name: CreateOrder :one
INSERT INTO orders (
    tenant_id,
    user_id,
    shipping_address_id,
    billing_address_id,
    subtotal_cents,
    shipping_cents,
    tax_cents,
    discount_cents,
    total_cents,
    currency,
    status,
    customer_notes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'pending', $11
) RETURNING *;

-- name: CreateOrderItem :one
INSERT INTO order_items (
    order_id,
    sku_id,
    quantity,
    unit_price_cents,
    total_price_cents
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetOrderByPaymentIntentID :one
SELECT o.*
FROM orders o
JOIN payments p ON p.order_id = o.id
WHERE p.tenant_id = $1
  AND p.provider_payment_id = $2
LIMIT 1;

-- name: CreatePayment :one
INSERT INTO payments (
    tenant_id,
    order_id,
    provider,
    provider_payment_id,
    amount_cents,
    currency,
    status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;
```

#### Addresses

```sql
-- name: CreateAddress :one
INSERT INTO addresses (
    tenant_id,
    address_type,
    full_name,
    company,
    address_line1,
    address_line2,
    city,
    state,
    postal_code,
    country,
    phone,
    email
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
) RETURNING *;
```

#### Inventory

```sql
-- name: DecrementSKUStock :exec
UPDATE product_skus
SET stock_quantity = stock_quantity - $2
WHERE id = $1
  AND stock_quantity >= $2;

-- name: GetSKUStock :one
SELECT stock_quantity
FROM product_skus
WHERE id = $1;
```

**Deliverables:**
- [ ] Migration 00016
- [ ] Migration 00017
- [ ] `sqlc/queries/orders.sql`
- [ ] `sqlc/queries/addresses.sql`
- [ ] Run `sqlc generate`

---

## Phase 5: Testing & Polish

**Duration:** 2-3 days
**Goal:** Comprehensive testing and UX refinement

### 5.1 Integration Tests

Test complete checkout flow:

```go
// internal/service/checkout_integration_test.go
func TestCheckout_CompletePurchase(t *testing.T) {
	// Setup test database
	db := setupTestDB(t)
	defer db.Close()

	// Create test data
	tenant := createTestTenant(db)
	product := createTestProduct(db, tenant.ID)
	cart := createTestCart(db, tenant.ID)
	addItemToCart(db, cart.ID, product.SKU[0].ID, 2)

	// Initialize services with real implementations
	repo := repository.New(db)
	taxCalc := tax.NewPercentageCalculator(0.08)
	shippingProv := shipping.NewFlatRateProvider(testRates)
	addrValidator := address.NewBasicValidator()
	billingProv := billing.NewStripeProvider(testStripeKey)

	checkoutSvc := service.NewCheckoutService(
		repo, cartSvc, billingProv, shippingProv, taxCalc, addrValidator, tenant.ID,
	)

	// Step 1: Get shipping rates
	rates, err := checkoutSvc.GetShippingRates(ctx, cart.ID, testAddress)
	assert.NoError(t, err)
	assert.NotEmpty(t, rates)

	// Step 2: Calculate total
	total, err := checkoutSvc.CalculateOrderTotal(ctx, service.OrderTotalParams{
		CartID:               cart.ID,
		ShippingAddress:      testAddress,
		SelectedShippingRate: rates[0],
	})
	assert.NoError(t, err)
	assert.Greater(t, total.TotalCents, int32(0))

	// Step 3: Create payment intent
	intent, err := checkoutSvc.CreatePaymentIntent(ctx, service.PaymentIntentParams{
		CartID:         cart.ID,
		OrderTotal:     total,
		CustomerEmail:  "test@example.com",
		IdempotencyKey: "test-" + uuid.NewString(),
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, intent.ClientSecret)

	// Simulate successful payment (use Stripe test mode)
	// ... confirm payment via Stripe API ...

	// Step 4: Complete checkout
	order, err := checkoutSvc.CompleteCheckout(ctx, service.CompleteCheckoutParams{
		CartID:          cart.ID,
		PaymentIntentID: intent.ID,
		ShippingAddress: testAddress,
		BillingAddress:  testAddress,
		ShippingRateID:  rates[0].RateID,
		IdempotencyKey:  intent.ID,
	})
	assert.NoError(t, err)
	assert.Equal(t, "pending", order.Status)

	// Verify order in database
	dbOrder := getOrderFromDB(db, order.ID)
	assert.Equal(t, total.TotalCents, dbOrder.TotalCents)

	// Verify inventory decremented
	sku := getSKUFromDB(db, product.SKU[0].ID)
	assert.Equal(t, originalStock-2, sku.StockQuantity)

	// Verify cart converted
	dbCart := getCartFromDB(db, cart.ID)
	assert.Equal(t, "converted", dbCart.Status)
}
```

### 5.2 Error Scenario Tests

```go
func TestCheckout_InsufficientInventory(t *testing.T) {
	// Create cart with item that has only 1 in stock
	// Try to checkout with quantity 2
	// Expect ErrInsufficientInventory
}

func TestCheckout_PaymentFailed(t *testing.T) {
	// Use Stripe test card that declines
	// Expect payment creation to fail
	// Verify cart remains active
	// Verify inventory not decremented
}

func TestCheckout_IdempotentOrderCreation(t *testing.T) {
	// Complete checkout once
	// Call CompleteCheckout again with same payment_intent_id
	// Expect same order returned, no duplicate created
}
```

### 5.3 UX Polish

- [ ] Loading spinners during async operations
- [ ] Address autocomplete (Google Places API - optional)
- [ ] Form field validation (client-side + server-side)
- [ ] Clear error messages for all failure cases
- [ ] Back button handling (allow return to previous step)
- [ ] Mobile responsive testing

**Deliverables:**
- [ ] Integration test suite
- [ ] Error scenario tests
- [ ] End-to-end manual testing
- [ ] UX refinements

---

## Implementation Checklist

### Phase 1: Interfaces & Providers ✓

- [ ] Tax package
  - [ ] Interface definition
  - [ ] PercentageCalculator
  - [ ] NoTaxCalculator
  - [ ] Mock implementation
  - [ ] Unit tests

- [ ] Shipping package
  - [ ] Interface definition
  - [ ] FlatRateProvider
  - [ ] Mock implementation
  - [ ] Unit tests

- [ ] Address package
  - [ ] Interface definition
  - [ ] BasicValidator
  - [ ] Mock implementation
  - [ ] Unit tests

### Phase 2: Checkout Service ✓

- [ ] CheckoutService interface
- [ ] GetShippingRates implementation
- [ ] CalculateOrderTotal implementation
- [ ] CompleteCheckout implementation with:
  - [ ] Idempotency check
  - [ ] Payment verification
  - [ ] Database transaction
  - [ ] Inventory check
  - [ ] Order creation
  - [ ] Cart conversion
- [ ] Stripe integration
  - [ ] CreatePaymentIntent
  - [ ] GetPaymentIntent
- [ ] Unit tests with mocks

### Phase 3: HTTP Handlers ✓

- [ ] Checkout handlers
  - [ ] GET /checkout (cart review)
  - [ ] POST /checkout/shipping
  - [ ] POST /checkout/shipping-method
  - [ ] POST /checkout/payment
  - [ ] POST /checkout/complete
  - [ ] GET /checkout/confirmation/:id
- [ ] Templates
  - [ ] cart_review.html
  - [ ] shipping_address.html
  - [ ] shipping_method.html
  - [ ] payment.html
  - [ ] confirmation.html
- [ ] Session helpers
- [ ] Handler tests

### Phase 4: Database ✓

- [ ] Migration 00016 (payment idempotency)
- [ ] Migration 00017 (indexes)
- [ ] sqlc queries
  - [ ] CreateOrder
  - [ ] CreateOrderItem
  - [ ] GetOrderByPaymentIntentID
  - [ ] CreatePayment
  - [ ] CreateAddress
  - [ ] DecrementSKUStock
  - [ ] GetSKUStock
- [ ] Run sqlc generate

### Phase 5: Testing ✓

- [ ] Integration tests
  - [ ] Happy path (cart → order)
  - [ ] Payment failure
  - [ ] Insufficient inventory
  - [ ] Idempotent order creation
- [ ] Manual testing
  - [ ] Desktop browsers
  - [ ] Mobile browsers
  - [ ] Stripe test cards
- [ ] UX polish
  - [ ] Loading states
  - [ ] Error messages
  - [ ] Form validation

---

## Configuration

### Environment Variables

```bash
# Stripe
STRIPE_SECRET_KEY=sk_test_...
STRIPE_PUBLISHABLE_KEY=pk_test_...

# Tax (if using percentage calculator)
DEFAULT_TAX_RATE=0.08

# Shipping (if using flat rate)
SHIPPING_STANDARD_CENTS=500
SHIPPING_EXPRESS_CENTS=1500
```

### main.go Wiring

```go
// Initialize providers
taxCalc := tax.NewPercentageCalculator(cfg.DefaultTaxRate)
shippingProv := shipping.NewFlatRateProvider([]shipping.FlatRate{
    {ServiceName: "Standard", ServiceCode: "STD", CostCents: 500, DaysMin: 3, DaysMax: 5},
    {ServiceName: "Express", ServiceCode: "EXP", CostCents: 1500, DaysMin: 1, DaysMax: 2},
})
addrValidator := address.NewBasicValidator()
billingProv := billing.NewStripeProvider(cfg.StripeSecretKey)

// Initialize checkout service
checkoutSvc := service.NewCheckoutService(
    repo,
    cartSvc,
    billingProv,
    shippingProv,
    taxCalc,
    addrValidator,
    cfg.TenantID,
)

// Initialize handlers
checkoutHandler := storefront.NewCheckoutHandler(checkoutSvc, renderer)

// Routes
r.Get("/checkout", checkoutHandler.CartReview)
r.Post("/checkout/shipping", checkoutHandler.SubmitShippingAddress)
r.Post("/checkout/shipping-method", checkoutHandler.SelectShippingMethod)
r.Post("/checkout/payment", checkoutHandler.CreatePayment)
r.Post("/checkout/complete", checkoutHandler.CompleteCheckout)
r.Get("/checkout/confirmation/{orderID}", checkoutHandler.Confirmation)
```

---

## Success Criteria

Checkout is complete when:

1. ✓ Customer can add items to cart and proceed to checkout
2. ✓ Shipping address is validated and stored
3. ✓ Shipping options are presented and selectable
4. ✓ Tax is calculated (or skipped if using Stripe Tax)
5. ✓ Order total is correct (subtotal + shipping + tax - discounts)
6. ✓ Payment Intent is created with correct amount
7. ✓ Stripe.js collects payment securely
8. ✓ Order is created in database after successful payment
9. ✓ Inventory is decremented
10. ✓ Cart is marked as converted
11. ✓ Customer sees order confirmation
12. ✓ Duplicate orders prevented (idempotency)
13. ✓ All error cases handled gracefully

---

## Future Enhancements (Post-MVP)

- [ ] Save payment methods for repeat customers
- [ ] One-click checkout for returning customers
- [ ] Guest checkout with email-only (no account required)
- [ ] Discount code support
- [ ] Gift card support
- [ ] Multiple shipping addresses per order
- [ ] Estimated delivery date display
- [ ] Real-time inventory updates
- [ ] Stripe webhooks for async payment confirmation
- [ ] Cart abandonment emails
- [ ] Shipping label generation (FedEx/UPS API)
- [ ] Advanced tax providers (TaxJar, Avalara)
- [ ] Address validation API (Lob, SmartyStreets)
- [ ] International shipping
- [ ] Multi-currency support

---

## Notes

- **Tax Flexibility:** The architecture supports three tax strategies:
  1. Application calculates (PercentageCalculator)
  2. Stripe calculates (StripeTaxCalculator - delegates to Stripe Tax)
  3. No tax (NoTaxCalculator - wholesale/exempt)

- **Idempotency:** All order creation is idempotent via Payment Intent ID. Multiple calls with same payment intent return existing order.

- **Transaction Safety:** Order creation is wrapped in database transaction. Any failure rolls back all changes.

- **Testing:** Use Stripe test mode and test cards:
  - `4242 4242 4242 4242` - Success
  - `4000 0000 0000 0002` - Decline
  - Full list: https://stripe.com/docs/testing

- **Security:** Payment details never touch our server - Stripe.js sends directly to Stripe (PCI compliant).
