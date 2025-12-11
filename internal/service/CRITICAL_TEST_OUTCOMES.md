# Critical Test Outcomes for Freyja E-Commerce Platform

This document defines the critical outcomes that MUST be verified by tests. Tests should focus on correctness and catching real bugs, not on passing. If a test reveals a bug, that's valuable.

---

## PHASE 1: CRITICAL PATH (Highest Risk)

### 1. Order Creation from Payment Intent (`order.go:CreateOrderFromPaymentIntent`)

**What this function does:** Converts a successful Stripe payment into a complete order with items, addresses, and inventory decrements.

**Critical Outcomes to Verify:**

#### 1.1 Idempotency
- [ ] Calling with the same `paymentIntentID` twice returns the existing order, not a new one
- [ ] No duplicate orders created on webhook retries (Stripe may send same event 3+ times)
- [ ] Returns `ErrPaymentAlreadyProcessed` sentinel error on duplicate, allowing caller to handle gracefully

#### 1.2 Payment Validation
- [ ] Rejects payment intents with status != "succeeded" with `ErrPaymentNotSucceeded`
- [ ] Rejects payment intents missing `cart_id` metadata with `ErrMissingCartID`
- [ ] Rejects payment intents missing `shipping_address` metadata with `ErrMissingShippingAddress`
- [ ] Rejects payment intents missing `billing_address` metadata with `ErrMissingBillingAddress`

#### 1.3 Tenant Isolation (SECURITY CRITICAL)
- [ ] Cart's `tenant_id` must match context `tenant_id`, returns `ErrTenantMismatch` otherwise
- [ ] Order cannot be created for cart belonging to different tenant
- [ ] All created records (order, items, addresses, payment) have correct `tenant_id`

#### 1.4 Cart State Management
- [ ] Cart status changes from "active" to "converted" after order creation
- [ ] Cannot create order from already-converted cart (returns `ErrCartAlreadyConverted`)
- [ ] Cannot create order from empty cart (returns `ErrEmptyCart`)

#### 1.5 Inventory Management
- [ ] Each order item decrements corresponding SKU's `inventory_quantity`
- [ ] Returns `ErrInsufficientStock` if any SKU has insufficient inventory
- [ ] Inventory decrement is atomic (no partial decrements on failure)

#### 1.6 Order Data Integrity
- [ ] Order totals match payment intent totals (subtotal + tax + shipping = total)
- [ ] Order items correctly capture product name, SKU, quantity, unit price, total price
- [ ] Order items preserve variant description (weight, grind)
- [ ] Shipping and billing addresses correctly stored and linked

#### 1.7 Guest Checkout
- [ ] Creates guest user account when cart has no `user_id`
- [ ] Links existing user if email already exists in system
- [ ] Extracts customer email from payment intent receipt_email or metadata

---

### 2. Checkout Service (`checkout.go`)

**Critical Outcomes to Verify:**

#### 2.1 GetShippingRates
- [ ] Returns `ErrCartEmpty` for carts with no items
- [ ] Returns `ErrNoShippingRates` when shipping provider returns empty rates
- [ ] Rates sorted by cost (cheapest first)
- [ ] Tenant warehouse address used as origin

#### 2.2 CalculateOrderTotal
- [ ] Subtotal correctly sums all cart item line totals
- [ ] Tax calculation uses customer's shipping address for jurisdiction
- [ ] Total = subtotal + shipping + tax - discount
- [ ] Tax breakdown returned for display

#### 2.3 CreatePaymentIntent
- [ ] Payment intent includes `tenant_id` in metadata (critical for webhook routing)
- [ ] Payment intent includes `cart_id` in metadata
- [ ] Payment intent includes serialized shipping/billing addresses
- [ ] Idempotency key prevents duplicate payment intents

---

### 3. Multi-Tenant Isolation (`middleware/tenant.go`, `middleware/auth.go`)

**Critical Outcomes to Verify:**

#### 3.1 Tenant Resolution
- [ ] Subdomain `acme.hiri.coffee` resolves to tenant with slug "acme"
- [ ] Custom domain `shop.acme.com` resolves via ByCustomDomain lookup
- [ ] `www.hiri.coffee` redirects to `hiri.coffee` (301)
- [ ] `app.hiri.coffee` bypasses tenant resolution (SaaS admin routes)
- [ ] Unknown subdomain returns 404

#### 3.2 Tenant Status Handling
- [ ] Active tenant: request continues normally, tenant in context
- [ ] Pending tenant: returns 404 (storefront not ready)
- [ ] Suspended tenant: returns 503 with Retry-After header
- [ ] Cancelled tenant: returns 404

#### 3.3 RequireTenant Middleware
- [ ] Returns 404 when no tenant in context
- [ ] Continues when tenant present

#### 3.4 User Authentication (auth.go)
- [ ] `WithUser` extracts user from session cookie
- [ ] Missing/invalid session continues without user (not an error)
- [ ] `RequireAuth` redirects to `/login?return_to=` when no user
- [ ] `RequireAdmin` redirects non-admins to storefront

---

### 4. Stripe Webhook Handling (`handler/webhook/stripe.go`)

**Critical Outcomes to Verify:**

#### 4.1 Security
- [ ] Rejects requests with missing Stripe-Signature header (401)
- [ ] Rejects requests with invalid signature (401)
- [ ] Only accepts POST requests (400 for other methods)

