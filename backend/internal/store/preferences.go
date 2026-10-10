package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Preferences mirrors user_preferences + user_excluded_ingredients.
type Preferences struct {
	HouseholdSize         int      `json:"household_size"`
	MealsPerDay           int      `json:"meals_per_day"`
	MealTypes             []string `json:"meal_types"`
	WeeklyBudget          *float64 `json:"weekly_budget"`
	MaxCookMinutes        *int     `json:"max_cook_minutes"`
	DietType              string   `json:"diet_type"`
	Allergens             []string `json:"allergens"`
	ExcludedIngredientIDs []int64  `json:"excluded_ingredient_ids"`
	Goal                  string   `json:"goal"`
}

// DefaultPreferences is returned when the user never saved preferences.
func DefaultPreferences() Preferences {
	return Preferences{
		HouseholdSize:         2,
		MealsPerDay:           3,
		MealTypes:             []string{"breakfast", "lunch", "dinner"},
		DietType:              "any",
		Allergens:             []string{},
		ExcludedIngredientIDs: []int64{},
	}
}

// GetPreferences returns the saved preferences or (defaults, found=false).
func (s *Store) GetPreferences(ctx context.Context, userID string) (Preferences, bool, error) {
	p := Preferences{}
	err := s.pool.QueryRow(ctx,
		`SELECT household_size, meals_per_day, meal_types, weekly_budget::float8,
		        max_cook_minutes, diet_type, allergens, goal
		 FROM user_preferences WHERE user_id = $1::uuid`, userID).
		Scan(&p.HouseholdSize, &p.MealsPerDay, &p.MealTypes, &p.WeeklyBudget,
			&p.MaxCookMinutes, &p.DietType, &p.Allergens, &p.Goal)
	if err != nil {
		if err == pgx.ErrNoRows {
			return DefaultPreferences(), false, nil
		}
		return p, false, fmt.Errorf("store: get preferences: %w", err)
	}
	ids, err := s.ExcludedIngredientIDs(ctx, userID)
	if err != nil {
		return p, false, err
	}
	p.ExcludedIngredientIDs = ids
	return p, true, nil
}

// ExcludedIngredientIDs lists the user's excluded ingredients.
func (s *Store) ExcludedIngredientIDs(ctx context.Context, userID string) ([]int64, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT ingredient_id FROM user_excluded_ingredients WHERE user_id = $1::uuid`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// SavePreferences upserts the row and replaces the excluded-ingredient set
// atomically. Unknown ingredient ids are ignored.
func (s *Store) SavePreferences(ctx context.Context, userID string, p Preferences) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO user_preferences
			(user_id, household_size, meals_per_day, meal_types, weekly_budget,
			 max_cook_minutes, diet_type, allergens, goal)
		 VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9)
		 ON CONFLICT (user_id) DO UPDATE SET
			household_size=EXCLUDED.household_size, meals_per_day=EXCLUDED.meals_per_day,
			meal_types=EXCLUDED.meal_types, weekly_budget=EXCLUDED.weekly_budget,
			max_cook_minutes=EXCLUDED.max_cook_minutes, diet_type=EXCLUDED.diet_type,
			allergens=EXCLUDED.allergens, goal=EXCLUDED.goal, updated_at=now()`,
		userID, p.HouseholdSize, p.MealsPerDay, p.MealTypes, p.WeeklyBudget,
		p.MaxCookMinutes, p.DietType, p.Allergens, p.Goal)
	if err != nil {
		return fmt.Errorf("store: upsert preferences: %w", err)
	}
	if _, err = tx.Exec(ctx,
		`DELETE FROM user_excluded_ingredients WHERE user_id = $1::uuid`, userID); err != nil {
		return fmt.Errorf("store: clear exclusions: %w", err)
	}
	if len(p.ExcludedIngredientIDs) > 0 {
		if _, err = tx.Exec(ctx,
			`INSERT INTO user_excluded_ingredients (user_id, ingredient_id)
			 SELECT $1::uuid, id FROM ingredients WHERE id = ANY($2::bigint[])`,
			userID, p.ExcludedIngredientIDs); err != nil {
			return fmt.Errorf("store: set exclusions: %w", err)
		}
	}
	return tx.Commit(ctx)
}
