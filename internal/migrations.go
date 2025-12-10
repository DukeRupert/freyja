package internal

import (
	"database/sql"
	"fmt"

	"github.com/dukerupert/hiri/migrations"
	"github.com/pressly/goose/v3"
)

// RunMigrations executes all pending database migrations
func RunMigrations(db *sql.DB) error {
	goose.SetBaseFS(migrations.MigrationsFS)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
