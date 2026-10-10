package store

import (
	"context"
	"fmt"
	"time"
)

// User is an account row. PasswordHash is never serialised to JSON.
type User struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	DisplayName    string    `json:"display_name"`
	HasPreferences bool      `json:"has_preferences"`
	PasswordHash   string    `json:"-"`
	CreatedAt      time.Time `json:"-"`
}

const userCols = `u.id::text, u.email, u.display_name,
	EXISTS(SELECT 1 FROM user_preferences p WHERE p.user_id = u.id)`

// CreateUser inserts a user; ErrEmailTaken on duplicate email.
func (s *Store) CreateUser(ctx context.Context, email, passwordHash, displayName string) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx,
		`INSERT INTO users AS u (email, password_hash, display_name) VALUES ($1,$2,$3)
		 RETURNING `+userCols+`, u.password_hash, u.created_at`,
		email, passwordHash, displayName,
	).Scan(&u.ID, &u.Email, &u.DisplayName, &u.HasPreferences, &u.PasswordHash, &u.CreatedAt)
	if isUniqueViolation(err) {
		return nil, ErrEmailTaken
	}
	if err != nil {
		return nil, fmt.Errorf("store: create user: %w", err)
	}
	return &u, nil
}

// UserByEmail loads a user including the password hash (for login).
func (s *Store) UserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx,
		`SELECT `+userCols+`, u.password_hash, u.created_at FROM users u WHERE u.email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.DisplayName, &u.HasPreferences, &u.PasswordHash, &u.CreatedAt)
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
	).Scan(&u.ID, &u.Email, &u.DisplayName, &u.HasPreferences, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &u, nil
}
