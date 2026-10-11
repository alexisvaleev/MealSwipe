package planner

import (
	"testing"
)

func cand(id string, mts []string, flags []string, tags []string, ings []int64) Candidate {
	return Candidate{
		RecipeID: id, MealTypes: mts, DietFlags: flags,
		AllergenTags: tags, IngredientIDs: ings,
		PrepMinutes: 10, CookMinutes: 10, Category: "cat-" + id,
	}
}

func prefs(mts []string) Prefs {
	return Prefs{
		MealTypes:             mts,
		DietType:              "any",
		Allergens:             map[string]bool{},
		ExcludedIngredientIDs: map[int64]bool{},
	}
}

func baseInput(pool []Candidate) Input {
	return Input{
		Prefs:           prefs([]string{"lunch"}),
		DefaultServings: 2,
		Pool:            pool,
		PrevWeek:        map[string]bool{},
		Seed:            1,
	}
}

func pickIDs(items []PlacedItem) map[string]int {
	out := map[string]int{}
	for _, p := range items {
		out[p.RecipeID]++
	}
	return out
}

func TestAllergenExclusion(t *testing.T) {
	in := baseInput([]Candidate{
		cand("safe", []string{"lunch"}, nil, nil, nil),
		cand("nutty", []string{"lunch"}, nil, []string{"nuts"}, nil),
	})
	in.Prefs.Allergens["nuts"] = true
	items, unfilled := Generate(in)
	if got := pickIDs(items); got["nutty"] != 0 {
		t.Fatalf("allergen recipe placed: %v", got)
	}
	if len(items) != 7 || len(unfilled) != 0 {
		// only "safe" exists; it fills day0, then pool empty -> 6 unfilled
		if len(items) != 1 || len(unfilled) != 6 {
			t.Fatalf("items=%d unfilled=%d", len(items), len(unfilled))
		}
	}
	for _, u := range unfilled {
		if u.Reason == "" {
			t.Fatal("unfilled without reason")
		}
	}
}

func TestDislikeExclusion(t *testing.T) {
	bad := cand("hated", []string{"lunch"}, nil, nil, nil)
	bad.Disliked = true
	good := cand("ok", []string{"lunch"}, nil, nil, nil)
	in := baseInput([]Candidate{bad, good})
	items, _ := Generate(in)
	if pickIDs(items)["hated"] != 0 {
		t.Fatal("disliked recipe placed")
	}
	if pickIDs(items)["ok"] != 1 {
		t.Fatal("expected ok recipe placed once")
	}
}

func TestDietFilter(t *testing.T) {
	meat := cand("meaty", []string{"lunch"}, nil, nil, nil)
	veg := cand("veggie", []string{"lunch"}, []string{"vegetarian"}, nil, nil)
	in := baseInput([]Candidate{meat, veg})
	in.Prefs.DietType = "vegetarian"
	items, _ := Generate(in)
	if pickIDs(items)["meaty"] != 0 {
		t.Fatal("non-vegetarian recipe placed under vegetarian diet")
	}
	if pickIDs(items)["veggie"] != 1 {
		t.Fatal("vegetarian recipe not placed")
	}
}

func TestDietVegan(t *testing.T) {
	veg := cand("veg", []string{"lunch"}, []string{"vegetarian"}, nil, nil)
	vegan := cand("vegan", []string{"lunch"}, []string{"vegan"}, nil, nil)
	in := baseInput([]Candidate{veg, vegan})
	in.Prefs.DietType = "vegan"
	items, _ := Generate(in)
	if pickIDs(items)["veg"] != 0 {
		t.Fatal("vegetarian (non-vegan) recipe placed under vegan diet")
	}
}

func TestMealTypeMatching(t *testing.T) {
	breakfast := cand("b", []string{"breakfast"}, nil, nil, nil)
	in := baseInput([]Candidate{breakfast})
	in.Prefs.MealTypes = []string{"lunch"}
	items, unfilled := Generate(in)
	if len(items) != 0 || len(unfilled) != 7 {
		t.Fatalf("breakfast recipe filled lunch slots: items=%d unfilled=%d", len(items), len(unfilled))
	}
}

func TestNoRecipeRepeats(t *testing.T) {
	pool := []Candidate{
		cand("a", []string{"lunch"}, nil, nil, nil),
		cand("b", []string{"lunch"}, nil, nil, nil),
	}
	in := baseInput(pool)
	items, unfilled := Generate(in)
	if len(items) != 2 || len(unfilled) != 5 {
		t.Fatalf("items=%d unfilled=%d", len(items), len(unfilled))
	}
	ids := pickIDs(items)
	if len(ids) != 2 {
		t.Fatalf("recipe repeated: %v", ids)
	}
}

func TestLockedSlotRespected(t *testing.T) {
	locked := cand("lockedR", []string{"lunch"}, nil, nil, nil)
	in := baseInput([]Candidate{
		locked,
		cand("free", []string{"lunch"}, nil, nil, nil),
	})
	in.Placed = []PlacedItem{{
		DayIndex: 0, MealType: "lunch", Servings: 2,
		RecipeID: "lockedR", Candidate: locked,
	}}
	items, _ := Generate(in)
	for _, p := range items {
		if p.DayIndex == 0 {
			t.Fatal("generated over locked slot")
		}
		if p.RecipeID == "lockedR" {
			t.Fatal("locked recipe reused")
		}
	}
	// Only "free" remains usable; no repeats -> 1 pick, 6 unfilled.
	if len(items) != 1 || items[0].RecipeID != "free" {
		t.Fatalf("expected 1 generated pick of free, got %+v", items)
	}
}

