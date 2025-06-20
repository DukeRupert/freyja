package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	// Database configuration
	DatabaseURL    string
	DatabaseHost   string
	DatabasePort   string
	DatabaseName   string
	DatabaseUser   string
	DatabasePassword string
	DatabaseSSLMode string
	
	ValkeyAddr     string
	NATSUrl        string
	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOUseSSL    bool
	Domain		 string
	Port           string
	Environment    string
	
	// Stripe configuration
	StripeSecretKey      string
	StripePublishableKey string
	StripeWebhookSecret  string

	// Admin configuration
	AdminDomain	string
	ApiVersion 	string
	ApiURL string
}

func Load() (*Config, error) {
	// Load .env file first
    if err := godotenv.Load(); err != nil {
        fmt.Println("No .env file found")
    }
	
	// Initialize Viper
	viper.AutomaticEnv()
	
	// Set all defaults
	viper.SetDefault("DOMAIN", "localhost")
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
	viper.SetDefault("ADMIN_DOMAIN", "localhost:8081")
	viper.SetDefault("API_VERSION", "v1")
	
	// Database defaults
	viper.SetDefault("DATABASE_HOST", "localhost")
	viper.SetDefault("DATABASE_PORT", "5432")
	viper.SetDefault("DATABASE_NAME", "coffee_ecommerce")
	viper.SetDefault("DATABASE_USER", "postgres")
	viper.SetDefault("DATABASE_PASSWORD", "password")
	viper.SetDefault("DATABASE_SSL_MODE", "disable")

	// Smart host detection
	isDocker := isRunningInDocker()

	// Set Docker-aware defaults
	if isDocker {
		viper.SetDefault("VALKEY_ADDR", "valkey:6379")
		viper.SetDefault("NATS_URL", "nats://nats:4222")
		viper.SetDefault("MINIO_ENDPOINT", "minio:9000")
		viper.SetDefault("DATABASE_HOST", "postgres")
	}

	// Build config struct
	cfg := &Config{
		Domain:				viper.GetString("DOMAIN"),
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
		AdminDomain: viper.GetString("ADMIN_DOMAIN"),
		ApiVersion: viper.GetString("API_VERSION"),
		
		// Database configuration
		DatabaseHost:     viper.GetString("DATABASE_HOST"),
		DatabasePort:     viper.GetString("DATABASE_PORT"),
		DatabaseName:     viper.GetString("DATABASE_NAME"),
		DatabaseUser:     viper.GetString("DATABASE_USER"),
		DatabasePassword: viper.GetString("DATABASE_PASSWORD"),
		DatabaseSSLMode:  viper.GetString("DATABASE_SSL_MODE"),
	}

	// Handle DATABASE_URL - if provided, use it directly; otherwise construct from components
	cfg.DatabaseURL = viper.GetString("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		cfg.DatabaseURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			cfg.DatabaseUser,
			cfg.DatabasePassword,
			cfg.DatabaseHost,
			cfg.DatabasePort,
			cfg.DatabaseName,
			cfg.DatabaseSSLMode,
		)
	}
	
	cfg.ApiURL = cfg.Domain + "api/" + cfg.ApiVersion

	return cfg, cfg.validate()
}

func (c *Config) validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	
	// Validate individual database components if DATABASE_URL is constructed
	if viper.GetString("DATABASE_URL") == "" {
		if c.DatabaseHost == "" {
			return fmt.Errorf("DATABASE_HOST is required")
		}
		if c.DatabasePort == "" {
			return fmt.Errorf("DATABASE_PORT is required")
		}
		if c.DatabaseName == "" {
			return fmt.Errorf("DATABASE_NAME is required")
		}
		if c.DatabaseUser == "" {
			return fmt.Errorf("DATABASE_USER is required")
		}
		if c.DatabasePassword == "" {
			return fmt.Errorf("DATABASE_PASSWORD is required")
		}
	}
	
	// Validate Stripe configuration for production
	if c.Environment == "production" {
		if c.StripeSecretKey == "" {
			return fmt.Errorf("STRIPE_SECRET_KEY is required in production")
		}
		if c.StripeWebhookSecret == "" {
			return fmt.Errorf("STRIPE_WEBHOOK_SECRET is required in production")
		}
	}
	
	return nil
}

func isRunningInDocker() bool {
	// Check if we're running in a Docker container
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	
	// Alternative check: look for docker in cgroup
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		return strings.Contains(string(data), "docker")
	}
	
	return false
}