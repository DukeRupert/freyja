package internal

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Config struct {
	env           string
	logLevel      string
	port          uint16
	databaseUrl   string
	sessionSecret string
	stripe        StripeConfig
	email         EmailConfig
}

type StripeConfig struct {
	secretKey     string
	webhookSecret string
}

type EmailConfig struct {
	host     string
	port     uint16
	username string
	password string
	from     string
}

var defaultConfig = Config{
	env:           "dev",
	logLevel:      "info",
	port:          3000,
	databaseUrl:   "postgres://freyja:password@localhost:5432/freyja?sslmode=disable",
	sessionSecret: "dev-secret-change-in-production",
	stripe: StripeConfig{
		secretKey:     "sk_test_your_key_here",
		webhookSecret: "whsec_your_webhook_secret_here",
	},
	email: EmailConfig{
		host:     "localhost",
		port:     1025,
		username: "",
		password: "",
		from:     "noreply@freyja.local",
	},
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
		env:           getEnv("ENV", "dev"),
		logLevel:      getEnv("LOG_LEVEL", "info"),
		port:          getEnvInt("PORT", 3000),
		databaseUrl:   getEnv("DATABASE_URL", "postgres://freyja:password@localhost:5432/freyja?sslmode=disable"),
		sessionSecret: getEnv("SESSION_SECRET", "dev-secret-change-in-production"),
		stripe: StripeConfig{
			secretKey:     getEnv("STRIPE_SECRET_KEY", "sk_test_your_key_here"),
			webhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", "whsec_your_webhook_secret_here"),
		},
		email: EmailConfig{
			host:     getEnv("SMTP_HOST", "localhost"),
			port:     getEnvInt("SMTP_PORT", 1025),
			username: getEnv("SMTP_USERNAME", ""),
			password: getEnv("SMTP_PASSWORD", ""),
			from:     getEnv("SMTP_FROM", "noreply@freyja.local"),
		},
	}

	// Validate env
	validEnv := cfg.env == "dev" || cfg.env == "prod"
	if !validEnv {
		slog.Default().Warn("Invalid environment. Using default: prod", slog.String("env", cfg.env))
		cfg.env = "prod"
	}
	
	// Validate log level
	validLevel := cfg.logLevel == "info" || cfg.logLevel == "debug" || cfg.logLevel == "warn" || cfg.logLevel == "error"
	if !validLevel {
		slog.Default().Warn("Invalid log level. Using default: info", slog.String("value", cfg.logLevel))
		cfg.logLevel = "info"
	}

	// Validate JWT secret in production
	if (cfg.env == "prod" || cfg.env == "production") && cfg.sessionSecret == "your-secret-key-change-in-production" {
		return nil, fmt.Errorf("SESSION_SECRET must be set in production environment")
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
