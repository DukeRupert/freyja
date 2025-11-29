# Order Service Skeleton - Summary

## Files Created

### 1. `/home/dukerupert/Repos/freyja/internal/service/order.go`

**OrderService Interface:**
- `CreateOrderFromPaymentIntent(ctx, paymentIntentID) -> OrderDetail`
- `GetOrder(ctx, orderID) -> OrderDetail`
- `GetOrderByNumber(ctx, orderNumber) -> OrderDetail`

**OrderDetail Type:**
Aggregates complete order information:
- Order record
- Order items
- Shipping address
- Billing address
- Payment record

**orderService Implementation:**
Private struct with:
- `repository.Querier` - Database access
- `tenant_id` - Multi-tenant isolation
- `billing.Provider` - Stripe integration
- `shipping.Provider` - Future shipping integration

**CreateOrderFromPaymentIntent Method:**
20-step implementation plan documented in detailed TODO comments:

1. Idempotency check (prevent duplicate orders)
2. Retrieve payment intent from Stripe
3. Validate payment status (must be "succeeded")
4. Extract cart_id from payment metadata
5. Retrieve cart and validate tenant isolation
6. Load cart items with product details
7. Parse shipping/billing addresses from metadata
8. Begin database transaction
9. Create shipping and billing address records
10. Create billing customer record (Stripe customer linkage)
11. Create payment method record (for saved cards)
12. Create payment transaction record
13. Generate unique order number (ORD-YYYYMMDD-XXXX)
14. Create order record with calculated totals
15. Create order items (snapshot cart state)
16. Decrement inventory with optimistic locking
17. Mark cart as converted
18. Link payment to order
19. Commit transaction
20. Return complete OrderDetail

**Error Handling:**
Uses existing error types from `service/errors.go`:
- `ErrPaymentAlreadyProcessed`
- `ErrPaymentNotSucceeded`
- `ErrMissingCartID`
- `ErrCartNotFound`
- `ErrTenantMismatch`
- `ErrCartAlreadyConverted`
- `ErrInsufficientStock`

### 2. `/home/dukerupert/Repos/freyja/internal/service/ORDER_IMPLEMENTATION_NOTES.md`

Comprehensive implementation guide covering:

**Architecture Context:**
- Dependency overview
- Data flow diagram
- Integration points

**Step-by-Step Implementation:**
- Detailed code examples for each step
- Error handling patterns
- Transaction management
- Helper function signatures

**Common Pitfalls:**
- Tenant ID isolation
- Transaction query usage
- Idempotency handling
- Inventory management
- Payment status validation

**Testing Strategy:**
- Unit test cases (9 scenarios)
- Integration test cases (3 scenarios)
- Mock setup guidance

**Performance Considerations:**
- Query optimization notes
- Expected timing benchmarks
- Caching guidance

**Monitoring:**
- Metrics to track
- Structured logging examples

**Future Enhancements:**
- Phase 1: MVP features
- Phase 2: Post-MVP features
- Phase 3: Advanced features

## SQL Queries Updated

### `/home/dukerupert/Repos/freyja/sqlc/queries/orders.sql`

**Existing Queries:**
- `GetOrderByPaymentIntentID` (fixed join condition)
- `CreateOrder`
- `CreateOrderItem`
- `CreateAddress`
- `CreateBillingCustomer`
- `CreatePayment`
- `DecrementSKUStock`
- `UpdateCartStatus`
- `GetOrder`
- `GetOrderByNumber`

**New Queries Added:**
- `GetOrderItems` - Loads all line items for an order
- `GetAddressByID` - Retrieves a single address
- `GetPaymentByID` - Retrieves a single payment
- `UpdateOrderPaymentID` - Links payment to order after creation

**Query Fix:**
Updated `GetOrderByPaymentIntentID` join condition:
- **Before:** `JOIN payments p ON p.order_id = o.id` (incorrect, no such column)
- **After:** `JOIN payments p ON p.id = o.payment_id` (correct relationship)

## Repository Code Generated

All new queries have been processed by `sqlc generate`:

### Generated Functions:
- `GetOrderItems(ctx, orderID) -> []OrderItem`
- `GetAddressByID(ctx, addressID) -> Address`
- `GetPaymentByID(ctx, paymentID) -> Payment`
- `UpdateOrderPaymentID(ctx, UpdateOrderPaymentIDParams) -> error`

### Generated Types:
- `UpdateOrderPaymentIDParams` struct with:
  - `TenantID pgtype.UUID`
  - `ID pgtype.UUID` (order ID)
  - `PaymentID pgtype.UUID`

## Verification

**Compilation Check:**
```bash
go build ./internal/service/...
```
Result: Success (no compilation errors)

**Code Generation:**
```bash
sqlc generate
```
Result: Success (4 new queries generated)

## Implementation Readiness

### Ready for Implementation:
- Complete function signatures
- Detailed step-by-step TODOs
- All required database queries available
- Error types defined
- Transaction pattern documented

