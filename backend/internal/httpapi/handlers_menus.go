package httpapi

import (
	"net/http"
	"time"

	"github.com/alexisvaleev/mealswipe/backend/internal/planner"
	"github.com/alexisvaleev/mealswipe/backend/internal/store"
)

func toPlannerPrefs(p store.Preferences) planner.Prefs {
	allergens := map[string]bool{}
	for _, a := range p.Allergens {
		allergens[a] = true
	}
	excl := map[int64]bool{}
	for _, id := range p.ExcludedIngredientIDs {
		excl[id] = true
	}
	return planner.Prefs{
		MealTypes:             p.MealTypes,
		DietType:              p.DietType,
		Allergens:             allergens,
		ExcludedIngredientIDs: excl,
		MaxCookMinutes:        p.MaxCookMinutes,
		WeeklyBudget:          p.WeeklyBudget,
	}
}

// plannerInput loads the shared inputs for generation/swap.
func (s *Server) plannerInput(r *http.Request, sess *sessionInfo, prefs store.Preferences,
	placed []planner.PlacedItem, weekStart string) (planner.Input, error) {
	pool, err := s.st.PlannerCandidates(r.Context(), sess.UserID)
	if err != nil {
		return planner.Input{}, err
	}
	prev, err := s.st.PrevWeekRecipeIDs(r.Context(), sess.UserID, weekStart)
	if err != nil {
		return planner.Input{}, err
	}
	return planner.Input{
		Prefs:           toPlannerPrefs(prefs),
		DefaultServings: float64(prefs.HouseholdSize),
		Pool:            pool,
		Placed:          placed,
		PrevWeek:        prev,
		Seed:            time.Now().UnixNano(),
	}, nil
}

// currentMenu returns the menu for ?week=YYYY-MM-DD (default: this week).
func (s *Server) currentMenu(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	week := monday(time.Now())
	if v := r.URL.Query().Get("week"); v != "" {
		t, err := parseDate(v)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "validation", "week must be YYYY-MM-DD")
			return
		}
		week = monday(t)
	}
	m, err := s.st.GetMenuByWeek(r.Context(), sess.UserID, week.Format("2006-01-02"))
	if s.storeErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"menu": m})
}

type generateReq struct {
	WeekStart string `json:"week_start"`
}

// generateMenu creates (or replaces non-locked items of) the week's menu.
func (s *Server) generateMenu(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	var in generateReq
	if !decode(w, r, &in) {
		return
	}
	t, err := parseDate(in.WeekStart)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "validation", "week_start must be YYYY-MM-DD")
		return
	}
	week := monday(t).Format("2006-01-02")
	m, err := s.st.GetOrCreateMenu(r.Context(), sess.UserID, week)
	if s.storeErr(w, err) {
		return
	}
	prefs, _, err := s.st.GetPreferences(r.Context(), sess.UserID)
	if s.storeErr(w, err) {
		return
	}
	locked, err := s.st.PlacedItems(r.Context(), sess.UserID, m.ID, true, "")
	if s.storeErr(w, err) {
		return
	}
	input, err := s.plannerInput(r, sess, prefs, locked, week)
	if s.storeErr(w, err) {
		return
	}
	picks, unfilled := planner.Generate(input)
	if s.storeErr(w, s.st.ReplaceUnlocked(r.Context(), m.ID, picks)) {
		return
	}
	m2, err := s.st.GetMenuByID(r.Context(), sess.UserID, m.ID)
	if s.storeErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"menu": m2, "unfilled": unfilled})
}

func (s *Server) getMenu(w http.ResponseWriter, r *http.Request) {
	if !requireUUID(w, r, "id") {
		return
	}
	sess := sessionFrom(r)
	m, err := s.st.GetMenuByID(r.Context(), sess.UserID, r.PathValue("id"))
	if s.storeErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"menu": m})
}

type addItemReq struct {
	DayIndex int     `json:"day_index"`
	MealType string  `json:"meal_type"`
	RecipeID string  `json:"recipe_id"`
	Servings float64 `json:"servings"`
}

func (s *Server) addMenuItem(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	var in addItemReq
	if !decode(w, r, &in) {
		return
	}
	if !requireUUID(w, r, "id") {
		return
	}
	if in.DayIndex < 0 || in.DayIndex > 6 || !validMealTypes[in.MealType] || !uuidLike(in.RecipeID) {
		writeErr(w, http.StatusBadRequest, "validation", "bad day_index, meal_type or recipe_id")
		return
	}
	if in.Servings <= 0 {
		p, _, _ := s.st.GetPreferences(r.Context(), sess.UserID)
		in.Servings = float64(p.HouseholdSize)
	}
	it, err := s.st.UpsertMenuItem(r.Context(), sess.UserID, r.PathValue("id"),
		in.DayIndex, in.MealType, in.RecipeID, in.Servings)
	if s.storeErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item": it})
}

