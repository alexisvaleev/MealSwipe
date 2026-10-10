package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// Session is a server-side auth session. The cookie holds the opaque token;
// only its SHA-256 hash is stored (SPEC §4).
type Session struct {
	UserID    string
	CSRF      string
	ExpiresAt time.Time
}

func randHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func tokenHash(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// CreateSession issues a new opaque token + session-bound CSRF token.
func (s *Store) CreateSession(ctx context.Context, userID string, ttl time.Duration) (token, csrf string, err error) {
	if token, err = randHex(32); err != nil {
		return "", "", fmt.Errorf("store: gen token: %w", err)
	}
	if csrf, err = randHex(32); err != nil {
		return "", "", fmt.Errorf("store: gen csrf: %w", err)
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO sessions (token_hash, user_id, csrf_token, expires_at) VALUES ($1,$2::uuid,$3,$4)`,
		tokenHash(token), userID, csrf, time.Now().Add(ttl))
	if err != nil {
		return "", "", fmt.Errorf("store: create session: %w", err)
	}
	return token, csrf, nil
}

// SessionByToken looks up a session by the raw cookie token.
func (s *Store) SessionByToken(ctx context.Context, token string) (*Session, error) {
	var sess Session
	err := s.pool.QueryRow(ctx,
		`SELECT user_id::text, csrf_token, expires_at FROM sessions WHERE token_hash = $1`,
		tokenHash(token)).Scan(&sess.UserID, &sess.CSRF, &sess.ExpiresAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &sess, nil
}

// RenewSession implements sliding expiry: bump expires_at for the same token.
func (s *Store) RenewSession(ctx context.Context, token string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx,
		`UPDATE sessions SET expires_at = $2 WHERE token_hash = $1`,
		tokenHash(token), expiresAt)
	return err
}

// DeleteSession removes a session (logout).
func (s *Store) DeleteSession(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash(token))
	return err
}

// DeleteExpiredSessions is periodic housekeeping (SPEC §11).
func (s *Store) DeleteExpiredSessions(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at < now()`)
	return err
}
