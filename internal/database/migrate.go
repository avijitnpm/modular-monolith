package database

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrate runs all SQL migration files from the given directory in order.
// It tracks applied migrations in a schema_migrations table.
func Migrate(pool *pgxpool.Pool, migrationsDir string, logger *slog.Logger) error {
	ctx := context.Background()

	// Create tracking table
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	// Get applied versions
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations ORDER BY version`)
	if err != nil {
		return fmt.Errorf("query migrations: %w", err)
	}
	defer rows.Close()

	applied := map[string]bool{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return err
		}
		applied[v] = true
	}

	// Read migration files
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, file := range files {
		if applied[file] {
			continue
		}

		content, err := os.ReadFile(filepath.Join(migrationsDir, file))
		if err != nil {
			return fmt.Errorf("read %s: %w", file, err)
		}

		// Only run the "up" portion (before the tern separator)
		sql := string(content)
		if idx := strings.Index(sql, "---- create above / drop below ----"); idx > 0 {
			sql = sql[:idx]
		}

		logger.Info("applying migration", "file", file)

		_, err = pool.Exec(ctx, sql)
		if err != nil {
			return fmt.Errorf("apply %s: %w", file, err)
		}

		_, err = pool.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, file)
		if err != nil {
			return fmt.Errorf("record %s: %w", file, err)
		}
	}

	return nil
}
