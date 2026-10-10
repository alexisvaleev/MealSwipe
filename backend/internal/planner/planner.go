// Package planner generates weekly menus from recipe candidates.
// It is a pure package: no I/O, deterministic for a given Input.Seed (SPEC §6).
package planner

import (
	"fmt"
	"hash/fnv"
	"math"
	"sort"
)

// Candidate is a recipe evaluated for a menu slot.
type Candidate struct {
	RecipeID       string
	MealTypes      []string
	DietFlags      []string
	PrepMinutes    int
	CookMinutes    int
	Category       string
	Servings       float64
	CostPerServing *float64
	IngredientIDs  []int64
	AllergenTags   []string
	Liked          bool
	Disliked       bool
	Favorited      bool
}

// Prefs holds the user-level constraints and signals.
type Prefs struct {
	MealTypes             []string
	DietType              string
	Allergens             map[string]bool
	ExcludedIngredientIDs map[int64]bool
	MaxCookMinutes        *int
	WeeklyBudget          *float64
}

// PlacedItem is a recipe assigned to one menu slot. Candidate carries the
// data the scorer needs when this item is already on the menu (context).
type PlacedItem struct {
	DayIndex  int
	MealType  string
	Servings  float64
	RecipeID  string
	Candidate Candidate
}

// Unfilled reports a slot for which no candidate passed the hard filters.
type Unfilled struct {
	DayIndex int    `json:"day_index"`
	MealType string `json:"meal_type"`
	Reason   string `json:"reason"`
}

// Input is everything Generate/SwapFor need.
type Input struct {
	Prefs           Prefs
	DefaultServings float64         // servings assigned to each generated item
	Pool            []Candidate     // recipes eligible for scoring
	Placed          []PlacedItem    // already on the menu (locked items; for swap: all other items)
	PrevWeek        map[string]bool // recipe ids that were in last week's menu
	Exclude         map[string]bool // extra recipe ids to skip (e.g. swapped-out recipe)
	Seed            int64           // controls the ±5 jitter deterministically
}

type slotKey struct {
	day int
	mt  string
}

// state accumulates the menu built so far for reuse/variety/budget scoring.
type state struct {
	ingredients map[int64]int
	categories  map[string]int
	spent       float64
}

func newState() *state {
	return &state{ingredients: map[int64]int{}, categories: map[string]int{}}
}

func (st *state) add(c Candidate, servings float64) {
	for _, id := range c.IngredientIDs {
		st.ingredients[id]++
	}
	st.categories[c.Category]++
	if c.CostPerServing != nil {
		st.spent += *c.CostPerServing * servings
	}
}

// Generate fills slots day 0..6 × Prefs.MealTypes (in the configured order).
// Slots already covered by Placed items are skipped; picks never repeat a
// recipe already on the menu and never violate hard filters.
func Generate(in Input) (picks []PlacedItem, unfilled []Unfilled) {
	pool := sortedPool(in.Pool)
	st := newState()
	used := map[string]bool{}
	occupied := map[slotKey]bool{}
	for _, p := range in.Placed {
		used[p.RecipeID] = true
		occupied[slotKey{p.DayIndex, p.MealType}] = true
		st.add(p.Candidate, p.Servings)
	}
	for day := 0; day < 7; day++ {
		for _, mt := range in.Prefs.MealTypes {
			if occupied[slotKey{day, mt}] {
				continue
			}
			item, ok := best(in, pool, st, day, mt, in.DefaultServings, used)
			if !ok {
				unfilled = append(unfilled, Unfilled{DayIndex: day, MealType: mt, Reason: "no_candidates"})
				continue
			}
			picks = append(picks, item)
			used[item.RecipeID] = true
			st.add(item.Candidate, item.Servings)
		}
	}
	return picks, unfilled
}

// SwapFor picks a replacement for one slot. in.Placed should hold every other
// menu item (for context and exclusion), in.Exclude the swapped-out recipe.
func SwapFor(in Input, dayIndex int, mealType string, servings float64) (PlacedItem, bool) {
	pool := sortedPool(in.Pool)
	st := newState()
	used := map[string]bool{}
	for _, p := range in.Placed {
		used[p.RecipeID] = true
		st.add(p.Candidate, p.Servings)
	}
	return best(in, pool, st, dayIndex, mealType, servings, used)
}

