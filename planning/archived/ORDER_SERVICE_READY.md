# Order Service - Implementation Ready

## Status: Skeleton Complete ✓

The OrderService skeleton is complete and ready for implementation. All database infrastructure, SQL queries, and service interfaces are in place.

## What Was Created

### 1. Database Layer
- **Migration**: `migrations/00016_order_idempotency.sql`
  - Unique constraint on `payments.provider_payment_id` to prevent duplicate orders
  - Indexes for fast idempotency checks and cart-to-order conversion tracking
  - Status: **Applied successfully**

### 2. SQL Queries
- **File**: `sqlc/queries/orders.sql`
- **Queries Created** (10 total):
  1. `GetOrderByPaymentIntentID` - Idempotency check
  2. `CreateOrder` - Create order record
  3. `CreateOrderItem` - Create order line items
  4. `CreateAddress` - Create shipping/billing addresses
  5. `CreateBillingCustomer` - Link to Stripe customer
  6. `CreatePayment` - Record payment transaction
  7. `DecrementSKUStock` - Reduce inventory with optimistic locking
  8. `UpdateCartStatus` - Mark cart as converted
  9. `GetOrder` - Retrieve order by ID
  10. `GetOrderByNumber` - Retrieve order by order number
  11. `GetOrderItems` - Load all line items for an order
  12. `GetAddressByID` - Retrieve address
  13. `GetPaymentByID` - Retrieve payment record
  14. `UpdateOrderPaymentID` - Link payment to order after creation

- **Repository Code**: Generated via `sqlc generate`
- **Location**: `internal/repository/orders.sql.go`

### 3. Service Layer
- **File**: `internal/service/order.go` (321 lines)
- **Interface**: `OrderService` with 3 methods
- **Domain Type**: `OrderDetail` (aggregates order data)
- **Implementation**: Complete skeleton with 20-step TODO guide

### 4. Error Handling
- **File**: `internal/service/errors.go`
- **New Errors Added**:
  - `ErrOrderNotFound`
  - `ErrPaymentNotSucceeded`
  - `ErrTenantMismatch`
  - `ErrCartAlreadyConverted`
  - `ErrInsufficientStock`
  - `ErrMissingCartID`
  - `ErrPaymentAlreadyProcessed`

### 5. Documentation
- **`ORDER_IMPLEMENTATION_NOTES.md`** (14KB)
  - Detailed implementation guide for all 20 steps
  - Code examples for each step
  - Common pitfalls and how to avoid them
  - Testing strategy
  - Performance and monitoring considerations

- **`ORDER_SKELETON_SUMMARY.md`** (9.4KB)
  - High-level overview of the implementation
  - Quick reference guide
  - Links to all related files

## Implementation Flow

The `CreateOrderFromPaymentIntent` method follows this 20-step flow:

1. **Idempotency check** - Return existing order if payment already processed
2. **Retrieve payment** from Stripe
3. **Validate payment status** (must be "succeeded")
4. **Extract cart_id** from payment metadata
5. **Retrieve and validate cart** (tenant isolation, not already converted)
6. **Load cart items** with product details
7. **Parse addresses** from payment metadata
8. **Begin database transaction** (all-or-nothing atomicity)
9. **Create address records** (shipping + billing)
10. **Create billing customer** (Stripe customer linkage)
11. **Create payment method** (for saved cards)
12. **Create payment record** (transaction log)
13. **Generate order number** (format: ORD-20250129-A3K9)
14. **Create order record** (with calculated totals)
15. **Create order items** (snapshot cart state)
16. **Decrement inventory** (with optimistic locking to prevent overselling)
17. **Mark cart as converted** (prevent duplicate orders)
18. **Link payment to order** (circular reference resolved)
19. **Commit transaction**
20. **Return order detail** (with all related data)

## Key Design Decisions

### Idempotency
- Uses `payment_intent_id` as the idempotency key
- Database unique constraint prevents duplicate payments
- Application-level check returns existing order on retry
- Safe for webhook retries from Stripe

### Multi-Tenant Isolation
- Every query includes `tenant_id` parameter
- Tenant validation at multiple checkpoints:
  - Payment intent metadata
  - Cart ownership
  - All repository queries

### Transaction Management
- Single database transaction for atomicity
- Automatic rollback on any failure
- All 20 steps succeed or fail together

### Inventory Management
- Optimistic locking via SQL WHERE clause
- `UPDATE ... WHERE inventory_quantity >= $3`
- Returns 0 rows if insufficient stock
- Prevents race conditions in concurrent orders

### Error Handling
- Specific error types for each failure mode
- Wrapped errors preserve context
- Clear error messages for debugging

## Next Steps

### To Implement the OrderService:

1. **Follow the TODO comments** in `internal/service/order.go`
   - Each step is documented with detailed comments
   - Reference the implementation notes for code examples

2. **Implement helper functions**:
   - `buildOrderDetail()` - Aggregates order data
   - `generateOrderNumber()` - Creates unique order numbers
   - `parseAddressFromMetadata()` - Extracts addresses from JSON
   - `calculateOrderTotals()` - Sums cart items

3. **Write tests**:
   - Unit tests for each error case
   - Integration tests for transaction behavior
   - Test idempotency guarantees
   - Test inventory race conditions

4. **Wire into webhook handler**:
   - Update `internal/handler/webhook/stripe.go`
   - Call `orderService.CreateOrderFromPaymentIntent()`
   - Handle errors appropriately
   - Log order creation success

5. **Test with Stripe CLI**:
   ```bash
   # Terminal 1: Start server
   go run cmd/server/main.go

   # Terminal 2: Forward webhooks
   stripe listen --forward-to localhost:3000/webhooks/stripe

   # Terminal 3: Trigger payment
   stripe trigger payment_intent.succeeded
   ```

## Verification Checklist

- [x] Database migration created and applied
- [x] SQL queries written and validated
- [x] Repository code generated successfully
- [x] Service skeleton compiles without errors
- [x] Error types defined
- [x] Documentation complete
- [ ] Implementation (pending)
- [ ] Unit tests (pending)
- [ ] Integration tests (pending)
- [ ] Webhook handler integration (pending)
- [ ] End-to-end testing (pending)

## Files Reference

```
freyja/
├── migrations/
│   └── 00016_order_idempotency.sql          # Database schema changes
├── sqlc/queries/
│   └── orders.sql                           # SQL queries for order operations
├── internal/
│   ├── repository/
│   │   └── orders.sql.go                    # Generated repository code
│   ├── service/
│   │   ├── order.go                         # OrderService skeleton (implement here)
│   │   ├── errors.go                        # Error definitions
│   │   ├── ORDER_IMPLEMENTATION_NOTES.md    # Detailed implementation guide
│   │   └── ORDER_SKELETON_SUMMARY.md        # Quick reference
│   └── handler/webhook/
│       └── stripe.go                        # Webhook handler (needs update)
└── ORDER_SERVICE_READY.md                   # This file
```

## Implementation Time Estimate

- **Core implementation**: 4-6 hours
- **Helper functions**: 1-2 hours
- **Unit tests**: 2-3 hours
- **Integration tests**: 2-3 hours
- **Webhook integration**: 1 hour
- **End-to-end testing**: 2-3 hours

**Total**: 12-18 hours for a complete, production-ready implementation.

## Questions or Issues?

Refer to:
- `ORDER_IMPLEMENTATION_NOTES.md` for detailed code examples
- `ORDER_SKELETON_SUMMARY.md` for architecture overview
- Existing services (`product.go`, `cart.go`) for patterns
- Generated repository code for available query methods
