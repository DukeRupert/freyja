# SKU Variants

SKU variants represent the specific purchasable combinations of weight and grind for each product.

## Understanding SKUs

A SKU (Stock Keeping Unit) is a unique identifier for each purchasable item. In Freyja, SKUs combine:

- **Product** - The base coffee (e.g., Ethiopia Yirgacheffe)
- **Weight** - Package size (e.g., 12 oz, 2 lb, 5 lb)
- **Grind** - Grind option (e.g., Whole Bean, Drip, Espresso)

Each combination has its own:
- SKU code
- Price
- Inventory count

## Creating Variants

1. Open a product for editing
2. Scroll to the **Variants** section
3. Click **Add Variant**
4. Fill in the variant details:
   - Weight
   - Grind
   - SKU code (auto-generated or custom)
   - Price
   - Stock quantity
5. Click **Save**

## Common Weight Options

| Weight | Typical Use |
|--------|-------------|
| 12 oz | Standard retail bag |
| 1 lb | Retail / small wholesale |
| 2 lb | Wholesale |
| 5 lb | Wholesale / bulk |

## Common Grind Options

| Grind | Description |
|-------|-------------|
| Whole Bean | Unground beans |
| Drip | Medium grind for drip coffee makers |
| Espresso | Fine grind for espresso machines |
| French Press | Coarse grind for immersion brewing |
| Pour Over | Medium-fine for pour over methods |
| Cold Brew | Extra coarse for cold extraction |

## SKU Naming Convention

A consistent SKU naming convention helps with inventory management. Example format:

`[ORIGIN]-[REGION]-[WEIGHT]-[GRIND]`

Examples:
- `ETH-YRG-12-WB` (Ethiopia Yirgacheffe, 12oz, Whole Bean)
- `COL-HUI-2LB-DR` (Colombia Huila, 2lb, Drip)
- `BRZ-SUL-5LB-WB` (Brazil Sul de Minas, 5lb, Whole Bean)

## Pricing Variants

Each variant has a base price. This price can be overridden per price list:

- **Base price** - Set on the variant itself
- **Price list price** - Override for specific customer tiers

If no price list override exists, the base price is used.

## Inventory

Each SKU tracks its own inventory:

- **Stock quantity** - Current units available
- Inventory decreases when orders are placed
- Update stock when you receive or roast new inventory

See [Inventory Management](inventory.md) for details.

## Editing Variants

1. Open the product
2. Find the variant in the **Variants** section
3. Click **Edit** on the variant
4. Make changes
5. Click **Save**

## Deleting Variants

Variants with order history cannot be deleted. Instead:
- Set stock to 0 to prevent new orders
- The variant will show as "Out of Stock"

---

Previous: [Creating Products](creating-products.md) | Next: [Product Images](images.md)