#### 4.2 Event Routing
- [ ] `payment_intent.succeeded` calls `CreateOrderFromPaymentIntent`
- [ ] `invoice.payment_succeeded` with subscription creates subscription order
- [ ] `customer.subscription.updated` syncs subscription status
- [ ] `customer.subscription.deleted` marks subscription expired

#### 4.3 Tenant Validation
- [ ] Event's `tenant_id` metadata must match handler's configured tenant
- [ ] Mismatched tenant_id logs warning and skips processing
- [ ] Test mode allows tenant mismatch (for Stripe CLI testing)

#### 4.4 Idempotency
- [ ] Duplicate webhook events handled gracefully (no duplicate orders)
- [ ] Returns 200 OK even on duplicate to prevent Stripe retries
- [ ] `ErrPaymentAlreadyProcessed` treated as success, not error

#### 4.5 Error Handling
- [ ] Always returns 200 to Stripe (prevents retry storms)
- [ ] Logs errors for investigation
- [ ] Tracks failures in telemetry metrics

---

## PHASE 2: BUSINESS LOGIC (High Risk)

### 5. Invoice Management (`invoice.go`)

**Critical Outcomes to Verify:**

#### 5.1 CreateInvoice
- [ ] Only wholesale users can have invoices created
- [ ] All orders must belong to the same user
- [ ] All orders must be wholesale type
- [ ] Payment terms correctly applied (Net 15/30/60)
- [ ] Due date calculated from payment terms

#### 5.2 RecordPayment
- [ ] Payment cannot exceed invoice balance
- [ ] Invoice status transitions: draft → sent → paid
- [ ] Partial payments update balance correctly
- [ ] Full payment marks invoice as paid

#### 5.3 GenerateConsolidatedInvoice
- [ ] Finds all uninvoiced orders in billing period
- [ ] Returns nil (not error) when no orders to invoice
- [ ] Creates single invoice for multiple orders

#### 5.4 MarkInvoicesOverdue
- [ ] Updates status from sent/viewed to overdue
- [ ] Only invoices past due date marked
- [ ] Enqueues overdue notification email

---

### 6. Subscription Lifecycle (`subscription_impl.go`)

**Critical Outcomes to Verify:**

#### 6.1 CreateSubscription
- [ ] Validates billing interval (weekly, biweekly, monthly, etc.)
- [ ] Creates Stripe product, price, and subscription
- [ ] Local subscription linked to Stripe via `provider_subscription_id`
- [ ] Subscription items created with correct quantities and prices

#### 6.2 PauseSubscription
- [ ] Only active subscriptions can be paused
- [ ] Stripe subscription paused (void pending invoices)
- [ ] Local status updated to "paused"

#### 6.3 ResumeSubscription
- [ ] Only paused subscriptions can be resumed
- [ ] Stripe subscription resumed
- [ ] Period dates updated from Stripe response

#### 6.4 CancelSubscription
- [ ] Can cancel at period end or immediately
- [ ] Cancellation reason stored
- [ ] Stripe subscription cancelled

#### 6.5 CreateOrderFromSubscriptionInvoice
- [ ] Idempotent: same invoice_id returns error, not new order
- [ ] Order linked to subscription via `subscription_id`
- [ ] Order items created from subscription items
- [ ] Inventory decremented

#### 6.6 SyncSubscriptionFromWebhook
- [ ] Idempotent via webhook_events table
- [ ] Updates local subscription status from Stripe
- [ ] Handles all status transitions (active, past_due, canceled)

---

### 7. Authentication & Session (`middleware/auth.go`)

**Critical Outcomes to Verify:**

#### 7.1 Session Validation
- [ ] Valid session token returns user
- [ ] Expired session treated as no session (not error)
- [ ] Invalid session token treated as no session

#### 7.2 Authorization
- [ ] Admin routes require admin account type
- [ ] Regular users redirected from admin routes
- [ ] Return URL preserved through login redirect

---

## PHASE 3: OPERATIONAL (Medium Risk)

### 8. Background Jobs (`worker/worker.go`, `jobs/`)

**Critical Outcomes to Verify:**

#### 8.1 Job Processing
- [ ] Jobs claimed with SKIP LOCKED (no double-processing)
- [ ] Failed jobs increment `attempt_count`
- [ ] Jobs exceed max attempts move to dead letter

#### 8.2 Invoice Jobs
- [ ] Invoice sent email contains correct data
- [ ] Overdue invoice job marks correct invoices

---

## Test Implementation Guidelines

### For Test Writers

1. **Focus on catching bugs, not making tests pass.** If a test fails, investigate whether the code is wrong.

2. **Use table-driven tests** for scenarios with multiple inputs/outputs.

3. **Mock external services** (Stripe, database) but verify correct calls were made.

4. **Test error paths** - most bugs hide in error handling.

5. **Verify tenant isolation** in every data access test - this is a security critical.

6. **Test idempotency** by calling the same operation twice.

7. **Test race conditions** for inventory decrement (concurrent orders).

### Mock Strategy

- `repository.Querier` - mock for unit tests, real DB for integration
- `billing.Provider` - always mock (never hit real Stripe in tests)
- `shipping.Provider` - mock
- `tax.Calculator` - mock

### Test File Naming

- Unit tests: `*_test.go` in same package
- Integration tests: `*_integration_test.go` with `// +build integration` tag
