# Quick Start Guide

Get your Freyja store up and running with this step-by-step guide.

## Step 1: Access Your Dashboard

Log in to your Freyja admin dashboard. You'll see an overview of your store with quick stats and navigation to all management sections.

## Step 2: Connect Stripe

Stripe handles all payment processing for Freyja. You'll need to connect your Stripe account before accepting orders.

1. Go to **Settings > Payments**
2. Click **Connect Stripe Account**
3. Follow the Stripe onboarding flow
4. Return to Freyja once connected

See [Connecting Stripe](stripe-setup.md) for detailed instructions.

## Step 3: Configure Tax Settings

Set up how taxes are calculated for your orders.

1. Go to **Settings > Tax**
2. Choose your tax calculation method:
   - **None** - No tax calculation
   - **Stripe Tax** - Automatic address-based rates
   - **Percentage** - Manual state-by-state rates
3. Configure rates if using percentage-based calculation

## Step 4: Set Up Shipping

Configure how shipping costs are calculated.

1. Go to **Settings > Shipping**
2. Choose your shipping provider:
   - **Flat Rate** - Fixed prices for standard/express shipping
   - **EasyPost** - Real-time rates from carriers (USPS, UPS, FedEx)
3. Enter your configuration details

## Step 5: Add Your First Product

1. Go to **Products** in the sidebar
2. Click **Add Product**
3. Fill in the product details:
   - Name and description
   - Coffee attributes (origin, roast level, tasting notes)
   - Base price
4. Add SKU variants (size and grind options)
5. Upload product images
6. Set visibility to **Public**
7. Save the product

See [Your First Product](first-product.md) for a detailed walkthrough.

## Step 6: Create a Price List

Price lists control what prices customers see.

1. Go to **Price Lists**
2. You'll see a default "Retail" price list
3. Click into it to set per-product pricing
4. Set prices for your products

## Step 7: Test Your Store

Before going live, place a test order:

1. Visit your storefront
2. Add a product to cart
3. Go through checkout using Stripe test mode
4. Verify the order appears in your dashboard
5. Check that confirmation emails are sent

## Step 8: Go Live

Once testing is complete:

1. Switch Stripe to live mode
2. Verify all settings are production-ready
3. Announce your store launch

See [Go Live Checklist](go-live-checklist.md) for a complete pre-launch checklist.

---

## Next Steps

- [Add more products](../products/creating-products.md)
- [Set up wholesale pricing](../pricing/wholesale-tiers.md)
- [Configure subscriptions](../subscriptions/plans.md)

---

Previous: [Getting Started](index.md) | Next: [Connecting Stripe](stripe-setup.md)
