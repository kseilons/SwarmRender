package migrator

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func MustRun(storagePath, migrationsPath string, log *slog.Logger) {
	if migrationsPath == "" {
		panic("migrations path is empty")
	}

	absPath, err := filepath.Abs(migrationsPath)
	if err != nil {
		panic(fmt.Sprintf("failed to get absolute path for migrations: %v", err))
	}

	// На Windows пути могут содержать обратные слэши, их нужно заменить для URL
	absPath = filepath.ToSlash(absPath)

	var m *migrate.Migrate
	for attempt := 1; attempt <= 30; attempt++ {
		m, err = migrate.New(
			"file://"+absPath,
			storagePath,
		)
		if err == nil {
			break
		}
		log.Warn("waiting for postgres database to become ready...", slog.Int("attempt", attempt), slog.Any("error", err))
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		panic(fmt.Sprintf("failed to initialize migrator: %v", err))
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info("no migrations to apply")
			return
		}
		panic(fmt.Sprintf("failed to run migrations: %v", err))
	}

	log.Info("migrations applied successfully")
}
