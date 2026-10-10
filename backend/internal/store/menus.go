package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/alexisvaleev/mealswipe/backend/internal/planner"
)

// Menu is a weekly menu (unique per user+week_start).
type Menu struct {
	ID        string     `json:"id"`
	WeekStart string     `json:"week_start"`
	Status    string     `json:"status"`
	Items     []MenuItem `json:"items"`
}

// MenuItem is one filled slot; it embeds its recipe summary (SPEC §5).
type MenuItem struct {
	ID       string        `json:"id"`
	DayIndex int           `json:"day_index"`
	MealType string        `json:"meal_type"`
	Servings float64       `json:"servings"`
	Locked   bool          `json:"locked"`
	Recipe   RecipeSummary `json:"recipe"`
}

const mealOrder = `array_position('{breakfast,lunch,dinner,snack}'::text[], mi.meal_type)`

func (s *Store) menuByWhere(ctx context.Context, where string, args ...any) (*Menu, error) {
	var m Menu
	err := s.pool.QueryRow(ctx,
		`SELECT id::text, week_start::text, status FROM menus WHERE `+where, args...).
		Scan(&m.ID, &m.WeekStart, &m.Status)
	if err != nil {
		return nil, mapErr(err)
	}
	if err := s.loadItems(ctx, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// GetMenuByID loads a menu only if it belongs to userID.
func (s *Store) GetMenuByID(ctx context.Context, userID, menuID string) (*Menu, error) {
	return s.menuByWhere(ctx, `user_id = $1::uuid AND id = $2::uuid`, userID, menuID)
}

// GetMenuByWeek loads the menu for a week (YYYY-MM-DD Monday), ErrNotFound if absent.
func (s *Store) GetMenuByWeek(ctx context.Context, userID, weekStart string) (*Menu, error) {
	return s.menuByWhere(ctx, `user_id = $1::uuid AND week_start = $2::date`, userID, weekStart)
}

// GetOrCreateMenu returns the existing menu or creates an active one.
func (s *Store) GetOrCreateMenu(ctx context.Context, userID, weekStart string) (*Menu, error) {
	var m Menu
	err := s.pool.QueryRow(ctx,
		`INSERT INTO menus (user_id, week_start) VALUES ($1::uuid,$2::date)
		 ON CONFLICT (user_id, week_start) DO UPDATE SET updated_at = now()
		 RETURNING id::text, week_start::text, status`, userID, weekStart).
		Scan(&m.ID, &m.WeekStart, &m.Status)
	if err != nil {
		return nil, fmt.Errorf("store: get-or-create menu: %w", err)
	}
	if err := s.loadItems(ctx, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

const itemCols = `mi.id::text, mi.day_index, mi.meal_type, mi.servings::float8, mi.locked,
	r.id::text, r.slug, r.title, r.description, r.image_url,
	r.prep_minutes, r.cook_minutes, r.servings::float8, r.meal_types, r.category,
	r.tags, r.diet_flags, r.cost_per_serving::float8, r.kcal_per_serving, r.nutrition_estimate`

func (s *Store) scanItem(scan func(...any) error) (*MenuItem, error) {
	var it MenuItem
	err := scan(&it.ID, &it.DayIndex, &it.MealType, &it.Servings, &it.Locked,
		&it.Recipe.ID, &it.Recipe.Slug, &it.Recipe.Title, &it.Recipe.Description,
		&it.Recipe.ImageURL, &it.Recipe.PrepMinutes, &it.Recipe.CookMinutes,
		&it.Recipe.Servings, &it.Recipe.MealTypes, &it.Recipe.Category, &it.Recipe.Tags,
		&it.Recipe.DietFlags, &it.Recipe.CostPerServing, &it.Recipe.KcalPerServing,
		&it.Recipe.NutritionEstimate)
	if err != nil {
		return nil, err
	}
	return &it, nil
}

func (s *Store) loadItems(ctx context.Context, m *Menu) error {
	rows, err := s.pool.Query(ctx,
		`SELECT `+itemCols+` FROM menu_items mi JOIN recipes r ON r.id = mi.recipe_id
		 WHERE mi.menu_id = $1::uuid ORDER BY mi.day_index, `+mealOrder+`, mi.created_at`, m.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	m.Items = []MenuItem{}
	for rows.Next() {
		it, err := s.scanItem(rows.Scan)
		if err != nil {
			return err
		}
		m.Items = append(m.Items, *it)
	}
	return rows.Err()
}

// GetMenuItem returns one item of a user's menu.
func (s *Store) GetMenuItem(ctx context.Context, userID, menuID, itemID string) (*MenuItem, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+itemCols+` FROM menu_items mi
		 JOIN recipes r ON r.id = mi.recipe_id
		 JOIN menus m ON m.id = mi.menu_id
		 WHERE m.user_id = $1::uuid AND mi.menu_id = $2::uuid AND mi.id = $3::uuid`,
		userID, menuID, itemID)
	it, err := s.scanItem(row.Scan)
	if err != nil {
		return nil, mapErr(err)
	}
	return it, nil
}

// UpsertMenuItem inserts or replaces the recipe in a slot (unique slot key).
func (s *Store) UpsertMenuItem(ctx context.Context, userID, menuID string, dayIndex int, mealType, recipeID string, servings float64) (*MenuItem, error) {
	var id string
	err := s.pool.QueryRow(ctx,
		`INSERT INTO menu_items (menu_id, day_index, meal_type, recipe_id, servings)
		 SELECT $2::uuid, $3, $4, $5::uuid, $6
		 WHERE EXISTS(SELECT 1 FROM menus WHERE id = $2::uuid AND user_id = $1::uuid)
		 ON CONFLICT (menu_id, day_index, meal_type)
		 DO UPDATE SET recipe_id = EXCLUDED.recipe_id, servings = EXCLUDED.servings
		 RETURNING id::text`,
		userID, menuID, dayIndex, mealType, recipeID, servings).Scan(&id)
	if err != nil {
		if isFKViolation(err) {
			return nil, ErrNotFound
		}
		return nil, mapErr(err)
	}
	return s.GetMenuItem(ctx, userID, menuID, id)
}

// ItemPatch describes optional updates to a menu item.
type ItemPatch struct {
	RecipeID *string
	Servings *float64
	DayIndex *int
	MealType *string
	Locked   *bool
}

// PatchMenuItem applies a partial update; ErrConflict on slot collision.
func (s *Store) PatchMenuItem(ctx context.Context, userID, menuID, itemID string, p ItemPatch) (*MenuItem, error) {
	sets, args := []string{}, []any{userID, menuID, itemID}
	n := 4
	if p.RecipeID != nil {
		sets = append(sets, fmt.Sprintf("recipe_id = $%d::uuid", n))
		args = append(args, *p.RecipeID)
		n++
	}
	if p.Servings != nil {
		sets = append(sets, fmt.Sprintf("servings = $%d", n))
		args = append(args, *p.Servings)
		n++
	}
	if p.DayIndex != nil {
		sets = append(sets, fmt.Sprintf("day_index = $%d", n))
		args = append(args, *p.DayIndex)
		n++
	}
	if p.MealType != nil {
		sets = append(sets, fmt.Sprintf("meal_type = $%d", n))
		args = append(args, *p.MealType)
		n++
	}
	if p.Locked != nil {
		sets = append(sets, fmt.Sprintf("locked = $%d", n))
		args = append(args, *p.Locked)
		n++
	}
	if len(sets) == 0 {
		return s.GetMenuItem(ctx, userID, menuID, itemID)
	}
	var id string
	err := s.pool.QueryRow(ctx,
		`UPDATE menu_items mi SET `+strings.Join(sets, ", ")+`
		 FROM menus m WHERE m.id = mi.menu_id AND m.user_id = $1::uuid
		   AND mi.menu_id = $2::uuid AND mi.id = $3::uuid
		 RETURNING mi.id::text`, args...).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		if isFKViolation(err) {
			return nil, ErrNotFound
		}
		return nil, mapErr(err)
	}
	return s.GetMenuItem(ctx, userID, menuID, id)
}

// DeleteMenuItem removes an item from a user's menu.
func (s *Store) DeleteMenuItem(ctx context.Context, userID, menuID, itemID string) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM menu_items mi USING menus m
		 WHERE m.id = mi.menu_id AND m.user_id = $1::uuid
		   AND mi.menu_id = $2::uuid AND mi.id = $3::uuid`, userID, menuID, itemID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ReplaceUnlocked deletes all non-locked items and inserts the picks in one tx.
func (s *Store) ReplaceUnlocked(ctx context.Context, menuID string, picks []planner.PlacedItem) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx,
		`DELETE FROM menu_items WHERE menu_id = $1::uuid AND NOT locked`, menuID); err != nil {
		return fmt.Errorf("store: clear unlocked: %w", err)
	}
	for _, p := range picks {
		if _, err := tx.Exec(ctx,
			`INSERT INTO menu_items (menu_id, day_index, meal_type, recipe_id, servings)
			 VALUES ($1::uuid,$2,$3,$4::uuid,$5)
			 ON CONFLICT (menu_id, day_index, meal_type) DO NOTHING`,
			menuID, p.DayIndex, p.MealType, p.RecipeID, p.Servings); err != nil {
			return fmt.Errorf("store: insert pick: %w", err)
		}
	}
	return tx.Commit(ctx)
}

// SetMenuItemRecipe points an item at a new recipe (swap).
func (s *Store) SetMenuItemRecipe(ctx context.Context, userID, menuID, itemID, recipeID string) (*MenuItem, error) {
	var id string
	err := s.pool.QueryRow(ctx,
		`UPDATE menu_items mi SET recipe_id = $4::uuid
		 FROM menus m WHERE m.id = mi.menu_id AND m.user_id = $1::uuid
		   AND mi.menu_id = $2::uuid AND mi.id = $3::uuid
		 RETURNING mi.id::text`, userID, menuID, itemID, recipeID).Scan(&id)
	if err != nil {
		if isFKViolation(err) {
			return nil, ErrNotFound
		}
		return nil, mapErr(err)
	}
	return s.GetMenuItem(ctx, userID, menuID, id)
}

// PlacedItems loads menu items as planner context (with candidate data).
// lockedOnly=true returns only locked items; excludeItemID skips one item.
func (s *Store) PlacedItems(ctx context.Context, userID, menuID string, lockedOnly bool, excludeItemID string) ([]planner.PlacedItem, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT mi.day_index, mi.meal_type, mi.servings::float8,
		        r.id::text, r.meal_types, r.diet_flags, r.prep_minutes, r.cook_minutes,
		        r.category, r.servings::float8, r.cost_per_serving::float8,
		        COALESCE(ii.ing_ids, '{}'::bigint[]), COALESCE(ii.tags, '{}'::text[])
		 FROM menu_items mi
		 JOIN recipes r ON r.id = mi.recipe_id
		 JOIN menus m ON m.id = mi.menu_id `+candidateAgg+`
		 WHERE m.user_id = $1::uuid AND mi.menu_id = $2::uuid
		   AND ($3::bool = false OR mi.locked)
		   AND ($4::text = '' OR mi.id <> $4::uuid)
		 ORDER BY mi.day_index, `+mealOrder,
		userID, menuID, lockedOnly, excludeItemID)
	if err != nil {
		return nil, fmt.Errorf("store: placed items: %w", err)
	}
	defer rows.Close()
	out := []planner.PlacedItem{}
	for rows.Next() {
		var p planner.PlacedItem
		var c Candidate
		if err := rows.Scan(&p.DayIndex, &p.MealType, &p.Servings,
			&c.RecipeID, &c.MealTypes, &c.DietFlags, &c.PrepMinutes, &c.CookMinutes,
			&c.Category, &c.Servings, &c.CostPerServing, &c.IngredientIDs, &c.AllergenTags); err != nil {
			return nil, err
		}
		p.RecipeID = c.RecipeID
		p.Candidate = c
		out = append(out, p)
	}
	return out, rows.Err()
}

// PrevWeekRecipeIDs returns recipe ids that were on the menu 7 days earlier.
func (s *Store) PrevWeekRecipeIDs(ctx context.Context, userID, weekStart string) (map[string]bool, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT mi.recipe_id::text FROM menus m
		 JOIN menu_items mi ON mi.menu_id = m.id
		 WHERE m.user_id = $1::uuid AND m.week_start = ($2::date - interval '7 days')::date`,
		userID, weekStart)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}
