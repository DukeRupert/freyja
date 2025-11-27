# Freyja Feature Roadmap

## Overview

This roadmap defines the path to MVP launch and the six months following. The MVP focuses on complete, reliable functionality for core use cases rather than breadth of features.

---

## MVP (12 Weeks)

Target: A roaster can sell coffee online to retail and wholesale customers with flexible pricing, subscriptions, and invoicing.

### Phase 1: Foundation (Weeks 1-2)

**Product Catalog**
- Coffee product management (name, description, images)
- Coffee-specific attributes: origin, region, producer, process, roast level, tasting notes, elevation
- SKU variants by weight and grind option
- Inventory tracking per SKU
- Product visibility controls (public, wholesale-only, restricted)
- Active/inactive status

**Customer Accounts**
- Email/password authentication
- Magic link authentication (passwordless option)
- Account types: retail and wholesale
- Profile management with saved addresses
- Wholesale account application flow

**Price List System**
- Default retail price list
- Named wholesale price lists (e.g., "Café Tier 1", "Restaurant Tier 2")
- Per-product pricing per list
- Price list assignment to customer accounts
- Restricted product access via price list entries

### Phase 2: Storefront & Cart (Weeks 3-4)

**Product Display**
- Product listing with filters (roast level, origin, process)
- Product detail pages with coffee metadata
- Dynamic pricing based on logged-in customer's price list
- Grind and size selection

**Shopping Cart**
- Add/remove/update cart items
- Cart persistence (session for guests, database for authenticated)
- Price recalculation on cart changes
- Minimum order enforcement for wholesale accounts

**Checkout Flow**
- Address entry and selection
- Shipping method selection (initially manual rates)
- Order summary with line items and totals
- Tax calculation (flat rate or simple API integration)

### Phase 3: Billing & Payments (Weeks 5-6)

**Billing Interface**
- Abstract billing provider interface
- Methods: customer management, one-time charges, subscriptions, invoicing
- Webhook handling abstraction

**Stripe Implementation**
- Customer creation and synchronization
- Payment method storage
- One-time payment processing via Payment Intents
- Webhook handlers for payment events
- Idempotent event processing

**Order Management**
- Order creation on successful payment
- Order status workflow: pending → paid → processing → shipped → delivered
- Order history for customers
- Admin order list and detail views

### Phase 4: Shipping (Weeks 7-8)

**Shipping Interface**
- Abstract shipping provider interface
- Methods: get rates, create label, track shipment

**Manual Fulfillment Provider**
- Admin enters shipping cost at checkout or order level
- Admin enters tracking number post-shipment
- Basic tracking status display

**Fulfillment Workflow**
- Pick list generation
- Mark orders as shipped
- Shipping confirmation emails with tracking

### Phase 5: Subscriptions (Weeks 9-10)

**Subscription Management**
- Subscription plans linked to products
- Frequency options: weekly, every 2 weeks, monthly, every 6 weeks, every 2 months
- Quantity and grind selection per subscription
- Subscription status: active, paused, canceled

**Stripe Subscription Integration**
- Create and manage subscriptions via Stripe Billing
- Handle subscription lifecycle webhooks
- Failed payment retry handling
- Dunning management (email notifications for payment issues)

**Customer Subscription Portal**
- View active subscriptions
- Pause/resume subscription
- Skip next delivery
- Change frequency, quantity, or grind
- Cancel subscription

### Phase 6: Wholesale & Invoicing (Weeks 11-12)

**Wholesale Account Management**
- Application review queue for admin
- Approval workflow with price list and terms assignment
- Wholesale-specific dashboard view

**Invoice Billing**
- Net terms configuration per account (Net 15, Net 30, etc.)
- Invoice generation on order placement
- Invoice status tracking: draft, sent, paid, overdue
- Stripe Invoice integration for payment collection

**Consolidated Billing**
- Billing cycle configuration per account (weekly, biweekly, monthly)
- Accumulate orders within billing period
- Generate consolidated invoice on cycle close
- Manual invoice generation option for admin

### MVP Admin Dashboard

**Included Throughout Phases**
- Product CRUD with image upload
- Customer management (view, edit, assign price lists, approve wholesale)
- Order management with fulfillment actions
- Subscription overview
- Invoice management
- Basic sales metrics (revenue, order count, active subscriptions)

### MVP Email Notifications

- Order confirmation
- Shipping confirmation with tracking
- Subscription renewal reminder
- Subscription payment failed
- Invoice sent
- Invoice payment reminder (approaching due date)
- Invoice overdue

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

## Milestone Summary

| Milestone | Target Date | Key Deliverable |
|-----------|-------------|-----------------|
| MVP | Week 12 | Full B2C + B2B ordering, subscriptions, invoicing |
| MVP + 2 mo | Month 2 | Shipping provider integration, automated labels |
| MVP + 4 mo | Month 4 | Inventory management, discounts, reviews |
| MVP + 6 mo | Month 6 | Accounting integration, platform hardening |