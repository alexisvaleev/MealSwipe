// Package db provides the pgx connection pool and the embedded migration runner.
package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	connectAttempts = 10
	connectDelay    = 3 * time.Second
)

// Connect opens a pgx pool, retrying while Postgres is still starting
// (10 attempts, ~30s total budget).
func Connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("db: parse config: %w", err)
	}
	if cfg.MaxConns == 0 || cfg.MaxConns > 20 {
		cfg.MaxConns = 10
	}

	var lastErr error
	for attempt := 1; attempt <= connectAttempts; attempt++ {
		pool, err := pgxpool.NewWithConfig(ctx, cfg)
		if err == nil {
			pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err = pool.Ping(pingCtx)
			cancel()
			if err == nil {
				return pool, nil
			}
			pool.Close()
		}
		lastErr = err
		if attempt < connectAttempts {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("db: connect cancelled: %w", ctx.Err())
			case <-time.After(connectDelay):
			}
		}
	}
	return nil, fmt.Errorf("db: connect after %d attempts: %w", connectAttempts, lastErr)
}

// Ping reports whether the database is reachable (used by /health if needed).
func Ping(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return errors.New("db: nil pool")
	}
	return pool.Ping(ctx)
}
