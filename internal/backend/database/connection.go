// internal/backend/database/connection.go
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/dukerupert/freyja/internal/database"
)

// SimplifiedDB holds just what the backend needs for read operations
type SimplifiedDB struct {
	conn    *pgx.Conn
	Queries *database.Queries
}

// NewSimplifiedDB creates a simplified database connection for backend read operations
func NewSimplifiedDB(databaseURL string) (*SimplifiedDB, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable required")
	}

	// Create pgx connection
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

	// Create SQLC queries instance
	queries := database.New(conn)

	return &SimplifiedDB{
		conn:    conn,
		Queries: queries,
	}, nil
}

// Close closes the database connection
func (db *SimplifiedDB) Close() error {
	return db.conn.Close(context.Background())
}

// GetQueries returns the queries instance for direct access
func (db *SimplifiedDB) GetQueries() *database.Queries {
	return db.Queries
}