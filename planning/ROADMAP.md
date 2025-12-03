# Freyja Feature Roadmap

## Overview

This roadmap defines the path to MVP launch and the six months following. The MVP focuses on complete, reliable functionality for core use cases rather than breadth of features.

**Last updated:** December 2, 2024

---

## Current Status

✅ **Phases 1-3 Complete** — Full B2C e-commerce with working checkout and payments
⏳ **Phase 4 Partial** — Flat-rate shipping working, carrier integration not started
✅ **Phase 5 Complete** — Subscriptions fully implemented with Stripe Billing
⏳ **Phase 6 Partial** — Wholesale service layer complete, admin UI not started
✅ **Email Notifications Complete** — Postmark/SMTP, background worker, 6 email types

**Codebase Metrics:**
- 100+ Go source files (~17,000 lines)
- 17 database migrations (44 tables)
- 63 HTML templates (including 7 email templates)
- 30+ HTTP handlers
- 3,100+ lines of test code

---

## MVP (12 Weeks)

Target: A roaster can sell coffee online to retail and wholesale customers with flexible pricing, subscriptions, and invoicing.

### Phase 1: Foundation ✅ COMPLETE

**Product Catalog** ✅
- ✅ Coffee product management (name, description, images)
- ✅ Coffee-specific attributes: origin, region, producer, process, roast level, tasting notes, elevation
- ✅ SKU variants by weight and grind option
- ✅ Inventory tracking per SKU (with optimistic locking)
- ✅ Product visibility controls (public, wholesale-only, restricted)
- ✅ Active/inactive status

**Customer Accounts** ✅
- ✅ Email/password authentication (bcrypt hashing)
- ✅ Password reset flow (forgot password → email token → reset)
- ⏳ Magic link authentication (passwordless option) — not implemented
- ✅ Account types: retail and wholesale (schema ready)
- ⏳ Profile management with saved addresses — partial
- ⏳ Wholesale account application flow — schema ready, UI not implemented

**Price List System** ✅
- ✅ Default retail price list
- ✅ Named wholesale price lists (e.g., "Café Tier 1", "Restaurant Tier 2")
- ✅ Per-product pricing per list
- ✅ Price list assignment to customer accounts
- ✅ Restricted product access via price list entries

### Phase 2: Storefront & Cart ✅ COMPLETE

**Product Display** ✅
- ✅ Product listing with responsive grid layout
- ✅ Product detail pages with coffee metadata
- ✅ Dynamic pricing based on logged-in customer's price list
- ✅ Grind and size selection via SKU variants
- ⏳ Filters (roast level, origin, process) — not implemented

**Shopping Cart** ✅
- ✅ Add/remove/update cart items (htmx dynamic updates)
- ✅ Cart persistence (session for guests, database for authenticated)
- ✅ Price recalculation on cart changes
- ⏳ Minimum order enforcement for wholesale accounts — not implemented

**Checkout Flow** ✅
- ✅ Multi-step checkout (Alpine.js)
- ✅ Contact information collection
- ✅ Address entry with validation
- ✅ Shipping method selection (flat-rate options)
- ✅ Billing address (same-as-shipping option)
- ✅ Order summary with line items and totals
- ✅ Stripe Elements payment form
- ⏳ Tax calculation — no-tax calculator implemented, Stripe Tax interface ready

### Phase 3: Billing & Payments ✅ COMPLETE

**Billing Interface** ✅
- ✅ Abstract billing provider interface
- ✅ Methods: customer management, one-time charges, payment intents
- ✅ Webhook handling abstraction
- ✅ Mock provider for testing

**Stripe Implementation** ✅
- ✅ Customer creation and synchronization
- ✅ Payment method storage
- ✅ One-time payment processing via Payment Intents
- ✅ Webhook handlers for payment events (payment_intent.succeeded, failed, canceled)
- ✅ Idempotent event processing (webhook_events table + idempotent_operations)
- ✅ Webhook signature verification

