# Hiri Business Case

## Purpose

Hiri is an e-commerce platform built exclusively for coffee roasters. It provides the business-critical features that small roasting operations need to sell both direct-to-consumer and wholesale—without the enterprise pricing or complexity of general-purpose platforms.

**Core thesis:** By building exclusively for coffee products, we eliminate the complexity that general-purpose platforms must carry. This lets a solo developer build and maintain what would otherwise require a team, passing the cost savings to customers.

## The Problem

Small coffee roasters (1-5 locations) face a difficult choice when setting up e-commerce:

**General platforms (Shopify, WooCommerce, BigCommerce)**
- Handle B2C well but B2B is an afterthought or expensive add-on
- Wholesale pricing tiers, net terms, and custom billing intervals require plugins that cost $50-300/month each
- Subscription functionality often requires yet another paid app
- None of these tools understand coffee—roasters end up hacking together solutions for roast dates, grind options, and origin metadata

**Wholesale-focused platforms**
- Built for large distributors, not small producers
- Overcomplicated for a roaster doing $200k-2M in annual revenue
- Pricing reflects enterprise sales cycles, not small business budgets

**The result:** Roasters either overpay for features they need, cobble together fragile integrations, or simply don't offer capabilities (like subscriptions or proper wholesale accounts) that would grow their business.

## Target Customer

Independent coffee roasters with 1-5 retail locations who:

- Sell roasted coffee online (B2C) and want to grow or formalize wholesale (B2B)
- Currently use a patchwork of tools or a platform that doesn't quite fit
- Have revenue between $200k-2M annually
- Value reliability and simplicity over endless customization options
- Would rather pay a fair, predictable price than negotiate enterprise contracts

## What Hiri Is Not

- A general-purpose e-commerce platform (we will never sell "products"—only coffee)
- A marketplace (each roaster gets their own store)
- A platform for large-scale distributors or multinational coffee companies
- A point-of-sale system (focused on online sales, not in-store)

## Success Criteria

Hiri succeeds if a small roaster can:

1. Set up a complete online store in a day, not a week
2. Manage both retail and wholesale customers in one place
3. Offer subscriptions without integrating a third-party service
4. Invoice wholesale accounts on flexible terms without spreadsheets
5. Pay a price that makes sense for their business size

---

## Market Opportunity

Small coffee roasters (1-5 locations, $200k-2M annual revenue) face a gap in e-commerce tooling:

| Solution | Monthly Cost | B2B Support | Subscriptions | Coffee-Specific |
|----------|--------------|-------------|---------------|-----------------|
| Shopify Basic | $39 | - | - | - |
| Shopify + plugins | $150-350 | Plugin | Plugin | - |
| BigCommerce + B2B | $300+ | Built-in | Plugin | - |
| Wholesale platforms | 15-25% cut | Yes | - | - |
| **Hiri** | **$149** | **Built-in** | **Built-in** | **Yes** |

**Example customer scenario:**

A roaster doing $50k/month in combined B2C and B2B sales currently pays:
- Shopify Basic: $39
- Bold Subscriptions: $100
- Wholesale Club: $30
- Invoice app: $30
- **Total: $199/month** across four separate tools

Hiri offers one integrated system for $149/month with no transaction fees.

---

## Pricing

**Model:** Flat monthly subscription

| Plan | Price |
|------|-------|
| Monthly | $149/month |
| Annual | $129/month ($1,548/year) |

**What's included:** Everything. No plugins, no tiers, no transaction fees.

**What's not included:** Stripe payment processing fees (2.9% + $0.30)—passed through at cost, industry standard.

---

## Unit Economics

### Cost Structure

**Fixed costs (monthly):**
- Base VPS (application + database): $50-100
- Domain/DNS: ~$2
- Transactional email: $0-20
- **Total: ~$75-125/month**

**Variable costs (per customer/month):**
- Incremental compute/database: $1-3
- Email volume: $1-3
- File storage: $0.50-1
- **Total: ~$3-7/month**

### Contribution Margin

$149 price - $5 average variable cost = **$144/month per customer**

### Break-Even

| Milestone | Customers | Monthly Revenue | Gross Margin |
|-----------|-----------|-----------------|--------------|
| Cash break-even | 1 | $149 | $44 |
| Time break-even* | 14 | $2,086 | $2,000 |
| Sustainable | 25 | $3,725 | $3,450 |
| Profitable | 50 | $7,450 | $7,000 |

*Assuming founder time valued at $100/hour, 20 hours/month at low customer counts.

---

## Validation Milestones

1. **Design partners (0-5 customers):** Find 3-5 roasters willing to use early builds and provide feedback. Free or heavily discounted.

2. **Paying pilot (5-10 customers):** Convert design partners and acquire new customers at $99-149/month. Validates willingness to pay.

3. **Sustainable operation (25+ customers):** Revenue covers costs with margin. Product validated; focus shifts to growth.

---

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Low customer acquisition | Medium | High | Validate with design partners before heavy development |
| Support volume overwhelms founder | Medium | Medium | Build good documentation, self-service portal |
| Larger competitor copies features | Low | Medium | Speed and focus; they can't justify eng time for small niche |
| Churn from customers outgrowing | Low | Low | Target market has natural ceiling |
| Stripe dependency | Low | High | Abstract billing interface allows future alternatives |