func TestExcludedIDs(t *testing.T) {
	in := baseInput([]Candidate{
		cand("x", []string{"lunch"}, nil, nil, nil),
		cand("y", []string{"lunch"}, nil, nil, nil),
	})
	in.Exclude = map[string]bool{"x": true}
	items, _ := Generate(in)
	if pickIDs(items)["x"] != 0 {
		t.Fatal("excluded recipe placed")
	}
}

func TestExcludedIngredient(t *testing.T) {
	in := baseInput([]Candidate{
		cand("hasMilk", []string{"lunch"}, nil, nil, []int64{1, 2}),
		cand("plain", []string{"lunch"}, nil, nil, []int64{3}),
	})
	in.Prefs.ExcludedIngredientIDs[2] = true
	items, _ := Generate(in)
	if pickIDs(items)["hasMilk"] != 0 {
		t.Fatal("recipe with excluded ingredient placed")
	}
}

func TestMaxCookMinutes(t *testing.T) {
	slow := cand("slow", []string{"lunch"}, nil, nil, nil)
	slow.CookMinutes = 120
	in := baseInput([]Candidate{slow})
	m := 30
	in.Prefs.MaxCookMinutes = &m
	items, unfilled := Generate(in)
	if len(items) != 0 || len(unfilled) != 7 {
		t.Fatal("too-slow recipe placed")
	}
}

func TestUnfilledWhenPoolEmpty(t *testing.T) {
	in := baseInput(nil)
	in.Prefs.MealTypes = []string{"breakfast", "dinner"}
	items, unfilled := Generate(in)
	if len(items) != 0 {
		t.Fatal("expected no picks")
	}
	if len(unfilled) != 14 {
		t.Fatalf("expected 14 unfilled, got %d", len(unfilled))
	}
	for _, u := range unfilled {
		if u.DayIndex < 0 || u.DayIndex > 6 || u.MealType == "" || u.Reason == "" {
			t.Fatalf("bad unfilled: %+v", u)
		}
	}
}

func TestDeterminism(t *testing.T) {
	pool := []Candidate{
		cand("a", []string{"lunch"}, nil, nil, nil),
		cand("b", []string{"lunch"}, nil, nil, nil),
		cand("c", []string{"lunch"}, nil, nil, nil),
	}
	in1 := baseInput(pool)
	in2 := baseInput(pool)
	got1, _ := Generate(in1)
	got2, _ := Generate(in2)
	if len(got1) != len(got2) {
		t.Fatal("different length")
	}
	for i := range got1 {
		if got1[i].RecipeID != got2[i].RecipeID ||
			got1[i].DayIndex != got2[i].DayIndex ||
			got1[i].MealType != got2[i].MealType {
			t.Fatalf("non-deterministic at %d: %v vs %v", i, got1[i], got2[i])
		}
	}
}

func TestLikedPreferred(t *testing.T) {
	plain := cand("plain", []string{"lunch"}, nil, nil, nil)
	liked := cand("liked", []string{"lunch"}, nil, nil, nil)
	liked.LikedBy = 1
	in := baseInput([]Candidate{plain, liked})
	items, _ := Generate(in)
	if len(items) == 0 || items[0].RecipeID != "liked" {
		t.Fatalf("liked recipe not picked first: %+v", items)
	}
}

func TestBothMembersLikedBeatsSingleLike(t *testing.T) {
	solo := cand("solo", []string{"lunch"}, nil, nil, nil)
	solo.LikedBy = 1
	shared := cand("shared", []string{"lunch"}, nil, nil, nil)
	shared.LikedBy = 2
	in := baseInput([]Candidate{solo, shared})
	items, _ := Generate(in)
	if len(items) == 0 || items[0].RecipeID != "shared" {
		t.Fatalf("recipe liked by both not picked first: %+v", items)
	}
}

func TestBudgetPrefersFittingRecipe(t *testing.T) {
	budget := 100.0
	expensiveCost := 200.0
	cheapCost := 5.0
	exp := cand("exp", []string{"lunch"}, nil, nil, nil)
	exp.CostPerServing = &expensiveCost
	cheap := cand("cheap", []string{"lunch"}, nil, nil, nil)
	cheap.CostPerServing = &cheapCost
	in := baseInput([]Candidate{exp, cheap})
	in.Prefs.WeeklyBudget = &budget
	items, _ := Generate(in)
	if len(items) == 0 || items[0].RecipeID != "cheap" {
		t.Fatalf("budget-fitting recipe not first: %+v", items)
	}
}

func TestPrevWeekPenalty(t *testing.T) {
	old := cand("old", []string{"lunch"}, nil, nil, nil)
	fresh := cand("fresh", []string{"lunch"}, nil, nil, nil)
	in := baseInput([]Candidate{old, fresh})
	in.PrevWeek["old"] = true
	items, _ := Generate(in)
	if len(items) == 0 || items[0].RecipeID != "fresh" {
		t.Fatalf("prev-week recipe preferred: %+v", items)
	}
}

func TestSwapFor(t *testing.T) {
	cur := cand("cur", []string{"lunch"}, nil, nil, nil)
	other := cand("other", []string{"lunch"}, nil, nil, nil)
	repl := cand("repl", []string{"lunch"}, nil, nil, nil)
	in := baseInput([]Candidate{cur, other, repl})
	in.Placed = []PlacedItem{{
		DayIndex: 1, MealType: "lunch", Servings: 2, RecipeID: "other", Candidate: other,
	}}
	in.Exclude = map[string]bool{"cur": true}
	pick, ok := SwapFor(in, 0, "lunch", 2)
	if !ok {
		t.Fatal("no swap candidate")
	}
	if pick.RecipeID != "repl" {
		t.Fatalf("swap picked %q, want repl", pick.RecipeID)
	}
}
