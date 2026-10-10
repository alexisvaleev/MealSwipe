package store

import (
	"context"
)

// Ingredient is a canonical ingredient row (shared catalog, not user-scoped).
type Ingredient struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	UnitFamily   string   `json:"unit_family"`
	AllergenTags []string `json:"allergen_tags"`
}

// SearchIngredients returns ingredients matching a name substring (for the
// exclusion picker and pantry UI).
func (s *Store) SearchIngredients(ctx context.Context, q string, limit int) ([]Ingredient, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, category, unit_family, allergen_tags FROM ingredients
		 WHERE ($1 = '' OR name ILIKE '%' || $1 || '%')
		 ORDER BY name LIMIT $2`, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIngredients(rows)
}

// IngredientsByIDs loads ingredients by primary key (pantry listing).
func (s *Store) IngredientsByIDs(ctx context.Context, ids []int64) ([]Ingredient, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, category, unit_family, allergen_tags
		 FROM ingredients WHERE id = ANY($1::bigint[]) ORDER BY name`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIngredients(rows)
}

func scanIngredients(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]Ingredient, error) {
	out := []Ingredient{}
	for rows.Next() {
		var in Ingredient
		if err := rows.Scan(&in.ID, &in.Name, &in.Category, &in.UnitFamily, &in.AllergenTags); err != nil {
			return nil, err
		}
		out = append(out, in)
	}
	return out, rows.Err()
}
