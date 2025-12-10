# Marketing Site Architecture

This document describes the architecture for hosting the Freyja marketing site at `hiri.coffee` alongside the application at `app.hiri.coffee`.

**Last updated:** December 9, 2024

## Current Implementation Status

The marketing site infrastructure is implemented and ready for deployment:

- [x] Domain configuration in `internal/config.go`
- [x] Host-based routing in `cmd/server/main.go`
- [x] Marketing page handler in `internal/handler/saas/landing.go`
- [x] Marketing routes in `internal/routes/saas.go`
- [x] Templates: landing, pricing, about, contact, privacy, terms
- [ ] SaaS checkout API endpoint (placeholder - needs Stripe setup)

---

## Domain Structure

| Domain | Purpose | Content |
|--------|---------|---------|
| `hiri.coffee` | Marketing site | Pricing, features, about, contact |
| `www.hiri.coffee` | Redirect | Redirects to `hiri.coffee` |
| `app.hiri.coffee` | Application | Admin dashboard, storefront, APIs |

This follows industry patterns (Stripe, SendGrid, YNAB use similar structures).

**Benefits:**
- **SEO:** Root domain accumulates authority for content marketing
- **Decoupling:** Marketing and app can evolve independently
- **User expectations:** B2B SaaS users expect this pattern
- **Deployment flexibility:** Can separate later if needed

---

## Signup Flow Integration

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  hiri.coffee    │     │     Stripe      │     │ app.hiri.coffee │
│  /pricing       │     │    Checkout     │     │                 │
└────────┬────────┘     └────────┬────────┘     └────────┬────────┘
         │                       │                       │
         │ 1. Click "Start"      │                       │
         │──────────────────────>│                       │
         │                       │                       │
         │ 2. POST /api/checkout/create                  │
         │──────────────────────────────────────────────>│
         │                       │                       │
         │ 3. Return Checkout URL│                       │
         │<──────────────────────────────────────────────│
         │                       │                       │
         │ 4. Redirect to Stripe │                       │
         │──────────────────────>│                       │
         │                       │                       │
         │                       │ 5. Payment succeeds   │
         │                       │──────────────────────>│
         │                       │    (redirect)         │
         │                       │                       │
         │                       │ 6. Webhook fires      │
         │                       │──────────────────────>│
```

The checkout API endpoint lives on `app.hiri.coffee` and is called cross-origin from the marketing site.

---

## Technical Implementation

### Host-Based Routing (Go 1.22+)

Go 1.22's enhanced `ServeMux` supports host-specific patterns:

```go
mux := http.NewServeMux()

// Marketing site routes (hiri.coffee)
mux.HandleFunc("hiri.coffee/", h.marketing.Home)
mux.HandleFunc("hiri.coffee/pricing", h.marketing.Pricing)
mux.HandleFunc("hiri.coffee/features", h.marketing.Features)
mux.HandleFunc("hiri.coffee/about", h.marketing.About)
mux.HandleFunc("hiri.coffee/contact", h.marketing.Contact)
mux.HandleFunc("www.hiri.coffee/", redirectToRoot)

// Application routes (app.hiri.coffee)
mux.HandleFunc("app.hiri.coffee/", h.storefront.Home)
mux.HandleFunc("app.hiri.coffee/admin/", h.admin.Dashboard)
mux.HandleFunc("app.hiri.coffee/api/checkout/create", h.saas.CreateCheckoutSession)
mux.HandleFunc("app.hiri.coffee/webhooks/stripe/saas", h.saas.HandleWebhook)
```

**Key points:**
- Host patterns take precedence over path-only patterns
- `www.hiri.coffee` redirects to `hiri.coffee` (canonical URL)
- Single binary serves both domains

### Environment Configuration

```bash
# Domain configuration (production)
HOST_ROUTING_ENABLED=true
MARKETING_DOMAIN=hiri.coffee
APP_DOMAIN=app.hiri.coffee
BASE_URL=https://app.hiri.coffee

