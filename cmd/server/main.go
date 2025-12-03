package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/dukerupert/freyja/internal"
	"github.com/dukerupert/freyja/internal/address"
	"github.com/dukerupert/freyja/internal/billing"
	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/handler/admin"
	"github.com/dukerupert/freyja/internal/handler/saas"
	"github.com/dukerupert/freyja/internal/handler/storefront"
	"github.com/dukerupert/freyja/internal/handler/webhook"
	"github.com/dukerupert/freyja/internal/middleware"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/router"
	"github.com/dukerupert/freyja/internal/routes"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/dukerupert/freyja/internal/shipping"
	"github.com/dukerupert/freyja/internal/tax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func run() error {
	ctx := context.Background()

	// Load configuration
	cfg, err := internal.NewConfig()
	if err != nil {
		return fmt.Errorf("config initialization failed: %w", err)
	}

	// Configure logger
	logger := internal.NewLogger(os.Stdout, cfg.Env, cfg.LogLevel)

	// Initialize database/sql connection for migrations
	logger.Info("Connecting to database...")
	sqlDB, err := sql.Open("pgx", cfg.DatabaseUrl)
	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	defer sqlDB.Close()

	// Verify database connection
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}
	logger.Info("Database connection established")

	// Run migrations
	logger.Info("Running database migrations...")
	if err := internal.RunMigrations(sqlDB); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}
	logger.Info("Database migrations completed successfully")

	// Initialize pgx connection pool for application
	pool, err := pgxpool.New(ctx, cfg.DatabaseUrl)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}
	defer pool.Close()

	// Initialize repository
	repo := repository.New(pool)

	// Initialize services
	productService, err := service.NewProductService(repo, cfg.TenantID)
	if err != nil {
		return fmt.Errorf("failed to initialize product service: %w", err)
	}

	cartService, err := service.NewCartService(repo, cfg.TenantID)
	if err != nil {
		return fmt.Errorf("failed to initialize cart service: %w", err)
	}

	userService, err := service.NewUserService(repo, cfg.TenantID)
	if err != nil {
		return fmt.Errorf("failed to initialize user service: %w", err)
	}

	// Parse tenant ID as UUID for password reset service
	tenantUUID, err := uuid.Parse(cfg.TenantID)
	if err != nil {
		return fmt.Errorf("failed to parse tenant ID: %w", err)
	}

	passwordResetService := service.NewPasswordResetService(repo)

	// Load templates with renderer
	logger.Info("Loading templates...")
	renderer, err := handler.NewRenderer("web/templates")
	if err != nil {
		return fmt.Errorf("failed to initialize renderer: %w", err)
	}
	logger.Info("Templates loaded successfully")

	// Initialize Stripe billing provider
	logger.Info("Initializing Stripe billing provider...")
	stripeConfig := billing.StripeConfig{
		APIKey:          cfg.Stripe.SecretKey,
		WebhookSecret:   cfg.Stripe.WebhookSecret,
		EnableStripeTax: false,
		MaxRetries:      3,
		TimeoutSeconds:  30,
	}
	billingProvider, err := billing.NewStripeProvider(stripeConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize Stripe provider: %w", err)
	}
	logger.Info("Stripe billing provider initialized", "test_mode", stripeConfig.IsTestMode())

	// Initialize shipping provider (flat rate for MVP)
	logger.Info("Initializing shipping provider...")
	shippingProvider := shipping.NewFlatRateProvider([]shipping.FlatRate{
		{ServiceName: "Standard Shipping", ServiceCode: "standard", CostCents: 795, DaysMin: 5, DaysMax: 7},
		{ServiceName: "Express Shipping", ServiceCode: "express", CostCents: 1495, DaysMin: 2, DaysMax: 3},
	})
	logger.Info("Shipping provider initialized")

	// Initialize order service
	logger.Info("Initializing order service...")
	orderService, err := service.NewOrderService(repo, cfg.TenantID, billingProvider, shippingProvider)
	if err != nil {
		return fmt.Errorf("failed to initialize order service: %w", err)
	}
	logger.Info("Order service initialized")

	// Initialize subscription service
	logger.Info("Initializing subscription service...")
	subscriptionService, err := service.NewSubscriptionService(repo, cfg.TenantID, billingProvider)
	if err != nil {
		return fmt.Errorf("failed to initialize subscription service: %w", err)
	}
	logger.Info("Subscription service initialized")

	// Initialize account service
	logger.Info("Initializing account service...")
	accountService := service.NewAccountService(repo)
	logger.Info("Account service initialized")

	// Initialize address validator (mock for MVP)
	logger.Info("Initializing address validator...")
	addressValidator := address.NewMockValidator()
	logger.Info("Address validator initialized")

	// Initialize tax calculator (no tax for MVP)
	logger.Info("Initializing tax calculator...")
	taxCalculator := tax.NewNoTaxCalculator()
	logger.Info("Tax calculator initialized")

	// Initialize checkout service
	logger.Info("Initializing checkout service...")
	checkoutService, err := service.NewCheckoutService(
		repo,
		cartService,
		billingProvider,
		shippingProvider,
		taxCalculator,
		addressValidator,
		cfg.TenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize checkout service: %w", err)
	}
	logger.Info("Checkout service initialized")

	// ==========================================================================
	// Build route dependencies
	// ==========================================================================

	// SaaS dependencies
	saasHandler, err := saas.NewPageHandler("web/templates")
	if err != nil {
		return fmt.Errorf("failed to initialize saas handler: %w", err)
	}
	saasDeps := routes.SaaSDeps{
		Handler: saasHandler,
	}

	// Storefront dependencies
	storefrontDeps := routes.StorefrontDeps{
		// Home
		HomeHandler: storefront.NewHomeHandler(productService, renderer),

		// Products
		ProductListHandler:   storefront.NewProductListHandler(productService, renderer),
		ProductDetailHandler: storefront.NewProductDetailHandler(productService, renderer),

		// Cart
		CartViewHandler:       storefront.NewCartViewHandler(cartService, renderer, cfg.Env != "development"),
		AddToCartHandler:      storefront.NewAddToCartHandler(cartService, renderer, cfg.Env != "development"),
		UpdateCartItemHandler: storefront.NewUpdateCartItemHandler(cartService, renderer),
		RemoveCartItemHandler: storefront.NewRemoveCartItemHandler(cartService, renderer),

		// Auth
		SignupHandler: storefront.NewSignupHandler(userService, renderer),
		LoginHandler:  storefront.NewLoginHandler(userService, renderer),
		LogoutHandler: storefront.NewLogoutHandler(userService),

		// Password Reset
		ForgotPasswordHandler: storefront.NewForgotPasswordHandler(renderer, passwordResetService, tenantUUID),
		ResetPasswordHandler:  storefront.NewResetPasswordHandler(renderer, passwordResetService, userService, tenantUUID),

		// Checkout
		CheckoutPageHandler:        storefront.NewCheckoutPageHandler(renderer, cartService, cfg.Stripe.PublishableKey),
		ValidateAddressHandler:     storefront.NewValidateAddressHandler(checkoutService),
		GetShippingRatesHandler:    storefront.NewGetShippingRatesHandler(checkoutService),
		CalculateTotalHandler:      storefront.NewCalculateTotalHandler(checkoutService),
		CreatePaymentIntentHandler: storefront.NewCreatePaymentIntentHandler(checkoutService),
		OrderConfirmationHandler:   storefront.NewOrderConfirmationHandler(renderer, cartService, orderService, repo, cfg.TenantID),

		// Account (authenticated)
		SubscriptionListHandler:     storefront.NewSubscriptionListHandler(subscriptionService, renderer, cfg.TenantID),
		SubscriptionDetailHandler:   storefront.NewSubscriptionDetailHandler(subscriptionService, renderer, cfg.TenantID),
		SubscriptionPortalHandler:   storefront.NewSubscriptionPortalHandler(subscriptionService, cfg.TenantID),
		SubscriptionCheckoutHandler: storefront.NewSubscriptionCheckoutHandler(productService, accountService, renderer, cfg.TenantID),
		CreateSubscriptionHandler:   storefront.NewCreateSubscriptionHandler(subscriptionService, renderer, cfg.TenantID),
	}

	// Admin dependencies
	adminDeps := routes.AdminDeps{
		DashboardHandler:          admin.NewDashboardHandler(repo, renderer, cfg.TenantID),
		ProductListHandler:        admin.NewProductListHandler(repo, renderer, cfg.TenantID),
		ProductFormHandler:        admin.NewProductFormHandler(repo, renderer, cfg.TenantID),
		ProductDetailHandler:      admin.NewProductDetailHandler(repo, renderer, cfg.TenantID),
		SKUFormHandler:            admin.NewSKUFormHandler(repo, renderer, cfg.TenantID),
		OrderListHandler:          admin.NewOrderListHandler(repo, renderer, cfg.TenantID),
		OrderDetailHandler:        admin.NewOrderDetailHandler(repo, renderer, cfg.TenantID),
		UpdateOrderStatusHandler:  admin.NewUpdateOrderStatusHandler(repo, cfg.TenantID),
		CreateShipmentHandler:     admin.NewCreateShipmentHandler(repo, cfg.TenantID),
		CustomerListHandler:       admin.NewCustomerListHandler(repo, renderer, cfg.TenantID),
		SubscriptionListHandler:   admin.NewSubscriptionListHandler(repo, renderer, cfg.TenantID),
		SubscriptionDetailHandler: admin.NewSubscriptionDetailHandler(repo, renderer, cfg.TenantID),
	}

	// Webhook dependencies
	stripeWebhookHandler := webhook.NewStripeHandler(billingProvider, orderService, subscriptionService, webhook.StripeWebhookConfig{
		WebhookSecret: cfg.Stripe.WebhookSecret,
		TenantID:      cfg.TenantID,
	})
	webhookDeps := routes.WebhookDeps{
		StripeHandler: stripeWebhookHandler.HandleWebhook,
	}

	// ==========================================================================
	// Initialize middleware
	// ==========================================================================

	// Initialize Prometheus metrics
	metrics := middleware.NewMetrics("freyja")

	// Configure security headers
	securityConfig := middleware.DefaultSecurityHeadersConfig()
	if cfg.Env == "development" {
		// Relax CSP in development for easier debugging
		securityConfig.ContentSecurityPolicy = ""
		securityConfig.HSTSMaxAge = 0 // Disable HSTS in development
	}

	// Configure CSRF protection
	csrfConfig := middleware.DefaultCSRFConfig()
	csrfConfig.CookieSecure = cfg.Env != "development"

	// Configure rate limiting
	defaultRateLimiter := middleware.NewRateLimiter(middleware.DefaultRateLimiterConfig())
	authRateLimiter := middleware.NewRateLimiter(middleware.StrictRateLimiterConfig())

	// ==========================================================================
	// Create routers and register routes
	// ==========================================================================

	// Main tenant router (storefront + admin + webhooks)
	r := router.New(
		router.Recovery(logger),
		middleware.RequestID,
		metrics.Middleware,
		middleware.SecurityHeaders(securityConfig),
		middleware.MaxBodySize(middleware.DefaultMaxBodySize),
		middleware.Timeout(middleware.DefaultTimeout),
		defaultRateLimiter.Middleware,
		router.Logger(logger),
		middleware.WithUser(userService),
		middleware.CSRF(csrfConfig),
	)

	// Static files
	r.Static("/static/", "./web/static")

	// Metrics endpoint (no auth required, but should be protected in production via firewall)
	r.Get("/metrics", func(w http.ResponseWriter, req *http.Request) {
		metrics.Handler().ServeHTTP(w, req)
	})

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Register route groups
	routes.RegisterStorefrontRoutes(r, storefrontDeps)
	routes.RegisterAdminRoutes(r, adminDeps)
	routes.RegisterWebhookRoutes(r, webhookDeps)

	// Apply stricter rate limiting to auth endpoints
	authRouter := r.Group(authRateLimiter.Middleware)
	authRouter.Post("/login", storefrontDeps.LoginHandler.ServeHTTP)
	authRouter.Post("/signup", storefrontDeps.SignupHandler.ServeHTTP)

	// SaaS marketing site router (separate, can be served on different port/domain)
	saasRouter := router.New(
		router.Recovery(logger),
		middleware.RequestID,
		middleware.SecurityHeaders(securityConfig),
		router.Logger(logger),
	)
	saasRouter.Static("/static/", "./web/static")
	routes.RegisterSaaSRoutes(saasRouter, saasDeps)

	// ==========================================================================
	// Start servers
	// ==========================================================================

	// For MVP: serve both on same port, SaaS on separate port
	// In production: SaaS would be on freyja.app, tenant on {tenant}.shop.freyja.app

	// Start SaaS server on port 3001
	saasAddr := ":3001"
	go func() {
		logger.Info("Starting SaaS marketing server", "address", saasAddr)
		if err := http.ListenAndServe(saasAddr, saasRouter); err != nil {
			logger.Error("SaaS server failed", "error", err)
		}
	}()

	// Start main tenant server on configured port
	addr := fmt.Sprintf(":%d", cfg.Port)
	logger.Info("Starting tenant server", "address", addr)

	if err := http.ListenAndServe(addr, r); err != nil {
		return fmt.Errorf("server failed: %w", err)
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
