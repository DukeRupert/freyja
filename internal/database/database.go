// internal/database/database.go
package database

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	"github.com/dukerupert/freyja/internal/database/migrations"
	"github.com/jackc/pgx/v5"

	_ "github.com/lib/pq" // for migrations only
)

type DB struct {
	conn    *pgx.Conn
	sqlDB   *sql.DB // Keep for migrations
	Queries *Queries
}

func NewDB(databaseURL string) (*DB, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable required")
	}

	// Parse the database URL to extract components
	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Extract database name from path
	databaseName := strings.TrimPrefix(parsedURL.Path, "/")
	if databaseName == "" {
		return nil, fmt.Errorf("database name not found in URL")
	}

	// Create database if it doesn't exist
	if err := ensureDatabaseExists(parsedURL, databaseName); err != nil {
		return nil, fmt.Errorf("failed to ensure database exists: %w", err)
	}

	// Create pgx connection for main operations
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test the connection
	if err := conn.Ping(ctx); err != nil {
		conn.Close(ctx)
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create standard sql.DB for migrations (goose compatibility)
	sqlDB, err := sql.Open("postgres", databaseURL)
	if err != nil {
		conn.Close(ctx)
		return nil, fmt.Errorf("failed to open sql database for migrations: %w", err)
	}

	// Create SQLC queries instance
	queries := New(conn)

	return &DB{
		conn:    conn,
		sqlDB:   sqlDB,
		Queries: queries,
	}, nil
}

// ensureDatabaseExists connects to PostgreSQL and creates the database if it doesn't exist
func ensureDatabaseExists(parsedURL *url.URL, databaseName string) error {
	// Create connection URL to postgres database (default db for admin operations)
	adminURL := *parsedURL
	adminURL.Path = "/postgres"
	
	ctx := context.Background()
	
	// Connect to postgres database to check/create target database
	adminConn, err := pgx.Connect(ctx, adminURL.String())
	if err != nil {
		return fmt.Errorf("failed to connect to postgres database: %w", err)
	}
	defer adminConn.Close(ctx)

	// Check if database exists
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)"
	err = adminConn.QueryRow(ctx, query, databaseName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if database exists: %w", err)
	}

	// Create database if it doesn't exist
	if !exists {
		// Note: Database names cannot be parameterized, so we need to validate and sanitize
		if err := validateDatabaseName(databaseName); err != nil {
			return fmt.Errorf("invalid database name: %w", err)
		}
		
		createQuery := fmt.Sprintf("CREATE DATABASE %s", pgx.Identifier{databaseName}.Sanitize())
		_, err = adminConn.Exec(ctx, createQuery)
		if err != nil {
			return fmt.Errorf("failed to create database %s: %w", databaseName, err)
		}
		
		fmt.Printf("Database '%s' created successfully\n", databaseName)
	} else {
		fmt.Printf("Database '%s' already exists\n", databaseName)
	}

	return nil
}

// validateDatabaseName ensures the database name is safe to use in SQL
func validateDatabaseName(name string) error {
	if name == "" {
		return fmt.Errorf("database name cannot be empty")
	}
	
	// Check for basic SQL injection patterns and invalid characters
	if strings.ContainsAny(name, "';\"\\-/*") {
		return fmt.Errorf("database name contains invalid characters")
	}
	
	// Check length (PostgreSQL limit is 63 characters)
	if len(name) > 63 {
		return fmt.Errorf("database name too long (max 63 characters)")
	}
	
	// Must start with letter or underscore
	if name[0] >= '0' && name[0] <= '9' {
		return fmt.Errorf("database name cannot start with a number")
	}
	
	return nil
}

func (db *DB) Close() {
	if db.conn != nil {
		db.conn.Close(context.Background())
	}
	if db.sqlDB != nil {
		db.sqlDB.Close()
	}
}

func (db *DB) Conn() *pgx.Conn {
	return db.conn
}

func (db *DB) RunMigrations(autoMigrate bool) error {
	config := migrations.MigrationConfig{
		AutoMigrate: autoMigrate,
		Direction:   "up",
	}

	// Use sql.DB for migrations
	return migrations.Run(db.sqlDB, config)
}

func (db *DB) MigrationStatus() error {
	config := migrations.MigrationConfig{
		Direction: "status",
	}

	// Use sql.DB for migrations
	return migrations.Run(db.sqlDB, config)
}