# Stripe SaaS configuration
STRIPE_SAAS_PRICE_ID=price_xxx           # $149/month price
STRIPE_SAAS_WEBHOOK_SECRET=whsec_xxx     # Separate from tenant webhook
STRIPE_LAUNCH_COUPON_ID=freyja-launch    # Optional introductory discount
```

**Development mode** (default when `HOST_ROUTING_ENABLED` is not set):
- Marketing site runs on port 3001
- App runs on configured PORT (default 3000)
- No host-based routing required

### Caddy Configuration

```caddyfile
# Marketing site
hiri.coffee {
    reverse_proxy localhost:8080

    # Cache static assets
    @static path /static/*
    header @static Cache-Control "public, max-age=31536000"
}

# Redirect www to root
www.hiri.coffee {
    redir https://hiri.coffee{uri} permanent
}

# Application
app.hiri.coffee {
    reverse_proxy localhost:8080
}
```

---

## File Structure (Actual Implementation)

```
internal/
├── handler/
│   ├── saas/                   # Marketing + SaaS onboarding handlers
│   │   ├── landing.go          # PageHandler for marketing pages
│   │   ├── setup.go            # GET/POST /saas/setup
│   │   ├── auth.go             # SaaS auth handlers
│   │   ├── billing.go          # Billing portal
│   │   └── webhook.go          # POST /webhooks/stripe/saas
│   ├── admin/                  # Admin dashboard
│   └── storefront/             # Customer storefront
├── routes/
│   ├── saas.go                 # Marketing site route registration
│   ├── deps.go                 # SaaSDeps includes CheckoutURL
│   └── ...                     # Other route files
└── config.go                   # DomainConfig struct

web/templates/
├── saas/                       # Marketing + SaaS templates
│   ├── layout.html             # Marketing layout
│   ├── landing.html            # Home page
│   ├── pricing.html            # Pricing with checkout button
│   ├── about.html
│   ├── contact.html
│   ├── privacy.html
│   └── terms.html
├── admin/                      # Admin templates
└── storefront/                 # Storefront templates
```

---

## Marketing Handler Implementation

### Handler Structure

```go
// internal/handler/marketing/handler.go
package marketing

import (
    "html/template"
    "log/slog"
    "net/http"

    "github.com/dukerupert/hiri/internal"
)

type Handler struct {
    templates *template.Template
    config    *internal.Config
    logger    *slog.Logger
}

type HandlerConfig struct {
    Config *internal.Config
    Logger *slog.Logger
}

func NewHandler(cfg HandlerConfig) (*Handler, error) {
    // Parse marketing templates
    tmpl, err := template.ParseGlob("web/templates/marketing/**/*.html")
    if err != nil {
        return nil, err
    }

    return &Handler{
        templates: tmpl,
        config:    cfg.Config,
        logger:    cfg.Logger,
    }, nil
}
```

### Page Data Structure

```go
// Common data for all marketing pages
type PageData struct {
    Title       string
    Description string
    CanonicalURL string
    AppDomain   string  // For checkout API calls
    CurrentPath string
}

func (h *Handler) basePageData(r *http.Request, title, description string) PageData {
    return PageData{
        Title:        title,
        Description:  description,
        CanonicalURL: "https://hiri.coffee" + r.URL.Path,
        AppDomain:    h.config.BaseURL, // e.g., https://app.hiri.coffee
        CurrentPath:  r.URL.Path,
    }
}
```

### Home Page Handler

```go
// internal/handler/marketing/home.go
package marketing

import "net/http"

func (h *Handler) Home(w http.ResponseWriter, r *http.Request) {
    // Catch-all for marketing domain - only serve home on exact "/"
    if r.URL.Path != "/" {
        http.NotFound(w, r)
        return
    }

    data := h.basePageData(r,
        "Freyja - E-commerce for Coffee Roasters",
        "Built for small coffee roasters. B2C + B2B in one platform. $149/month.",
    )

    if err := h.templates.ExecuteTemplate(w, "home.html", data); err != nil {
        h.logger.Error("failed to render home page", "error", err)
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
    }
}
```

### Pricing Page Handler

```go
// internal/handler/marketing/pricing.go
package marketing

