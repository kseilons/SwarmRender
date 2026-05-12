package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

func RunMigrations(ctx context.Context, connStr string, migrationsDir string) error {
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}
	defer conn.Close(ctx)

	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("unable to read migrations directory: %w", err)
	}

	var migrationFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".sql") {
			migrationFiles = append(migrationFiles, f.Name())
		}
	}
	sort.Strings(migrationFiles)

	for _, f := range migrationFiles {
		content, err := os.ReadFile(filepath.Join(migrationsDir, f))
		if err != nil {
			return fmt.Errorf("unable to read migration file %s: %w", f, err)
		}

		_, err = conn.Exec(ctx, string(content))
		if err != nil {
			return fmt.Errorf("error executing migration %s: %w", f, err)
		}
	}

	return nil
}
