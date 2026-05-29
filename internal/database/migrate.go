package database

import (
	"errors"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"money-tracker-service/internal/config"
)

func RunMigrations(cfg config.Config, path string) error {
	m, err := migrate.New("file://"+path, cfg.MigrationDatabaseURL())
	if err != nil {
		return err
	}
	err = m.Up()
	if errors.Is(err, migrate.ErrNoChange) {
		return nil
	}
	return err
}