import "net/http"

type PricingData struct {
    PageData
    MonthlyPrice    int    // 149
    IntroPrice      int    // 5
    IntroMonths     int    // 3
    CheckoutAPIURL  string // https://app.hiri.coffee/api/checkout/create
}

func (h *Handler) Pricing(w http.ResponseWriter, r *http.Request) {
    data := PricingData{
        PageData: h.basePageData(r,
            "Pricing - Freyja",
            "Simple pricing. $149/month flat fee. No transaction fees.",
        ),
        MonthlyPrice:   149,
        IntroPrice:     5,
        IntroMonths:    3,
        CheckoutAPIURL: h.config.BaseURL + "/api/checkout/create",
    }

    if err := h.templates.ExecuteTemplate(w, "pricing.html", data); err != nil {
        h.logger.Error("failed to render pricing page", "error", err)
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
    }
}
```

---

## Route Registration

### Marketing Routes

```go
// internal/routes/marketing.go
package routes

import (
    "net/http"

    "github.com/dukerupert/hiri/internal/handler/marketing"
)

func RegisterMarketingRoutes(mux *http.ServeMux, h *marketing.Handler, domain string) {
    // Home page
    mux.HandleFunc(domain+"/", h.Home)

    // Static pages
    mux.HandleFunc(domain+"/pricing", h.Pricing)
    mux.HandleFunc(domain+"/features", h.Features)
    mux.HandleFunc(domain+"/about", h.About)
    mux.HandleFunc(domain+"/contact", h.Contact)

    // www redirect
    mux.HandleFunc("www."+domain+"/", func(w http.ResponseWriter, r *http.Request) {
        target := "https://" + domain + r.URL.Path
        if r.URL.RawQuery != "" {
            target += "?" + r.URL.RawQuery
        }
        http.Redirect(w, r, target, http.StatusMovedPermanently)
    })
}
```

### Main Router Integration

```go
// In cmd/server/main.go or internal/routes/router.go

func setupRoutes(cfg *internal.Config, services *Services) *http.ServeMux {
    mux := http.NewServeMux()

    // Marketing site (hiri.coffee)
    marketingHandler, _ := marketing.NewHandler(marketing.HandlerConfig{
        Config: cfg,
        Logger: services.Logger,
    })
    routes.RegisterMarketingRoutes(mux, marketingHandler, cfg.MarketingDomain)

    // Application (app.hiri.coffee)
    appDomain := cfg.AppDomain

    // SaaS checkout API (called from marketing site)
    mux.HandleFunc(appDomain+"/api/checkout/create", services.SaaS.CreateCheckoutSession)

    // SaaS webhooks
    mux.HandleFunc(appDomain+"/webhooks/stripe/saas", services.SaaS.HandleWebhook)

    // Admin routes
    routes.RegisterAdminRoutes(mux, services.Admin, appDomain)

    // Storefront routes
    routes.RegisterStorefrontRoutes(mux, services.Storefront, appDomain)

    // Static files (served from both domains)
    mux.Handle("/static/", http.StripPrefix("/static/",
        http.FileServer(http.Dir("web/static"))))

    return mux
}
```

---

## Template Structure

### Marketing Layout

```html
<!-- web/templates/marketing/layouts/base.html -->
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <meta name="description" content="{{.Description}}">
    <link rel="canonical" href="{{.CanonicalURL}}">

    <!-- Open Graph -->
    <meta property="og:title" content="{{.Title}}">
    <meta property="og:description" content="{{.Description}}">
    <meta property="og:type" content="website">
    <meta property="og:url" content="{{.CanonicalURL}}">

    <!-- Styles -->
    <link rel="stylesheet" href="/static/css/marketing.css">
</head>
<body class="bg-neutral-50 text-neutral-900">
    {{template "header" .}}

    <main>
        {{template "content" .}}
    </main>

    {{template "footer" .}}

    <!-- Scripts -->
    <script src="/static/js/marketing.js"></script>
