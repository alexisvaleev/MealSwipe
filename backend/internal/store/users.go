package store

import (
	"context"
	"fmt"
	"time"
)

// User is an account row. PasswordHash is never serialised to JSON.
type User struct {
	ID             string    `json:"id"`
	Username       string    `json:"username"`
	DisplayName    string    `json:"display_name"`
	HasPreferences bool      `json:"has_preferences"`
	PasswordHash   string    `json:"-"`
	CreatedAt      time.Time `json:"-"`
}

const userCols = `u.id::text, u.username, u.display_name,
	EXISTS(SELECT 1 FROM user_preferences p WHERE p.user_id = u.id)`

// CreateUser registers a user together with their personal household
// (every user belongs to exactly one household; joining re-points it).
func (s *Store) CreateUser(ctx context.Context, username, passwordHash, displayName string) (*User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var householdID string
	for i := 0; i < 5; i++ {
		code, err := newInviteCode()
		if err != nil {
			return nil, err
		}
		err = tx.QueryRow(ctx,
			`INSERT INTO households (invite_code) VALUES ($1) RETURNING id::text`, code).
			Scan(&householdID)
		if err == nil {
			break
		}
		if !isUniqueViolation(err) {
			return nil, fmt.Errorf("store: create household: %w", err)
		}
		if i == 4 {
			return nil, fmt.Errorf("store: create household: invite code collisions")
		}
	}

	var u User
	err = tx.QueryRow(ctx,
		`INSERT INTO users AS u (username, password_hash, display_name, household_id)
		 VALUES ($1,$2,$3,$4::uuid)
		 RETURNING `+userCols+`, u.password_hash, u.created_at`,
		username, passwordHash, displayName, householdID,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &u.HasPreferences, &u.PasswordHash, &u.CreatedAt)
	if isUniqueViolation(err) {
		return nil, ErrUsernameTaken
	}
	if err != nil {
		return nil, fmt.Errorf("store: create user: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &u, nil
}

// UserByUsername loads a user including the password hash (for login).
func (s *Store) UserByUsername(ctx context.Context, username string) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx,
		`SELECT `+userCols+`, u.password_hash, u.created_at FROM users u WHERE u.username = $1`,
		username,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &u.HasPreferences, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &u, nil
}

// UserByID loads a user by primary key.
func (s *Store) UserByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx,
		`SELECT `+userCols+`, u.password_hash, u.created_at FROM users u WHERE u.id = $1::uuid`,
		id,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &u.HasPreferences, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &u, nil
}

// HouseholdID is the hot lookup used by every shared-scope handler.
func (s *Store) HouseholdID(ctx context.Context, userID string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx,
		`SELECT household_id::text FROM users WHERE id = $1::uuid`, userID).Scan(&id)
	if err != nil {
		return "", mapErr(err)
	}
	return id, nil
}
