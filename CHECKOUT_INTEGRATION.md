# Checkout Templates - Integration Guide

## Overview

This guide explains how to integrate the checkout templates with your Go handlers and API endpoints.

## Files Created

### Template Files
1. **`/web/templates/storefront/checkout.html`** - Main checkout page (28KB)
2. **`/web/templates/storefront/order-confirmation.html`** - Order confirmation page (7.3KB)
3. **`/web/templates/storefront/checkout_partials.html`** - htmx partial responses (7.6KB)

### Documentation
1. **`/CHECKOUT_UX_FLOW.md`** - Complete UX flow documentation
2. **`/CHECKOUT_INTEGRATION.md`** - This file

## Required API Endpoints

### 1. Main Checkout Page
**Route:** `GET /checkout`

**Handler Responsibilities:**
- Retrieve cart contents from session
- Calculate cart subtotal
- Render checkout template

**Template Data:**
```go
type CheckoutPageData struct {
    Cart struct {
        Items []struct {
            ProductName    string
            SKU           string
            Quantity      int
            UnitPriceCents int
            LineSubtotal  int
        }
        SubtotalCents int
        ItemCount     int
    }
}
```

**Example Handler:**
```go
func (h *Handler) GetCheckout(w http.ResponseWriter, r *http.Request) {
    cart, err := h.cartService.GetCart(r.Context(), sessionID)
    if err != nil {
        http.Error(w, "Failed to load cart", http.StatusInternalServerError)
        return
    }

    data := CheckoutPageData{Cart: cart}
    h.templates.ExecuteTemplate(w, "checkout.html", data)
}
```

---

### 2. Validate Addresses
**Route:** `POST /api/checkout/validate-addresses`

**Request Body (form-encoded):**
```
shipping_name: string
shipping_address1: string
shipping_address2: string (optional)
shipping_city: string
shipping_state: string (2 letters)
shipping_postal_code: string
```

**Response Templates:**
- Success: `{{template "address_validation_success" .}}`
- Error: `{{template "address_validation_error" .}}`

**Success Data:**
```go
type AddressValidationSuccess struct {
    AddressNormalized bool
    NormalizedAddress *Address // if normalized
}
```

**Error Data:**
```go
type AddressValidationError struct {
    ErrorMessage string
}
```

**Example Handler:**
```go
func (h *Handler) ValidateAddress(w http.ResponseWriter, r *http.Request) {
    r.ParseForm()

    addr := Address{
        Name:       r.FormValue("shipping_name"),
        Address1:   r.FormValue("shipping_address1"),
        Address2:   r.FormValue("shipping_address2"),
        City:       r.FormValue("shipping_city"),
        State:      r.FormValue("shipping_state"),
        PostalCode: r.FormValue("shipping_postal_code"),
    }

    validated, normalized, err := h.checkoutService.ValidateAddress(r.Context(), addr)
    if err != nil {
        data := AddressValidationError{ErrorMessage: err.Error()}
        h.templates.ExecuteTemplate(w, "address_validation_error", data)
        return
    }

    data := AddressValidationSuccess{
        AddressNormalized: normalized != nil,
        NormalizedAddress: normalized,
    }
    h.templates.ExecuteTemplate(w, "address_validation_success", data)
}
```

---

### 3. Get Shipping Rates
**Route:** `POST /api/checkout/shipping-rates`

**Request Body:** (pulls from session - shipping address already validated)

**Response Template:**
- `{{template "shipping_rates" .}}`

**Response Data:**
```go
type ShippingRatesResponse struct {
    Rates []struct {
        ID            string
        Carrier       string
        Service       string
        RateCents     int
        EstimatedDays string
    }
}
```

**Example Handler:**
```go
func (h *Handler) GetShippingRates(w http.ResponseWriter, r *http.Request) {
    // Get shipping address from session
    addr, err := h.sessionService.GetShippingAddress(r.Context(), sessionID)
    if err != nil {
        http.Error(w, "Shipping address not found", http.StatusBadRequest)
        return
    }

    // Get cart items for weight calculation
    cart, _ := h.cartService.GetCart(r.Context(), sessionID)

    rates, err := h.checkoutService.GetShippingRates(r.Context(), addr, cart)
    if err != nil {
        data := ShippingRatesResponse{Rates: nil}
        h.templates.ExecuteTemplate(w, "shipping_rates", data)
        return
    }

    data := ShippingRatesResponse{Rates: rates}
    h.templates.ExecuteTemplate(w, "shipping_rates", data)
}
```

---

### 4. Calculate Total
**Route:** `POST /api/checkout/calculate-total`

**Request Body (form-encoded):**
```
shipping_cents: int
```

**Response Template:**
- `{{template "order_total" .}}`

**Response Data:**
```go
type OrderTotalResponse struct {
    SubtotalCents int
    ShippingCents int
    TaxCents      int
    TotalCents    int
}
```