func sortedPool(pool []Candidate) []Candidate {
	out := append([]Candidate(nil), pool...)
	sort.Slice(out, func(i, j int) bool { return out[i].RecipeID < out[j].RecipeID })
	return out
}

// best returns the argmax-scored eligible candidate for a slot.
func best(in Input, pool []Candidate, st *state, day int, mt string, servings float64, used map[string]bool) (PlacedItem, bool) {
	bestIdx, bestScore := -1, math.Inf(-1)
	for i, c := range pool {
		if used[c.RecipeID] || in.Exclude[c.RecipeID] {
			continue
		}
		if ok, _ := eligible(c, in.Prefs, mt); !ok {
			continue
		}
		if sc := score(c, in, st, day, mt, servings); sc > bestScore {
			bestScore, bestIdx = sc, i
		}
	}
	if bestIdx < 0 {
		return PlacedItem{}, false
	}
	c := pool[bestIdx]
	return PlacedItem{DayIndex: day, MealType: mt, Servings: servings, RecipeID: c.RecipeID, Candidate: c}, true
}

// eligible applies the hard filters for a slot. An empty mealType means "any
// of the configured meal types" (used by discovery-style filtering).
func eligible(c Candidate, p Prefs, mealType string) (bool, string) {
	if mealType == "" {
		if !intersects(c.MealTypes, p.MealTypes) {
			return false, "meal_type"
		}
	} else if !contains(c.MealTypes, mealType) {
		return false, "meal_type"
	}
	if !dietOK(c.DietFlags, p.DietType) {
		return false, "diet"
	}
	for _, a := range c.AllergenTags {
		if p.Allergens[a] {
			return false, "allergen"
		}
	}
	for _, id := range c.IngredientIDs {
		if p.ExcludedIngredientIDs[id] {
			return false, "excluded_ingredient"
		}
	}
	if c.Disliked {
		return false, "disliked"
	}
	if p.MaxCookMinutes != nil && c.PrepMinutes+c.CookMinutes > *p.MaxCookMinutes {
		return false, "too_slow"
	}
	return true, ""
}

func dietOK(flags []string, diet string) bool {
	switch diet {
	case "vegetarian":
		return contains(flags, "vegetarian") || contains(flags, "vegan")
	case "vegan":
		return contains(flags, "vegan")
	case "pescatarian":
		return contains(flags, "vegetarian") || contains(flags, "vegan") || contains(flags, "pescatarian")
	default:
		return true
	}
}

// score implements the SPEC §6 scoring table.
func score(c Candidate, in Input, st *state, day int, mt string, servings float64) float64 {
	s := 0.0
	if c.Liked {
		s += 100
	}
	if c.Favorited {
		s += 60
	}
	shared := 0
	for _, id := range c.IngredientIDs {
		if st.ingredients[id] > 0 {
			shared++
		}
	}
	s += math.Min(25, 8*float64(shared))
	if st.categories[c.Category] == 0 {
		s += 10
	} else {
		s += math.Max(-45, -15*float64(st.categories[c.Category]))
	}
	if c.PrepMinutes+c.CookMinutes <= 30 {
		s += 10
	}
	if in.Prefs.WeeklyBudget != nil && c.CostPerServing != nil {
		cost := *c.CostPerServing * servings
		remaining := *in.Prefs.WeeklyBudget - st.spent
		switch {
		case cost <= remaining:
			s += 15
		case remaining <= 0:
			s -= 30
		default:
			over := (cost - remaining) / remaining
			s -= 30 * math.Min(1, over)
		}
	}
	if in.PrevWeek[c.RecipeID] {
		s -= 40
	}
	s += float64(jitter(in.Seed, day, mt, c.RecipeID))
	return s
}

// jitter derives a deterministic 0..5 bonus per (seed, slot, recipe).
func jitter(seed int64, day int, mt, recipeID string) int64 {
	h := fnv.New64a()
	fmt.Fprintf(h, "%d|%d|%s|%s", seed, day, mt, recipeID)
	return int64(h.Sum64() % 6)
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func intersects(xs, ys []string) bool {
	for _, x := range xs {
		if contains(ys, x) {
			return true
		}
	}
	return false
}
