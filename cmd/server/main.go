package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dukerupert/freyja/internal"
	"github.com/dukerupert/freyja/internal/address"
	"github.com/dukerupert/freyja/internal/billing"
	"github.com/dukerupert/freyja/internal/bootstrap"
	"github.com/dukerupert/freyja/internal/crypto"
	"github.com/dukerupert/freyja/internal/email"
	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/handler/admin"
	"github.com/dukerupert/freyja/internal/handler/api"
	"github.com/dukerupert/freyja/internal/handler/saas"
	"github.com/dukerupert/freyja/internal/handler/storefront"
	"github.com/dukerupert/freyja/internal/handler/webhook"
	"github.com/dukerupert/freyja/internal/middleware"
	"github.com/dukerupert/freyja/internal/onboarding"
	"github.com/dukerupert/freyja/internal/provider"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/router"
	"github.com/dukerupert/freyja/internal/routes"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/dukerupert/freyja/internal/shipping"
	"github.com/dukerupert/freyja/internal/storage"
	"github.com/dukerupert/freyja/internal/tax"
	"github.com/dukerupert/freyja/internal/telemetry"
	"github.com/dukerupert/freyja/internal/worker"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
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

	// Initialize telemetry
	logger.Info("Initializing telemetry...")

	// Initialize business metrics (Prometheus)
	telemetry.InitBusinessMetrics("freyja")
	logger.Info("Business metrics initialized")

	// Initialize Sentry error tracking
	sentryCleanup, err := telemetry.InitSentry(telemetry.SentryConfig{
		DSN:              cfg.Sentry.DSN,
		Enabled:          cfg.Sentry.Enabled,
		Environment:      cfg.Sentry.Environment,
		Release:          cfg.Sentry.Release,
		SampleRate:       cfg.Sentry.SampleRate,
		TracesSampleRate: cfg.Sentry.TracesSampleRate,
		Debug:            cfg.Sentry.Debug,
	}, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize Sentry: %w", err)
	}
	defer sentryCleanup()

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

	// Convert to pgtype.UUID for repository queries
	var tenantPgUUID pgtype.UUID
	if err := tenantPgUUID.Scan(cfg.TenantID); err != nil {
		return fmt.Errorf("failed to convert tenant ID to pgtype.UUID: %w", err)
	}

	// Bootstrap: ensure master admin user exists
	logger.Info("Checking for master admin user...")
	adminCfg := &bootstrap.AdminConfig{
		Email:     cfg.Admin.Email,
		Password:  cfg.Admin.Password,
		FirstName: cfg.Admin.FirstName,
		LastName:  cfg.Admin.LastName,
	}
	if err := bootstrap.EnsureMasterAdmin(ctx, repo, tenantPgUUID, adminCfg, logger); err != nil {
		return fmt.Errorf("failed to ensure master admin: %w", err)
	}

	passwordResetService := service.NewPasswordResetService(repo)
	emailVerificationService := service.NewEmailVerificationService(repo, pool, cfg.BaseURL)

	// Initialize email service
	logger.Info("Initializing email service...")
	var emailSender email.Sender
	if cfg.Email.PostmarkToken != "" {
		logger.Info("Using Postmark email sender")
		emailSender = email.NewPostmarkSender(cfg.Email.PostmarkToken, logger)
	} else {
		logger.Info("Using SMTP email sender (development)")
		emailSender = email.NewSMTPSender(
			cfg.Email.Host,
			int(cfg.Email.Port),
			cfg.Email.Username,
			cfg.Email.Password,
			cfg.Email.From,
			cfg.Email.FromName,
		)
	}

	emailService, err := email.NewService(emailSender, cfg.Email.From, cfg.Email.FromName, "web/templates", logger)
	if err != nil {
		return fmt.Errorf("failed to initialize email service: %w", err)
	}
	logger.Info("Email service initialized")

	// Initialize file storage for product images
	logger.Info("Initializing file storage...", "provider", cfg.Storage.Provider)
	fileStorage, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		return fmt.Errorf("failed to initialize file storage: %w", err)
	}
	logger.Info("File storage initialized", "provider", cfg.Storage.Provider)

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

		// Products (consolidated: list, detail, subscription products)
		ProductHandler: storefront.NewProductHandler(productService, repo, renderer, cfg.TenantID),

		// Cart (consolidated handler)
		CartHandler: storefront.NewCartHandler(cartService, renderer, cfg.Env != "development", cfg.TenantID),

		// Auth (consolidated: signup, login, logout, password reset, email verification)
		AuthHandler: storefront.NewAuthHandler(
			userService,
			emailVerificationService,
			passwordResetService,
			repo,
			renderer,
			tenantUUID,
		),

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

		// Subscriptions (consolidated: list, detail, portal, checkout, create)
		SubscriptionHandler: storefront.NewSubscriptionHandler(
			subscriptionService,
			productService,
			accountService,
			renderer,
			cfg.TenantID,
		),

		// Account (consolidated: dashboard, orders, addresses, payment methods, profile)
		AccountHandler: storefront.NewAccountHandler(
			accountService,
			subscriptionService,
			repo,
			renderer,
			cfg.TenantID,
		),

		// Wholesale
		WholesaleApplicationHandler: storefront.NewWholesaleApplicationHandler(repo, renderer, cfg.TenantID),
		WholesaleOrderingHandler:    storefront.NewWholesaleOrderingHandler(repo, cartService, renderer, cfg.TenantID, cfg.Env != "development"),
	}

	// ==========================================================================
	// Initialize provider configuration system
	// ==========================================================================

	// Initialize encryptor for provider credentials
	var encryptor crypto.Encryptor
	if cfg.EncryptionKey != "" {
		encryptionKey, err := crypto.DecodeKeyBase64(cfg.EncryptionKey)
		if err != nil {
			return fmt.Errorf("invalid encryption key: %w", err)
		}
		encryptor, err = crypto.NewAESEncryptor(encryptionKey)
		if err != nil {
			return fmt.Errorf("failed to create encryptor: %w", err)
		}
	} else {
		// Generate a temporary key for development (not persisted)
		logger.Warn("ENCRYPTION_KEY not set - generating temporary key (credentials will not persist across restarts)")
		tempKey, _ := crypto.GenerateKey()
		encryptor, _ = crypto.NewAESEncryptor(tempKey)
	}

	// Initialize provider system components
	providerValidator := provider.NewDefaultValidator()
	providerFactory := provider.MustNewDefaultFactory(providerValidator) // Panics only during startup if validator is nil
	providerRegistry := provider.NewDefaultRegistry(repo, providerFactory, encryptor, 0) // 0 = default 1 hour TTL

	// Initialize onboarding service
	onboardingService := onboarding.NewService(repo)

	// Initialize custom domain service
	customDomainService := service.NewCustomDomainService(repo, logger)

	// Admin dependencies (consolidated handlers)
	adminDeps := routes.AdminDeps{
		LoginHandler:        admin.NewLoginHandler(userService, renderer),
		LogoutHandler:       admin.NewLogoutHandler(userService),
		DashboardHandler:    admin.NewDashboardHandler(repo, renderer, cfg.TenantID, onboardingService),
		ProductHandler:      admin.NewProductHandler(repo, renderer, fileStorage, cfg.TenantID),
		OrderHandler:        admin.NewOrderHandler(repo, renderer, cfg.TenantID),
		CustomerHandler:     admin.NewCustomerHandler(repo, invoiceService, renderer, cfg.TenantID),
		SubscriptionHandler: admin.NewSubscriptionHandler(repo, renderer, cfg.TenantID),
		InvoiceHandler:      admin.NewInvoiceHandler(invoiceService, repo, renderer, cfg.TenantID),
		PriceListHandler:    admin.NewPriceListHandler(repo, renderer, cfg.TenantID),
		TaxRateHandler:      admin.NewTaxRateHandler(repo, renderer, cfg.TenantID),
		IntegrationsHandler: admin.NewIntegrationsHandler(repo, renderer, cfg.TenantID, encryptor, providerValidator, providerRegistry),
		CustomDomainHandler: admin.NewCustomDomainHandler(customDomainService, renderer),
		OnboardingHandler:   admin.NewOnboardingHandler(onboardingService, renderer, cfg.TenantID),
	}

	// Webhook dependencies
	// TestMode allows Stripe CLI trigger testing without full metadata validation.
	// Enable via STRIPE_WEBHOOK_TEST_MODE=true (development only!)
	webhookTestMode := os.Getenv("STRIPE_WEBHOOK_TEST_MODE") == "true"
	if webhookTestMode {
		slog.Warn("Stripe webhook TEST MODE enabled - tenant isolation checks bypassed")
	}
	stripeWebhookHandler := webhook.NewStripeHandler(billingProvider, orderService, subscriptionService, webhook.StripeWebhookConfig{
		WebhookSecret: cfg.Stripe.WebhookSecret,
		TenantID:      cfg.TenantID,
		TestMode:      webhookTestMode,
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
		// Skip CSRF validation for webhook endpoints (they use signature verification)
		SkipPaths: []string{"/webhooks/"},
	}

	// ==========================================================================
	// Create routers and register routes
	// ==========================================================================

	// Create user extractor for Sentry context
	userExtractor := func(ctx context.Context) *telemetry.UserInfo {
		user := middleware.GetUserFromContext(ctx)
		if user == nil {
			return nil
		}
		return &telemetry.UserInfo{
			ID:    user.ID.String(),
			Email: user.Email,
		}
	}

	// Main tenant router (storefront + admin + webhooks)
	r := router.New(
		router.Recovery(logger),
		telemetry.SentryMiddleware(), // Capture panics and add request context to Sentry
		middleware.RequestID(),
		metrics.Middleware(),
		middleware.SecurityHeaders(securityConfig),
		middleware.MaxBodySize(),
		middleware.Timeout(),
		middleware.RateLimit(),
		router.Logger(logger),
		middleware.WithUser(userService),
		telemetry.SentryContextMiddleware(cfg.TenantID, userExtractor), // Set tenant/user context for Sentry
		middleware.WithRequestLogger(logger),
		middleware.CSRF(csrfConfig),
	)

	// Static files
	r.Static("/static/", "./web/static")
	r.Static("/uploads/", "./web/static/uploads")

	// Metrics endpoint (no auth required, but should be protected in production via firewall)
	r.Get("/metrics", func(w http.ResponseWriter, req *http.Request) {
		metrics.Handler().ServeHTTP(w, req)
	})

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// API dependencies
	apiDeps := routes.APIDeps{
		DomainValidationHandler: api.NewDomainValidationHandler(customDomainService, logger),
	}

	// Register route groups
	routes.RegisterStorefrontRoutes(r, storefrontDeps)
	routes.RegisterAdminRoutes(r, adminDeps)
	routes.RegisterAPIRoutes(r, apiDeps)
	routes.RegisterWebhookRoutes(r, webhookDeps)

	// Apply stricter rate limiting to auth endpoints
	authRouter := r.Group(middleware.StrictRateLimit())
	authRouter.Post("/login", storefrontDeps.AuthHandler.HandleLogin)
	authRouter.Post("/signup", storefrontDeps.AuthHandler.HandleSignup)
	authRouter.Post("/admin/login", adminDeps.LoginHandler.HandleSubmit)

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
