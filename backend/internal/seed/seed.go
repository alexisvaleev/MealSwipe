// Package seed loads the embedded curated recipe dataset (SPEC §8) into the DB.
// It is idempotent: recipes upsert by slug, recipe_ingredients are rebuilt.
package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type file struct {
	Recipes []recipe `json:"recipes"`
}

type recipe struct {
	Slug           string             `json:"slug"`
	Title          string             `json:"title"`
	Description    string             `json:"description"`
	Instructions   []string           `json:"instructions"`
	PrepMinutes    int                `json:"prep_minutes"`
	CookMinutes    int                `json:"cook_minutes"`
	Servings       float64            `json:"servings"`
	MealTypes      []string           `json:"meal_types"`
	Category       string             `json:"category"`
	Tags           []string           `json:"tags"`
	DietFlags      []string           `json:"diet_flags"`
	ImageURL       *string            `json:"image_url"`
	ImageCredit    *string            `json:"image_credit"`
	CostPerServing *float64           `json:"cost_per_serving"`
	KcalPerServing *int               `json:"kcal_per_serving"`
	Ingredients    []recipeIngredient `json:"ingredients"`
	IngredientMeta map[string]meta    `json:"ingredient_meta"`
}

type recipeIngredient struct {
	Name     string  `json:"name"`
	Quantity float64 `json:"quantity"`
	Unit     string  `json:"unit"`
	Note     string  `json:"note"`
}

type meta struct {
	Category     string   `json:"category"`
	UnitFamily   string   `json:"unit_family"`
	AllergenTags []string `json:"allergen_tags"`
}

// Seed parses raw JSON and upserts everything in one transaction.
// Returns the number of recipes processed.
func Seed(ctx context.Context, pool *pgxpool.Pool, raw []byte) (int, error) {
	var f file
	if err := json.Unmarshal(raw, &f); err != nil {
		return 0, fmt.Errorf("seed: parse json: %w", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	for _, r := range f.Recipes {
		if err := upsertRecipe(ctx, tx, r); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("seed: commit: %w", err)
	}
	return len(f.Recipes), nil
}

func upsertRecipe(ctx context.Context, tx pgx.Tx, r recipe) error {
	if r.Slug == "" || r.Title == "" {
		return fmt.Errorf("seed: recipe missing slug/title")
	}
	if r.Servings <= 0 {
		r.Servings = 2
	}
	var recipeID string
	err := tx.QueryRow(ctx,
		`INSERT INTO recipes
			(slug, title, description, instructions, prep_minutes, cook_minutes,
			 servings, meal_types, category, tags, diet_flags, image_url,
			 image_credit, cost_per_serving, kcal_per_serving, source)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,'seed')
		 ON CONFLICT (slug) DO UPDATE SET
			title=EXCLUDED.title, description=EXCLUDED.description,
			instructions=EXCLUDED.instructions, prep_minutes=EXCLUDED.prep_minutes,
			cook_minutes=EXCLUDED.cook_minutes, servings=EXCLUDED.servings,
			meal_types=EXCLUDED.meal_types, category=EXCLUDED.category,
			tags=EXCLUDED.tags, diet_flags=EXCLUDED.diet_flags,
			image_url=EXCLUDED.image_url, image_credit=EXCLUDED.image_credit,
			cost_per_serving=EXCLUDED.cost_per_serving,
			kcal_per_serving=EXCLUDED.kcal_per_serving
		 RETURNING id::text`,
		r.Slug, r.Title, r.Description, r.Instructions, r.PrepMinutes, r.CookMinutes,
		r.Servings, r.MealTypes, r.Category, r.Tags, r.DietFlags, r.ImageURL,
		r.ImageCredit, r.CostPerServing, r.KcalPerServing).Scan(&recipeID)
	if err != nil {
		return fmt.Errorf("seed: upsert recipe %q: %w", r.Slug, err)
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM recipe_ingredients WHERE recipe_id = $1::uuid`, recipeID); err != nil {
		return fmt.Errorf("seed: clear ingredients %q: %w", r.Slug, err)
	}
	for pos, ing := range r.Ingredients {
		ingID, err := upsertIngredient(ctx, tx, ing.Name, r.IngredientMeta)
		if err != nil {
			return err
		}
		unit := ing.Unit
		if unit == "" {
			unit = "pcs"
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO recipe_ingredients (recipe_id, ingredient_id, quantity, unit, note, position)
			 VALUES ($1::uuid,$2,$3,$4,$5,$6)`,
			recipeID, ingID, ing.Quantity, unit, ing.Note, pos); err != nil {
			return fmt.Errorf("seed: link ingredient %q: %w", ing.Name, err)
		}
	}
	return nil
}

func upsertIngredient(ctx context.Context, tx pgx.Tx, name string, metas map[string]meta) (int64, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	m := metas[name]
	if m.Category == "" {
		m.Category = "other"
	}
	if m.UnitFamily == "" {
		m.UnitFamily = "count"
	}
	if m.AllergenTags == nil {
		m.AllergenTags = []string{}
	}
	var id int64
	err := tx.QueryRow(ctx,
		`INSERT INTO ingredients (name, category, unit_family, allergen_tags)
		 VALUES ($1,$2,$3,$4)
		 ON CONFLICT (name) DO UPDATE SET
			category=EXCLUDED.category, unit_family=EXCLUDED.unit_family,
			allergen_tags=EXCLUDED.allergen_tags
		 RETURNING id`, name, m.Category, m.UnitFamily, m.AllergenTags).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("seed: upsert ingredient %q: %w", name, err)
	}
	return id, nil
}