**Order Management** ✅
- ✅ Order creation on successful payment (20-step atomic workflow)
- ✅ Order status workflow: pending → paid → processing → shipped → delivered
- ✅ Order history for customers
- ✅ Admin order list and detail views
- ✅ Order number generation
- ✅ Inventory decrement on order creation

### Phase 4: Shipping ⏳ PARTIAL

**Shipping Interface** ✅
- ✅ Abstract shipping provider interface
- ✅ Methods: get rates, validate address
- ⏳ Create label, track shipment — interface defined, not implemented

**Flat-Rate Provider** ✅
- ✅ Standard Shipping: $7.95 (5-7 days)
- ✅ Express Shipping: $14.95 (2-3 days)
- ✅ Rate selection in checkout

**Fulfillment Workflow** ⏳
- ✅ Shipment creation with carrier/tracking number
- ✅ Mark orders as shipped (admin UI)
- ✅ Shipment status tracking
- ⏳ Pick list generation — not implemented
- ⏳ Shipping confirmation emails — not implemented

### Phase 5: Subscriptions ✅ COMPLETE

**Database Schema** ✅
- ✅ subscription_plans table
- ✅ subscriptions table
- ✅ subscription_items table
- ✅ subscription_schedule table

**Subscription Management** ✅
- ✅ Subscription plans linked to products (via product SKU)
- ✅ Frequency options: weekly, every 2 weeks, monthly, every 6 weeks, every 2 months
- ✅ Quantity and grind selection per subscription
- ✅ Subscription status: active, paused, cancelled, past_due, expired

