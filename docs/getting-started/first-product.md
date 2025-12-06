# Your First Product

This guide walks through adding your first coffee product to Freyja.

## Creating a Product

1. Go to **Products** in the admin sidebar
2. Click **Add Product**
3. You'll see the product form with several sections

## Basic Information

Fill in the core product details:

- **Name** - The product name (e.g., "Ethiopia Yirgacheffe")
- **Description** - Detailed description for customers
- **Slug** - URL-friendly identifier (auto-generated from name)

## Coffee Attributes

These coffee-specific fields help customers find and understand your products:

- **Origin** - Country of origin (e.g., "Ethiopia")
- **Region** - Specific growing region (e.g., "Yirgacheffe")
- **Producer** - Farm or cooperative name
- **Process** - Processing method (Washed, Natural, Honey, etc.)
- **Roast Level** - Light, Medium-Light, Medium, Medium-Dark, Dark
- **Tasting Notes** - Flavor descriptors (e.g., "Blueberry, jasmine, citrus")
- **Elevation** - Growing elevation in meters

## Pricing

- **Base Price** - The starting price for this product
- Actual prices are determined by price lists (set up separately)

## Visibility

Control who can see this product:

- **Public** - Visible to all customers
- **Wholesale Only** - Only visible to approved wholesale customers
- **Hidden** - Not visible on storefront (draft products)

## Adding SKU Variants

SKUs represent the specific purchasable items with weight and grind options.

1. In the **Variants** section, click **Add Variant**
2. Fill in variant details:
   - **Weight** - Package size (e.g., "12 oz", "2 lb", "5 lb")
   - **Grind** - Grind option (Whole Bean, Drip, Espresso, French Press, etc.)
   - **SKU** - Unique identifier (auto-generated or custom)
   - **Price** - Price for this specific variant
   - **Stock** - Current inventory count
3. Repeat for each weight/grind combination you offer

### Example SKU Setup

For a product available in 12oz and 2lb bags:

| Weight | Grind | SKU | Price |
|--------|-------|-----|-------|
| 12 oz | Whole Bean | ETH-YRG-12-WB | $18.00 |
| 12 oz | Drip | ETH-YRG-12-DR | $18.00 |
| 12 oz | Espresso | ETH-YRG-12-ES | $18.00 |
| 2 lb | Whole Bean | ETH-YRG-2LB-WB | $52.00 |
| 2 lb | Drip | ETH-YRG-2LB-DR | $52.00 |

## Adding Images

1. In the **Images** section, click **Upload Image**
2. Select your product photo
3. First image becomes the primary/featured image
4. Add additional images as needed
5. Drag to reorder images

**Image recommendations:**
- Square format (1:1 ratio)
- Minimum 800x800 pixels
- Clean, well-lit product photos
- Show the bag/packaging clearly

## Saving Your Product

1. Review all information
2. Click **Save Product**
3. The product is now in your catalog

## After Creating

Once saved, you can:

- View the product on your storefront
- Edit details anytime
- Set specific prices in price lists
- Create subscription plans for this product

---

Previous: [Connecting Stripe](stripe-setup.md) | Next: [Go Live Checklist](go-live-checklist.md)
