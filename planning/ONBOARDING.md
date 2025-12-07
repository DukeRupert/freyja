# Tenant Onboarding Checklist

This document defines the onboarding flow for new coffee roaster tenants on the Freyja platform. It serves as the source of truth for what must be configured before a store can go live.

## Overview

When a new tenant signs up via the SaaS checkout flow, they receive a pending tenant record and must complete several configuration steps before their store can accept orders. This checklist tracks their progress and guides them through setup.

## Onboarding Phases

### Phase 1: Critical Path (Required for Launch)

These steps are **mandatory** before the store can accept orders.

| Step ID | Step Name | Description | Validation Criteria |
|---------|-----------|-------------|---------------------|
| `account_activated` | Activate Account | Set password via setup email link | Operator has hashed password, setup_completed_at is set |
| `stripe_connected` | Connect Stripe | Configure payment processing | Valid Stripe API keys in tenant_provider_configs, connection tested |
| `email_configured` | Configure Email | Set up transactional email provider | Valid email provider config in tenant_provider_configs, connection tested |
| `first_product` | Add First Product | Create at least one active product | At least 1 product with status='active' exists |
| `first_sku` | Create Product SKU | Add weight/grind variant to product | At least 1 SKU exists for an active product |
| `pricing_set` | Set Up Pricing | Configure retail prices for SKUs | Price list entries exist for all SKUs in default price list |
| `shipping_configured` | Configure Shipping | Add at least one shipping method | At least 1 active shipping method exists |
| `tax_configured` | Configure Taxes | Select tax calculation method | Tax provider configured (even if 'none') |

### Phase 2: Recommended (Better Experience)

These steps improve the customer experience but aren't required to accept orders.

| Step ID | Step Name | Description | Validation Criteria |
|---------|-----------|-------------|---------------------|
| `business_info` | Business Information | Store name, contact details | Tenant has name, email, phone populated |
| `warehouse_address` | Warehouse Address | Fulfillment origin for shipping | Primary warehouse address exists |
| `product_images` | Product Images | Upload images for products | All active products have at least 1 image |
| `coffee_attributes` | Coffee Details | Origin, roast level, tasting notes | Active products have origin and roast_level set |

### Phase 3: Wholesale (B2B Only)

These steps enable wholesale/B2B functionality.

| Step ID | Step Name | Description | Validation Criteria |
|---------|-----------|-------------|---------------------|
| `wholesale_pricing` | Wholesale Pricing | Create B2B price list | At least 1 wholesale price list with entries exists |
| `payment_terms` | Payment Terms | Configure net terms | At least 1 payment term marked as default |

### Phase 4: Advanced (Post-Launch)

These are optional enhancements that can be configured after launch.

| Step ID | Step Name | Description | Validation Criteria |
|---------|-----------|-------------|---------------------|
| `subscriptions` | Subscriptions | Enable recurring orders | Subscription products configured in Stripe |
| `multiple_shipping` | Multiple Shipping Options | Offer shipping choices | 2+ active shipping methods exist |
| `advanced_tax` | Advanced Tax Integration | TaxJar/Avalara integration | Third-party tax provider configured |

## Checklist States

Each checklist item can be in one of these states:

| State | Value | Description |
|-------|-------|-------------|
| Not Started | `not_started` | User hasn't begun this step |
| In Progress | `in_progress` | User has started but not completed |
| Completed | `completed` | Validation criteria met |
| Skipped | `skipped` | User explicitly skipped (optional items only) |

## Launch Readiness

A store is considered **ready to launch** when all Phase 1 (Critical Path) items are completed:

```
launch_ready = ALL of:
  - account_activated = completed
  - stripe_connected = completed
  - email_configured = completed
  - first_product = completed
  - first_sku = completed
  - pricing_set = completed
  - shipping_configured = completed
  - tax_configured = completed
```

## Validation Logic

Each step has specific validation logic to determine completion:

### account_activated
```sql
SELECT EXISTS(
  SELECT 1 FROM tenant_operators
  WHERE tenant_id = ? AND setup_completed_at IS NOT NULL
)
```

### stripe_connected
```sql
SELECT EXISTS(
  SELECT 1 FROM tenant_provider_configs
  WHERE tenant_id = ?
    AND provider_type = 'billing'
    AND provider_name = 'stripe'
    AND is_active = true
    AND config->>'api_key' IS NOT NULL
)
```

