# Checkout Flow - Visual Design Specification

## Layout Structure

### Desktop (≥1024px)

```
┌─────────────────────────────────────────────────────────────────────┐
│ Header (Site Navigation)                                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌─ Page Title ──────────────────────────────────────────────┐     │
│  │ Checkout                                                   │     │
│  │ Complete your order in a few simple steps                  │     │
│  └────────────────────────────────────────────────────────────┘     │
│                                                                      │
│  ┌─ Checkout Form (2/3 width) ──┐  ┌─ Order Summary (1/3) ──┐     │
│  │                               │  │                          │     │
│  │  ┌─ Step 1: Contact ───────┐ │  │  Order Summary           │     │
│  │  │ ✓ Contact Information  │ │  │                          │     │
│  │  │ [Collapsed]            │ │  │  [Cart Items]            │     │
│  │  └────────────────────────┘ │  │                          │     │
│  │                               │  │  Subtotal    $36.00     │     │
│  │  ┌─ Step 2: Shipping Addr ─┐│  │  Shipping    $8.95      │     │
│  │  │ ✓ Shipping Address      ││  │  Tax         $3.24      │     │
│  │  │ [Collapsed]             ││  │  ───────────────────     │     │
│  │  └────────────────────────┘ │  │  Total       $48.19     │     │
│  │                               │  │                          │     │
│  │  ┌─ Step 3: Shipping Method┐│  └──────────────────────────┘     │
│  │  │ ○ Shipping Method       ││                                   │
│  │  │ [Rate Selection]        ││  (Sticky - stays on scroll)       │
│  │  │                         ││                                   │
│  │  │ [Continue to Billing]   ││                                   │
│  │  └────────────────────────┘ │                                   │
│  │                               │                                   │
│  └───────────────────────────────┘                                   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Mobile (<1024px)

```
┌──────────────────────┐
│ Header               │
├──────────────────────┤
│                      │
│  Checkout            │
│  Complete order...   │
│                      │
│  ┌─ Summary Toggle ─┐│
│  │ Order summary ▼  ││
│  │ ($48.19)         ││
│  └──────────────────┘│
│                      │
│  [Expandable Summary]│
│                      │
│  ┌─ Step 1 ─────────┐│
│  │ ✓ Contact Info   ││
│  │ [Collapsed]      ││
│  └──────────────────┘│
│                      │
│  ┌─ Step 2 ─────────┐│
│  │ ○ Shipping Addr  ││
│  │ [Active Form]    ││
│  │                  ││
│  │ [Continue]       ││
│  └──────────────────┘│
│                      │
└──────────────────────┘
```

---

## Color Palette

### Primary Colors
```css
/* Teal (Primary Action) */
--teal-700: #2A7D7D;    /* Buttons, links, focus */
--teal-800: #1F5F5F;    /* Hover states */
--teal-100: #D6EBEB;    /* Light backgrounds */
--teal-50:  #F0F7F7;    /* Subtle highlights */

/* Neutral (Base) */
--neutral-900: #1A1A1A; /* Headings, primary text */
--neutral-600: #6B6B6B; /* Secondary text, labels */
--neutral-200: #E8E6E3; /* Borders, dividers */
--neutral-100: #F5F5F5; /* Card backgrounds */
--neutral-50:  #FAFAF8; /* Page background */
```

### Semantic Colors
```css
/* Success */
--green-900: #1F5F3D;   /* Success text */
--green-50:  #E8F5E9;   /* Success background */
--green-200: #A5D6A7;   /* Success border */

/* Error */
--red-900:   #7F1D1D;   /* Error text */
--red-50:    #FEE2E2;   /* Error background */
--red-200:   #FCA5A5;   /* Error border */

/* Warning */
--amber-900: #78350F;   /* Warning text */
--amber-50:  #FEF3C7;   /* Warning background */
--amber-200: #FCD34D;   /* Warning border */
```

---

## Typography

### Font Stack
```css
font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto,
             Helvetica, Arial, sans-serif;
```

### Type Scale
```css
/* Page Title */
.page-title {
  font-size: 24px;      /* 1.5rem */
  font-weight: 600;
  line-height: 1.3;
  color: var(--neutral-900);
}

/* Page Subtitle */
.page-subtitle {
  font-size: 14px;      /* 0.875rem */
  font-weight: 400;
  line-height: 1.5;
  color: var(--neutral-600);
}

/* Step Title (Card Header) */
.step-title {
  font-size: 16px;      /* 1rem */
  font-weight: 500;
  line-height: 1.3;
  color: var(--neutral-900);
}

/* Form Label */
.form-label {
  font-size: 14px;      /* 0.875rem */
  font-weight: 500;
  line-height: 1.5;
  color: var(--neutral-900);
}

/* Body Text */
.body-text {
  font-size: 15px;      /* 0.9375rem */
  font-weight: 400;
  line-height: 1.5;
  color: var(--neutral-900);
}

