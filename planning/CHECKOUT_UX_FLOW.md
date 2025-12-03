# Checkout Flow - UX Design Documentation

## Overview

The Freyja checkout experience is designed around **progressive disclosure**, **inline validation**, and **confidence-inspiring clarity**. Users progress through five distinct steps, with each step collapsing upon completion to maintain context while reducing visual noise.

## Design Philosophy

**Guiding Principles:**
- **One thing at a time**: Show only the current step to reduce cognitive load
- **Always show progress**: Visual indicators show completion and current position
- **Enable correction**: Completed steps remain editable via "Edit" buttons
- **Validate early**: Catch errors immediately with inline feedback
- **Inspire confidence**: Clear CTAs, secure payment indicators, transparent pricing

**Voice & Tone:**
- Direct, concise copy ("Continue to Shipping" not "Next Step")
- Calm confidence ("Address validated" not "Great! Your address looks good!")
- Helpful errors ("Address validation failed. Please check your address" with specific guidance)
- Security reassurance ("Your payment information is securely processed by Stripe")

## User Journey Map

### Entry Point
**From:** Cart page → "Proceed to Checkout" button
**Context:** User has items in cart, ready to complete purchase
**Goal:** Complete purchase quickly and confidently

### Step 1: Contact Information
**What:** Email address (required) + Phone number (optional)
**Why First:** Captures abandonment recovery info early
**Validation:** Email format validation on blur
**Success State:** Green checkmark in step indicator, step collapses showing email/phone
**Error Handling:** Inline error below field with red text

**User Flow:**
```
1. User enters email address
2. Field validates on blur (format check)
3. User enters phone (optional)
4. User clicks "Continue to Shipping"
5. Step collapses, shows summary, advances to Step 2
```

**Collapsed State Shows:**
- Email address (full)
- Phone number or "No phone provided"
- "Edit" button (top-right)

---

### Step 2: Shipping Address
**What:** Full shipping address form
**Why:** Required for shipping rate calculation
**Validation:** Real-time validation via `/api/checkout/validate-addresses`
**Success State:** Green success message with normalized address
**Error Handling:** Red error message with specific issue and retry guidance

**User Flow:**
```
1. User fills out address form
   - Full name
   - Street address
   - Apt/suite (optional)
   - City, State, ZIP
2. User clicks "Validate Address"
3. htmx POST to /api/checkout/validate-addresses
4. On success:
   - Green success message appears
   - "Continue to Shipping Method" button appears
   - Step advances to 3
5. On error:
   - Red error message with specific issue
   - User corrects and retries
```

**Validation Logic:**
- Address format validation (USPS API)
- Address normalization (correct formatting)
- Deliverability check

**Collapsed State Shows:**
- Full name
- Street address
- Apt/suite (if provided)
- City, State ZIP
- "Edit" button

---

### Step 3: Shipping Method
**What:** Radio button selection of shipping options
**Why:** User chooses speed vs. cost tradeoff
**Auto-load:** Fetches rates automatically when step loads
**Success State:** Selected rate highlighted with teal border/background
**Error Handling:** Amber warning if no rates available with contact prompt

**User Flow:**
```
1. Step loads, triggers htmx GET to /api/checkout/shipping-rates
2. Loading spinner shows "Loading shipping options..."
3. Rates injected into DOM as radio buttons
4. User selects preferred rate
   - JavaScript updates order summary (shipping cost)
   - Triggers htmx POST to /api/checkout/calculate-total
   - Tax calculated and displayed
   - Order total updated
5. "Continue to Billing" button enables
6. User clicks continue
7. Step collapses, advances to Step 4
```

**Rate Display Format:**
```
┌─────────────────────────────────────────┐
│ ○ USPS Priority Mail          $8.95    │
│   2-3 business days                     │
└─────────────────────────────────────────┘
```

**Collapsed State Shows:**
- Carrier + Service (e.g., "USPS Priority Mail")
- Estimated delivery + Cost
- "Edit" button

---

### Step 4: Billing Address
**What:** Billing address form (conditional)
**Default:** "Same as shipping address" checkbox (checked)
**Conditional Display:** Form only shows if checkbox unchecked
**Success State:** Immediate advancement if same as shipping

