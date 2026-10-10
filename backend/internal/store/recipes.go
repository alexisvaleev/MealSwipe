package store

import (
	"context"
	"fmt"

	"github.com/alexisvaleev/mealswipe/backend/internal/planner"
)

// Candidate is a type alias so callers use store.Candidate.
type Candidate = planner.Candidate

// RecipeSummary is the card-level recipe payload.
type RecipeSummary struct {
	ID                string   `json:"id"`
	Slug              string   `json:"slug"`
	Title             string   `json:"title"`
	Description       string   `json:"description"`
	ImageURL          *string  `json:"image_url"`
	PrepMinutes       int      `json:"prep_minutes"`
	CookMinutes       int      `json:"cook_minutes"`
	Servings          float64  `json:"servings"`
	MealTypes         []string `json:"meal_types"`
	Category          string   `json:"category"`
	Tags              []string `json:"tags"`
	DietFlags         []string `json:"diet_flags"`
	CostPerServing    *float64 `json:"cost_per_serving"`
	KcalPerServing    *int     `json:"kcal_per_serving"`
	NutritionEstimate bool     `json:"nutrition_estimate"`
	Liked             bool     `json:"liked,omitempty"`
	Favorited         bool     `json:"favorited,omitempty"`
	FavoritedAt       *string  `json:"favorited_at,omitempty"`
}

// RecipeIngredient is a recipe line with joined ingredient data.
type RecipeIngredient struct {
	IngredientID int64    `json:"ingredient_id"`
	Name         string   `json:"name"`
	Quantity     float64  `json:"quantity"`
	Unit         string   `json:"unit"`
	Note         string   `json:"note"`
	Category     string   `json:"category"`
	AllergenTags []string `json:"allergen_tags"`
}

// RecipeDetail is the full recipe payload.
type RecipeDetail struct {
	RecipeSummary
	Instructions []string           `json:"instructions"`
	Ingredients  []RecipeIngredient `json:"ingredients"`
}

const summaryCols = `r.id::text, r.slug, r.title, r.description, r.image_url,
	r.prep_minutes, r.cook_minutes, r.servings::float8, r.meal_types, r.category,
	r.tags, r.diet_flags, r.cost_per_serving::float8, r.kcal_per_serving, r.nutrition_estimate`

func scanSummary(scan func(...any) error, rs *RecipeSummary, extra ...any) error {
	args := append([]any{
		&rs.ID, &rs.Slug, &rs.Title, &rs.Description, &rs.ImageURL,
		&rs.PrepMinutes, &rs.CookMinutes, &rs.Servings, &rs.MealTypes, &rs.Category,
		&rs.Tags, &rs.DietFlags, &rs.CostPerServing, &rs.KcalPerServing, &rs.NutritionEstimate,
	}, extra...)
	return scan(args...)
}

const flagCols = `,
	EXISTS(SELECT 1 FROM swipes sw WHERE sw.user_id = $1::uuid AND sw.recipe_id = r.id AND sw.direction = 'like'),
	EXISTS(SELECT 1 FROM favorites fv WHERE fv.user_id = $1::uuid AND fv.recipe_id = r.id)`

// ListRecipes returns card summaries + total count for pagination.
func (s *Store) ListRecipes(ctx context.Context, userID, mealType, q string, page, pageSize int) ([]RecipeSummary, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 24
	}
	whereCount := `WHERE ($1 = '' OR r.meal_types @> ARRAY[$1]::text[])
		AND ($2 = '' OR r.title ILIKE '%' || $2 || '%' OR r.description ILIKE '%' || $2 || '%')`
	var total int
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM recipes r `+whereCount, mealType, q).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("store: count recipes: %w", err)
	}
	where := `WHERE ($2 = '' OR r.meal_types @> ARRAY[$2]::text[])
		  AND ($3 = '' OR r.title ILIKE '%' || $3 || '%' OR r.description ILIKE '%' || $3 || '%')`
	rows, err := s.pool.Query(ctx,
		`SELECT `+summaryCols+flagCols+` FROM recipes r `+where+
			` ORDER BY r.title LIMIT $4 OFFSET $5`,
		userID, mealType, q, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("store: list recipes: %w", err)
	}
	defer rows.Close()
	out := []RecipeSummary{}
	for rows.Next() {
		var rs RecipeSummary
		if err := scanSummary(rows.Scan, &rs, &rs.Liked, &rs.Favorited); err != nil {
			return nil, 0, err
		}
		out = append(out, rs)
	}
	return out, total, rows.Err()
}

// GetRecipe returns a full recipe with joined ingredients.
func (s *Store) GetRecipe(ctx context.Context, userID, id string) (*RecipeDetail, error) {
	var d RecipeDetail
	err := scanSummary(s.pool.QueryRow(ctx,
		`SELECT `+summaryCols+flagCols+`, r.instructions FROM recipes r WHERE r.id = $2::uuid`,
		userID, id).Scan,
		&d.RecipeSummary, &d.Liked, &d.Favorited, &d.Instructions)
	if err != nil {
		return nil, mapErr(err)
	}
	rows, err := s.pool.Query(ctx,
		`SELECT ri.ingredient_id, i.name, ri.quantity::float8, ri.unit, ri.note,
		        i.category, i.allergen_tags
		 FROM recipe_ingredients ri JOIN ingredients i ON i.id = ri.ingredient_id
		 WHERE ri.recipe_id = $1::uuid ORDER BY ri.position`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	d.Ingredients = []RecipeIngredient{}
	for rows.Next() {
		var ri RecipeIngredient
		if err := rows.Scan(&ri.IngredientID, &ri.Name, &ri.Quantity, &ri.Unit,
			&ri.Note, &ri.Category, &ri.AllergenTags); err != nil {
			return nil, err
		}
		d.Ingredients = append(d.Ingredients, ri)
	}
	return &d, rows.Err()
}

// RecipeCount is used by SEED_ON_START.
func (s *Store) RecipeCount(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM recipes`).Scan(&n)
	return n, err
}