### email_configured
```sql
SELECT EXISTS(
  SELECT 1 FROM tenant_provider_configs
  WHERE tenant_id = ?
    AND provider_type = 'email'
    AND is_active = true
)
```

### first_product
```sql
SELECT EXISTS(
  SELECT 1 FROM products
  WHERE tenant_id = ? AND status = 'active'
)
```

### first_sku
```sql
SELECT EXISTS(
  SELECT 1 FROM product_skus ps
  JOIN products p ON p.id = ps.product_id
  WHERE p.tenant_id = ? AND p.status = 'active'
)
```

### pricing_set
```sql
-- All SKUs of active products have entries in the default price list
SELECT NOT EXISTS(
  SELECT 1 FROM product_skus ps
  JOIN products p ON p.id = ps.product_id
  JOIN price_lists pl ON pl.tenant_id = p.tenant_id AND pl.list_type = 'default'
  LEFT JOIN price_list_entries ple ON ple.price_list_id = pl.id AND ple.sku_id = ps.id
  WHERE p.tenant_id = ? AND p.status = 'active' AND ple.id IS NULL
)
```

### shipping_configured
```sql
SELECT EXISTS(
  SELECT 1 FROM shipping_methods
  WHERE tenant_id = ? AND is_active = true
)
```

### tax_configured
```sql
SELECT EXISTS(
  SELECT 1 FROM tenant_provider_configs
  WHERE tenant_id = ?
    AND provider_type = 'tax'
    AND is_active = true
)
```

### business_info
```sql
SELECT EXISTS(
  SELECT 1 FROM tenants
  WHERE id = ?
    AND name IS NOT NULL
    AND email IS NOT NULL
)
```

### warehouse_address
```sql
SELECT EXISTS(
  SELECT 1 FROM addresses
  WHERE tenant_id = ?
    AND address_type = 'warehouse'
)
```

### product_images
```sql
-- All active products have at least one image
SELECT NOT EXISTS(
  SELECT 1 FROM products p
  LEFT JOIN product_images pi ON pi.product_id = p.id
  WHERE p.tenant_id = ? AND p.status = 'active'
  GROUP BY p.id
  HAVING COUNT(pi.id) = 0
)
```

### coffee_attributes
```sql
SELECT NOT EXISTS(
  SELECT 1 FROM products
  WHERE tenant_id = ?
    AND status = 'active'
    AND (origin IS NULL OR roast_level IS NULL)
)
```

### wholesale_pricing
```sql
SELECT EXISTS(
  SELECT 1 FROM price_lists pl
  JOIN price_list_entries ple ON ple.price_list_id = pl.id
  WHERE pl.tenant_id = ? AND pl.list_type = 'wholesale'
)
```

### payment_terms
```sql
SELECT EXISTS(
  SELECT 1 FROM payment_terms
  WHERE tenant_id = ? AND is_default = true
)
```

## UI Presentation

The onboarding checklist should be presented as:

1. **Dashboard Widget** - Persistent on admin dashboard until all Phase 1 items complete
2. **Progress Indicator** - "X of 8 steps complete" with progress bar
3. **Contextual Links** - Each item links to the relevant settings page
4. **Completion Celebration** - Toast/modal when store becomes launch-ready
5. **Dismissable** - After launch-ready, user can dismiss but access via settings

### Checklist Display Priority

1. Show Phase 1 items first (always visible until complete)
2. Show Phase 2 items after Phase 1 complete (collapsible)
3. Show Phase 3/4 items only when relevant or user expands

### Item Display Format

```
[Status Icon] Step Name
Brief description of what this step accomplishes
[Action Button: "Configure" / "Complete" / "View"]
```

## Critical Gotchas (Surface in UI)

These are common issues that should be highlighted in the onboarding UI:

1. **Products must be "active"** - Draft products won't appear in storefront
2. **SKUs need price list entries** - Unpiced SKUs can't be purchased
3. **At least 1 shipping method required** - Checkout fails without shipping
4. **Test Stripe connection** - Validate keys before going live
5. **Email provider required** - Order confirmations won't send without it

## Future Enhancements

- **Guided Tours** - Step-by-step walkthrough overlay for complex tasks
- **Sample Data** - Option to populate with example coffee products
- **Video Tutorials** - Embedded help videos for each step
- **AI Assistant** - Context-aware help based on current step
- **Mobile Onboarding** - Responsive checklist for mobile admin access
