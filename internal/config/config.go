// config/config.go
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	App        AppConfig
	DB         DBConfig
	Stripe     StripeConfig
	JWT        JWTConfig
	MessageBus MessageBusConfig
}

// AppConfig holds application-specific configuration
type AppConfig struct {
	Name  string
	Env   string
	Port  int
	Debug bool
}

// DBConfig holds database configuration
type DBConfig struct {
	Host       string
	Port       int
	Name       string
	User       string
	Password   string
	SSLMode    string
	DSN        string
	MigrateURL string
}

// StripeConfig holds Stripe API configuration
type StripeConfig struct {
	SecretKey     string
	WebhookSecret string
}

// JWTConfig holds JWT authentication configuration
type JWTConfig struct {
	Secret     string
	Expiration string
}

type MessageBusConfig struct {
	URL       string
	Username  string
	Password  string
	Namespace string // prefix for all topics
}

// Load loads configuration from environment variables
func Load(path string) (*Config, error) {
	// Load .env file if it exists
	godotenv.Load(path)

	cfg := &Config{
		App: AppConfig{
			Name:  getEnv("APP_NAME", "freyja"),
			Env:   getEnv("APP_ENV", "development"),
			Port:  getEnvAsIntWithValidation("APP_PORT", 8080, 1, 65535),
			Debug: getEnvAsBool("APP_DEBUG", true),
		},
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsIntWithValidation("DB_PORT", 5432, 1, 65535),
			Name:     getEnv("DB_NAME", "coffee_subscriptions"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		Stripe: StripeConfig{
			SecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
			WebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", "your_jwt_secret_key"),
			Expiration: getEnv("JWT_EXPIRATION", "24h"),
		},
		MessageBus: MessageBusConfig{
			URL:       getEnv("NATS_URL", "nats://localhost:4222"),
			Username:  getEnv("NATS_USERNAME", ""),
			Password:  getEnv("NATS_PASSWORD", ""),
			Namespace: getEnv("NATS_NAMESPACE", "walkingdrum"),
		},
	}

	// Validate configuration before proceeding
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	// Construct database connection string only after validation passes
	cfg.DB.DSN = fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.DB.Host, cfg.DB.Port, cfg.DB.User, cfg.DB.Password, cfg.DB.Name, cfg.DB.SSLMode,
	)

	// Construct URL-formatted connection string for golang-migrate
	cfg.DB.MigrateURL = fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.Name, cfg.DB.SSLMode,
	)

	// Only show debug info in development
	if cfg.App.Debug && cfg.App.Env == "development" {
		fmt.Println("====== Database Configuration ======")
		fmt.Println("DSN:", cfg.DB.DSN)
		fmt.Println("MigrateURL:", cfg.DB.MigrateURL)
		fmt.Println("===================================")
	}

	return cfg, nil
}