**User Flow (Same as Shipping):**
```
1. Step loads with checkbox checked
2. "Continue to Payment" button visible
3. User clicks continue
4. Step collapses showing "Same as shipping address"
5. Advances to Step 5
```

**User Flow (Different Address):**
```
1. User unchecks "Same as shipping address"
2. Billing address form appears
3. User fills out form (same fields as shipping)
4. User clicks "Continue to Payment"
5. Step collapses showing billing address
6. Advances to Step 5
```

**Collapsed State Shows:**
- If same: "Same as shipping address"
- If different: Full billing address
- "Edit" button

---

### Step 5: Payment
**What:** Stripe Payment Element + Submit button
**Auto-load:** Creates payment intent when step loads
**Security:** Stripe.js handles sensitive data
**Success:** Redirects to order confirmation
**Error Handling:** Inline error message with retry

**User Flow:**
```
1. Step loads
2. htmx POST to /api/checkout/create-payment-intent
3. Loading spinner shows "Preparing payment..."
4. Stripe Payment Element mounts
5. User enters payment details (handled by Stripe)
6. User clicks "Place Order"
7. Button shows "Processing..." (disabled)
8. Stripe confirms payment
9. On success: Redirect to /checkout/complete → /order-confirmation
10. On error: Show error message, re-enable button
```

