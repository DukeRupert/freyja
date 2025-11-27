# Freyja Business Analysis

## Market Opportunity

Small coffee roasters (1-5 locations, $200k-2M annual revenue) face a gap in e-commerce tooling. Their options:

**General platforms with plugins ($150-350/month)**
Shopify or similar plus subscriptions app plus wholesale app plus invoicing app. Works, but expensive, fragmented, and none of it understands coffee.

**Wholesale-specific platforms (15-25% commission or $300+/month)**
Built for large distributors, overbuilt and overpriced for small producers.

**Basic e-commerce ($30-80/month)**
Handles B2C adequately but offers no real B2B support. Roasters outgrow it quickly once wholesale accounts matter.

Freyja targets the gap: business-critical features at small-business prices, purpose-built for coffee.

---

## Competitive Positioning

| Solution | Monthly Cost | B2B Support | Subscriptions | Coffee-Specific |
|----------|--------------|-------------|---------------|-----------------|
| Shopify Basic | $39 | ❌ | ❌ | ❌ |
| Shopify + plugins | $150-350 | Plugin | Plugin | ❌ |
| BigCommerce + B2B | $300+ | Built-in | Plugin | ❌ |
| Wholesale platforms | 15-25% cut | ✅ | ❌ | ❌ |
| **Freyja** | **$149** | **Built-in** | **Built-in** | **✅** |

**Example customer scenario:**

A roaster doing $50k/month in combined B2C and B2B sales currently pays:
- Shopify Basic: $39
- Bold Subscriptions: $100
- Wholesale Club: $30
- Invoice app: $30
- **Total: $199/month** across four separate tools

Freyja offers one integrated system for $149/month with no transaction fees. Clear value, simpler operations.

---

## Pricing Strategy

**Model:** Flat monthly subscription

**Price point:** $149/month

**Annual option:** $129/month ($1,548/year paid upfront)

**Rationale:**
- Simple to communicate: "Everything you need for $149/month, no plugins, no transaction fees"
- Undercuts typical Shopify + plugins stack by $50-200/month
- Predictable for customers—no surprise costs as they grow
- Annual discount improves cash flow and reduces churn
- Room to add tiers later based on observed usage patterns

**What's not included:**
- Stripe payment processing fees (2.9% + $0.30) — passed through to customer, industry standard
- No Freyja transaction fees — cost transparency is a selling point

---

## Architecture Decision: Multi-Tenant

Freyja will use a multi-tenant architecture (shared database, all customers in one instance).

**Why multi-tenant:**

| Benefit | Impact |
|---------|--------|
| Lower infrastructure cost per customer | ~$3-7 variable cost vs $15-30 for single-tenant |
| Single deployment updates all customers | Manageable by solo developer |
| One database to back up and monitor | Simpler operations |
| Scales efficiently to 100+ customers | Infrastructure grows slowly with customer count |

**Tradeoffs accepted:**

| Risk | Mitigation |
|------|------------|
| Noisy neighbor (one customer slows others) | Unlikely at target scale; monitor and address if needed |
| Data isolation concerns | Rigorous tenant_id scoping, tested in CI |
| One bug affects everyone | Staged rollouts, good test coverage |

**Future option:** Offer single-tenant as a premium tier if larger customers demand dedicated resources.

---

## Cost Structure

### Fixed Costs (monthly, regardless of customer count)

| Item | Cost |
|------|------|
| Base VPS (application + database) | $50-100 |
| Domain and DNS | ~$2 |
| Transactional email service | $0-20 |
| Error tracking (free tier) | $0 |
| **Total fixed** | **~$75-125/month** |

### Variable Costs (per customer/month)

| Item | Cost |
|------|------|
| Incremental compute and database | $1-3 |
| Email volume | $1-3 |
| File storage (product images) | $0.50-1 |
| **Total variable** | **~$3-7/month** |

Support (email only, handled by founder) is not included as a cash cost but is a significant time investment, especially in early stages.

---

## Break-Even Analysis

**Contribution margin per customer:**
$149 price - $5 average variable cost = **$144/month**

**Cash break-even:**
$100 fixed costs ÷ $144 contribution = **1 customer**

Infrastructure costs are covered almost immediately. The meaningful break-even is founder time.

**Time-adjusted break-even:**

If founder time is valued at $100/hour and Freyja requires 20 hours/month at low customer counts:
- Opportunity cost: $2,000/month
- Break-even: ~14 customers

This is a rough heuristic. Actual time investment will vary based on support load and development pace.

---

## Growth Projections

| Customers | Monthly Revenue | Variable Costs | Fixed Costs | Gross Margin |
|-----------|-----------------|----------------|-------------|--------------|
| 5 | $745 | $25 | $100 | $620 |
| 10 | $1,490 | $50 | $100 | $1,340 |
| 25 | $3,725 | $125 | $150* | $3,450 |
| 50 | $7,450 | $250 | $200* | $7,000 |
| 100 | $14,900 | $500 | $400* | $14,000 |

*Fixed costs increase modestly with scale (larger VPS, paid tiers of services).

**At 25 customers:** Revenue supports part-time support hire or continued solo operation with healthy margin.

**At 50+ customers:** Business is sustainable and profitable. Decision point on whether to grow aggressively or maintain as lifestyle business.

---

## Support Model

**Initial approach:** Email only, handled by founder

**Rationale:**
- Keeps costs at zero until revenue supports hiring
- Direct customer contact builds product insight
- Email creates natural async buffer, manageable alongside development

**Transition point:** At ~25 customers, evaluate whether to:
- Hire part-time support help
- Implement self-service knowledge base to reduce volume
- Continue solo if volume is manageable

**Not planned:**
- Live chat (high interrupt cost for solo developer)
- Phone support (doesn't scale, not expected at this price point)

---

## Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Low customer acquisition | Medium | High | Validate with 5-10 design partners before heavy development |
| Support volume overwhelms founder | Medium | Medium | Build good documentation, consider self-service portal early |
| Larger competitor copies features | Low | Medium | Speed and focus; they can't justify eng time for small niche |
| Churn from customers outgrowing platform | Low | Low | Target market has natural ceiling; larger roasters aren't the goal |
| Stripe dependency | Low | High | Abstract billing interface allows future alternatives |

---

## Validation Milestones

Before heavy investment, validate demand:

1. **Design partners (0-5 customers):** Find 3-5 roasters willing to use early builds and provide feedback. Can be free or heavily discounted.

2. **Paying pilot (5-10 customers):** Convert design partners and acquire new customers at $99-149/month. Validates willingness to pay.

3. **Sustainable operation (25+ customers):** Revenue covers costs with margin. Product is validated; focus shifts to growth.

---

## Summary

The niche exists: small roasters paying $150-350/month for fragmented tools, or making do without proper B2B features. Freyja offers an integrated solution at $149/month with clear value.

Multi-tenant architecture keeps variable costs around $5/customer, yielding strong unit economics. Cash break-even is nearly immediate; time break-even is ~14 customers.

The question isn't whether the product can work economically—it's whether the target customers can be reached efficiently. That's a marketing and distribution problem to solve alongside product development.