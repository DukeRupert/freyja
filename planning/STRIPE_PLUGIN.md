Architecture Overview

  The skeleton follows Freyja's established plugin patterns with a phased implementation approach:

  Core Structure:
  - internal/billing/stripe_config.go - Configuration and validation
  - internal/billing/errors.go - Billing-specific error types
  - internal/billing/stripe.go - Stripe provider implementation (functions signatures only)
  - internal/billing/stripe_tax.go - Stripe Tax calculator adapter
  - internal/billing/mock.go - Mock provider for testing

  Key Design Decisions

  1. Phased Implementation (Recommended Option C):
  - MVP methods with full TODOs: CreatePaymentIntent, GetPaymentIntent, UpdatePaymentIntent, CancelPaymentIntent, VerifyWebhookSignature
  - Post-MVP methods returning ErrNotImplemented: Customer management, Subscriptions, Refunds
  - Better than panics, clearer than omitting methods entirely

  2. Payment Flow Integration:
  Checkout → CreatePaymentIntent → Frontend (Stripe.js)
    → Webhook (async) → Complete endpoint → GetPaymentIntent
    → Verify succeeded → Create order

  3. Notable Features:
  - Comprehensive error types with Stripe-specific context
  - Idempotency support via cart_id
  - Stripe Tax integration option
  - Multi-tenant metadata requirements
  - Mock provider for testing without API calls

  4. Configuration:
  - API key validation and test mode detection
  - Webhook secret for signature verification
  - Optional Stripe Tax enablement
  - Retry and timeout configuration

  The design includes 8 open questions about multi-tenancy, idempotency keys, tax calculation timing, and webhook setup that should be validated during implementation.
