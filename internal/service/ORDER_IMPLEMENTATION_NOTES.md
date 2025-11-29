# Order Service Implementation Notes

This document provides guidance for implementing the OrderService skeleton defined in `order.go`.

## Overview

The OrderService handles order creation from successful Stripe payments, with a focus on:
- Idempotency (handling webhook retries)
- Multi-tenant isolation
- Atomic transactions
- Inventory management
- Complete order state capture

## Architecture Context

### Dependencies

The OrderService depends on:
- `repository.Querier` - Database operations (sqlc-generated)
- `billing.Provider` - Payment provider integration (Stripe)
- `shipping.Provider` - Shipping rate calculation (future use)

### Data Flow

```
Stripe Webhook (payment_intent.succeeded)
  → OrderService.CreateOrderFromPaymentIntent()
    → Check idempotency (existing order?)
    → Validate payment succeeded
    → Load cart and items
    → Begin transaction
      → Create addresses
      → Create billing customer
      → Create payment record
      → Generate order number
      → Create order
      → Create order items
      → Decrement inventory
      → Mark cart as converted
      → Link payment to order
    → Commit transaction
  → Return OrderDetail
```

## Implementation Guide

### Step-by-Step Breakdown

#### Step 1: Idempotency Check

**Purpose:** Prevent duplicate order creation from webhook retries.

**Implementation:**
```go
order, err := s.repo.GetOrderByPaymentIntentID(ctx, repository.GetOrderByPaymentIntentIDParams{
    TenantID:          s.tenantID,
    ProviderPaymentID: paymentIntentID,
})
if err == nil {
    // Order already exists - return it
    return s.buildOrderDetail(ctx, order)
}
if !errors.Is(err, sql.ErrNoRows) {
    return nil, fmt.Errorf("idempotency check failed: %w", err)
}
// No existing order - proceed with creation
```

**Error Handling:**
- If order found: Return existing order (success case)
- If `sql.ErrNoRows`: Continue to step 2
- Any other error: Return error

#### Step 2: Retrieve Payment Intent

**Purpose:** Get payment details and verify it's from Stripe.

**Implementation:**
```go
paymentIntent, err := s.billingProvider.GetPaymentIntent(ctx, billing.GetPaymentIntentParams{
    PaymentIntentID: paymentIntentID,
    TenantID:        s.tenantID.Bytes[:], // Convert UUID to string
})
if err != nil {
    return nil, fmt.Errorf("failed to retrieve payment intent: %w", err)
}
```

**Notes:**
- The billing provider will verify the tenant_id in metadata matches
- This prevents cross-tenant payment processing
- Payment intent contains all checkout metadata (cart_id, addresses, etc.)

#### Step 3: Validate Payment Status

**Purpose:** Ensure payment actually succeeded before creating order.

**Implementation:**
```go
if paymentIntent.Status != "succeeded" {
    return nil, ErrPaymentNotSucceeded
}
```

**Why:** Webhooks may fire for various payment intent events. Only "succeeded" creates orders.

#### Step 4: Extract Cart ID

**Purpose:** Link order to the cart that was converted.

**Implementation:**
```go
cartID, ok := paymentIntent.Metadata["cart_id"]
if !ok || cartID == "" {
    return nil, ErrMissingCartID
}

var cartUUID pgtype.UUID
if err := cartUUID.Scan(cartID); err != nil {
    return nil, fmt.Errorf("invalid cart_id in payment metadata: %w", err)
}
```

**Metadata Structure:**
Payment intent metadata should contain:
- `cart_id` - UUID of cart being converted
- `tenant_id` - For validation
- `shipping_address` - JSON encoded address
- `billing_address` - JSON encoded address
- `customer_notes` - Optional notes

#### Step 5: Retrieve and Validate Cart

**Purpose:** Load cart, verify tenant isolation, check conversion status.

**Implementation:**
```go
cart, err := s.repo.GetCartByID(ctx, cartUUID)
if err != nil {
    if errors.Is(err, sql.ErrNoRows) {
        return nil, ErrCartNotFound
    }
    return nil, fmt.Errorf("failed to get cart: %w", err)
}

// Tenant isolation check
if !bytes.Equal(cart.TenantID.Bytes[:], s.tenantID.Bytes[:]) {
    return nil, ErrTenantMismatch
}

// Prevent double conversion
if cart.Status == "converted" {
    return nil, ErrCartAlreadyConverted
}
```

**Security:** The tenant_id check is critical for multi-tenant isolation.

#### Step 6: Load Cart Items

**Purpose:** Get product details and pricing for order creation.

**Implementation:**
```go
cartItems, err := s.repo.GetCartItems(ctx, cartUUID)
if err != nil {
    return nil, fmt.Errorf("failed to get cart items: %w", err)
}

if len(cartItems) == 0 {
    return nil, fmt.Errorf("cannot create order from empty cart")
}
```