**Stripe Subscription Integration** ✅
- ✅ Create and manage subscriptions via Stripe Billing
- ✅ Handle subscription lifecycle webhooks (invoice.payment_succeeded, .failed, customer.subscription.updated, .deleted)
- ✅ Order creation from subscription invoice payments
- ✅ Failed payment handling (status → past_due)
- ⏳ Dunning email notifications — not implemented (uses Stripe's built-in)

**Subscription Checkout Flow** ✅
- ✅ Product detail page with one-time/subscribe toggle
- ✅ Dedicated subscription checkout page (/subscribe/checkout)
- ✅ Select from saved addresses and payment methods
- ✅ Delivery frequency selector
- ✅ Subscription creation via POST /subscribe

**Customer Subscription Portal** ✅
- ✅ View active subscriptions (/account/subscriptions)
- ✅ View subscription details (/account/subscriptions/{id})
- ✅ Stripe Customer Portal integration for pause/resume/cancel
- ⏳ Skip next delivery — deferred to post-MVP
- ⏳ Change frequency/quantity in-app — uses Stripe Portal for now

### Phase 6: Wholesale & Invoicing ⏳ PARTIAL

**Database Schema** ✅
- ✅ invoices table
- ✅ invoice_items table
- ✅ invoice_payments table
- ✅ invoice_status_history table
- ✅ payment_terms table
- ✅ invoice_orders linking table (consolidated invoicing)

**Service Layer** ✅
- ✅ PaymentTermsService — CRUD for payment terms (Net 15/30/60), due date calculation
- ✅ FulfillmentService — Partial shipment tracking, quantity_dispatched management
- ✅ InvoiceService — Full invoice lifecycle management
- ✅ Billing Provider Extensions — Stripe Invoicing methods (create, finalize, send, void, pay)

**Invoice Billing** ✅
- ✅ Net terms configuration per account (Net 15, Net 30, etc.)
- ✅ Invoice generation from single or multiple orders
- ✅ Invoice status tracking: draft, sent, viewed, paid, overdue, void
- ✅ Stripe Invoice integration for payment collection
- ✅ Payment recording with balance tracking

**Consolidated Billing** ✅
- ✅ Billing period configuration on invoices
- ✅ Accumulate multiple orders into single invoice
- ✅ Generate consolidated invoice on cycle close

**Background Jobs** ✅
- ✅ invoice:generate_consolidated — Create consolidated invoices
- ✅ invoice:mark_overdue — Nightly job to detect and mark overdue invoices
- ✅ invoice:send_reminder — Payment reminder scheduling
- ✅ invoice:sync_stripe — Stripe webhook synchronization

**Wholesale Account Management** 🔲
- 🔲 Application review queue for admin
- 🔲 Approval workflow with price list and terms assignment
- 🔲 Wholesale-specific dashboard view

**Admin UI** 🔲
- 🔲 Invoice list and detail views
- 🔲 Manual invoice creation
- 🔲 Payment recording interface
- 🔲 Wholesale customer management

### MVP Admin Dashboard ⏳ PARTIAL

**Implemented** ✅
- ✅ Dashboard with order/revenue statistics
- ✅ Product CRUD with image management
- ✅ SKU variant management
- ✅ Order list with status filtering
- ✅ Order detail with fulfillment actions (status updates, shipment creation)
- ✅ Customer list view

**Not Yet Implemented** 🔲
- 🔲 Customer editing and price list assignment
- 🔲 Wholesale approval workflow
- 🔲 Subscription overview
- 🔲 Invoice management

### MVP Email Notifications ✅ COMPLETE

**Infrastructure** ✅
- ✅ Email provider interface with Postmark implementation
- ✅ SMTP fallback for development (Mailhog)
- ✅ Background job worker for async sending
- ✅ Job queue with retry logic and concurrency control
- ✅ Email templates with base layout

**Transactional Emails** ✅
- ✅ Password reset
- ✅ Order confirmation
- ✅ Shipping confirmation with tracking
- ✅ Subscription welcome
- ✅ Subscription payment failed
- ✅ Subscription cancelled

**Not Yet Implemented** 🔲
- 🔲 Invoice sent (requires wholesale/invoicing feature)
- 🔲 Invoice payment reminder
- 🔲 Invoice overdue

---

## MVP + 6 Months

Target: Operational efficiency, customer retention tools, and preparation for scale.

### Month 1-2: Shipping Provider Integration

**Shipping Rate Calculation**
- Integrate EasyPost, Shippo, or ShipStation (evaluate based on pricing and Go SDK quality)
- Real-time rate fetching at checkout
- Support for multiple carriers (USPS, UPS, FedEx)
- Flat rate and free shipping threshold options

**Label Purchasing**
- Purchase labels from admin interface
- Batch label printing for multiple orders
- Automatic tracking number population
- Shipping cost tracking for margin analysis

### Month 2-3: Inventory & Operations

**Inventory Management**
- Low stock alerts with configurable thresholds
- Inventory adjustment logging (who, when, why)
- Expected restock dates
- Backorder acceptance (optional per product)

**Roast Date Management**
- Roast date entry at fulfillment
- "Roasted on" display on packing slips
- Rest period configuration (don't ship until N days after roast)
- Subscription alignment with roast schedule

**Reporting**
- Sales by period, product, customer type
- Subscription metrics (new, churned, MRR)
- Wholesale account performance
- Exportable reports (CSV)

### Month 3-4: Customer Retention

**Discount System**
- Percentage and fixed amount discounts
- Discount codes with usage limits and expiration
- Automatic discounts (e.g., 10% off first subscription)
- Wholesale volume discounts (automatic tier pricing)

**Customer Communication**
- Transactional email customization
- Reorder reminders for retail customers
- Win-back emails for lapsed customers
- Subscription upsell prompts

**Reviews & Ratings**
- Product review collection (post-delivery email)
- Review display on product pages
- Admin moderation queue

### Month 4-5: Financial Integration

**Accounting Integration**
- QuickBooks Online integration
- Invoice synchronization
- Payment recording
- Basic chart of accounts mapping

**Tax Automation**
- Tax calculation service integration (TaxJar or similar)
- Automatic rate determination by address
- Tax reporting exports

### Month 5-6: Platform Hardening

**Multi-Tenancy Preparation**
- Evaluate architecture changes for future SaaS model
- Tenant isolation strategy
- Shared vs. dedicated database approach

**Performance & Reliability**
- Query optimization and indexing review
- Response time monitoring
- Error tracking and alerting
- Automated backup verification
- Disaster recovery documentation

**Security Audit**
- Dependency vulnerability scanning
- Authentication flow review
- Input validation audit
- Rate limiting implementation
- Penetration testing (external or self-conducted)

---

## Future Considerations (Beyond MVP + 6 Months)

These are noted for architectural awareness but not scheduled:

- Multi-location inventory
- POS integration
- Mobile app for wholesale ordering
- Coffee grading and cupping notes
- Green coffee inventory (pre-roast tracking)
- Customer segmentation and targeted marketing
- Affiliate/referral program
- International shipping and multi-currency
- API access for customer integrations

---

## Implementation Summary

### What's Working Now

| Component | Status | Details |
|-----------|--------|---------|
| Product Catalog | ✅ Complete | Full CRUD, SKU variants, coffee-specific attributes |
| Customer Auth | ✅ Complete | Signup, login, password reset, sessions with bcrypt |
| Price Lists | ✅ Complete | Multi-tier pricing, customer assignment |
| Shopping Cart | ✅ Complete | Add/update/remove, htmx updates |
| Checkout | ✅ Complete | 5-step flow, address validation, Stripe Elements |
| Payments | ✅ Complete | Payment intents, webhooks, idempotent processing |
| Orders | ✅ Complete | Creation, status tracking, admin management |
| Shipping | ⏳ Partial | Flat-rate working, no carrier integration |
| Admin Dashboard | ⏳ Partial | Products, orders, customers, subscriptions; missing invoices |
| Subscriptions | ✅ Complete | Full Stripe Billing integration, checkout flow, webhooks |
| Invoicing | ⏳ Partial | Service layer complete, admin UI pending |
| Email | ✅ Complete | Postmark + SMTP, 6 templates, background worker |
| Background Jobs | ✅ Complete | Worker with concurrency, retry logic, graceful shutdown |

### Architecture Highlights

- **44+ database tables** across 18 migrations
- **30+ HTTP handlers** for storefront, admin, and webhooks
- **11 service layers** (product, cart, user, order, checkout, subscription, account, password reset, payment terms, fulfillment, invoice)
- **Interface-based abstractions** for billing, shipping, email, storage, tax
- **Multi-tenant isolation** on all queries (tenant_id scoping)
- **Idempotent webhook processing** for payment reliability
- **Comprehensive test coverage** for checkout (1,735 lines) and orders (1,374 lines)
- **Stripe Invoicing integration** for wholesale billing

### Remaining MVP Work

1. ~~**Subscriptions**~~ ✅ Complete — Full Stripe Billing integration with checkout flow
2. ~~**Email Notifications**~~ ✅ Complete — Postmark + SMTP, background worker, 6 email templates
3. ~~**Wholesale Service Layer**~~ ✅ Complete — InvoiceService, PaymentTermsService, FulfillmentService, Stripe Invoicing
4. **Wholesale Admin UI** — Invoice list/detail, payment recording, wholesale approval workflow
5. **Carrier Integration** (Optional for MVP) — EasyPost/Shippo, real-time rates, label purchasing
6. **Polish** — Product filters, wholesale minimums, pick lists, customer profile editing

---

## Milestone Summary

| Milestone | Status | Key Deliverable |
|-----------|--------|-----------------|
| Phase 1-3 | ✅ Complete | B2C checkout with Stripe payments |
| Phase 4 | ⏳ Partial | Flat-rate shipping, fulfillment workflow |
| Phase 5 | ✅ Complete | Subscriptions with Stripe Billing |
| Phase 6 | ⏳ Partial | Wholesale service layer (admin UI pending) |
| MVP + 2 mo | — | Shipping provider integration, automated labels |
| MVP + 4 mo | — | Inventory management, discounts, reviews |
| MVP + 6 mo | — | Accounting integration, platform hardening |