**Example Handler:**
```go
func (h *Handler) CalculateTotal(w http.ResponseWriter, r *http.Request) {
    r.ParseForm()

    shippingCents, _ := strconv.Atoi(r.FormValue("shipping_cents"))

    cart, _ := h.cartService.GetCart(r.Context(), sessionID)
    addr, _ := h.sessionService.GetShippingAddress(r.Context(), sessionID)

    total, err := h.checkoutService.CalculateTotal(r.Context(), cart, addr, shippingCents)
    if err != nil {
        http.Error(w, "Failed to calculate total", http.StatusInternalServerError)
        return
    }

    h.templates.ExecuteTemplate(w, "order_total", total)
}
```

---

### 5. Create Payment Intent
**Route:** `POST /api/checkout/create-payment-intent`

**Request Body:** (pulls from session)

**Response Templates:**
- Success: `{{template "payment_intent_created" .}}`
- Error: `{{template "payment_intent_error" .}}`

**Success Data:**
```go
type PaymentIntentCreated struct {
    ClientSecret          string
    StripePublishableKey  string
}
```

**Error Data:**
```go
type PaymentIntentError struct {
    ErrorMessage string
}
```

**Example Handler:**
```go
func (h *Handler) CreatePaymentIntent(w http.ResponseWriter, r *http.Request) {
    // Get order total from session
    total, _ := h.sessionService.GetOrderTotal(r.Context(), sessionID)

    intent, err := h.checkoutService.CreatePaymentIntent(r.Context(), total.TotalCents)
    if err != nil {
        data := PaymentIntentError{ErrorMessage: err.Error()}
        h.templates.ExecuteTemplate(w, "payment_intent_error", data)
        return
    }

    data := PaymentIntentCreated{
        ClientSecret:         intent.ClientSecret,
        StripePublishableKey: h.config.StripePublishableKey,
    }
    h.templates.ExecuteTemplate(w, "payment_intent_created", data)
}
```

---

### 6. Complete Checkout
**Route:** `GET /checkout/complete`

**Query Params:**
- `payment_intent` - Stripe payment intent ID
- `payment_intent_client_secret` - Client secret for verification

**Handler Responsibilities:**
1. Verify payment intent with Stripe
2. Create order record in database
3. Clear cart session
4. Redirect to order confirmation

**Example Handler:**
```go
func (h *Handler) CompleteCheckout(w http.ResponseWriter, r *http.Request) {
    paymentIntentID := r.URL.Query().Get("payment_intent")

    // Verify payment with Stripe
    intent, err := h.checkoutService.VerifyPayment(r.Context(), paymentIntentID)
    if err != nil || intent.Status != "succeeded" {
        http.Redirect(w, r, "/checkout?error=payment_failed", http.StatusSeeOther)
        return
    }

    // Create order
    order, err := h.checkoutService.CreateOrder(r.Context(), sessionID, paymentIntentID)
    if err != nil {
        http.Error(w, "Failed to create order", http.StatusInternalServerError)
        return
    }

    // Clear cart
    h.cartService.Clear(r.Context(), sessionID)

    // Redirect to confirmation
    http.Redirect(w, r, "/order-confirmation?order="+order.OrderNumber, http.StatusSeeOther)
}
```

---

### 7. Order Confirmation Page
**Route:** `GET /order-confirmation`

**Query Params:**
- `order` - Order number

**Template Data:**
```go
type OrderConfirmationData struct {
    Order struct {
        OrderNumber                string
        Email                      string
        SubtotalCents              int
        ShippingCents              int
        TaxCents                   int
        TotalCents                 int
        BillingAddressSameAsShipping bool
        CreatedAt                  time.Time
    }
    ShippingAddress Address
    BillingAddress  Address
    Items []struct {
        ProductName    string
        SKU           string
        Quantity      int
        UnitPriceCents int
        LineSubtotal  int
    }
    User *User // nil if guest checkout
}
```

**Example Handler:**
```go
func (h *Handler) GetOrderConfirmation(w http.ResponseWriter, r *http.Request) {
    orderNumber := r.URL.Query().Get("order")

    order, err := h.orderService.GetOrderByNumber(r.Context(), orderNumber)
    if err != nil {
        http.Error(w, "Order not found", http.StatusNotFound)
        return
    }

    data := OrderConfirmationData{
        Order:           order,
        ShippingAddress: order.ShippingAddress,
        BillingAddress:  order.BillingAddress,
        Items:           order.Items,
        User:            getCurrentUser(r),
    }

    h.templates.ExecuteTemplate(w, "order-confirmation.html", data)
}
```

---

## Session Management

The checkout flow relies on session storage for maintaining state between steps:

