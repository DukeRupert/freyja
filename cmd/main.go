package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/dukerupert/freyja/internal/database"
	h "github.com/dukerupert/freyja/internal/server/handler"
	customMiddleware "github.com/dukerupert/freyja/internal/server/middleware"
	"github.com/dukerupert/freyja/internal/server/provider"
	"github.com/dukerupert/freyja/internal/shared/config"
	"github.com/dukerupert/freyja/internal/shared/interfaces"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

// NewServer creates and configures an Echo instance
func NewServer(
	logger zerolog.Logger,
	cfg *config.Config,
	db *database.DB,
	eventBus interfaces.EventPublisher,
	stripeProvider *provider.StripeProvider,
) *echo.Echo {
	e := echo.New()

	// Configure Echo settings
	e.HideBanner = true
	e.HidePort = true
	e.Validator = &CustomValidator{validator: validator.New()}

	// Global middleware
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(customMiddleware.ZerologMiddleware(logger))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"https://refactored-umbrella-rp9xx597vq535wg6-8081.app.github.dev", "http://localhost:8081"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions, http.MethodHead},
		AllowCredentials: false,
	}))
	e.Use(customMiddleware.PrometheusMiddleware())

	// Add routes
	addRoutes(e, logger, cfg, db, eventBus, stripeProvider)

	return e
}

func addRoutes(
	e *echo.Echo,
	logger zerolog.Logger,
	cfg *config.Config,
	db *database.DB,
	eventBus interfaces.EventPublisher,
	stripeProvider *provider.StripeProvider,
) {
	// System endpoints
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
	e.GET("/health", handleHealthCheck())
	e.GET("/", handleHelloWorld())

	// API routes
	// Webhooks
	// webhooks := e.Group("/webhooks")
	// webhooks.POST("/stripe", handleStripeWebhook(stripeProvider, db, eventBus, logger))

	// API v1
	api := e.Group("/api/v1")

	// Products
	products := api.Group("/products")
	products.GET("", h.HandleGetProducts(db, logger))
	// products.GET("/in-stock", handleGetInStockProducts(db, logger))
	// products.GET("/low-stock", handleGetLowStockProducts(db, logger))
	// products.GET("/stats", handleGetProductStats(db, logger))
	products.GET("/:id", h.HandleGetProduct(db, logger))
	// products.GET("/:id/variants", handleGetProductVariants(db, logger))
	// products.GET("/variants/search", handleSearchProductVariants(db, logger))

	// Cart
	// cart := api.Group("/cart")
	// cart.GET("", handleGetCart(db, logger))
	// cart.DELETE("", handleClearCart(db, eventBus, logger))
	// cart.POST("/items", handleAddCartItem(db, eventBus, logger))
	// cart.PUT("/items/:id", handleUpdateCartItem(db, eventBus, logger))
	// cart.DELETE("/items/:id", handleRemoveCartItem(db, eventBus, logger))

	// Checkout
	// checkout := api.Group("/checkout")
	// checkout.POST("", handleCreateCheckoutSession(db, stripeProvider, eventBus, logger))

	// Orders
	// orders := api.Group("/orders")
	// orders.GET("", handleGetOrders(db, logger))
	// orders.GET("/:id", handleGetOrder(db, logger))

	// Customers
	// customers := api.Group("/customers")
	// customers.POST("", handleCreateCustomer(db, stripeProvider, eventBus, logger))
	// customers.GET("", handleGetCustomers(db, logger))
	// customers.GET("/:id", handleGetCustomerByID(db, logger))
	// customers.PUT("/:id", handleUpdateCustomer(db, stripeProvider, eventBus, logger))
	// customers.DELETE("/:id", handleDeleteCustomer(db, stripeProvider, eventBus, logger))
	// customers.GET("/by-email/:email", handleGetCustomerByEmail(db, logger))
	// customers.GET("/search", handleSearchCustomers(db, logger))
	// customers.POST("/:id/stripe", handleEnsureStripeCustomer(db, stripeProvider, eventBus, logger))
	// customers.GET("/stats", handleGetCustomerStats(db, logger))

	// Admin
	// admin := api.Group("/admin")
	// admin.GET("/orders", handleGetAllOrders(db, logger))
	// admin.PUT("/orders/:id/status", handleUpdateOrderStatus(db, eventBus, logger))
	// admin.GET("/orders/stats", handleGetOrderStats(db, logger))
	// admin.POST("/products", handleCreateProduct(db, eventBus, logger))
	// admin.PUT("/products/:id", handleUpdateProduct(db, eventBus, logger))

	// Variants
	// admin.POST("/variants", handleCreateVariant(db, eventBus, logger))
	// admin.GET("/variants/:id", handleGetVariant(db, logger))
	// admin.PUT("/variants/:id", handleUpdateVariant(db, eventBus, logger))
	// admin.DELETE("/variants/:id", handleArchiveVariant(db, eventBus, logger))
	// admin.POST("/variants/:id/activate", handleActivateVariant(db, eventBus, logger))
	// admin.POST("/variants/:id/deactivate", handleDeactivateVariant(db, eventBus, logger))

	// Options
	// admin.POST("/products/:product_id/options", handleCreateProductOption(db, eventBus, logger))
	// admin.GET("/products/:product_id/options", handleGetProductOptions(db, logger))
	// admin.GET("/options/:id", handleGetProductOption(db, logger))
	// admin.PUT("/options/:id", handleUpdateProductOption(db, eventBus, logger))
	// admin.DELETE("/options/:id", handleDeleteProductOption(db, eventBus, logger))
}

