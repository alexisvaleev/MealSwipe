// Package store contains all SQL access (pgx), one file per entity.
// All queries are parameterised; every user-scoped query takes userID.
package store

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Sentinel errors mapped to HTTP status codes by the httpapi layer.
var (
	ErrNotFound   = errors.New("store: not found")
	ErrConflict   = errors.New("store: conflict")
	ErrEmailTaken = errors.New("store: email taken")
)

// Store wraps the connection pool; per-entity methods live in their own files.
type Store struct {
	pool *pgxpool.Pool
}

// New returns a Store over the pool.
func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Pool exposes the raw pool (migrations, seeding, tests).
func (s *Store) Pool() *pgxpool.Pool { return s.pool }

func mapErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func isUniqueViolation(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
}

func isFKViolation(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23503"
}