**Session Keys:**
- `cart` - Cart contents
- `checkout.email` - Contact email
- `checkout.phone` - Contact phone
- `checkout.shipping_address` - Validated shipping address
- `checkout.shipping_rate_id` - Selected shipping rate
- `checkout.billing_address` - Billing address (if different)
- `checkout.order_total` - Calculated total with tax

**Session Flow:**
1. Step 1 completes → Store email/phone
2. Step 2 completes → Store validated shipping address
3. Step 3 completes → Store selected rate, calculated total
4. Step 4 completes → Store billing address (if different)
5. Step 5 completes → Create order, clear session

---

## Template Registration

Ensure all templates are registered with your template engine:

```go
// Parse all storefront templates
templates := template.Must(template.ParseGlob("web/templates/storefront/*.html"))
templates = template.Must(templates.ParseGlob("web/templates/*.html"))

// Make available to handlers
handler := &Handler{
    templates: templates,
    // ... other dependencies
}
```

---

## Routing Configuration

Add these routes to your router:

```go
// Storefront checkout routes
r.Get("/checkout", handler.GetCheckout)
r.Get("/checkout/complete", handler.CompleteCheckout)
r.Get("/order-confirmation", handler.GetOrderConfirmation)

// API routes (htmx endpoints)
r.Post("/api/checkout/validate-addresses", handler.ValidateAddress)
r.Post("/api/checkout/shipping-rates", handler.GetShippingRates)
r.Post("/api/checkout/calculate-total", handler.CalculateTotal)
r.Post("/api/checkout/create-payment-intent", handler.CreatePaymentIntent)
```

---

## Environment Variables

Required configuration:

```bash
# Stripe (required for payment processing)
STRIPE_PUBLISHABLE_KEY=pk_test_...
STRIPE_SECRET_KEY=sk_test_...

# Address validation (optional, fallback to basic validation)
USPS_API_KEY=...

# Tax calculation (required)
TAX_JAR_API_KEY=...  # or alternative tax service
```

---

## Testing Checklist

### Unit Tests
- [ ] Address validation logic
- [ ] Shipping rate calculation
- [ ] Tax calculation
- [ ] Payment intent creation
- [ ] Order creation

### Integration Tests
- [ ] Full checkout flow (happy path)
- [ ] Address validation errors
- [ ] No shipping rates available
- [ ] Payment failures
- [ ] Session persistence

### Manual Testing
- [ ] Complete checkout on desktop
- [ ] Complete checkout on mobile
- [ ] Edit completed steps
- [ ] Address validation with various inputs
- [ ] Different billing address flow
- [ ] Payment with test cards
- [ ] Order confirmation displays correctly

### Accessibility Testing
- [ ] Keyboard navigation through all steps
- [ ] Screen reader announces step changes
- [ ] Error messages announced
- [ ] All form fields labeled correctly
- [ ] Color contrast meets WCAG AA

---

## Error Scenarios

Handle these edge cases:

1. **Empty Cart:** Redirect to cart page with message
2. **Invalid Session:** Redirect to login or cart
3. **Address Validation Timeout:** Show error, allow retry
4. **No Shipping Rates:** Show contact prompt
5. **Payment Intent Creation Fails:** Show error, allow retry
6. **Payment Confirmation Fails:** Don't create order, show error
7. **Stripe Webhook Delay:** Handle async confirmation

---

## Performance Optimization

1. **Cache Shipping Rates:** Store in session for 10 minutes
2. **Lazy Load Stripe.js:** Only load when payment step visible
3. **Debounce Tax Calculation:** Avoid excessive API calls
4. **Session Cleanup:** Remove expired checkout sessions

---

## Security Considerations

1. **CSRF Protection:** All POST endpoints require CSRF token
2. **Session Validation:** Verify session on every step
3. **Price Recalculation:** Never trust client-side totals
4. **Payment Verification:** Always verify payment intent server-side
5. **Order Deduplication:** Check for duplicate orders by payment intent ID

---

## Monitoring & Logging

Log these events:

- Checkout initiated (cart value, item count)
- Address validation (success/failure, corrections made)
- Shipping rates retrieved (number of options, selected rate)
- Payment intent created (amount, currency)
- Payment succeeded (order number, total)
- Payment failed (reason, step)
- Checkout abandoned (step, time spent)

**Metrics to Track:**
- Checkout completion rate
- Average time per step
- Most common abandonment points
- Address validation success rate
- Payment success rate

---

## Next Steps

1. Implement backend handlers for all API endpoints
2. Set up Stripe webhook listener for payment confirmation
3. Configure tax calculation service (TaxJar/Avalara)
4. Set up email notifications (order confirmation, shipping updates)
5. Add error tracking (Sentry/Rollbar)
6. Configure monitoring dashboards

---

**Document Version:** 1.0
**Last Updated:** 2025-11-29
**Status:** Ready for Implementation
