package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"

	"github.com/dukerupert/freyja/internal"
	"github.com/dukerupert/freyja/internal/handler/storefront"
	"github.com/dukerupert/freyja/internal/repository"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type (
	Middleware func(http.Handler) http.Handler
	Router     struct {
		*http.ServeMux
		chain []Middleware
	}
)

func NewRouter(mx ...Middleware) *Router {
	return &Router{ServeMux: &http.ServeMux{}, chain: mx}
}

func (r *Router) Use(mx ...Middleware) {
	r.chain = append(r.chain, mx...)
}

func (r *Router) Group(fn func(r *Router)) {
	fn(&Router{ServeMux: r.ServeMux, chain: slices.Clone(r.chain)})
}

func (r *Router) Get(path string, fn http.HandlerFunc, mx ...Middleware) {
	r.handle(http.MethodGet, path, fn, mx)
}

func (r *Router) Post(path string, fn http.HandlerFunc, mx ...Middleware) {
	r.handle(http.MethodPost, path, fn, mx)
}

func (r *Router) Put(path string, fn http.HandlerFunc, mx ...Middleware) {
	r.handle(http.MethodPut, path, fn, mx)
}

func (r *Router) Delete(path string, fn http.HandlerFunc, mx ...Middleware) {
	r.handle(http.MethodDelete, path, fn, mx)
}

func (r *Router) Head(path string, fn http.HandlerFunc, mx ...Middleware) {
	r.handle(http.MethodHead, path, fn, mx)
}

func (r *Router) Options(path string, fn http.HandlerFunc, mx ...Middleware) {
	r.handle(http.MethodOptions, path, fn, mx)
}

func (r *Router) handle(method, path string, fn http.HandlerFunc, mx []Middleware) {
	r.Handle(method+" "+path, r.wrap(fn, mx))
}

func (r *Router) wrap(fn http.HandlerFunc, mx []Middleware) (out http.Handler) {
	out, mx = http.Handler(fn), append(slices.Clone(r.chain), mx...)

	slices.Reverse(mx)

	for _, m := range mx {
		out = m(out)
	}

	return
}

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

	// Initialize handlers
	// Note: templates will be nil for now (will be added when we create templates)
	productListHandler := storefront.NewProductListHandler(productService, nil)
	productDetailHandler := storefront.NewProductDetailHandler(productService, nil)
	cartViewHandler := storefront.NewCartViewHandler(cartService, nil, cfg.Env != "development")
	addToCartHandler := storefront.NewAddToCartHandler(cartService, nil, cfg.Env != "development")
	updateCartItemHandler := storefront.NewUpdateCartItemHandler(cartService, nil)
	removeCartItemHandler := storefront.NewRemoveCartItemHandler(cartService, nil)

	// Initialize router
	r := NewRouter()

	// Storefront routes
	r.Get("/products", productListHandler.ServeHTTP)
	r.Get("/products/{slug}", productDetailHandler.ServeHTTP)
	r.Get("/cart", cartViewHandler.ServeHTTP)
	r.Post("/cart/add", addToCartHandler.ServeHTTP)
	r.Post("/cart/update", updateCartItemHandler.ServeHTTP)
	r.Post("/cart/remove", removeCartItemHandler.ServeHTTP)

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
