package httpapi

import (
	"net/http"
	"unicode/utf8"

	"github.com/alexisvaleev/mealswipe/backend/internal/store"
)

var (
	validMealTypes = map[string]bool{"breakfast": true, "lunch": true, "dinner": true, "snack": true}
	validDiets     = map[string]bool{"any": true, "vegetarian": true, "vegan": true, "pescatarian": true}
	validAllergens = map[string]bool{
		"milk": true, "eggs": true, "gluten": true, "nuts": true, "peanuts": true,
		"fish": true, "shellfish": true, "soy": true, "sesame": true,
	}
	validCategories = map[string]bool{
		"produce": true, "dairy": true, "meat": true, "fish": true, "bakery": true,
		"grains": true, "frozen": true, "canned": true, "spices": true, "oils": true, "other": true,
	}
)

func (s *Server) getPrefs(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	p, _, err := s.st.GetPreferences(r.Context(), sess.UserID)
	if s.storeErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"preferences": p})
}

func (s *Server) putPrefs(w http.ResponseWriter, r *http.Request) {
	sess := sessionFrom(r)
	var p store.Preferences
	if !decode(w, r, &p) {
		return
	}
	if msg := validatePrefs(&p); msg != "" {
		writeErr(w, http.StatusBadRequest, "validation", msg)
		return
	}
	if s.storeErr(w, s.st.SavePreferences(r.Context(), sess.UserID, p)) {
		return
	}
	p2, _, err := s.st.GetPreferences(r.Context(), sess.UserID)
	if s.storeErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"preferences": p2})
}

func validatePrefs(p *store.Preferences) string {
	if p.HouseholdSize < 1 || p.HouseholdSize > 20 {
		return "household_size must be 1..20"
	}
	if p.MealsPerDay < 1 || p.MealsPerDay > 6 {
		return "meals_per_day must be 1..6"
	}
	if len(p.MealTypes) == 0 {
		return "meal_types must not be empty"
	}
	for _, mt := range p.MealTypes {
		if !validMealTypes[mt] {
			return "unknown meal_type: " + mt
		}
	}
	if !validDiets[p.DietType] {
		return "unknown diet_type"
	}
	for _, a := range p.Allergens {
		if !validAllergens[a] {
			return "unknown allergen: " + a
		}
	}
	if p.WeeklyBudget != nil && *p.WeeklyBudget < 0 {
		return "weekly_budget must be >= 0"
	}
	if p.MaxCookMinutes != nil && *p.MaxCookMinutes <= 0 {
		return "max_cook_minutes must be > 0"
	}
	if utf8.RuneCountInString(p.Goal) > 200 {
		return "goal too long"
	}
	if p.ExcludedIngredientIDs == nil {
		p.ExcludedIngredientIDs = []int64{}
	}
	if p.Allergens == nil {
		p.Allergens = []string{}
	}
	return ""
}
