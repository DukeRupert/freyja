# Pricing

Manage price lists, wholesale tiers, and product visibility for different customer segments.

## In This Section

- [Understanding Price Lists](price-lists.md) - How pricing tiers work
- [Setting Up Wholesale Tiers](wholesale-tiers.md) - Create pricing for wholesale customers
- [Product Visibility](visibility.md) - Control who sees which products

## Overview

Freyja uses **price lists** to manage pricing for different customer segments:

- **Retail customers** see retail prices
- **Wholesale customers** see their assigned tier prices
- **Anonymous visitors** see default retail prices

This allows you to maintain different pricing without creating duplicate products.

## How Price Lists Work

```
Product: Ethiopia Yirgacheffe (12oz)
├── Retail Price List: $18.00
├── Café Tier 1: $14.00
├── Café Tier 2: $12.00
└── Restaurant: $10.00
```

Each customer is assigned to one price list. When they view products or check out, they see prices from their assigned list.

## Default Price List

All customers start with the default retail price list. Wholesale customers are assigned to other price lists when approved.

## Quick Actions

- **View price lists:** Price Lists in sidebar
- **Create new list:** Price Lists > Add Price List
- **Set product prices:** Price Lists > Click list > Set prices
- **Assign customer:** Customers > Edit customer > Assign price list

---

Next: [Understanding Price Lists](price-lists.md)
