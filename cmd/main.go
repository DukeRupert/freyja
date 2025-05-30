// cmd/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	custommiddleware "github.com/dukerupert/freyja/internal/middleware"

	"github.com/dukerupert/freyja/internal/api"
	"github.com/dukerupert/freyja/internal/config"
	"github.com/dukerupert/freyja/internal/handler"
	"github.com/dukerupert/freyja/internal/repo"
	"github.com/dukerupert/freyja/internal/service"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	strictecho "github.com/oapi-codegen/runtime/strictmiddleware/echo"
	"github.com/rs/zerolog"
)

func init() {
	// UNIX Time is faster and smaller than most timestamps
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
}

func main() {
	// Initialize logger
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: zerolog.TimeFormatUnix}).With().Timestamp().Logger()

	debug := flag.Bool("debug", false, "sets log level to debug")
	flag.Parse()
	// Default level for this example is info, unless debug flag is present
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Initialize database connection
	db, err := pgxpool.New(context.Background(), cfg.DB.DSN)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(context.Background()); err != nil {
		logger.Fatal().Err(err).Msg("Failed to ping database")
	}
	logger.Info().Msg("Database connection established")

	// Initialize repositories and services
	queries := repo.New(db)
	productService := service.NewProductService(queries)
	productHandler := handler.NewProductHandler(productService)

	// Initialize Echo server
	e := echo.New()

	// Add middleware
	e.Use(custommiddleware.RequestLogger(&logger))
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Health check endpoint
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status": "healthy",
			"service": "freyja",
		})
	})

	// Create strict server interface
	strictHandler := api.NewStrictHandler(productHandler, []strictecho.StrictEchoMiddlewareFunc{})

	// Register API routes with base path
	api.RegisterHandlersWithBaseURL(e, strictHandler, "/api/v1")

	// Start server
	logger.Info().Int("port", cfg.App.Port).Msg("Starting server")
	if err := e.Start(fmt.Sprintf(":%d", cfg.App.Port)); err != nil && err != http.ErrServerClosed {
		logger.Fatal().Err(err).Msg("Failed to start server")
	}
}

// Example of how to use the service directly (for testing or other purposes)
func ExampleDirectUsage() {
	// This is just an example of how you might use the service layer directly
	
	// Initialize database connection (same as above)
	cfg, _ := config.Load()
	db, _ := pgxpool.New(context.Background(), cfg.DB.DSN)
	defer db.Close()

	// Initialize service
	queries := repo.New(db)
	productService := service.NewProductService(queries)

	// Create a product
	createReq := api.CreateProductRequest{
		Title:  "Ethiopian Yirgacheffe",
		Handle: "ethiopian-yirgacheffe",
		// Add other fields as needed
	}

	ctx := context.Background()
	product, err := productService.CreateProduct(ctx, createReq)
	if err != nil {
		// Handle error
		return
	}

	// Use the created product
	_ = product
}