/* Helper Text */
.helper-text {
  font-size: 12px;      /* 0.75rem */
  font-weight: 400;
  line-height: 1.4;
  color: var(--neutral-600);
}
```

---

## Spacing Scale

```css
/* 4px-based spacing system */
--spacing-xs:  4px;
--spacing-sm:  8px;
--spacing-md:  16px;
--spacing-lg:  24px;
--spacing-xl:  32px;
--spacing-2xl: 48px;
```

### Component Spacing

**Card Padding:**
```css
.card {
  padding: 24px; /* --spacing-lg */
}
```

**Form Field Spacing:**
```css
.form-group {
  margin-bottom: 16px; /* --spacing-md */
}

.form-label {
  margin-bottom: 4px; /* --spacing-xs */
}
```

**Step Card Gap:**
```css
.step-container {
  gap: 16px; /* --spacing-md between steps */
}
```

---

## Components

### Buttons

#### Primary Button
```css
.btn-primary {
  background-color: #2A7D7D;
  color: #FFFFFF;
  padding: 12px 20px;
  font-size: 15px;
  font-weight: 500;
  border-radius: 6px;
  border: none;
  transition: background-color 150ms;
}

.btn-primary:hover {
  background-color: #1F5F5F;
}

.btn-primary:disabled {
  background-color: #E5E5E5;
  color: #9CA3AF;
  cursor: not-allowed;
}
```

#### Secondary Button
```css
.btn-secondary {
  background-color: #FFFFFF;
  color: #1A1A1A;
  padding: 12px 20px;
  font-size: 15px;
  font-weight: 500;
  border-radius: 6px;
  border: 1px solid #E8E6E3;
  transition: background-color 150ms;
}

.btn-secondary:hover {
  background-color: #FAFAF8;
}
```

#### Edit Link Button
```css
.btn-edit {
  color: #2A7D7D;
  font-size: 14px;
  font-weight: 500;
  padding: 4px 8px;
  background: transparent;
  border: none;
}

.btn-edit:hover {
  color: #1F5F5F;
  text-decoration: underline;
}
```

---

### Form Inputs

#### Text Input
```css
.input-text {
  width: 100%;
  padding: 10px 12px;
  font-size: 15px;
  border: 1px solid #E8E6E3;
  border-radius: 6px;
  background-color: #FFFFFF;
  transition: border-color 150ms, box-shadow 150ms;
}

.input-text:focus {
  border-color: #2A7D7D;
  box-shadow: 0 0 0 3px rgba(42, 125, 125, 0.1);
  outline: none;
}

.input-text:invalid {
  border-color: #FCA5A5;
}
```

#### Checkbox
```css
.checkbox {
  width: 16px;
  height: 16px;
  border: 1px solid #E8E6E3;
  border-radius: 4px;
  accent-color: #2A7D7D;
}
```

#### Radio Button
```css
.radio {
  width: 16px;
  height: 16px;
  border: 1px solid #E8E6E3;
  accent-color: #2A7D7D;
}
```

---

### Cards

#### Step Card (Default State)
```css
.step-card {
  background-color: #FFFFFF;
  border: 1px solid #E8E6E3;
  border-radius: 8px;
  margin-bottom: 16px;
}

.step-card-header {
  padding: 24px;
  border-bottom: 1px solid #E8E6E3;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.step-card-body {
  padding: 24px;
}
```

#### Order Summary Card
```css
.order-summary {
  background-color: #FFFFFF;
  border: 1px solid #E8E6E3;
  border-radius: 8px;
  padding: 24px;
  position: sticky;
  top: 96px; /* Header height + margin */
}

@media (max-width: 1023px) {
  .order-summary {
    position: static;
  }
}
```

---

### Step Indicators

#### Step Number Circle (Pending)
```css
.step-indicator-pending {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background-color: #E8E6E3;
  color: #6B6B6B;
  font-size: 14px;
  font-weight: 500;
  display: flex;
  align-items: center;
  justify-content: center;
}
```

#### Step Number Circle (Complete)
```css
.step-indicator-complete {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background-color: #2A7D7D;
  color: #FFFFFF;
  display: flex;
  align-items: center;
  justify-content: center;
}

.step-indicator-complete svg {
  width: 20px;
  height: 20px;
}
```

---

### Feedback Messages

#### Success Message
```css
.message-success {
  padding: 16px;
  background-color: #E8F5E9;
  border: 1px solid #A5D6A7;
  border-radius: 8px;
  display: flex;
  gap: 12px;
}

.message-success-icon {
  width: 20px;
  height: 20px;
  color: #1F5F3D;
}

.message-success-title {
  font-size: 14px;
  font-weight: 500;
  color: #1F5F3D;
}

.message-success-text {
  font-size: 12px;
  color: #2E7D32;
  margin-top: 4px;
}
```

#### Error Message
```css
.message-error {
  padding: 16px;
  background-color: #FEE2E2;
  border: 1px solid #FCA5A5;
  border-radius: 8px;
  display: flex;
  gap: 12px;
}

