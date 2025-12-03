package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dukerupert/freyja/internal"
	"github.com/dukerupert/freyja/internal/address"
	"github.com/dukerupert/freyja/internal/billing"
	"github.com/dukerupert/freyja/internal/email"
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
	"github.com/dukerupert/freyja/internal/worker"
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
	emailVerificationService := service.NewEmailVerificationService(repo, cfg.BaseURL)

	// Initialize email service
	logger.Info("Initializing email service...")
	var emailSender email.Sender
	if cfg.Email.PostmarkToken != "" {
		logger.Info("Using Postmark email sender")
		emailSender = email.NewPostmarkSender(cfg.Email.PostmarkToken)
	} else {
		logger.Info("Using SMTP email sender (development)")
		emailSender = email.NewSMTPSender(
			cfg.Email.Host,
			int(cfg.Email.Port),
			cfg.Email.Username,
			cfg.Email.Password,
			cfg.Email.From,
		)
	}

	emailService, err := email.NewService(emailSender, cfg.Email.From, cfg.Email.FromName, "web/templates")
	if err != nil {
		return fmt.Errorf("failed to initialize email service: %w", err)
	}
	logger.Info("Email service initialized")

	// Note: Background worker initialization moved after service creation

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

	// Initialize payment terms service
	logger.Info("Initializing payment terms service...")
	paymentTermsService, err := service.NewPaymentTermsService(repo, cfg.TenantID)
	if err != nil {
		return fmt.Errorf("failed to initialize payment terms service: %w", err)
	}
	logger.Info("Payment terms service initialized")

	// Initialize invoice service
	logger.Info("Initializing invoice service...")
	invoiceService, err := service.NewInvoiceService(repo, paymentTermsService, billingProvider, cfg.TenantID)
	if err != nil {
		return fmt.Errorf("failed to initialize invoice service: %w", err)
	}
	logger.Info("Invoice service initialized")

	// Initialize background worker
	logger.Info("Initializing background worker...")
	workerConfig := worker.Config{
		WorkerID:       fmt.Sprintf("worker-%s", uuid.New().String()[:8]),
		PollInterval:   1 * time.Second,
		MaxConcurrency: 5,
		Queue:          "", // Process all queues
		TenantID:       &tenantUUID,
	}
	bgWorker := worker.NewWorker(repo, emailService, invoiceService, workerConfig, logger)
	logger.Info("Background worker initialized")

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

		// Cart (consolidated handler)
		CartHandler: storefront.NewCartHandler(cartService, renderer, cfg.Env != "development"),

		// Auth
		SignupHandler:        storefront.NewSignupHandler(userService, emailVerificationService, renderer, tenantUUID),
		SignupSuccessHandler: storefront.NewSignupSuccessHandler(renderer),
		LoginHandler:         storefront.NewLoginHandler(userService, renderer),
		LogoutHandler:        storefront.NewLogoutHandler(userService),

		// Password Reset
		ForgotPasswordHandler: storefront.NewForgotPasswordHandler(renderer, passwordResetService, tenantUUID),
		ResetPasswordHandler:  storefront.NewResetPasswordHandler(renderer, passwordResetService, userService, tenantUUID),

		// Email Verification
		VerifyEmailHandler:        storefront.NewVerifyEmailHandler(emailVerificationService, renderer, tenantUUID),
		ResendVerificationHandler: storefront.NewResendVerificationHandler(emailVerificationService, repo, renderer, tenantUUID),

		// Checkout (consolidated handler)
		CheckoutHandler: storefront.NewCheckoutHandler(
			renderer,
			cartService,
			checkoutService,
			orderService,
			repo,
			cfg.Stripe.PublishableKey,
			cfg.TenantID,
		),

		// Account (authenticated)
		AccountDashboardHandler:     storefront.NewAccountDashboardHandler(accountService, subscriptionService, renderer, cfg.TenantID),
		SubscriptionListHandler:     storefront.NewSubscriptionListHandler(subscriptionService, renderer, cfg.TenantID),
		SubscriptionDetailHandler:   storefront.NewSubscriptionDetailHandler(subscriptionService, renderer, cfg.TenantID),
		SubscriptionPortalHandler:   storefront.NewSubscriptionPortalHandler(subscriptionService, cfg.TenantID),
		SubscriptionCheckoutHandler: storefront.NewSubscriptionCheckoutHandler(productService, accountService, renderer, cfg.TenantID),
		CreateSubscriptionHandler:   storefront.NewCreateSubscriptionHandler(subscriptionService, renderer, cfg.TenantID),
	}

	// Admin dependencies (consolidated handlers)
	adminDeps := routes.AdminDeps{
		DashboardHandler:    admin.NewDashboardHandler(repo, renderer, cfg.TenantID),
		ProductHandler:      admin.NewProductHandler(repo, renderer, cfg.TenantID),
		OrderHandler:        admin.NewOrderHandler(repo, renderer, cfg.TenantID),
		CustomerHandler:     admin.NewCustomerHandler(repo, invoiceService, renderer, cfg.TenantID),
		SubscriptionHandler: admin.NewSubscriptionHandler(repo, renderer, cfg.TenantID),
		InvoiceHandler:      admin.NewInvoiceHandler(invoiceService, repo, renderer, cfg.TenantID),
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

	// Environment-specific middleware configuration
	isDev := cfg.Env == "development"

	// Security headers config (relaxed in development)
	var securityConfig middleware.SecurityHeadersConfig
	if isDev {
		securityConfig = middleware.SecurityHeadersConfig{
			// Relax CSP and HSTS in development for easier debugging
			ContentSecurityPolicy: "",
			HSTSMaxAge:            0,
		}
	}

	// CSRF config (insecure cookies allowed in development)
	csrfConfig := middleware.CSRFConfig{
		CookieSecure: !isDev,
	}

	// ==========================================================================
	// Create routers and register routes
	// ==========================================================================

	// Main tenant router (storefront + admin + webhooks)
	r := router.New(
		router.Recovery(logger),
		middleware.RequestID(),
		metrics.Middleware(),
		middleware.SecurityHeaders(securityConfig),
		middleware.MaxBodySize(),
		middleware.Timeout(),
		middleware.RateLimit(),
		router.Logger(logger),
		middleware.WithUser(userService),
		middleware.WithRequestLogger(logger),
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
	authRouter := r.Group(middleware.StrictRateLimit())
	authRouter.Post("/login", storefrontDeps.LoginHandler.HandleSubmit)
	authRouter.Post("/signup", storefrontDeps.SignupHandler.HandleSubmit)

	// SaaS marketing site router (separate, can be served on different port/domain)
	saasRouter := router.New(
		router.Recovery(logger),
		middleware.RequestID(),
		middleware.SecurityHeaders(securityConfig),
		middleware.RateLimit(),
		router.Logger(logger),
	)
	saasRouter.Static("/static/", "./web/static")
	routes.RegisterSaaSRoutes(saasRouter, saasDeps)

	// ==========================================================================
	// Start servers and background workers
	// ==========================================================================

	// Create a context for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	defer shutdownCancel()

	// Start background worker
	go func() {
		logger.Info("Starting background worker")
		if err := bgWorker.Start(shutdownCtx); err != nil && err != context.Canceled {
			logger.Error("Background worker error", "error", err)
		}
	}()

	// For MVP: serve both on same port, SaaS on separate port
	// In production: SaaS would be on freyja.app, tenant on {tenant}.shop.freyja.app

	// Start SaaS server on port 3001
	saasAddr := ":3001"
	saasServer := &http.Server{
		Addr:    saasAddr,
		Handler: saasRouter,
	}
	go func() {
		logger.Info("Starting SaaS marketing server", "address", saasAddr)
		if err := saasServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("SaaS server failed", "error", err)
		}
	}()

	// Start main tenant server on configured port
	addr := fmt.Sprintf(":%d", cfg.Port)
	mainServer := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// Channel to listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start main server in goroutine
	go func() {
		logger.Info("Starting tenant server", "address", addr)
		if err := mainServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Main server failed", "error", err)
		}
	}()

	// Wait for interrupt signal
	<-sigChan
	logger.Info("Shutdown signal received, initiating graceful shutdown...")

	// Cancel worker context to stop background jobs
	shutdownCancel()

	// Create shutdown context with timeout
	shutdownTimeout := 30 * time.Second
	shutdownTimeoutCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// Shutdown servers gracefully
	if err := mainServer.Shutdown(shutdownTimeoutCtx); err != nil {
		logger.Error("Main server shutdown error", "error", err)
	}
	if err := saasServer.Shutdown(shutdownTimeoutCtx); err != nil {
		logger.Error("SaaS server shutdown error", "error", err)
	}

	logger.Info("Graceful shutdown complete")
	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
