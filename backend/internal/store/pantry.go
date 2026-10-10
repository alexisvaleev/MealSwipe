package store

import (
	"context"
)

// PantryAdd marks an ingredient as already owned. The "already have it"
// shopping flow is: DELETE the list item + PUT /pantry/{ingredient_id}.
func (s *Store) PantryAdd(ctx context.Context, userID string, ingredientID int64) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO pantry_items (user_id, ingredient_id) VALUES ($1::uuid,$2)
		 ON CONFLICT DO NOTHING`, userID, ingredientID)
	if isFKViolation(err) {
		return ErrNotFound
	}
	return err
}

// PantryRemove removes a pantry marker (idempotent).
func (s *Store) PantryRemove(ctx context.Context, userID string, ingredientID int64) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM pantry_items WHERE user_id = $1::uuid AND ingredient_id = $2`,
		userID, ingredientID)
	return err
}

// PantryIDs returns the user's pantry ingredient id set.
func (s *Store) PantryIDs(ctx context.Context, userID string) (map[int64]bool, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT ingredient_id FROM pantry_items WHERE user_id = $1::uuid`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}
