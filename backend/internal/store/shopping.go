package store

import (
	"context"
	"fmt"

	"github.com/alexisvaleev/mealswipe/backend/internal/shopping"
	"github.com/jackc/pgx/v5"
)

// ShoppingList is a list bound (usually) to a menu.
type ShoppingList struct {
	ID     string         `json:"id"`
	MenuID *string        `json:"menu_id"`
	Name   string         `json:"name"`
	Items  []ShoppingItem `json:"items"`
}

// ShoppingItem is one line; ingredient_id is NULL for manual entries.
type ShoppingItem struct {
	ID           string   `json:"id"`
	IngredientID *int64   `json:"ingredient_id"`
	Name         string   `json:"name"`
	Quantity     *float64 `json:"quantity"`
	Unit         string   `json:"unit"`
	Category     string   `json:"category"`
	Checked      bool     `json:"checked"`
	IsManual     bool     `json:"is_manual"`
	Position     int      `json:"position"`
}

func (s *Store) listByWhere(ctx context.Context, where string, args ...any) (*ShoppingList, error) {
	var l ShoppingList
	err := s.pool.QueryRow(ctx,
		`SELECT id::text, menu_id::text, name FROM shopping_lists WHERE `+where, args...).
		Scan(&l.ID, &l.MenuID, &l.Name)
	if err != nil {
		return nil, mapErr(err)
	}
	if err := s.loadListItems(ctx, &l); err != nil {
		return nil, err
	}
	return &l, nil
}

// GetList returns a list owned by userID with its items.
func (s *Store) GetList(ctx context.Context, userID, listID string) (*ShoppingList, error) {
	return s.listByWhere(ctx, `user_id = $1::uuid AND id = $2::uuid`, userID, listID)
}

// ListByMenu returns the list bound to a menu (ErrNotFound when absent).
func (s *Store) ListByMenu(ctx context.Context, userID, menuID string) (*ShoppingList, error) {
	return s.listByWhere(ctx, `user_id = $1::uuid AND menu_id = $2::uuid`, userID, menuID)
}

// GetOrCreateListForMenu returns (list, created, err).
func (s *Store) GetOrCreateListForMenu(ctx context.Context, userID, menuID string) (*ShoppingList, bool, error) {
	var id string
	err := s.pool.QueryRow(ctx,
		`INSERT INTO shopping_lists (user_id, menu_id) VALUES ($1::uuid,$2::uuid)
		 ON CONFLICT (user_id, menu_id) DO NOTHING RETURNING id::text`,
		userID, menuID).Scan(&id)
	if err == pgx.ErrNoRows {
		l, err := s.ListByMenu(ctx, userID, menuID)
		return l, false, err
	}
	if err != nil {
		return nil, false, fmt.Errorf("store: create list: %w", err)
	}
	l, err := s.GetList(ctx, userID, id)
	return l, true, err
}

