// internal/database/database.go
package database

import (
	"database/sql"
	"fmt"

	"github.com/dukerupert/freyja/internal/database/migrations"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func New(url string) (*DB, error) {
	if url == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable required")
	}

	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{DB: db}, nil
}

func (db *DB) RunMigrations(autoMigrate bool) error {
	config := migrations.MigrationConfig{
		AutoMigrate: autoMigrate,
		Direction:   "up",
	}

	return migrations.Run(db.DB, config)
}

func (db *DB) MigrationStatus() error {
	config := migrations.MigrationConfig{
		Direction: "status",
	}

	return migrations.Run(db.DB, config)
}
