package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/dukerupert/freyja/internal"
	"github.com/dukerupert/freyja/internal/handler/saas"
	"github.com/dukerupert/freyja/internal/router"
	"github.com/dukerupert/freyja/internal/routes"
)

func run() error {
	// Load configuration (only need env and port for SaaS)
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	port := os.Getenv("SAAS_PORT")
	if port == "" {
		port = "3001"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	// Configure logger
	logger := internal.NewLogger(os.Stdout, env, logLevel)

	// Initialize SaaS page handler
	logger.Info("Loading SaaS templates...")
	saasHandler, err := saas.NewPageHandler("web/templates")
	if err != nil {
		return fmt.Errorf("failed to initialize saas handler: %w", err)
	}
	logger.Info("SaaS templates loaded successfully")

	// Build route dependencies
	saasDeps := routes.SaaSDeps{
		Handler: saasHandler,
	}

	// Create router
	r := router.New(
		router.Recovery(logger),
		router.Logger(logger),
	)

	// Static files
	r.Static("/static/", "./web/static")

	// Register SaaS routes
	routes.RegisterSaaSRoutes(r, saasDeps)

	// Start server
	addr := fmt.Sprintf(":%s", port)
	logger.Info("Starting SaaS marketing server", "address", addr)

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
