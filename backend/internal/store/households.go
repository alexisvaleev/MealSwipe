package store

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"
)

// Household is a shared kitchen: members share favorites, menus, shopping
// lists and pantry; preferences and swipes stay personal.
type Household struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	InviteCode string            `json:"invite_code"`
	Members    []HouseholdMember `json:"members"`
}

// HouseholdMember is a user inside a household.
type HouseholdMember struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name"`
	JoinedAt    time.Time `json:"joined_at"`
}

// inviteAlphabet avoids ambiguous characters (no 0/O, 1/l/I).
const inviteAlphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZ"

// newInviteCode returns an "XXXX-XXXX" code (~32 bits of entropy).
func newInviteCode() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("store: invite code: %w", err)
	}
	out := make([]byte, 9)
	for i, v := range b {
		j := i
		if i >= 4 {
			j = i + 1 // skip the '-' position
		}
		out[j] = inviteAlphabet[int(v)%len(inviteAlphabet)]
	}
	out[4] = '-'
	return string(out), nil
}

func (s *Store) loadMembers(ctx context.Context, h *Household) error {
	rows, err := s.pool.Query(ctx,
		`SELECT id::text, display_name, household_joined_at
		 FROM users WHERE household_id = $1::uuid ORDER BY household_joined_at`, h.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	h.Members = []HouseholdMember{}
	for rows.Next() {
		var m HouseholdMember
		if err := rows.Scan(&m.ID, &m.DisplayName, &m.JoinedAt); err != nil {
			return err
		}
		h.Members = append(h.Members, m)
	}
	return rows.Err()
}

// GetHousehold returns the user's household with its member list.
func (s *Store) GetHousehold(ctx context.Context, userID string) (*Household, error) {
	var h Household
	err := s.pool.QueryRow(ctx,
		`SELECT h.id::text, h.name, h.invite_code
		 FROM households h JOIN users u ON u.household_id = h.id
		 WHERE u.id = $1::uuid`, userID).
		Scan(&h.ID, &h.Name, &h.InviteCode)
	if err != nil {
		return nil, mapErr(err)
	}
	if err := s.loadMembers(ctx, &h); err != nil {
		return nil, err
	}
	return &h, nil
}

// JoinHousehold moves the user into the household matching invite code.
// Their personal data (preferences, swipes) travels with the user; shared
// data stays behind with the old household. A now-empty old household is
// removed together with its cascaded rows.
func (s *Store) JoinHousehold(ctx context.Context, userID, code string) (*Household, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var targetID, name, invite string
	err = tx.QueryRow(ctx,
		`SELECT id::text, name, invite_code FROM households WHERE upper(invite_code) = upper($1)`,
		code).Scan(&targetID, &name, &invite)
	if err != nil {
		return nil, mapErr(err)
	}

	var oldID string
	if err := tx.QueryRow(ctx,
		`SELECT household_id::text FROM users WHERE id = $1::uuid FOR UPDATE`, userID).
		Scan(&oldID); err != nil {
		return nil, mapErr(err)
	}
	if oldID == targetID {
		return s.GetHousehold(ctx, userID) // already a member — idempotent no-op
	}
	if _, err := tx.Exec(ctx,
		`UPDATE users SET household_id = $1::uuid, household_joined_at = now() WHERE id = $2::uuid`,
		targetID, userID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM households h WHERE h.id = $1::uuid
		   AND NOT EXISTS(SELECT 1 FROM users u WHERE u.household_id = h.id)`, oldID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetHousehold(ctx, userID)
}

// LeaveHousehold moves the user into a fresh personal household. A solo
// household has nothing to leave — return it unchanged (leaving would wipe
// the shared rows via the empty-household cleanup).
func (s *Store) LeaveHousehold(ctx context.Context, userID string) (*Household, error) {
	var cnt int
	err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM users m WHERE m.household_id =
		   (SELECT household_id FROM users WHERE id = $1::uuid)`, userID).
		Scan(&cnt)
	if err != nil {
		return nil, mapErr(err)
	}
	if cnt <= 1 {
		return s.GetHousehold(ctx, userID)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var newID string
	for i := 0; i < 5; i++ {
		code, err := newInviteCode()
		if err != nil {
			return nil, err
		}
		err = tx.QueryRow(ctx,
			`INSERT INTO households (invite_code) VALUES ($1) RETURNING id::text`, code).
			Scan(&newID)
		if err == nil {
			break
		}
		if !isUniqueViolation(err) {
			return nil, fmt.Errorf("store: leave household: %w", err)
		}
	}
	if newID == "" {
		return nil, errors.New("store: leave household: too many invite collisions")
	}

	var oldID string
	if err := tx.QueryRow(ctx,
		`SELECT household_id::text FROM users WHERE id = $1::uuid FOR UPDATE`, userID).
		Scan(&oldID); err != nil {
		return nil, mapErr(err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE users SET household_id = $1::uuid, household_joined_at = now() WHERE id = $2::uuid`,
		newID, userID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM households h WHERE h.id = $1::uuid
		   AND NOT EXISTS(SELECT 1 FROM users u WHERE u.household_id = h.id)`, oldID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.GetHousehold(ctx, userID)
}

// RotateInvite issues a new invite code (old code stops working).
func (s *Store) RotateInvite(ctx context.Context, householdID string) error {
	for i := 0; i < 5; i++ {
		code, err := newInviteCode()
		if err != nil {
			return err
		}
		tag, err := s.pool.Exec(ctx,
			`UPDATE households SET invite_code = $1 WHERE id = $2::uuid`, code, householdID)
		if err == nil {
			if tag.RowsAffected() == 0 {
				return ErrNotFound
			}
			return nil
		}
		if !isUniqueViolation(err) {
			return err
		}
	}
	return errors.New("store: rotate invite: too many collisions")
}

// RenameHousehold updates the display name (validated at the handler).
func (s *Store) RenameHousehold(ctx context.Context, householdID, name string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE households SET name = $2 WHERE id = $1::uuid`, householdID, name)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MemberPrefs bundles a member's personal constraints for menu generation.
type MemberPrefs struct {
	UserID string
	Prefs  Preferences
}

// HouseholdPreferences loads every member's saved preferences (defaults for
// members who never saved) so the planner can merge hard constraints.
func (s *Store) HouseholdPreferences(ctx context.Context, householdID string) ([]MemberPrefs, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT u.id::text,
		        COALESCE(p.household_size, 2), COALESCE(p.meals_per_day, 3),
		        COALESCE(p.meal_types, '{breakfast,lunch,dinner}'::text[]),
		        p.weekly_budget::float8, p.max_cook_minutes,
		        COALESCE(p.diet_type, 'any'), COALESCE(p.allergens, '{}'::text[]),
		        COALESCE(p.goal, '')
		 FROM users u LEFT JOIN user_preferences p ON p.user_id = u.id
		 WHERE u.household_id = $1::uuid
		 ORDER BY u.household_joined_at`, householdID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MemberPrefs
	for rows.Next() {
		var mp MemberPrefs
		if err := rows.Scan(&mp.UserID, &mp.Prefs.HouseholdSize, &mp.Prefs.MealsPerDay,
			&mp.Prefs.MealTypes, &mp.Prefs.WeeklyBudget, &mp.Prefs.MaxCookMinutes,
			&mp.Prefs.DietType, &mp.Prefs.Allergens, &mp.Prefs.Goal); err != nil {
			return nil, err
		}
		ids, err := s.ExcludedIngredientIDs(ctx, mp.UserID)
		if err != nil {
			return nil, err
		}
		mp.Prefs.ExcludedIngredientIDs = ids
		out = append(out, mp)
	}
	return out, rows.Err()
}