### Next Steps for Implementation:
1. Replace TODOs with actual code following the commented steps
2. Implement helper functions:
   - `buildOrderDetail(ctx, order) -> OrderDetail`
   - `generateOrderNumber() -> string`
   - `parseAddressFromMetadata(json) -> AddressData`
3. Add unit tests for each error case
4. Add integration tests for transaction behavior
5. Implement proper logging at each step
6. Add metrics collection

### Dependencies Required:
- `crypto/rand` - For order number generation
- `encoding/base32` - For order number encoding
- `encoding/json` - For metadata parsing
- `database/sql` - For error checking
- `bytes` - For tenant ID comparison

## Design Patterns Used

### Service Layer Pattern:
- Business logic separated from HTTP handlers
- Repository abstraction for database access
- Provider abstraction for external services

### Repository Pattern:
- Type-safe database queries via sqlc
- Multi-tenant isolation at query level
- Transaction support

### Transaction Script Pattern:
- Single transaction for entire order creation flow
- All-or-nothing semantics
- Rollback on any failure

### Idempotency Pattern:
- Check for existing order before creation
- Use payment_intent_id as idempotency key
- Safe for webhook retries

### Optimistic Locking:
- Inventory decrements include quantity check
- Prevents overselling
- No explicit row locks needed

## Testing Approach

### Unit Test Structure:
```go
type mockRepository struct {
    // Implement repository.Querier interface
}

type mockBillingProvider struct {
    // Implement billing.Provider interface
}

func TestCreateOrderFromPaymentIntent_Success(t *testing.T) {
    // Setup mocks
    // Call CreateOrderFromPaymentIntent
    // Assert order created correctly
}

func TestCreateOrderFromPaymentIntent_Idempotent(t *testing.T) {
    // Setup mock to return existing order
    // Call twice with same payment_intent_id
    // Assert same order returned both times
}

// ... more test cases
```

### Integration Test Structure:
```go
func TestCreateOrderFromPaymentIntent_Integration(t *testing.T) {
    // Requires: PostgreSQL test database
    // Setup: Create tenant, products, cart, payment intent
    // Execute: CreateOrderFromPaymentIntent
    // Assert: Order, items, addresses created in database
    // Assert: Inventory decremented
    // Assert: Cart marked as converted
}
```

## Security Considerations

### Multi-Tenant Isolation:
- All queries include `tenant_id` parameter
- Cart validation checks tenant ownership
- Payment intent metadata validated for tenant match

### Payment Validation:
- Only "succeeded" payments create orders
- Payment amount verified against calculated total
- Stripe signature verification in webhook handler (not in service)

### Inventory Protection:
- Optimistic locking prevents overselling
- Transaction rollback if insufficient stock
- No race conditions between check and decrement

### Data Integrity:
- Transaction ensures all-or-nothing
- Foreign key constraints prevent orphaned records
- Status transitions validated (cart can't be converted twice)

## Performance Targets

### Order Creation:
- **Target:** < 100ms (p95)
- **Breakdown:**
  - Idempotency check: < 5ms
  - Payment intent retrieval: < 20ms (external API)
  - Database transaction: < 50ms
  - Commit: < 10ms

### Query Performance:
- Idempotency check: Index on `(tenant_id, provider_payment_id)`
- Cart items load: JOIN query, typically 1-5 items
- Inventory decrement: Single UPDATE per SKU

### Scalability:
- No locks held across external API calls
- Transaction scope minimized
- Stateless service (horizontally scalable)

## Open Questions

The implementation notes document includes a section with questions that need clarification:

1. Guest checkout support (NULL user_id in orders?)
2. Multi-currency support or USD-only?
3. Tax calculation source (Stripe Tax vs manual)?
4. Shipping rate calculation (real-time vs flat rate)?
5. Order number uniqueness (check vs rely on randomness)?
6. Payment method storage policy (always vs subscriptions-only)?

These should be answered before implementation begins.

## Related Files

**Database Migrations:**
- `/home/dukerupert/Repos/freyja/migrations/00016_order_idempotency.sql`

**Error Definitions:**
- `/home/dukerupert/Repos/freyja/internal/service/errors.go`

**Repository Code:**
- `/home/dukerupert/Repos/freyja/internal/repository/orders.sql.go` (generated)

**Billing Provider:**
- `/home/dukerupert/Repos/freyja/internal/billing/billing.go`
- `/home/dukerupert/Repos/freyja/internal/billing/stripe.go`

**Shipping Provider:**
- `/home/dukerupert/Repos/freyja/internal/shipping/shipping.go`
- `/home/dukerupert/Repos/freyja/internal/shipping/flatrate.go`

## Conclusion

The OrderService skeleton is complete and ready for implementation. All interfaces are defined, database queries are written and generated, and detailed implementation guidance is documented.

The next phase is to replace the TODO comments with actual implementation code, following the step-by-step guide provided in the implementation notes.
