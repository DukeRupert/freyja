package internal

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Config struct {
	Env           string
	LogLevel      string
	Port          uint16
	DatabaseUrl   string
	TenantID      string
	SessionSecret string
	BaseURL       string
	EncryptionKey string // Base64-encoded 32-byte key for encrypting provider credentials
	Stripe        StripeConfig
	Email         EmailConfig
	Admin         AdminConfig
	Storage       StorageConfig
}

// AdminConfig contains initial admin user configuration.
// These values are only used on first startup to create the admin user.
type AdminConfig struct {
	Email     string
	Password  string
	FirstName string
	LastName  string
}

type StripeConfig struct {
	SecretKey      string
	PublishableKey string
	WebhookSecret  string
}

type EmailConfig struct {
	Host          string
	Port          uint16
	Username      string
	Password      string
	From          string
	FromName      string
	PostmarkToken string
}

type StorageConfig struct {
	Provider      string // "local" or "r2"
	LocalPath     string
	LocalURL      string
	R2AccountID   string
	R2AccessKeyID string
	R2SecretKey   string
	R2BucketName  string
	R2PublicURL   string
}

func NewConfig() (*Config, error) {
	// Try to load .env from current directory, then walk up to find it (max 2 levels)
	err := godotenv.Load()
	if err != nil {
		// Walk up directories to find .env (max 2 parent directories)
		dir, _ := os.Getwd()
		found := false
		for i := 0; i < 2; i++ {
			dir = filepath.Join(dir, "..")
			if err := godotenv.Load(filepath.Join(dir, ".env")); err == nil {
				found = true
				break
			}
		}
		if !found {
			slog.Default().Warn("Warning: .env file not found, using environment variables and defaults")
		}
	}

	cfg := &Config{
		Env:           getEnv("ENV", "dev"),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
		Port:          getEnvInt("PORT", 3000),
		DatabaseUrl:   getEnv("DATABASE_URL", "postgres://freyja:password@localhost:5432/freyja?sslmode=disable"),
		TenantID:      getEnv("TENANT_ID", "00000000-0000-0000-0000-000000000001"),
		SessionSecret: getEnv("SESSION_SECRET", "dev-secret-change-in-production"),
		BaseURL:       getEnv("BASE_URL", "http://localhost:3000"),
		EncryptionKey: getEnv("ENCRYPTION_KEY", ""), // Must be set in production
		Stripe: StripeConfig{
			SecretKey:      getEnv("STRIPE_SECRET_KEY", "sk_test_your_key_here"),
			PublishableKey: getEnv("STRIPE_PUBLISHABLE_KEY", "pk_test_your_key_here"),
			WebhookSecret:  getEnv("STRIPE_WEBHOOK_SECRET", "whsec_your_webhook_secret_here"),
		},
		Email: EmailConfig{
			Host:          getEnv("SMTP_HOST", "localhost"),
			Port:          getEnvInt("SMTP_PORT", 1025),
			Username:      getEnv("SMTP_USERNAME", ""),
			Password:      getEnv("SMTP_PASSWORD", ""),
			From:          getEnv("SMTP_FROM", "noreply@freyja.local"),
			FromName:      getEnv("EMAIL_FROM_NAME", "Freyja Coffee"),
			PostmarkToken: getEnv("POSTMARK_API_TOKEN", ""),
		},
		Admin: AdminConfig{
			Email:     getEnv("FREYJA_ADMIN_EMAIL", ""),
			Password:  getEnv("FREYJA_ADMIN_PASSWORD", ""),
			FirstName: getEnv("FREYJA_ADMIN_FIRST_NAME", ""),
			LastName:  getEnv("FREYJA_ADMIN_LAST_NAME", ""),
		},
		Storage: StorageConfig{
			Provider:      getEnv("STORAGE_PROVIDER", "local"),
			LocalPath:     getEnv("LOCAL_STORAGE_PATH", "./web/static/uploads"),
			LocalURL:      getEnv("LOCAL_STORAGE_URL", "/uploads"),
			R2AccountID:   getEnv("R2_ACCOUNT_ID", ""),
			R2AccessKeyID: getEnv("R2_ACCESS_KEY_ID", ""),
			R2SecretKey:   getEnv("R2_SECRET_ACCESS_KEY", ""),
			R2BucketName:  getEnv("R2_BUCKET_NAME", ""),
			R2PublicURL:   getEnv("R2_PUBLIC_URL", ""),
		},
	}

	// Validate env
	validEnv := cfg.Env == "dev" || cfg.Env == "prod"
	if !validEnv {
		slog.Default().Warn("Invalid environment. Using default: prod", slog.String("env", cfg.Env))
		cfg.Env = "prod"
	}

	// Validate log level
	validLevel := cfg.LogLevel == "info" || cfg.LogLevel == "debug" || cfg.LogLevel == "warn" || cfg.LogLevel == "error"
	if !validLevel {
		slog.Default().Warn("Invalid log level. Using default: info", slog.String("value", cfg.LogLevel))
		cfg.LogLevel = "info"
	}

	// Validate JWT secret in production
	if (cfg.Env == "prod" || cfg.Env == "production") && cfg.SessionSecret == "your-secret-key-change-in-production" {
		return nil, fmt.Errorf("SESSION_SECRET must be set in production environment")
	}

	// Validate R2 configuration in production
	if cfg.Env == "prod" && cfg.Storage.Provider == "r2" {
		if cfg.Storage.R2AccountID == "" {
			return nil, fmt.Errorf("R2_ACCOUNT_ID required when using R2 storage in production")
		}
		if cfg.Storage.R2AccessKeyID == "" || cfg.Storage.R2SecretKey == "" {
			return nil, fmt.Errorf("R2 credentials required when using R2 storage in production")
		}
		if cfg.Storage.R2BucketName == "" {
			return nil, fmt.Errorf("R2_BUCKET_NAME required when using R2 storage in production")
		}
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue uint16) uint16 {
	if value := os.Getenv(key); value != "" {
		var intValue uint16
		if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
			return intValue
		}
	}
	return defaultValue
}
