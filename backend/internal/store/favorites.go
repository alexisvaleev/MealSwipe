package store

import (
	"context"
	"fmt"
)

// AddFavorite bookmarks a recipe for the whole household; userID records who did.
func (s *Store) AddFavorite(ctx context.Context, householdID, userID, recipeID string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO favorites (household_id, recipe_id, added_by) VALUES ($1::uuid,$2::uuid,$3::uuid)
		 ON CONFLICT (household_id, recipe_id) DO UPDATE SET added_by = EXCLUDED.added_by`,
		householdID, recipeID, userID)
	if isFKViolation(err) {
		return ErrNotFound
	}
	return err
}

// RemoveFavorite deletes the bookmark (idempotent).
func (s *Store) RemoveFavorite(ctx context.Context, householdID, recipeID string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM favorites WHERE household_id = $1::uuid AND recipe_id = $2::uuid`,
		householdID, recipeID)
	return err
}

// ListFavorites returns the household's favorited recipes with favorited_at.
// sort: "recent" (default), "oldest", "title". category/q filter the recipe.
func (s *Store) ListFavorites(ctx context.Context, userID, householdID, category, q, sort string) ([]RecipeSummary, error) {
	order := "fv.created_at DESC"
	switch sort {
	case "oldest":
		order = "fv.created_at ASC"
	case "title":
		order = "r.title ASC"
	}
	rows, err := s.pool.Query(ctx,
		`SELECT `+summaryCols+flagCols+`, fv.created_at::text
		 FROM favorites fv JOIN recipes r ON r.id = fv.recipe_id
		 WHERE fv.household_id = $2::uuid
		   AND ($3 = '' OR r.category = $3)
		   AND ($4 = '' OR r.title ILIKE '%' || $4 || '%')
		 ORDER BY `+order, userID, householdID, category, q)
	if err != nil {
		return nil, fmt.Errorf("store: list favorites: %w", err)
	}
	defer rows.Close()
	out := []RecipeSummary{}
	for rows.Next() {
		var rs RecipeSummary
		var at string
		if err := scanSummary(rows.Scan, &rs, &rs.Liked, &rs.Favorited, &at); err != nil {
			return nil, err
		}
		rs.FavoritedAt = &at
		out = append(out, rs)
	}
	return out, rows.Err()
}
