// internal/database/database.go
package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/dukerupert/freyja/internal/database/migrations"
	"github.com/jackc/pgx/v5"

	_ "github.com/lib/pq" // for migrations only
)

type DB struct {
	conn    *pgx.Conn
	sqlDB   *sql.DB // Keep for migrations
	Queries *Queries
}

func NewDB(url string) (*DB, error) {
	if url == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable required")
	}

	// Create pgx connection for main operations
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test the connection
	if err := conn.Ping(ctx); err != nil {
		conn.Close(ctx)
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Create standard sql.DB for migrations (goose compatibility)
	sqlDB, err := sql.Open("postgres", url)
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