</body>
</html>
```

### Marketing Header

```html
<!-- web/templates/marketing/partials/header.html -->
{{define "header"}}
<header class="bg-white border-b border-neutral-200">
    <nav class="max-w-6xl mx-auto px-4 py-4 flex items-center justify-between">
        <a href="/" class="text-xl font-semibold text-neutral-900">
            Freyja
        </a>

        <div class="hidden md:flex items-center space-x-8">
            <a href="/features" class="text-neutral-600 hover:text-neutral-900 {{if eq .CurrentPath "/features"}}text-neutral-900{{end}}">
                Features
            </a>
            <a href="/pricing" class="text-neutral-600 hover:text-neutral-900 {{if eq .CurrentPath "/pricing"}}text-neutral-900{{end}}">
                Pricing
            </a>
            <a href="/about" class="text-neutral-600 hover:text-neutral-900 {{if eq .CurrentPath "/about"}}text-neutral-900{{end}}">
                About
            </a>
        </div>

        <div class="flex items-center space-x-4">
            <a href="{{.AppDomain}}/login" class="text-neutral-600 hover:text-neutral-900">
                Log in
            </a>
            <a href="/pricing" class="px-4 py-2 bg-teal-700 text-white rounded-md hover:bg-teal-800">
                Get Started
            </a>
        </div>
    </nav>
</header>
{{end}}
```

### Pricing Page with Checkout

```html
<!-- web/templates/marketing/pages/pricing.html -->
{{define "content"}}
<section class="py-20">
    <div class="max-w-4xl mx-auto px-4 text-center">
        <h1 class="text-4xl font-bold mb-4">Simple, Transparent Pricing</h1>
        <p class="text-xl text-neutral-600 mb-12">
            One plan. Everything included. No transaction fees.
        </p>

        <div class="bg-white rounded-lg shadow-lg p-8 max-w-md mx-auto">
            <div class="mb-6">
                <span class="text-5xl font-bold">${{.MonthlyPrice}}</span>
                <span class="text-neutral-600">/month</span>
            </div>

            <div class="bg-amber-50 border border-amber-200 rounded-md p-4 mb-6">
                <p class="text-amber-800 font-medium">
                    Launch Special: ${{.IntroPrice}}/month for {{.IntroMonths}} months
                </p>
            </div>

            <ul class="text-left space-y-3 mb-8">
                <li class="flex items-center">
                    <svg class="w-5 h-5 text-teal-600 mr-2" fill="currentColor" viewBox="0 0 20 20">
                        <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"/>
                    </svg>
                    Unlimited products
                </li>
                <li class="flex items-center">
                    <svg class="w-5 h-5 text-teal-600 mr-2" fill="currentColor" viewBox="0 0 20 20">
                        <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"/>
                    </svg>
                    B2C + B2B sales
                </li>
                <li class="flex items-center">
                    <svg class="w-5 h-5 text-teal-600 mr-2" fill="currentColor" viewBox="0 0 20 20">
                        <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"/>
                    </svg>
                    Subscriptions included
                </li>
                <li class="flex items-center">
                    <svg class="w-5 h-5 text-teal-600 mr-2" fill="currentColor" viewBox="0 0 20 20">
                        <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"/>
                    </svg>
                    Wholesale invoicing
                </li>
                <li class="flex items-center">
                    <svg class="w-5 h-5 text-teal-600 mr-2" fill="currentColor" viewBox="0 0 20 20">
                        <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd"/>
                    </svg>
                    No transaction fees
                </li>
            </ul>

            <button
                id="checkout-btn"
                data-checkout-url="{{.CheckoutAPIURL}}"
                class="w-full py-3 bg-teal-700 text-white rounded-md hover:bg-teal-800 font-medium"
            >
                Start Your Store
            </button>

            <p class="text-sm text-neutral-500 mt-4">
                Cancel anytime. No contracts.
            </p>
        </div>
    </div>
</section>
{{end}}
```

### Checkout JavaScript

```javascript
// web/static/js/marketing.js