**Payment Element Configuration:**
- Theme: Stripe default with Freyja color overrides
- Primary color: Teal (#2A7D7D)
- Border radius: 6px
- Font: System sans-serif

**Security Messaging:**
- Below submit button: "Your payment information is securely processed by Stripe"
- Reassures without being verbose

---

## Order Summary Sidebar

**Desktop (≥1024px):**
- Fixed width (1/3 of layout)
- Sticky positioning (stays visible on scroll)
- Always visible

**Mobile (<1024px):**
- Collapsible with toggle button
- Shows total in collapsed state
- Expands to full summary on tap

**Contents:**
1. Cart items with thumbnails, quantities, prices
2. Subtotal
3. Shipping (updates after Step 3)
4. Tax (updates after Step 3)
5. Total (bold, prominent)

**Dynamic Updates:**
- Shipping cost updates when rate selected (Step 3)
- Tax updates when total calculated (Step 3)
- Total recalculates automatically

---

## Visual Progress Indicators

**Step Indicators:**
- Pending: Gray circle with number
- Active: Gray circle with number
- Complete: Teal circle with white checkmark

**Step State:**
- Active: Full form visible
- Complete: Collapsed summary with "Edit" button
- Future: Not visible until previous step complete

**Edit Capability:**
- Any completed step can be reopened
- Click "Edit" button (top-right of collapsed step)
- Step expands, user makes changes
- Must re-complete to advance

---

## Loading States

**Address Validation:**
- Button shows "Validating..." (disabled) during API call
- Result appears inline (success or error)

**Shipping Rates:**
- Spinner with message: "Loading shipping options..."
- Uses htmx indicator pattern

**Total Calculation:**
- Occurs automatically in background
- No explicit loading state (fast operation)
- Updates DOM when complete

**Payment Intent:**
- Spinner with message: "Preparing payment..."
- Stripe Element mounts when ready

**Payment Submission:**
- Button text changes to "Processing..." (disabled)
- Remains disabled until success or error

---

## Error Handling

### Validation Errors (Step 2)
**Appearance:** Red background, red border, error icon
**Content:** Specific issue + guidance
**Example:**
```
⚠ Address validation failed
  The postal code format is invalid. Please check and try again.
```

### No Shipping Rates (Step 3)
**Appearance:** Amber background, amber border, warning icon
**Content:** Explanation + contact prompt
**Example:**
```
⚠ No shipping rates available
  We couldn't find shipping options for this address.
  Please contact us for assistance.
```

### Payment Errors (Step 5)
**Appearance:** Red background, red border, error icon
**Content:** Error message from Stripe + retry option
**Example:**
```
⚠ Payment failed
  Your card was declined. Please try a different payment method.
```

**Retry Flow:**
- Error message appears inline
- Submit button re-enables
- User can edit payment info
- Click "Place Order" to retry

---

## Accessibility Considerations

**Keyboard Navigation:**
- All form fields accessible via Tab
- Radio buttons selectable with arrow keys
- Submit buttons triggered with Enter/Space

**Screen Reader Support:**
- Labels associated with inputs (for/id)
- Error messages announced (aria-live)
- Step indicators use semantic HTML
- Stripe Payment Element inherently accessible

**Color Contrast:**
- All text meets WCAG AA standards
- Error messages use icon + text (not color alone)
- Focus states have visible ring

**Touch Targets (Mobile):**
- Minimum 44px height for all interactive elements
- Radio button labels expand touch area
- Buttons full-width on mobile

---

## Mobile Optimization

**Layout Changes:**
- Single column (no sidebar)
- Order summary collapsible at top
- Buttons full-width
- Larger form inputs for touch

**Responsive Breakpoints:**
- Mobile: <640px (single column, collapsible summary)
- Tablet: 640-1024px (single column, sticky summary)
- Desktop: >1024px (two column, sticky sidebar)

**Form Adaptations:**
- City/State/ZIP stacked on narrow screens
- Larger input padding (12px vs 10px)
- Ample spacing between fields

---

## Performance Considerations

**Progressive Enhancement:**
- Works without JavaScript (server-rendered)
- htmx provides dynamic updates
- Alpine.js manages local UI state
- Graceful degradation if scripts fail

**API Call Optimization:**
- Address validation: On blur (not on every keystroke)
- Shipping rates: Once per valid address
- Total calculation: Once per shipping selection
- Payment intent: Once when step loads

**Perceived Performance:**
- Loading spinners for async operations
- Optimistic UI updates where safe
- Skeleton states for content injection

---

## Success Metrics

**Conversion Goals:**
- Complete checkout in <3 minutes
- <5% abandonment after Step 2
- <2% validation errors per session

**UX Quality Indicators:**
- Zero confusion about next action
- No unexpected behavior
- Clear error recovery paths
- Confidence-inspiring payment flow

---

## Technical Implementation Notes

**Template Files:**
1. `/web/templates/storefront/checkout.html` - Main checkout page
2. `/web/templates/storefront/order-confirmation.html` - Success page
3. `/web/templates/storefront/checkout_partials.html` - htmx responses

**API Endpoints:**
- `POST /api/checkout/validate-addresses` - Address validation
- `POST /api/checkout/shipping-rates` - Get shipping options
- `POST /api/checkout/calculate-total` - Calculate order total with tax
- `POST /api/checkout/create-payment-intent` - Initialize Stripe payment

**JavaScript Dependencies:**
- htmx 1.9.10 (dynamic content)
- Alpine.js 3.x (state management)
- Stripe.js v3 (payment processing)

**CSS Framework:**
- Tailwind CSS with custom Freyja theme
- Custom components: `.btn-primary`, `.input-text`, `.card`

---

## Future Enhancements (Post-MVP)

1. **Guest Checkout:** Allow checkout without account creation
2. **Address Autocomplete:** Google Places API integration
3. **Saved Addresses:** Quick select for returning customers
4. **Shipping Insurance:** Optional add-on in Step 3
5. **Gift Options:** Gift message and wrap in cart
6. **Apple Pay / Google Pay:** Express checkout buttons
7. **Order Notes:** Special instructions field
8. **Estimated Delivery Dates:** Calendar-based selection

---

## Questions Resolved During Design

**Q: Should we show all steps at once or progressive disclosure?**
**A:** Progressive disclosure. Reduces overwhelm, maintains focus, industry best practice.

**Q: When should address validation occur?**
**A:** On explicit "Validate Address" button click. Gives user control, clear validation state.

**Q: Should billing address be required or optional?**
**A:** Required, but defaults to "Same as shipping". Simplifies most common case.

**Q: Where should order summary live on mobile?**
**A:** Collapsible at top. Keeps it accessible without consuming vertical space.

**Q: How should we handle shipping rate errors?**
**A:** Amber warning (not red error). It's not user's fault, provide contact option.

**Q: Should payment step auto-advance after other steps?**
**A:** No. Payment is final, requires explicit user action. No auto-advancement.

---

**Document Version:** 1.0
**Last Updated:** 2025-11-29
**Author:** UX Design Lead (Claude)
**Status:** Production Ready
