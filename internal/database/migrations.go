package database

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RunMigrations runs all SQL migration files in the migrations directory
func (db *DB) RunMigrations(ctx context.Context, migrationsPath string) error {
	// Create migrations table if it doesn't exist
	err := db.createMigrationsTable(ctx)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get list of migration files
	migrationFiles, err := getMigrationFiles(migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to get migration files: %w", err)
	}

	// Get already applied migrations
	appliedMigrations, err := db.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Run pending migrations
	for _, file := range migrationFiles {
		if _, applied := appliedMigrations[file]; applied {
			continue
		}

		err := db.runMigration(ctx, migrationsPath, file)
		if err != nil {
			return fmt.Errorf("failed to run migration %s: %w", file, err)
		}

		// Record migration as applied
		err = db.recordMigration(ctx, file)
		if err != nil {
			return fmt.Errorf("failed to record migration %s: %w", file, err)
		}

		db.logger.Info("migration_applied", fmt.Sprintf("Applied migration: %s", file), "startup", nil)
	}

	return nil
}

// createMigrationsTable creates the migrations tracking table
func (db *DB) createMigrationsTable(ctx context.Context) error {
	sql := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id SERIAL PRIMARY KEY,
			migration_name VARCHAR(255) NOT NULL UNIQUE,
			applied_at TIMESTAMPTZ DEFAULT NOW()
		)
	`
	return db.Exec(ctx, sql)
}

// getMigrationFiles returns a sorted list of migration files
func getMigrationFiles(migrationsPath string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(migrationsPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && strings.HasSuffix(path, ".sql") {
			files = append(files, filepath.Base(path))
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort files to ensure they run in order
	sort.Strings(files)

	return files, nil
}

// getAppliedMigrations returns a map of already applied migrations
func (db *DB) getAppliedMigrations(ctx context.Context) (map[string]bool, error) {
	applied := make(map[string]bool)

	rows, err := db.Query(ctx, "SELECT migration_name FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var migrationName string
		err := rows.Scan(&migrationName)
		if err != nil {
			return nil, err
		}
		applied[migrationName] = true
	}

	return applied, rows.Err()
}

// runMigration executes a single migration file
func (db *DB) runMigration(ctx context.Context, migrationsPath, filename string) error {
	filePath := filepath.Join(migrationsPath, filename)
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Execute the migration in a transaction
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, string(content))
	if err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	return tx.Commit(ctx)
}

// recordMigration records that a migration has been applied
func (db *DB) recordMigration(ctx context.Context, filename string) error {
	sql := "INSERT INTO schema_migrations (migration_name) VALUES ($1)"
	return db.Exec(ctx, sql, filename)
}