document.addEventListener('DOMContentLoaded', function() {
    const checkoutBtn = document.getElementById('checkout-btn');

    if (checkoutBtn) {
        checkoutBtn.addEventListener('click', async function() {
            const url = this.dataset.checkoutUrl;

            // Disable button during request
            this.disabled = true;
            this.textContent = 'Redirecting...';

            try {
                const response = await fetch(url, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    credentials: 'omit', // Cross-origin, no cookies needed
                });

                if (!response.ok) {
                    throw new Error('Failed to create checkout session');
                }

                const data = await response.json();

                // Redirect to Stripe Checkout
                window.location.href = data.url;
            } catch (error) {
                console.error('Checkout error:', error);
                this.disabled = false;
                this.textContent = 'Start Your Store';
                alert('Something went wrong. Please try again.');
            }
        });
    }
});
```

---

## CORS Configuration

The checkout API needs to accept cross-origin requests from the marketing site:

```go
// internal/middleware/cors.go
package middleware

import "net/http"

func CORSForCheckout(marketingDomain string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            origin := r.Header.Get("Origin")

            // Allow requests from marketing domain
            allowedOrigins := []string{
                "https://" + marketingDomain,
                "https://www." + marketingDomain,
            }

            for _, allowed := range allowedOrigins {
                if origin == allowed {
                    w.Header().Set("Access-Control-Allow-Origin", origin)
                    w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
                    w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
                    break
                }
            }

            // Handle preflight
            if r.Method == "OPTIONS" {
                w.WriteHeader(http.StatusNoContent)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

---

## Local Development

### /etc/hosts Configuration

```
# Add to /etc/hosts for local testing
127.0.0.1 hiri.coffee
127.0.0.1 www.hiri.coffee
127.0.0.1 app.hiri.coffee
```

### Development Environment

```bash
# .env for local development
MARKETING_DOMAIN=hiri.coffee
APP_DOMAIN=app.hiri.coffee
BASE_URL=http://app.hiri.coffee:8080

# Stripe test mode
STRIPE_SECRET_KEY=sk_test_xxx
STRIPE_SAAS_PRICE_ID=price_xxx
STRIPE_SAAS_WEBHOOK_SECRET=whsec_xxx
```

### Testing Flow

```bash
# Start server
go run cmd/server/main.go

# Test marketing site
curl http://hiri.coffee:8080/
curl http://hiri.coffee:8080/pricing

# Test app
curl http://app.hiri.coffee:8080/
curl http://app.hiri.coffee:8080/admin

# Test checkout API
curl -X POST http://app.hiri.coffee:8080/api/checkout/create

# Forward Stripe webhooks
stripe listen --forward-to http://app.hiri.coffee:8080/webhooks/stripe/saas
```

---

## Implementation Phases

### Phase 1: Infrastructure

1. Add `MARKETING_DOMAIN` and `APP_DOMAIN` to config
2. Update main router to use host-based patterns
3. Create marketing handler package structure
4. Test host routing locally

### Phase 2: Marketing Templates

1. Create marketing layout template
2. Create header/footer partials
3. Create home page
4. Create pricing page with checkout button
5. Create features page
6. Create about page

### Phase 3: Checkout Integration

1. Add CORS middleware for checkout endpoint
2. Implement checkout button JavaScript
3. Create checkout session API endpoint
4. Test full signup flow

### Phase 4: Deployment

1. Configure Caddy for both domains
2. Set up DNS records
3. Deploy and test production flow

---

## Security Considerations

- [ ] CORS only allows marketing domain origin
- [ ] Checkout endpoint rate limited
- [ ] No sensitive data in marketing pages
- [ ] CSP headers for marketing site
- [ ] HTTPS enforced on both domains

---

## SEO Considerations

- [ ] Unique `<title>` and `<meta description>` per page
- [ ] Canonical URLs set correctly
- [ ] Open Graph tags for social sharing
- [ ] Sitemap.xml for marketing pages
- [ ] robots.txt allows crawling
