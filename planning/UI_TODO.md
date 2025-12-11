# UI Direction Compliance Checklist

Survey completed: 2025-12-11

This checklist tracks UI components and templates that need updates to comply with `UI_DIRECTION.md`.

---

## High Priority

### Color System Alignment

- [x] **Button component colors** (`web/templates/components/button.html`)
  - Lines 72-96: Change `bg-zinc-900` to brand teal `#2D7A7A`
  - Change `bg-blue-600` to brand teal for primary actions
  - Update focus states from `blue-500` to teal

- [x] **Empty state button colors** (`web/templates/components/empty-state.html`)
  - Lines 53-54, 63-64: Change `bg-blue-600` to brand teal

- [x] **Input focus states** (`web/templates/components/input.html`)
  - Line 52: Change `focus:outline-blue-500` to teal

### Voice & Tone Fixes

- [x] **wholesale_approved.html** (`web/templates/email/wholesale_approved.html`)
  - Line 1: Change "Has Been Approved!" to "Approved"
  - Line 9: Remove "Great news!" - start with "Your wholesale account..."

- [x] **shipping_confirmation.html** (`web/templates/email/shipping_confirmation.html`)
  - Line 9: Remove "Great news!" - start with "Your order has shipped..."

- [x] **subscription_cancelled.html** (`web/templates/email/subscription_cancelled.html`)
  - Line 32: Remove "We're sorry to see you go!" - start with "If you'd like to restart..."

- [x] **settings.html** (`web/templates/storefront/settings.html`)
  - Line 30: Change "Profile updated successfully." to "Profile updated"
  - Line 39: Change "Password changed successfully." to "Password changed"

---

## Medium Priority

### Typography Consistency

- [x] **Admin layout font** (`web/templates/admin/layout.html`)
  - Lines 10-12: Remove Inter font import from Google Fonts
  - Lines 18-25: Remove custom font-family style block
  - Use system font stack per spec: `-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif`

### Badge Semantic Colors

- [x] **Badge component** (`web/templates/components/badge.html`)
  - Update color variants to use brand semantic colors:
    - Success: `#3D8B6E` (Coastal green) instead of generic green
    - Warning: `#C4873A` (Trade amber) instead of generic amber
    - Error: `#B85C4A` (Ochre red) instead of generic red

### Focus States

- [ ] **Field component** (`web/templates/components/field.html`)
  - Verify focus states use teal accent

- [ ] **Select component** (`web/templates/components/select.html`)
  - Update focus states to teal

- [ ] **Textarea component** (`web/templates/components/textarea.html`)
  - Update focus states to teal

---

## Low Priority

### Background Colors

- [ ] **Admin layout background** (`web/templates/admin/layout.html`)
  - Line 29: Consider changing `bg-zinc-50` to warmer `#FAF9F7` (spec background)

- [ ] **Tailwind config** (`web/static/css/input.css`)
  - Add custom background color: `--color-background: #FAF9F7`
  - Update base layer body background

### Border Radius Audit

- [ ] **Button border radius** (`web/templates/components/button.html`)
  - Line 49: `rounded-lg` is 8px, spec says 6px for buttons
  - Consider custom `rounded-[6px]` or update Tailwind config

- [ ] **Input border radius** (`web/templates/components/input.html`)
  - Line 41: Verify 6px radius per spec

### Additional Voice/Tone Review

- [ ] **wholesale_rejected.html** (`web/templates/email/wholesale_rejected.html`)
  - Line 19: Review "enjoy our great products" phrasing

- [ ] **storefront/home.html** (`web/templates/storefront/home.html`)
  - Line 275: Review testimonial content (may be acceptable as customer quote)

---

## Design System Unification

### Long-term: Consolidate Dual Systems

The codebase currently has two competing design approaches:

1. **Admin/Components**: Catalyst UI Kit style with generic Tailwind colors (zinc, blue, red)
2. **Storefront/CSS**: Brand-aligned with teal/amber custom colors

Consider:

- [ ] Create unified color tokens in Tailwind config
- [ ] Update component library to use brand tokens
- [ ] Document component usage guidelines
- [ ] Audit all templates for consistent color usage

---

## Reference

### Spec Colors (from UI_DIRECTION.md)

| Role | Color | Current Implementation |
|------|-------|----------------------|
| Primary action | `#2D7A7A` (Ocean teal) | `bg-zinc-900` or `bg-blue-600` |
| Primary hover | `#1E5E5E` (Deep teal) | Various |
| Secondary accent | `#C4873A` (Sunset amber) | `amber-700` ✓ |
| Success | `#3D8B6E` (Coastal green) | Generic green |
| Warning | `#C4873A` (Trade amber) | Generic amber |
| Error | `#B85C4A` (Ochre red) | Generic red |
| Background | `#FAF9F7` (Warm off-white) | `zinc-50` |

### Spec Typography

```css
font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
```

### Spec Border Radius

| Element | Radius |
|---------|--------|
| Buttons | 6px |
| Inputs | 6px |
| Cards | 8px |
| Badges | 4px |
| Modals | 12px |
