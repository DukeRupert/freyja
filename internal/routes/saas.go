package routes

import (
	"net/http"

	"github.com/dukerupert/hiri/internal/router"
)

// RegisterSaaSRoutes registers all SaaS marketing routes.
// These routes are for the Hiri platform marketing site itself,
// not for tenant storefronts.
//
// When ready to split, these routes can be served on a separate
// port or domain (e.g., hiri.coffee vs app.hiri.coffee).
func RegisterSaaSRoutes(r *router.Router, deps SaaSDeps) {
	// Marketing pages
	r.Get("/", deps.Handler.Landing())
	r.Get("/pricing", deps.Handler.Pricing(deps.CheckoutURL))
	r.Get("/about", deps.Handler.About())
	r.Get("/contact", deps.Handler.Contact())
	r.Get("/privacy", deps.Handler.Privacy())
	r.Get("/terms", deps.Handler.Terms())

	// Redirect /signup to /pricing (the signup flow starts from pricing page checkout)
	r.Get("/signup", redirectToPricing())

	// Checkout API (creates Stripe checkout session)
	if deps.CheckoutHandler != nil {
		r.Post("/api/saas/checkout", deps.CheckoutHandler.HandleCreateCheckoutSession)
		r.Options("/api/saas/checkout", deps.CheckoutHandler.HandleCheckoutOptions)
	}

	// Auth routes (login, logout, password reset)
	if deps.AuthHandler != nil {
		r.Get("/login", deps.AuthHandler.ShowLoginForm)
		r.Post("/login", deps.AuthHandler.HandleLogin)
		r.Post("/logout", deps.AuthHandler.HandleLogout)
		r.Get("/forgot-password", deps.AuthHandler.ShowForgotPasswordForm)
		r.Post("/forgot-password", deps.AuthHandler.HandleForgotPassword)
		r.Get("/reset-password", deps.AuthHandler.ShowResetPasswordForm)
		r.Post("/reset-password", deps.AuthHandler.HandleResetPassword)
	}

	// Setup routes (account setup after checkout)
	if deps.SetupHandler != nil {
		r.Get("/setup", deps.SetupHandler.ShowSetupForm)
		r.Post("/setup", deps.SetupHandler.HandleSetup)
		r.Get("/resend-setup", deps.SetupHandler.ShowResendSetupForm)
		r.Post("/resend-setup", deps.SetupHandler.HandleResendSetup)

		// Success page after Stripe checkout (before email setup)
		r.Get("/setup/success", setupSuccessHandler())
	}

	// Stripe webhook for SaaS subscription events
	if deps.WebhookHandler != nil {
		r.Post("/webhooks/stripe/saas", deps.WebhookHandler.HandleStripeWebhook)
	}
}

// redirectToPricing returns a handler that redirects to the pricing page.
// This provides a clean /signup URL for marketing while the actual
// signup flow starts from the Stripe checkout on the pricing page.
func redirectToPricing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/pricing", http.StatusSeeOther)
	}
}

// setupSuccessHandler shows a success message after Stripe checkout.
// The user will receive an email with the actual setup link.
func setupSuccessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// For now, return a simple HTML page
		// TODO: Create a proper template for this page
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Welcome to Hiri</title>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-neutral-50 min-h-screen flex items-center justify-center">
    <div class="max-w-md mx-auto px-4 text-center">
        <div class="bg-white rounded-2xl shadow-sm ring-1 ring-neutral-900/5 p-8">
            <div class="mx-auto w-12 h-12 bg-teal-100 rounded-full flex items-center justify-center mb-4">
                <svg class="w-6 h-6 text-teal-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
                </svg>
            </div>
            <h1 class="text-2xl font-semibold text-neutral-900 mb-2">Payment successful</h1>
            <p class="text-neutral-600 mb-6">
                Check your email for a link to set up your account. The link will expire in 48 hours.
            </p>
            <p class="text-sm text-neutral-500">
                Didn't receive the email? Check your spam folder or <a href="/resend-setup" class="text-teal-600 hover:text-teal-700">request a new link</a>.
            </p>
        </div>
    </div>
</body>
</html>`))
	}
}
