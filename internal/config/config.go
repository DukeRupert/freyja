package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	DatabaseURL    string
	ValkeyAddr     string
	NATSUrl        string
	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOUseSSL    bool
	Port           string
	Environment    string
	
	// Stripe configuration
	StripeSecretKey      string
	StripePublishableKey string
	StripeWebhookSecret  string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:                 getEnv("PORT", "8080"),
		Environment:          getEnv("ENV", "development"),
		MinIOAccessKey:       getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey:       getEnv("MINIO_SECRET_KEY", "minioadmin123"),
		MinIOUseSSL:          getEnv("MINIO_USE_SSL", "false") == "true",
		
		// Stripe configuration
		StripeSecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
		StripePublishableKey: getEnv("STRIPE_PUBLISHABLE_KEY", ""),
		StripeWebhookSecret:  getEnv("STRIPE_WEBHOOK_SECRET", ""),
	}

	// Smart host detection
	isDocker := isRunningInDocker()

	// Database URL
	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		host := "localhost"
		if isDocker {
			host = "postgres"
		}
		cfg.DatabaseURL = fmt.Sprintf("postgres://postgres:password@%s:5432/coffee_ecommerce?sslmode=disable", host)
	}

	// Valkey/Redis
	cfg.ValkeyAddr = getEnv("VALKEY_ADDR", "localhost:6379")
	if isDocker && cfg.ValkeyAddr == "localhost:6379" {
		cfg.ValkeyAddr = "valkey:6379"
	}

	// NATS
	cfg.NATSUrl = getEnv("NATS_URL", "nats://localhost:4222")
	if isDocker && cfg.NATSUrl == "nats://localhost:4222" {
		cfg.NATSUrl = "nats://nats:4222"
	}

	// MinIO
	cfg.MinIOEndpoint = getEnv("MINIO_ENDPOINT", "localhost:9000")
	if isDocker && cfg.MinIOEndpoint == "localhost:9000" {
		cfg.MinIOEndpoint = "minio:9000"
	}

	return cfg, cfg.validate()
}

func (c *Config) validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	
	// Validate Stripe configuration for production
	if c.Environment == "production" {
		if c.StripeSecretKey == "" {
			return fmt.Errorf("STRIPE_SECRET_KEY is required in production")
		}
		if c.StripeWebhookSecret == "" {
			return fmt.Errorf("STRIPE_WEBHOOK_SECRET is required in production")
		}
		if !strings.HasPrefix(c.StripeSecretKey, "sk_live_") {
			return fmt.Errorf("production environment requires live Stripe keys")
		}
	}
	
	// For development, warn if Stripe keys are missing but don't fail
	if c.Environment == "development" {
		if c.StripeSecretKey == "" {
			fmt.Println("⚠️  Warning: STRIPE_SECRET_KEY not set - Stripe functionality will be disabled")
		}
		if c.StripeWebhookSecret == "" {
			fmt.Println("⚠️  Warning: STRIPE_WEBHOOK_SECRET not set - webhook verification will be disabled")
		}
	}
	
	return nil
}

// IsStripeConfigured returns true if Stripe is properly configured
func (c *Config) IsStripeConfigured() bool {
	return c.StripeSecretKey != "" && c.StripeWebhookSecret != ""
}

// IsStripeLiveMode returns true if using live Stripe keys
func (c *Config) IsStripeLiveMode() bool {
	return strings.HasPrefix(c.StripeSecretKey, "sk_live_")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func isRunningInDocker() bool {
	// Check for Docker environment indicators
	if os.Getenv("DOCKER_CONTAINER") == "true" {
		return true
	}

	// Check if we're in a container by looking at cgroup
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		return strings.Contains(string(data), "docker") || strings.Contains(string(data), "containerd")
	}

	// Check hostname (Docker containers often have random hostnames)
	if hostname, err := os.Hostname(); err == nil {
		// Docker compose containers often have predictable names
		if strings.Contains(hostname, "coffee-") || len(hostname) == 12 {
			return true
		}
	}

	return false
}