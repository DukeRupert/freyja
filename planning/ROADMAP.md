# Freyja Feature Roadmap

## Overview

This roadmap defines the path to MVP launch and the six months following. The MVP focuses on complete, reliable functionality for core use cases rather than breadth of features.

**Last updated:** December 2024

---

## Current Status

✅ **Phases 1-3 Complete** — Full B2C e-commerce with working checkout and payments
⏳ **Phase 4 Partial** — Flat-rate shipping working, advanced shipping not started
🔲 **Phases 5-6** — Database schema complete, service layer not implemented

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

### Phase 5: Subscriptions 🔲 NOT STARTED

**Database Schema** ✅
- ✅ subscription_plans table
- ✅ subscriptions table
- ✅ subscription_items table
- ✅ subscription_schedule table

**Subscription Management** 🔲
- 🔲 Subscription plans linked to products
- 🔲 Frequency options: weekly, every 2 weeks, monthly, every 6 weeks, every 2 months
- 🔲 Quantity and grind selection per subscription
- 🔲 Subscription status: active, paused, canceled

**Stripe Subscription Integration** 🔲
- 🔲 Create and manage subscriptions via Stripe Billing
- 🔲 Handle subscription lifecycle webhooks
- 🔲 Failed payment retry handling
- 🔲 Dunning management (email notifications for payment issues)

**Customer Subscription Portal** 🔲
- 🔲 View active subscriptions
- 🔲 Pause/resume subscription
- 🔲 Skip next delivery
- 🔲 Change frequency, quantity, or grind
- 🔲 Cancel subscription

### Phase 6: Wholesale & Invoicing 🔲 NOT STARTED

**Database Schema** ✅
- ✅ invoices table
- ✅ invoice_items table
- ✅ invoice_payments table
- ✅ invoice_status_history table

**Wholesale Account Management** 🔲
- 🔲 Application review queue for admin
- 🔲 Approval workflow with price list and terms assignment
- 🔲 Wholesale-specific dashboard view

**Invoice Billing** 🔲
- 🔲 Net terms configuration per account (Net 15, Net 30, etc.)
- 🔲 Invoice generation on order placement
- 🔲 Invoice status tracking: draft, sent, paid, overdue
- 🔲 Stripe Invoice integration for payment collection

**Consolidated Billing** 🔲
- 🔲 Billing cycle configuration per account (weekly, biweekly, monthly)
- 🔲 Accumulate orders within billing period
- 🔲 Generate consolidated invoice on cycle close
- 🔲 Manual invoice generation option for admin

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

### MVP Email Notifications 🔲 NOT STARTED

**Interface Ready** ✅
- ✅ Email provider interface defined
- ✅ Mock provider for testing

**Notifications to Implement** 🔲
- 🔲 Order confirmation
- 🔲 Shipping confirmation with tracking
- 🔲 Subscription renewal reminder
- 🔲 Subscription payment failed
- 🔲 Invoice sent
- 🔲 Invoice payment reminder (approaching due date)
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
| Customer Auth | ✅ Complete | Signup, login, sessions with bcrypt |
| Price Lists | ✅ Complete | Multi-tier pricing, customer assignment |
| Shopping Cart | ✅ Complete | Add/update/remove, htmx updates |
| Checkout | ✅ Complete | 5-step flow, address validation, Stripe Elements |
| Payments | ✅ Complete | Payment intents, webhooks, idempotent processing |
| Orders | ✅ Complete | Creation, status tracking, admin management |
| Shipping | ⏳ Partial | Flat-rate working, no carrier integration |
| Admin Dashboard | ⏳ Partial | Products, orders, customers; missing subscriptions/invoices |
| Subscriptions | 🔲 Schema only | 4 tables ready, no service layer |
| Invoicing | 🔲 Schema only | 4 tables ready, no service layer |
| Email | 🔲 Interface only | Provider interface defined, no implementation |

### Architecture Highlights

- **44 database tables** across 16 migrations
- **30+ HTTP handlers** for storefront, admin, and webhooks
- **5 service layers** with comprehensive test coverage
- **Interface-based abstractions** for billing, shipping, email, storage, tax
- **Multi-tenant isolation** on all queries
- **Idempotent webhook processing** for payment reliability

### Remaining MVP Work

1. **Subscriptions** — SubscriptionService, Stripe Billing integration, customer portal
2. **Wholesale/Invoicing** — InvoiceService, net terms, consolidated billing
3. **Email notifications** — Provider implementation, transactional emails
4. **Polish** — Product filters, wholesale minimums, pick lists

---

## Milestone Summary

| Milestone | Status | Key Deliverable |
|-----------|--------|-----------------|
| Phase 1-3 | ✅ Complete | B2C checkout with Stripe payments |
| Phase 4 | ⏳ Partial | Flat-rate shipping, fulfillment workflow |
| Phase 5 | 🔲 Not started | Subscriptions |
| Phase 6 | 🔲 Not started | Wholesale & invoicing |
| MVP + 2 mo | — | Shipping provider integration, automated labels |
| MVP + 4 mo | — | Inventory management, discounts, reviews |
| MVP + 6 mo | — | Accounting integration, platform hardening |