// HasGeneratedItems reports whether the list already has non-manual items.
func (s *Store) HasGeneratedItems(ctx context.Context, listID string) (bool, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM shopping_list_items WHERE list_id = $1::uuid AND NOT is_manual`,
		listID).Scan(&n)
	return n > 0, err
}

func (s *Store) loadListItems(ctx context.Context, l *ShoppingList) error {
	rows, err := s.pool.Query(ctx,
		`SELECT id::text, ingredient_id, name, quantity::float8, unit, category,
		        checked, is_manual, position
		 FROM shopping_list_items WHERE list_id = $1::uuid
		 ORDER BY position, id`, l.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	l.Items = []ShoppingItem{}
	for rows.Next() {
		var it ShoppingItem
		if err := rows.Scan(&it.ID, &it.IngredientID, &it.Name, &it.Quantity,
			&it.Unit, &it.Category, &it.Checked, &it.IsManual, &it.Position); err != nil {
			return err
		}
		l.Items = append(l.Items, it)
	}
	return rows.Err()
}

// GenerateListItems rebuilds non-manual items from the bound menu (SPEC §7):
// manual rows survive; checked state is preserved by (ingredient_id,unit,name).
func (s *Store) GenerateListItems(ctx context.Context, userID, listID string) error {
	parts, err := s.listParts(ctx, userID, listID)
	if err != nil {
		return err
	}
	pantry, err := s.PantryIDs(ctx, userID)
	if err != nil {
		return err
	}
	items := shopping.Generate(parts, pantry)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	checked := map[string]bool{}
	rows, err := tx.Query(ctx,
		`SELECT ingredient_id, unit, name FROM shopping_list_items
		 WHERE list_id = $1::uuid AND checked AND ingredient_id IS NOT NULL`, listID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int64
		var unit, name string
		if err := rows.Scan(&id, &unit, &name); err != nil {
			rows.Close()
			return err
		}
		checked[checkKey(id, unit, name)] = true
	}
	rows.Close()

	if _, err := tx.Exec(ctx,
		`DELETE FROM shopping_list_items WHERE list_id = $1::uuid AND NOT is_manual`, listID); err != nil {
		return fmt.Errorf("store: clear generated: %w", err)
	}
	for _, it := range items {
		isChecked := it.IngredientID != nil && checked[checkKey(*it.IngredientID, it.Unit, it.Name)]
		if _, err := tx.Exec(ctx,
			`INSERT INTO shopping_list_items
				(list_id, ingredient_id, name, quantity, unit, category, checked, is_manual, position)
			 VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,false,$8)`,
			listID, it.IngredientID, it.Name, it.Quantity, it.Unit, it.Category,
			isChecked, it.Position); err != nil {
			return fmt.Errorf("store: insert generated item: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func checkKey(id int64, unit, name string) string {
	return fmt.Sprintf("%d\x00%s\x00%s", id, unit, name)
}

// listParts loads the menu's recipe lines for the shopping generator.
func (s *Store) listParts(ctx context.Context, userID, listID string) ([]shopping.Part, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT mi.id::text, mi.servings::float8, r.servings::float8,
		        ri.ingredient_id, i.name, i.category, i.unit_family,
		        ri.quantity::float8, ri.unit
		 FROM shopping_lists l
		 JOIN menu_items mi ON mi.menu_id = l.menu_id
		 JOIN recipes r ON r.id = mi.recipe_id
		 JOIN recipe_ingredients ri ON ri.recipe_id = r.id
		 JOIN ingredients i ON i.id = ri.ingredient_id
		 WHERE l.id = $1::uuid AND l.user_id = $2::uuid
		 ORDER BY mi.day_index, mi.id, ri.position`, listID, userID)
	if err != nil {
		return nil, fmt.Errorf("store: list parts: %w", err)
	}
	defer rows.Close()
	byItem := map[string]*shopping.Part{}
	var order []*shopping.Part
	for rows.Next() {
		var itemID string
		var part struct {
			itemServ, recipeServ float64
		}
		var ln shopping.Line
		if err := rows.Scan(&itemID, &part.itemServ, &part.recipeServ,
			&ln.IngredientID, &ln.Name, &ln.Category, &ln.UnitFamily,
			&ln.Quantity, &ln.Unit); err != nil {
			return nil, err
		}
		p, ok := byItem[itemID]
		if !ok {
			p = &shopping.Part{RecipeServings: part.recipeServ, ItemServings: part.itemServ}
			byItem[itemID] = p
			order = append(order, p)
		}
		p.Lines = append(p.Lines, ln)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]shopping.Part, len(order))
	for i, p := range order {
		out[i] = *p
	}
	return out, nil
}

// AddListItem appends a manual item.
func (s *Store) AddListItem(ctx context.Context, listID, name string, quantity *float64, unit, category string) (*ShoppingItem, error) {
	var it ShoppingItem
	err := s.pool.QueryRow(ctx,
		`INSERT INTO shopping_list_items (list_id, name, quantity, unit, category, is_manual, position)
		 VALUES ($1::uuid, $2, $3, $4, $5, true,
		         COALESCE((SELECT max(position) FROM shopping_list_items WHERE list_id = $1::uuid), -1) + 1)
		 RETURNING id::text, ingredient_id, name, quantity::float8, unit, category, checked, is_manual, position`,
		listID, name, quantity, unit, category).
		Scan(&it.ID, &it.IngredientID, &it.Name, &it.Quantity, &it.Unit,
			&it.Category, &it.Checked, &it.IsManual, &it.Position)
	if err != nil {
		return nil, fmt.Errorf("store: add list item: %w", err)
	}
	return &it, nil
}

// ListItemPatch describes optional updates to a shopping item.
type ListItemPatch struct {
	Checked  *bool
	Quantity *float64
	Name     *string
}

// PatchListItem applies a partial update to an item of a user's list.
func (s *Store) PatchListItem(ctx context.Context, userID, listID, itemID string, p ListItemPatch) (*ShoppingItem, error) {
	var it ShoppingItem
	err := s.pool.QueryRow(ctx,
		`UPDATE shopping_list_items si SET
			checked  = COALESCE($4, si.checked),
			quantity = COALESCE($5, si.quantity),
			name     = COALESCE($6, si.name)
		 FROM shopping_lists l
		 WHERE l.id = si.list_id AND l.user_id = $1::uuid
		   AND si.list_id = $2::uuid AND si.id = $3::uuid
		 RETURNING si.id::text, si.ingredient_id, si.name, si.quantity::float8,
		           si.unit, si.category, si.checked, si.is_manual, si.position`,
		userID, listID, itemID, p.Checked, p.Quantity, p.Name).
		Scan(&it.ID, &it.IngredientID, &it.Name, &it.Quantity, &it.Unit,
			&it.Category, &it.Checked, &it.IsManual, &it.Position)
	if err != nil {
		return nil, mapErr(err)
	}
	return &it, nil
}

// DeleteListItem removes an item from a user's list.
func (s *Store) DeleteListItem(ctx context.Context, userID, listID, itemID string) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM shopping_list_items si USING shopping_lists l
		 WHERE l.id = si.list_id AND l.user_id = $1::uuid
		   AND si.list_id = $2::uuid AND si.id = $3::uuid`, userID, listID, itemID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