type patchItemReq struct {
	RecipeID *string  `json:"recipe_id"`
	Servings *float64 `json:"servings"`
	DayIndex *int     `json:"day_index"`
	MealType *string  `json:"meal_type"`
	Locked   *bool    `json:"locked"`
}

func (s *Server) patchMenuItem(w http.ResponseWriter, r *http.Request) {
	if !requireUUID(w, r, "id", "item_id") {
		return
	}
	sess := sessionFrom(r)
	var in patchItemReq
	if !decode(w, r, &in) {
		return
	}
	if in.DayIndex != nil && (*in.DayIndex < 0 || *in.DayIndex > 6) {
		writeErr(w, http.StatusBadRequest, "validation", "day_index must be 0..6")
		return
	}
	if in.MealType != nil && !validMealTypes[*in.MealType] {
		writeErr(w, http.StatusBadRequest, "validation", "unknown meal_type")
		return
	}
	if in.RecipeID != nil && !uuidLike(*in.RecipeID) {
		writeErr(w, http.StatusBadRequest, "validation", "bad recipe_id")
		return
	}
	if in.Servings != nil && *in.Servings <= 0 {
		writeErr(w, http.StatusBadRequest, "validation", "servings must be > 0")
		return
	}
	it, err := s.st.PatchMenuItem(r.Context(), sess.UserID, r.PathValue("id"),
		r.PathValue("item_id"), store.ItemPatch{
			RecipeID: in.RecipeID, Servings: in.Servings,
			DayIndex: in.DayIndex, MealType: in.MealType, Locked: in.Locked,
		})
	if s.storeErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item": it})
}

func (s *Server) deleteMenuItem(w http.ResponseWriter, r *http.Request) {
	if !requireUUID(w, r, "id", "item_id") {
		return
	}
	sess := sessionFrom(r)
	if s.storeErr(w, s.st.DeleteMenuItem(r.Context(), sess.UserID,
		r.PathValue("id"), r.PathValue("item_id"))) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// swapMenuItem replaces one slot's recipe with a planner pick that honours
// the same hard filters (current recipe + all other menu recipes excluded).
func (s *Server) swapMenuItem(w http.ResponseWriter, r *http.Request) {
	if !requireUUID(w, r, "id", "item_id") {
		return
	}
	sess := sessionFrom(r)
	menuID, itemID := r.PathValue("id"), r.PathValue("item_id")
	item, err := s.st.GetMenuItem(r.Context(), sess.UserID, menuID, itemID)
	if s.storeErr(w, err) {
		return
	}
	menu, err := s.st.GetMenuByID(r.Context(), sess.UserID, menuID)
	if s.storeErr(w, err) {
		return
	}
	prefs, _, err := s.st.GetPreferences(r.Context(), sess.UserID)
	if s.storeErr(w, err) {
		return
	}
	others, err := s.st.PlacedItems(r.Context(), sess.UserID, menuID, false, itemID)
	if s.storeErr(w, err) {
		return
	}
	input, err := s.plannerInput(r, sess, prefs, others, menu.WeekStart)
	if s.storeErr(w, err) {
		return
	}
	input.Exclude = map[string]bool{item.Recipe.ID: true}
	pick, ok := planner.SwapFor(input, item.DayIndex, item.MealType, item.Servings)
	if !ok {
		writeErr(w, http.StatusNotFound, "no_candidates", "no replacement recipe available")
		return
	}
	it, err := s.st.SetMenuItemRecipe(r.Context(), sess.UserID, menuID, itemID, pick.RecipeID)
	if s.storeErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"item": it})
}

// menuShoppingList auto-creates the list bound to the menu and generates
// items when the list has no generated lines yet; explicit rebuilds go
// through POST /shopping-lists/{id}/regenerate.
func (s *Server) menuShoppingList(w http.ResponseWriter, r *http.Request) {
	if !requireUUID(w, r, "id") {
		return
	}
	sess := sessionFrom(r)
	m, err := s.st.GetMenuByID(r.Context(), sess.UserID, r.PathValue("id"))
	if s.storeErr(w, err) {
		return
	}
	list, created, err := s.st.GetOrCreateListForMenu(r.Context(), sess.UserID, m.ID)
	if s.storeErr(w, err) {
		return
	}
	has, err := s.st.HasGeneratedItems(r.Context(), list.ID)
	if s.storeErr(w, err) {
		return
	}
	if created || !has {
		if s.storeErr(w, s.st.GenerateListItems(r.Context(), sess.UserID, list.ID)) {
			return
		}
	}
	l2, err := s.st.GetList(r.Context(), sess.UserID, list.ID)
	if s.storeErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"list": l2})
}