// validate checks if all required configuration is present
func (c *Config) validate() error {
	var errors []string

	// Validate essential app configuration
	if c.App.Name == "" {
		errors = append(errors, "APP_NAME is required")
	}
	
	if c.App.Port <= 0 || c.App.Port > 65535 {
		errors = append(errors, fmt.Sprintf("APP_PORT must be between 1 and 65535, got: %d", c.App.Port))
	}

	// Validate database configuration
	if c.DB.Host == "" {
		errors = append(errors, "DB_HOST is required")
	}
	
	if c.DB.Port <= 0 || c.DB.Port > 65535 {
		errors = append(errors, fmt.Sprintf("DB_PORT must be between 1 and 65535, got: %d", c.DB.Port))
	}
	
	if c.DB.Name == "" {
		errors = append(errors, "DB_NAME is required")
	} else {
		// Verify that the provided database name is valid
		valid, msg := isValidPostgresIdentifier(c.DB.Name)
		if !valid {
			errors = append(errors, fmt.Sprintf("invalid database name '%s': %s", c.DB.Name, msg))
		}
	}
	
	if c.DB.User == "" {
		errors = append(errors, "DB_USER is required")
	}
	
	// DB_PASSWORD can be empty for some setups (like peer authentication), so we'll just warn
	if c.DB.Password == "" && c.App.Env == "production" {
		errors = append(errors, "DB_PASSWORD should be set in production for security")
	}

	// Validate environment-specific requirements
	switch c.App.Env {
	case "production":
		// Production-specific validations
		if c.Stripe.SecretKey == "" {
			errors = append(errors, "STRIPE_SECRET_KEY is required in production")
		}
		if c.Stripe.WebhookSecret == "" {
			errors = append(errors, "STRIPE_WEBHOOK_SECRET is required in production")
		}
		if c.JWT.Secret == "" || c.JWT.Secret == "your_jwt_secret_key" {
			errors = append(errors, "JWT_SECRET must be set to a secure value in production")
		}
		if len(c.JWT.Secret) < 32 {
			errors = append(errors, "JWT_SECRET should be at least 32 characters long in production")
		}
		if c.MessageBus.URL == "" {
			errors = append(errors, "NATS_URL is required in production")
		}
	case "development", "dev":
		// Development-specific validations (more lenient)
		if c.JWT.Secret == "" {
			errors = append(errors, "JWT_SECRET is required even in development")
		}
	case "test", "testing":
		// Test-specific validations
		if c.JWT.Secret == "" {
			c.JWT.Secret = "test_jwt_secret_key_32_characters_long" // Set default for tests
		}
	default:
		// Unknown environment
		errors = append(errors, fmt.Sprintf("unknown environment '%s', expected: production, development, or test", c.App.Env))
	}

	// Validate JWT configuration
	if c.JWT.Expiration == "" {
		errors = append(errors, "JWT_EXPIRATION is required")
	} else {
		// Validate that expiration is a valid duration
		if _, err := time.ParseDuration(c.JWT.Expiration); err != nil {
			errors = append(errors, fmt.Sprintf("JWT_EXPIRATION must be a valid duration (e.g., '24h', '30m'), got: %s", c.JWT.Expiration))
		}
	}

	// Validate MessageBus configuration
	if c.MessageBus.URL != "" {
		// Basic URL validation
		if !strings.HasPrefix(c.MessageBus.URL, "nats://") {
			errors = append(errors, "NATS_URL must start with 'nats://'")
		}
	}

	// Return all validation errors at once
	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

// validateDuration checks if a string is a valid time duration
func validateDuration(duration string) error {
	_, err := time.ParseDuration(duration)
	return err
}

// Additional helper function to validate URL format
func isValidURL(urlStr string) bool {
	if urlStr == "" {
		return false
	}
	_, err := url.Parse(urlStr)
	return err == nil
}

// Enhanced getEnvAsInt with validation
func getEnvAsIntWithValidation(key string, defaultValue int, min int, max int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	
	if value, err := strconv.Atoi(valueStr); err == nil {
		if value < min || value > max {
			// Log warning but return default
			fmt.Printf("Warning: %s value %d is out of range [%d, %d], using default %d\n", key, value, min, max, defaultValue)
			return defaultValue
		}
		return value
	}
	
	fmt.Printf("Warning: %s value '%s' is not a valid integer, using default %d\n", key, valueStr, defaultValue)
	return defaultValue
}

// Helper functions to get environment variables with default values
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}

// IsValidPostgresIdentifier checks if the provided name is a valid PostgreSQL identifier
// according to PostgreSQL naming rules.
func isValidPostgresIdentifier(name string) (bool, string) {
	// Handle empty name
	if name == "" {
		return false, "identifier cannot be empty"
	}

	// Check if the name is quoted
	if strings.HasPrefix(name, "\"") && strings.HasSuffix(name, "\"") {
		// For quoted identifiers, we need to:
		// 1. Remove the surrounding quotes
		// 2. Check for any embedded double quotes (they must be escaped as "")
		// 3. Check if the resulting name is not empty

		// Remove surrounding quotes
		unquotedName := name[1 : len(name)-1]

		// Check for proper escaping of embedded quotes
		for i := 0; i < len(unquotedName); i++ {
			if unquotedName[i] == '"' {
				// If this is the last character or the next character is not a double quote
				if i == len(unquotedName)-1 || unquotedName[i+1] != '"' {
					return false, "embedded double quote in identifier must be escaped by doubling"
				}
				// Skip the next quote (the escape)
				i++
			}
		}

		// Check if the unquoted name is empty
		if len(unquotedName) == 0 {
			return false, "quoted identifier cannot be empty"
		}

		// Check length (after removing quotes and handling escaped quotes)
		// Note: This is simplified; a proper implementation would count "" as a single character
		if len(unquotedName) > 31 {
			return false, "identifier too long (maximum is 31 characters)"
		}

		return true, ""
	}

	// For unquoted identifiers
	// Check if first character is a letter or underscore
	if len(name) == 0 || (!unicode.IsLetter(rune(name[0])) && name[0] != '_') {
		return false, "identifier must begin with a letter or underscore"
	}

	// Check subsequent characters
	for i := 1; i < len(name); i++ {
		ch := rune(name[i])
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '_' {
			return false, fmt.Sprintf("identifier contains invalid character: %c", ch)
		}
	}

	// Check length
	if len(name) > 31 {
		return false, "identifier too long (maximum is 31 characters)"
	}

	// Check if it's a reserved keyword (simplified - would need a comprehensive list)
	keywords := map[string]bool{
		"select": true, "from": true, "where": true, "insert": true,
		"update": true, "delete": true, "create": true, "drop": true,
		"table": true, "index": true, "view": true, "sequence": true,
		"trigger": true, "function": true, "procedure": true, "schema": true,
		"database": true, "in": true, "between": true, "like": true,
		"and": true, "or": true, "not": true, "null": true, "true": true, "false": true,
	}

	if keywords[strings.ToLower(name)] {
		return false, fmt.Sprintf("%s is a reserved keyword", name)
	}

	return true, ""
}