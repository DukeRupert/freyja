// internal/database/migrations/embed.go
package migrations

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var EmbeddedMigrations embed.FS

// Config for migration behavior
type MigrationConfig struct {
	AutoMigrate bool
	Direction   string // "up", "down", "status"
}

func Run(db *sql.DB, config MigrationConfig) error {
	goose.SetBaseFS(EmbeddedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	switch config.Direction {
	case "up":
		return goose.Up(db, ".")
	case "down":
		return goose.Down(db, ".")
	case "status":
		return goose.Status(db, ".")
	default:
		if config.AutoMigrate {
			return goose.Up(db, ".")
		}
		return nil
	}
}

// GetVersion returns current migration version
func GetVersion(db *sql.DB) (int64, error) {
	goose.SetBaseFS(EmbeddedMigrations)
	if err := goose.SetDialect("postgres"); err != nil {
		return 0, err
	}
	return goose.GetDBVersion(db)
}