// candidateAgg is the shared per-recipe ingredient/allergen aggregation.
const candidateAgg = `
	LEFT JOIN LATERAL (
		SELECT array_agg(DISTINCT ri.ingredient_id) AS ing_ids,
			(SELECT array_agg(DISTINCT t) FROM recipe_ingredients ri2
			   JOIN ingredients i2 ON i2.id = ri2.ingredient_id
			   CROSS JOIN LATERAL unnest(i2.allergen_tags) t
			   WHERE ri2.recipe_id = r.id) AS tags
		FROM recipe_ingredients ri WHERE ri.recipe_id = r.id
	) ii ON true`

// PlannerCandidates returns every recipe with the signals the planner needs.
func (s *Store) PlannerCandidates(ctx context.Context, userID string) ([]Candidate, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT r.id::text, r.meal_types, r.diet_flags, r.prep_minutes, r.cook_minutes,
		        r.category, r.servings::float8, r.cost_per_serving::float8,
		        COALESCE(ii.ing_ids, '{}'::bigint[]), COALESCE(ii.tags, '{}'::text[]),
		        EXISTS(SELECT 1 FROM swipes sw WHERE sw.user_id = $1::uuid AND sw.recipe_id = r.id AND sw.direction = 'like'),
		        EXISTS(SELECT 1 FROM swipes sw WHERE sw.user_id = $1::uuid AND sw.recipe_id = r.id AND sw.direction = 'dislike'),
		        EXISTS(SELECT 1 FROM favorites fv WHERE fv.user_id = $1::uuid AND fv.recipe_id = r.id)
		 FROM recipes r `+candidateAgg, userID)
	if err != nil {
		return nil, fmt.Errorf("store: planner candidates: %w", err)
	}
	defer rows.Close()
	return scanCandidates(rows)
}

func scanCandidates(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]Candidate, error) {
	out := []Candidate{}
	for rows.Next() {
		var c Candidate
		if err := rows.Scan(&c.RecipeID, &c.MealTypes, &c.DietFlags, &c.PrepMinutes,
			&c.CookMinutes, &c.Category, &c.Servings, &c.CostPerServing,
			&c.IngredientIDs, &c.AllergenTags, &c.Liked, &c.Disliked, &c.Favorited); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// DiscoverFilter holds the SQL-applied hard filters for the swipe deck.
type DiscoverFilter struct {
	Allergens      []string
	ExcludedIDs    []int64
	MealTypes      []string
	DietType       string
	MaxCookMinutes *int
	Limit          int
}

// DiscoverRecipes returns unseen, hard-filtered recipes in random order.
func (s *Store) DiscoverRecipes(ctx context.Context, userID string, f DiscoverFilter) ([]RecipeSummary, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 20
	}
	diet := `true`
	switch f.DietType {
	case "vegetarian":
		diet = `r.diet_flags && '{vegetarian,vegan}'::text[]`
	case "vegan":
		diet = `r.diet_flags @> '{vegan}'::text[]`
	case "pescatarian":
		diet = `r.diet_flags && '{vegetarian,vegan,pescatarian}'::text[]`
	}
	rows, err := s.pool.Query(ctx,
		`SELECT `+summaryCols+flagCols+`
		 FROM recipes r `+candidateAgg+`
		 WHERE NOT EXISTS(SELECT 1 FROM swipes sw WHERE sw.user_id = $1::uuid AND sw.recipe_id = r.id)
		   AND NOT EXISTS(SELECT 1 FROM recipe_ingredients ri JOIN ingredients i ON i.id = ri.ingredient_id
		                  WHERE ri.recipe_id = r.id AND i.allergen_tags && $2::text[])
		   AND NOT EXISTS(SELECT 1 FROM recipe_ingredients ri
		                  WHERE ri.recipe_id = r.id AND ri.ingredient_id = ANY($3::bigint[]))
		   AND (cardinality($4::text[]) = 0 OR r.meal_types && $4::text[])
		   AND (`+diet+`)
		   AND ($5::int IS NULL OR r.prep_minutes + r.cook_minutes <= $5)
		 ORDER BY random() LIMIT $6`,
		userID, f.Allergens, f.ExcludedIDs, f.MealTypes, f.MaxCookMinutes, f.Limit)
	if err != nil {
		return nil, fmt.Errorf("store: discover: %w", err)
	}
	defer rows.Close()
	out := []RecipeSummary{}
	for rows.Next() {
		var rs RecipeSummary
		if err := scanSummary(rows.Scan, &rs, &rs.Liked, &rs.Favorited); err != nil {
			return nil, err
		}
		out = append(out, rs)
	}
	return out, rows.Err()
}
