package store

import (
	"context"
	"fmt"
)

// AddFavorite bookmarks a recipe.
func (s *Store) AddFavorite(ctx context.Context, userID, recipeID string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO favorites (user_id, recipe_id) VALUES ($1::uuid,$2::uuid)
		 ON CONFLICT DO NOTHING`, userID, recipeID)
	if isFKViolation(err) {
		return ErrNotFound
	}
	return err
}

// RemoveFavorite deletes the bookmark (idempotent).
func (s *Store) RemoveFavorite(ctx context.Context, userID, recipeID string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM favorites WHERE user_id = $1::uuid AND recipe_id = $2::uuid`, userID, recipeID)
	return err
}

// ListFavorites returns favorited recipes with favorited_at.
// sort: "recent" (default), "oldest", "title". category/q filter the recipe.
func (s *Store) ListFavorites(ctx context.Context, userID, category, q, sort string) ([]RecipeSummary, error) {
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
		 WHERE fv.user_id = $1::uuid
		   AND ($2 = '' OR r.category = $2)
		   AND ($3 = '' OR r.title ILIKE '%' || $3 || '%')
		 ORDER BY `+order, userID, category, q)
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
