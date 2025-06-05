package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
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
	// Load .env file first
    if err := godotenv.Load(); err != nil {
        fmt.Println("No .env file found")
    }
	
	// Initialize Viper
	viper.AutomaticEnv()
	
	// Set all defaults
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("ENV", "development")
	viper.SetDefault("MINIO_ACCESS_KEY", "minioadmin")
	viper.SetDefault("MINIO_SECRET_KEY", "minioadmin123")
	viper.SetDefault("MINIO_USE_SSL", false)
	viper.SetDefault("STRIPE_SECRET_KEY", "")
	viper.SetDefault("STRIPE_PUBLISHABLE_KEY", "")
	viper.SetDefault("STRIPE_WEBHOOK_SECRET", "")
	viper.SetDefault("VALKEY_ADDR", "localhost:6379")
	viper.SetDefault("NATS_URL", "nats://localhost:4222")
	viper.SetDefault("MINIO_ENDPOINT", "localhost:9000")

	// Smart host detection
	isDocker := isRunningInDocker()

	// Set Docker-aware defaults
	if isDocker {
		viper.SetDefault("VALKEY_ADDR", "valkey:6379")
		viper.SetDefault("NATS_URL", "nats://nats:4222")
		viper.SetDefault("MINIO_ENDPOINT", "minio:9000")
	}

	// Build config struct
	cfg := &Config{
		Port:                 viper.GetString("PORT"),
		Environment:          viper.GetString("ENV"),
		MinIOAccessKey:       viper.GetString("MINIO_ACCESS_KEY"),
		MinIOSecretKey:       viper.GetString("MINIO_SECRET_KEY"),
		MinIOUseSSL:          viper.GetBool("MINIO_USE_SSL"),
		StripeSecretKey:      viper.GetString("STRIPE_SECRET_KEY"),
		StripePublishableKey: viper.GetString("STRIPE_PUBLISHABLE_KEY"),
		StripeWebhookSecret:  viper.GetString("STRIPE_WEBHOOK_SECRET"),
		ValkeyAddr:           viper.GetString("VALKEY_ADDR"),
		NATSUrl:              viper.GetString("NATS_URL"),
		MinIOEndpoint:        viper.GetString("MINIO_ENDPOINT"),
	}

	// Handle DATABASE_URL with smart defaults
	cfg.DatabaseURL = viper.GetString("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		host := "localhost"
		if isDocker {
			host = "postgres"
		}
		cfg.DatabaseURL = fmt.Sprintf("postgres://postgres:password@%s:5432/coffee_ecommerce?sslmode=disable", host)
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