**Notes:**
- GetCartItems returns enriched data with product names, SKUs, etc.
- This data is captured in order_items to preserve state at purchase time

#### Step 7: Parse Addresses

**Purpose:** Extract shipping/billing addresses from payment metadata.

**Implementation:**
```go
type AddressData struct {
    FullName     string `json:"full_name"`
    Company      string `json:"company"`
    AddressLine1 string `json:"address_line1"`
    AddressLine2 string `json:"address_line2"`
    City         string `json:"city"`
    State        string `json:"state"`
    PostalCode   string `json:"postal_code"`
    Country      string `json:"country"`
    Phone        string `json:"phone"`
}

var shippingAddr AddressData
if err := json.Unmarshal([]byte(paymentIntent.Metadata["shipping_address"]), &shippingAddr); err != nil {
    return nil, fmt.Errorf("invalid shipping address in payment metadata: %w", err)
}

var billingAddr AddressData
if err := json.Unmarshal([]byte(paymentIntent.Metadata["billing_address"]), &billingAddr); err != nil {
    return nil, fmt.Errorf("invalid billing address in payment metadata: %w", err)
}
```

**Why JSON?** Stripe metadata values are strings, so complex data must be JSON-encoded.

#### Step 8-19: Transaction Management

**Purpose:** Ensure all database operations succeed or roll back together.

**Pattern:**
```go
tx, err := s.repo.BeginTx(ctx)
if err != nil {
    return nil, fmt.Errorf("failed to begin transaction: %w", err)
}
defer tx.Rollback(ctx) // Safe to call after commit

txQueries := s.repo.WithTx(tx)

// ... perform all operations using txQueries ...

if err := tx.Commit(ctx); err != nil {
    return nil, fmt.Errorf("failed to commit order transaction: %w", err)
}
```

**Critical:** All operations between steps 9-18 must use `txQueries`, not `s.repo`.

#### Step 13: Order Number Generation

**Purpose:** Generate human-readable, unique order identifier.

**Format:** `ORD-YYYYMMDD-XXXX`
- `YYYYMMDD` - Date stamp (e.g., 20250129)
- `XXXX` - Random alphanumeric (4 chars, uppercase)

**Implementation:**
```go
import (
    "crypto/rand"
    "encoding/base32"
    "time"
)

func generateOrderNumber() (string, error) {
    datePart := time.Now().Format("20060102")

    // Generate 4-character random suffix
    randomBytes := make([]byte, 3) // 3 bytes = 4.8 base32 chars, take first 4
    if _, err := rand.Read(randomBytes); err != nil {
        return "", fmt.Errorf("failed to generate random suffix: %w", err)
    }

    randomPart := base32.StdEncoding.EncodeToString(randomBytes)[:4]

    return fmt.Sprintf("ORD-%s-%s", datePart, randomPart), nil
}
```

**Notes:**
- Uses crypto/rand for security (no collisions)
- Base32 encoding avoids ambiguous characters (0/O, 1/I/l)
- Format is easy for customers to read/type

#### Step 14: Calculate Totals

**Purpose:** Compute order totals from cart items and payment data.

**Implementation:**
```go
var subtotalCents int32
for _, item := range cartItems {
    subtotalCents += item.Quantity * item.UnitPriceCents
}

// Tax and shipping come from Stripe (either calculated or manual)
taxCents := paymentIntent.TaxCents           // From Stripe Tax or manual
shippingCents := paymentIntent.ShippingCents // From metadata

totalCents := subtotalCents + taxCents + shippingCents

// Verify total matches payment intent amount (sanity check)
if totalCents != paymentIntent.AmountCents {
    return nil, fmt.Errorf("order total mismatch: calculated %d, payment intent %d",
        totalCents, paymentIntent.AmountCents)
}
```

**Why Verify?** Catches bugs in frontend calculation or race conditions.

#### Step 16: Inventory Decrement

**Purpose:** Atomically reduce inventory with optimistic locking.

**Implementation:**
```go
for _, item := range cartItems {
    err := txQueries.DecrementSKUStock(ctx, repository.DecrementSKUStockParams{
        TenantID:          s.tenantID,
        ID:                item.ProductSkuID,
        InventoryQuantity: item.Quantity,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to decrement inventory for SKU %s: %w",
            item.Sku, err)
    }

    // Check if update succeeded (rows affected)
    // If the WHERE clause fails (insufficient stock), no rows are updated
    // This is enforced by the SQL query: WHERE inventory_quantity >= $3
}
```

**Optimistic Locking:** The SQL query includes `WHERE inventory_quantity >= $3`, preventing overselling.

**Error Recovery:** If this step fails, the entire transaction rolls back - no order is created.

### Helper Functions

You'll need to implement:

