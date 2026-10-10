package db

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"github.com/alexisvaleev/mealswipe/backend/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

type migrationFile struct {
	version int
	name    string
}

// Migrate applies all pending *.up.sql migrations from the embedded FS in
// numeric order, each inside its own transaction.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	// The fixed schema (0001_init.up.sql) uses the citext type; the extension
	// is part of the standard Postgres contrib package. Ensured here because
	// migration files themselves must not be modified.
	if _, err := pool.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS citext"); err != nil {
		return fmt.Errorf("db: ensure citext extension: %w", err)
	}
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    int PRIMARY KEY,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`); err != nil {
		return fmt.Errorf("db: create schema_migrations: %w", err)
	}

	applied := map[int]bool{}
	rows, err := pool.Query(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("db: read schema_migrations: %w", err)
	}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return fmt.Errorf("db: scan version: %w", err)
		}
		applied[v] = true
	}
	rows.Close()

	ups, err := listMigrations(".up.sql")
	if err != nil {
		return err
	}
	for _, m := range ups {
		if applied[m.version] {
			continue
		}
		body, err := fs.ReadFile(migrations.FS, m.name)
		if err != nil {
			return fmt.Errorf("db: read %s: %w", m.name, err)
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("db: begin tx for %s: %w", m.name, err)
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("db: apply %s: %w", m.name, err)
		}
		if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", m.version); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("db: record %s: %w", m.name, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("db: commit %s: %w", m.name, err)
		}
	}
	return nil
}

// Down rolls back the last n applied migrations. Used by tests and manual ops.
func Down(ctx context.Context, pool *pgxpool.Pool, n int) error {
	rows, err := pool.Query(ctx, "SELECT version FROM schema_migrations ORDER BY version DESC")
	if err != nil {
		return fmt.Errorf("db: read schema_migrations: %w", err)
	}
	var versions []int
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return err
		}
		versions = append(versions, v)
	}
	rows.Close()

	for i := 0; i < n && i < len(versions); i++ {
		v := versions[i]
		name := downName(v)
		body, err := fs.ReadFile(migrations.FS, name)
		if err != nil {
			return fmt.Errorf("db: read %s: %w", name, err)
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("db: apply %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, "DELETE FROM schema_migrations WHERE version=$1", v); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

func listMigrations(suffix string) ([]migrationFile, error) {
	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		return nil, fmt.Errorf("db: list migrations: %w", err)
	}
	var out []migrationFile
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, suffix) {
			continue
		}
		num, _, found := strings.Cut(name, "_")
		if !found {
			continue
		}
		v, err := strconv.Atoi(num)
		if err != nil {
			return nil, fmt.Errorf("db: bad migration name %q", name)
		}
		out = append(out, migrationFile{version: v, name: name})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].version < out[j].version })
	return out, nil
}

func downName(version int) string {
	entries, _ := migrations.FS.ReadDir(".")
	prefix := fmt.Sprintf("%04d_", version)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix) && strings.HasSuffix(e.Name(), ".down.sql") {
			return e.Name()
		}
	}
	return prefix + "unknown.down.sql"
}
