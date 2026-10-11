package store

import (
	"context"
	"fmt"
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

// DeleteSwipe removes a user's swipe on a recipe (unlike/undislike), idempotent.
func (s *Store) DeleteSwipe(ctx context.Context, userID, recipeID string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM swipes WHERE user_id = $1::uuid AND recipe_id = $2::uuid`,
		userID, recipeID)
	return err
}

// ListLikes returns recipes the user swiped "like" on. Personal scope: likes
// are per-member even inside a shared household.
// sort: "recent" (default), "oldest", "title".
func (s *Store) ListLikes(ctx context.Context, userID, householdID, category, q, sort string) ([]RecipeSummary, error) {
	order := "sw.updated_at DESC"
	switch sort {
	case "oldest":
		order = "sw.updated_at ASC"
	case "title":
		order = "r.title ASC"
	}
	rows, err := s.pool.Query(ctx,
		`SELECT `+summaryCols+flagCols+`, sw.updated_at::text
		 FROM swipes sw JOIN recipes r ON r.id = sw.recipe_id
		 WHERE sw.user_id = $1::uuid AND sw.direction = 'like'
		   AND ($3 = '' OR r.category = $3)
		   AND ($4 = '' OR r.title ILIKE '%' || $4 || '%')
		 ORDER BY `+order, userID, householdID, category, q)
	if err != nil {
		return nil, fmt.Errorf("store: list likes: %w", err)
	}
	defer rows.Close()
	out := []RecipeSummary{}
	for rows.Next() {
		var rs RecipeSummary
		var at string
		if err := scanSummary(rows.Scan, &rs, &rs.Liked, &rs.Favorited, &at); err != nil {
			return nil, err
		}
		out = append(out, rs)
	}
	return out, rows.Err()
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