**buildOrderDetail:**
```go
func (s *orderService) buildOrderDetail(ctx context.Context, order repository.Order) (*OrderDetail, error) {
    // Load items, addresses, payment
    // Construct OrderDetail struct
}
```

**convertPgUUIDToString:**
```go
func convertPgUUIDToString(uuid pgtype.UUID) string {
    // Handle conversion for billing provider calls
}
```

## Testing Strategy

### Unit Tests

**Mock Dependencies:**
- Mock repository (sqlc queries)
- Mock billing provider
- Mock shipping provider

**Test Cases:**
1. **Happy Path:** Successful order creation
2. **Idempotency:** Calling twice with same payment_intent_id returns same order
3. **Payment Not Succeeded:** Returns error for non-succeeded payment
4. **Missing Cart ID:** Returns error when metadata lacks cart_id
5. **Tenant Mismatch:** Returns error when cart belongs to different tenant
6. **Cart Already Converted:** Returns error when cart status is "converted"
7. **Empty Cart:** Returns error when cart has no items
8. **Inventory Insufficient:** Transaction rolls back when SKU out of stock
9. **Transaction Rollback:** Verify rollback on any step failure

### Integration Tests

**Database Required:**
- Test actual transaction behavior
- Test inventory decrement optimistic locking
- Test idempotency query performance

**Test Cases:**
1. **Concurrent Orders:** Two webhooks arrive simultaneously
2. **Inventory Race:** Multiple orders for same SKU
3. **Large Cart:** Order with many items (10+)

## Common Pitfalls

### 1. Forgetting Tenant ID
**Problem:** Queries without tenant_id can leak data across tenants.
**Solution:** Every query must include `tenant_id`. Code review checklist item.

### 2. Not Using Transaction Queries
**Problem:** Using `s.repo` instead of `txQueries` after starting transaction.
**Solution:** Always use `txQueries` within transaction scope.

### 3. Ignoring Idempotency
**Problem:** Webhooks retry, creating duplicate orders.
**Solution:** Always check GetOrderByPaymentIntentID first.

### 4. Missing Inventory Check
**Problem:** Creating order when SKU is out of stock.
**Solution:** DecrementSKUStock query uses optimistic locking. Check for errors.

### 5. Not Validating Payment Status
**Problem:** Creating orders for pending or failed payments.
**Solution:** Only create orders for status == "succeeded".

## Performance Considerations

### Query Optimization

**Idempotency Check:**
- Uses index on `(tenant_id, provider_payment_id)`
- Should be < 5ms

**Cart Items Load:**
- Uses JOIN to load product details in single query
- Typical cart: 1-5 items, < 10ms

**Transaction Duration:**
- Target: < 100ms for typical order
- Monitor for lock contention on inventory updates

### Caching Opportunities

**Don't Cache:**
- Order creation (each is unique)
- Inventory levels (changes frequently)

**Future Caching:**
- Product details (if performance issues arise)
- Price list lookups (rarely change)

## Monitoring and Observability

### Metrics to Track

1. **Order Creation Duration** - Histogram of transaction time
2. **Idempotency Hit Rate** - % of duplicate webhook calls
3. **Inventory Failures** - Count of insufficient stock errors
4. **Transaction Rollbacks** - Count and reasons

### Logging

**Structured Logs:**
```go
log.Info("creating order from payment intent",
    "payment_intent_id", paymentIntentID,
    "cart_id", cartID,
    "tenant_id", s.tenantID,
)
```

**Error Context:**
```go
log.Error("order creation failed",
    "error", err,
    "step", "decrement_inventory",
    "sku", item.Sku,
    "payment_intent_id", paymentIntentID,
)
```

## Future Enhancements

### Phase 1 (MVP)
- Basic order creation from payment intent
- Idempotency handling
- Inventory management

### Phase 2 (Post-MVP)
- Partial refunds
- Order amendments
- Wholesale orders with net terms
- Subscription order creation

### Phase 3 (Future)
- Order splitting (multiple shipments)
- Backorder handling
- Advanced inventory reservations

## Questions for Clarification

Before implementing, verify these assumptions:

1. **Guest Checkout:** Can `user_id` be NULL in orders table?
2. **Currency:** Always USD or support multiple currencies?
3. **Tax Source:** Stripe Tax, manual calculation, or both?
4. **Shipping Calculation:** Real-time rates or flat rate for MVP?
5. **Order Number Collisions:** Should we check uniqueness or rely on randomness?
6. **Payment Method Storage:** Always save payment method or only for subscriptions?

## Additional Resources

- **Stripe Webhooks:** https://stripe.com/docs/webhooks
- **Idempotency:** https://stripe.com/docs/api/idempotent_requests
- **PostgreSQL Transactions:** https://www.postgresql.org/docs/current/tutorial-transactions.html
- **sqlc Transaction Patterns:** https://docs.sqlc.dev/en/latest/howto/transactions.html