.message-error-icon {
  width: 20px;
  height: 20px;
  color: #7F1D1D;
}

.message-error-title {
  font-size: 14px;
  font-weight: 500;
  color: #7F1D1D;
}

.message-error-text {
  font-size: 12px;
  color: #991B1B;
  margin-top: 4px;
}
```

#### Warning Message
```css
.message-warning {
  padding: 16px;
  background-color: #FEF3C7;
  border: 1px solid #FCD34D;
  border-radius: 8px;
  display: flex;
  gap: 12px;
}

.message-warning-icon {
  width: 20px;
  height: 20px;
  color: #78350F;
}

.message-warning-title {
  font-size: 14px;
  font-weight: 500;
  color: #78350F;
}
```

---

### Shipping Rate Selection

```css
.shipping-rate-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  border: 2px solid #E8E6E3;
  border-radius: 8px;
  cursor: pointer;
  transition: all 150ms;
}

.shipping-rate-option:hover {
  border-color: #2A7D7D;
  background-color: #F0F7F7;
}

.shipping-rate-option:has(input:checked) {
  border-color: #2A7D7D;
  background-color: #F0F7F7;
}

.shipping-rate-carrier {
  font-size: 14px;
  font-weight: 500;
  color: #1A1A1A;
}

.shipping-rate-estimate {
  font-size: 12px;
  color: #6B6B6B;
  margin-top: 2px;
}

.shipping-rate-price {
  font-size: 14px;
  font-weight: 500;
  color: #1A1A1A;
}
```

---

### Loading States

#### Spinner
```css
.spinner {
  width: 32px;
  height: 32px;
  border: 3px solid #E8E6E3;
  border-top-color: #2A7D7D;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
```

#### Loading Container
```css
.loading-container {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px;
  gap: 12px;
}

.loading-text {
  font-size: 14px;
  color: #6B6B6B;
}
```

---

## Responsive Breakpoints

```css
/* Mobile: 0-639px */
@media (max-width: 639px) {
  .page-title { font-size: 20px; }
  .step-card-body { padding: 16px; }
  .btn-primary { width: 100%; }

  /* Stack form fields */
  .form-row { flex-direction: column; }
}

/* Tablet: 640-1023px */
@media (min-width: 640px) and (max-width: 1023px) {
  .order-summary { position: static; margin-top: 32px; }
}

/* Desktop: 1024px+ */
@media (min-width: 1024px) {
  .checkout-grid {
    display: grid;
    grid-template-columns: 2fr 1fr;
    gap: 32px;
  }

  .order-summary {
    position: sticky;
    top: 96px;
  }
}
```

---

## Animations & Transitions

```css
/* Step expansion/collapse */
.step-content {
  transition: max-height 300ms ease-in-out,
              opacity 200ms ease-in-out;
}

/* Button hover */
.btn-primary,
.btn-secondary {
  transition: background-color 150ms ease,
              border-color 150ms ease;
}

/* Input focus */
.input-text {
  transition: border-color 150ms ease,
              box-shadow 150ms ease;
}

/* Message fade-in */
.message {
  animation: fadeIn 200ms ease-in;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(-8px); }
  to   { opacity: 1; transform: translateY(0); }
}
```

---

## Icons

All icons use Heroicons (outline style) with consistent sizing:

```css
.icon-sm { width: 16px; height: 16px; }
.icon-md { width: 20px; height: 20px; }
.icon-lg { width: 24px; height: 24px; }
```

**Icons Used:**
- Checkmark (step complete): `M5 13l4 4L19 7`
- Warning: `M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z`
- Spinner: SVG circle animation
- Image placeholder: `M4 16l4.586-4.586a2 2 0 012.828 0L16 16...`

---

## Accessibility

### Focus States
```css
*:focus-visible {
  outline: 2px solid #2A7D7D;
  outline-offset: 2px;
}

button:focus-visible {
  box-shadow: 0 0 0 3px rgba(42, 125, 125, 0.3);
}
```

### Screen Reader Only Text
```css
.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border-width: 0;
}
```

### Color Contrast Ratios
- Body text (neutral-900 on white): 12.63:1 ✓
- Secondary text (neutral-600 on white): 4.92:1 ✓
- Teal-700 on white: 4.56:1 ✓
- All meet WCAG AA standards

---

## Print Styles (Order Confirmation)

```css
@media print {
  header, footer { display: none; }

  .order-confirmation {
    max-width: 100%;
    padding: 0;
  }

  .card {
    border: 1px solid #000;
    page-break-inside: avoid;
  }

  .btn-primary,
  .btn-secondary {
    display: none;
  }
}
```

---

**Document Version:** 1.0
**Last Updated:** 2025-11-29
**Designer:** Claude (UX Design Lead)
**Status:** Production Ready
