# Products

Manage your coffee catalog, including products, variants, pricing, and inventory.

## In This Section

- [Creating Products](creating-products.md) - Add new products to your catalog
- [SKU Variants](variants.md) - Manage weight and grind options
- [Product Images](images.md) - Upload and manage product photos
- [Inventory Management](inventory.md) - Track and update stock levels
- [Coffee Attributes](coffee-attributes.md) - Origin, roast level, and tasting notes

## Overview

Products in Freyja are designed specifically for coffee. Each product represents a single coffee offering (like "Ethiopia Yirgacheffe") with:

- **Coffee-specific attributes** - Origin, region, process, roast level, tasting notes
- **SKU variants** - Different weight and grind combinations
- **Visibility controls** - Public, wholesale-only, or hidden
- **Multiple pricing** - Different prices for different customer tiers via price lists

## Product Structure

```
Product (Ethiopia Yirgacheffe)
├── Attributes
│   ├── Origin: Ethiopia
│   ├── Region: Yirgacheffe
│   ├── Roast Level: Light
│   └── Tasting Notes: Blueberry, jasmine, citrus
├── SKUs
│   ├── 12oz Whole Bean - $18.00
│   ├── 12oz Drip - $18.00
│   ├── 2lb Whole Bean - $52.00
│   └── 2lb Drip - $52.00
├── Images
│   ├── Primary image
│   └── Additional images
└── Visibility: Public
```

## Quick Actions

- **Add a product:** Products > Add Product
- **Edit a product:** Products > Click product name
- **Manage inventory:** Products > Click product > SKU section
- **Update images:** Products > Click product > Images section

---

Next: [Creating Products](creating-products.md)
