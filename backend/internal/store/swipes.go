package store

import (
	"context"
	"time"
)

// Swipe is a like/dislike signal on a recipe.
type Swipe struct {
	RecipeID  string    `json:"recipe_id"`
	Direction string    `json:"direction"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UpsertSwipe records or overwrites a swipe (re-swiping flips direction).
func (s *Store) UpsertSwipe(ctx context.Context, userID, recipeID, direction string) (*Swipe, error) {
	var sw Swipe
	err := s.pool.QueryRow(ctx,
		`INSERT INTO swipes (user_id, recipe_id, direction) VALUES ($1::uuid,$2::uuid,$3)
		 ON CONFLICT (user_id, recipe_id) DO UPDATE SET direction = EXCLUDED.direction, updated_at = now()
		 RETURNING recipe_id::text, direction, updated_at`,
		userID, recipeID, direction).Scan(&sw.RecipeID, &sw.Direction, &sw.UpdatedAt)
	if err != nil {
		if isFKViolation(err) {
			return nil, ErrNotFound
		}
		return nil, mapErr(err)
	}
	return &sw, nil
}

// DeleteLatestSwipe pops the most recently updated swipe (undo).
func (s *Store) DeleteLatestSwipe(ctx context.Context, userID string) (bool, error) {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM swipes WHERE (user_id, recipe_id) IN (
			SELECT user_id, recipe_id FROM swipes
			WHERE user_id = $1::uuid ORDER BY updated_at DESC LIMIT 1)`, userID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