func run(
	ctx context.Context,
	args []string,
	getenv func(string) string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) error {
	// Signal handling
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	// Parse flags and config
	logLevel := getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	logFormat := getenv("LOG_FORMAT")
	if logFormat == "" {
		logFormat = "json"
	}

	port := getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Setup logger
	logger := customMiddleware.SetupLogger(logLevel, logFormat)

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config load failed: %w", err)
	}

	// Initialize database
	db, err := database.NewDB(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	defer db.Close()

	// Run migrations
	autoMigrate := getenv("ENV") == "development"
	if err := db.RunMigrations(autoMigrate); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	// Initialize event bus
	eventBus, err := provider.NewNATSEventPublisher(cfg.NATSUrl, logger)
	if err != nil {
		return fmt.Errorf("failed to create event publisher: %w", err)
	}
	defer eventBus.Close()

	// Initialize Stripe
	stripeProvider, err := provider.NewStripeProvider(
		cfg.StripeSecretKey, cfg.StripeWebhookSecret, eventBus, logger,
	)
	if err != nil {
		return fmt.Errorf("stripe provider initialization failed: %w", err)
	}

	// Start event subscribers (simplified)
	go startEventSubscribers(ctx, db, eventBus, logger)

	// Create server
	e := NewServer(logger, cfg, db, eventBus, stripeProvider)

	// Start server
	go func() {
		addr := ":" + port
		logger.Info().Str("port", port).Msg("starting server")
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("server error")
		}
	}()

	// Wait for shutdown
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logger.Info().Msg("shutting down server")
	if err := e.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown failed: %w", err)
	}

	return nil
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args, os.Getenv, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

// System handlers
func handleHealthCheck() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":  "healthy",
			"service": "freyja-ecommerce-api",
			"version": "1.0.0",
		})
	}
}

func handleHelloWorld() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "Welcome to Freyja E-commerce API!",
			"version": "1.0.0",
			"status":  "running",
		})
	}
}

// helpers
type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

func startEventSubscribers(ctx context.Context, db *database.DB, eventBus interfaces.EventPublisher, logger zerolog.Logger) {
	// Simplified event subscriber startup
	logger.Info().Msg("event subscribers started")
}
