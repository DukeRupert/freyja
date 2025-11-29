package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/dukerupert/freyja/internal"
	"github.com/dukerupert/freyja/internal/billing"
	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/handler/admin"
	"github.com/dukerupert/freyja/internal/handler/storefront"
	"github.com/dukerupert/freyja/internal/handler/webhook"
	"github.com/dukerupert/freyja/internal/middleware"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/router"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/dukerupert/freyja/internal/shipping"
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

	// Initialize webhook handler
	stripeWebhookHandler := webhook.NewStripeHandler(billingProvider, orderService, webhook.StripeWebhookConfig{
		WebhookSecret: cfg.Stripe.WebhookSecret,
		TenantID:      cfg.TenantID,
	})

	// Initialize admin handlers
	adminDashboardHandler := admin.NewDashboardHandler(repo, renderer, cfg.TenantID)
	adminProductListHandler := admin.NewProductListHandler(repo, renderer, cfg.TenantID)
	adminProductFormHandler := admin.NewProductFormHandler(repo, renderer, cfg.TenantID)
	adminProductDetailHandler := admin.NewProductDetailHandler(repo, renderer, cfg.TenantID)
	adminSKUFormHandler := admin.NewSKUFormHandler(repo, renderer, cfg.TenantID)

	// Create router with global middleware
	r := router.New(
		router.Recovery(logger),
		router.Logger(logger),
		middleware.WithUser(userService),
	)

	// Static files
	r.Static("/static/", "./web/static")

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

	// Webhook routes (no authentication - Stripe handles signature verification)
	r.Post("/webhooks/stripe", stripeWebhookHandler.HandleWebhook)

	// Admin routes (require admin authentication)
	adminRouter := r.Group(middleware.RequireAdmin)
	adminRouter.Get("/admin", adminDashboardHandler.ServeHTTP)
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
