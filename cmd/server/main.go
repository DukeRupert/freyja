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
	"github.com/dukerupert/freyja/internal/service"
	"github.com/dukerupert/freyja/internal/shipping"
	"github.com/dukerupert/freyja/internal/tax"
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

	// Load templates with renderer
	logger.Info("Loading templates...")
	renderer, err := handler.NewRenderer("web/templates")
	if err != nil {
		return fmt.Errorf("failed to initialize renderer: %w", err)
	}
	logger.Info("Templates loaded successfully")

	// Initialize handlers
	productListHandler := storefront.NewProductListHandler(productService, renderer)
	productDetailHandler := storefront.NewProductDetailHandler(productService, renderer)
	cartViewHandler := storefront.NewCartViewHandler(cartService, renderer, cfg.Env != "development")
	addToCartHandler := storefront.NewAddToCartHandler(cartService, renderer, cfg.Env != "development")
	updateCartItemHandler := storefront.NewUpdateCartItemHandler(cartService, renderer)
	removeCartItemHandler := storefront.NewRemoveCartItemHandler(cartService, renderer)

	// Auth handlers
	signupHandler := storefront.NewSignupHandler(userService, renderer)
	loginHandler := storefront.NewLoginHandler(userService, renderer)
	logoutHandler := storefront.NewLogoutHandler(userService)

	// Checkout handlers (will be initialized after checkout service is created)
	var checkoutPageHandler *storefront.CheckoutPageHandler
	var validateAddressHandler *storefront.ValidateAddressHandler
	var getShippingRatesHandler *storefront.GetShippingRatesHandler
	var calculateTotalHandler *storefront.CalculateTotalHandler
	var createPaymentIntentHandler *storefront.CreatePaymentIntentHandler

	// Initialize Stripe billing provider
	logger.Info("Initializing Stripe billing provider...")
	stripeConfig := billing.StripeConfig{
		APIKey:          cfg.Stripe.SecretKey,
		WebhookSecret:   cfg.Stripe.WebhookSecret,
		EnableStripeTax: false, // Set to true if Stripe Tax is enabled
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

	// Initialize checkout handlers
	checkoutPageHandler = storefront.NewCheckoutPageHandler(renderer, cartService, cfg.Stripe.PublishableKey)
	validateAddressHandler = storefront.NewValidateAddressHandler(checkoutService)
	getShippingRatesHandler = storefront.NewGetShippingRatesHandler(checkoutService)
	calculateTotalHandler = storefront.NewCalculateTotalHandler(checkoutService)
	createPaymentIntentHandler = storefront.NewCreatePaymentIntentHandler(checkoutService)
	orderConfirmationHandler := storefront.NewOrderConfirmationHandler(renderer, cartService, orderService, repo, cfg.TenantID)

	// Initialize webhook handler
	stripeWebhookHandler := webhook.NewStripeHandler(billingProvider, orderService, subscriptionService, webhook.StripeWebhookConfig{
		WebhookSecret: cfg.Stripe.WebhookSecret,
		TenantID:      cfg.TenantID,
	})

	// Initialize admin handlers
	adminDashboardHandler := admin.NewDashboardHandler(repo, renderer, cfg.TenantID)
	adminProductListHandler := admin.NewProductListHandler(repo, renderer, cfg.TenantID)
	adminProductFormHandler := admin.NewProductFormHandler(repo, renderer, cfg.TenantID)
	adminProductDetailHandler := admin.NewProductDetailHandler(repo, renderer, cfg.TenantID)
	adminSKUFormHandler := admin.NewSKUFormHandler(repo, renderer, cfg.TenantID)
	adminOrderListHandler := admin.NewOrderListHandler(repo, renderer, cfg.TenantID)
	adminOrderDetailHandler := admin.NewOrderDetailHandler(repo, renderer, cfg.TenantID)
	adminCustomerListHandler := admin.NewCustomerListHandler(repo, renderer, cfg.TenantID)
	updateOrderStatusHandler := admin.NewUpdateOrderStatusHandler(repo, cfg.TenantID)
	createShipmentHandler := admin.NewCreateShipmentHandler(repo, cfg.TenantID)
	adminSubscriptionListHandler := admin.NewSubscriptionListHandler(repo, renderer, cfg.TenantID)
	adminSubscriptionDetailHandler := admin.NewSubscriptionDetailHandler(repo, renderer, cfg.TenantID)

	// Initialize storefront subscription handlers
	subscriptionListHandler := storefront.NewSubscriptionListHandler(subscriptionService, renderer, cfg.TenantID)
	subscriptionDetailHandler := storefront.NewSubscriptionDetailHandler(subscriptionService, renderer, cfg.TenantID)
	subscriptionPortalHandler := storefront.NewSubscriptionPortalHandler(subscriptionService, cfg.TenantID)
	subscriptionCheckoutHandler := storefront.NewSubscriptionCheckoutHandler(productService, accountService, renderer, cfg.TenantID)
	createSubscriptionHandler := storefront.NewCreateSubscriptionHandler(subscriptionService, renderer, cfg.TenantID)

	// Initialize SaaS landing page handler
	landingHandler, err := saas.NewLandingHandler("web/templates")
	if err != nil {
		return fmt.Errorf("failed to initialize landing handler: %w", err)
	}

	// Create router with global middleware
	r := router.New(
		router.Recovery(logger),
		router.Logger(logger),
		middleware.WithUser(userService),
	)

	// Static files
	r.Static("/static/", "./web/static")

	// SaaS landing page (root)
	r.Get("/", landingHandler.ServeHTTP)

	// Auth routes
	r.Get("/signup", signupHandler.ServeHTTP)
	r.Post("/signup", signupHandler.ServeHTTP)
	r.Get("/login", loginHandler.ServeHTTP)
	r.Post("/login", loginHandler.ServeHTTP)
	r.Post("/logout", logoutHandler.ServeHTTP)

	// Storefront routes
	r.Get("/products", productListHandler.ServeHTTP)
	r.Get("/products/{slug}", productDetailHandler.ServeHTTP)
	r.Get("/cart", cartViewHandler.ServeHTTP)
	r.Post("/cart/add", addToCartHandler.ServeHTTP)
	r.Post("/cart/update", updateCartItemHandler.ServeHTTP)
	r.Post("/cart/remove", removeCartItemHandler.ServeHTTP)

	// Checkout routes
	r.Get("/checkout", checkoutPageHandler.ServeHTTP)
	r.Post("/checkout/validate-address", validateAddressHandler.ServeHTTP)
	r.Post("/checkout/shipping-rates", getShippingRatesHandler.ServeHTTP)
	r.Post("/checkout/calculate-total", calculateTotalHandler.ServeHTTP)
	r.Post("/checkout/create-payment-intent", createPaymentIntentHandler.ServeHTTP)
	r.Get("/order-confirmation", orderConfirmationHandler.ServeHTTP)

	// Account routes (require authentication)
	accountRouter := r.Group(middleware.RequireAuth)
	accountRouter.Get("/account/subscriptions", subscriptionListHandler.ServeHTTP)
	accountRouter.Get("/account/subscriptions/portal", subscriptionPortalHandler.ServeHTTP)
	accountRouter.Get("/account/subscriptions/{id}", subscriptionDetailHandler.ServeHTTP)
	accountRouter.Get("/subscribe/checkout", subscriptionCheckoutHandler.ServeHTTP)
	accountRouter.Post("/subscribe", createSubscriptionHandler.ServeHTTP)

	// Webhook routes (no authentication - Stripe handles signature verification)
	r.Post("/webhooks/stripe", stripeWebhookHandler.HandleWebhook)

	// Admin routes (require admin authentication)
	adminRouter := r.Group(middleware.RequireAdmin)
	adminRouter.Get("/admin", adminDashboardHandler.ServeHTTP)

	// Product routes
	adminRouter.Get("/admin/products", adminProductListHandler.ServeHTTP)
	adminRouter.Get("/admin/products/new", adminProductFormHandler.ServeHTTP)
	adminRouter.Post("/admin/products/new", adminProductFormHandler.ServeHTTP)
	adminRouter.Get("/admin/products/{id}", adminProductDetailHandler.ServeHTTP)
	adminRouter.Get("/admin/products/{id}/edit", adminProductFormHandler.ServeHTTP)
	adminRouter.Post("/admin/products/{id}/edit", adminProductFormHandler.ServeHTTP)
	adminRouter.Get("/admin/products/{product_id}/skus/new", adminSKUFormHandler.ServeHTTP)
	adminRouter.Post("/admin/products/{product_id}/skus/new", adminSKUFormHandler.ServeHTTP)
	adminRouter.Get("/admin/products/{product_id}/skus/{sku_id}/edit", adminSKUFormHandler.ServeHTTP)
	adminRouter.Post("/admin/products/{product_id}/skus/{sku_id}/edit", adminSKUFormHandler.ServeHTTP)

	// Order routes
	adminRouter.Get("/admin/orders", adminOrderListHandler.ServeHTTP)
	adminRouter.Get("/admin/orders/{id}", adminOrderDetailHandler.ServeHTTP)
	adminRouter.Post("/admin/orders/{id}/status", updateOrderStatusHandler.ServeHTTP)
	adminRouter.Post("/admin/orders/{id}/shipments", createShipmentHandler.ServeHTTP)

	// Customer routes
	adminRouter.Get("/admin/customers", adminCustomerListHandler.ServeHTTP)

	// Subscription routes
	adminRouter.Get("/admin/subscriptions", adminSubscriptionListHandler.ServeHTTP)
	adminRouter.Get("/admin/subscriptions/{id}", adminSubscriptionDetailHandler.ServeHTTP)

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Port)
	logger.Info("Starting server", "address", addr)

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
