// Package shopping turns menu items into a consolidated shopping list.
// Pure package: unit normalisation, servings scaling, merging, grouping (SPEC §7).
package shopping

import (
	"math"
	"sort"
	"strings"
)

// Line is one recipe_ingredient usage as stored for a recipe.
type Line struct {
	IngredientID int64
	Name         string
	Category     string
	UnitFamily   string
	Quantity     float64 // per RecipeServings
	Unit         string
}

// Part is a menu item: a recipe plus the servings it is cooked for.
type Part struct {
	RecipeServings float64 // recipe.servings
	ItemServings   float64 // menu_items.servings
	Lines          []Line
}

// Item is one shopping-list line.
type Item struct {
	IngredientID *int64
	Name         string
	Quantity     *float64
	Unit         string
	Category     string
	Position     int
}

// Canonical unit tables. Mass → g, volume → ml, count units keep their own
// unit and merge only with identical unit (SPEC §7).
var massUnits = map[string]float64{
	"g": 1, "gram": 1, "grams": 1, "gr": 1,
	"kg": 1000, "kilogram": 1000, "kilograms": 1000,
}

var volumeUnits = map[string]float64{
	"ml": 1, "milliliter": 1, "milliliters": 1, "millilitre": 1, "millilitres": 1,
	"l": 1000, "liter": 1000, "liters": 1000, "litre": 1000, "litres": 1000,
	"tbsp": 15, "tablespoon": 15, "tablespoons": 15,
	"tsp": 5, "teaspoon": 5, "teaspoons": 5,
	"cup": 240, "cups": 240,
}

var countAlias = map[string]string{
	"": "pcs", "pc": "pcs", "pcs": "pcs", "piece": "pcs", "pieces": "pcs",
	"шт": "pcs", "шт.": "pcs", "x": "pcs",
	"slice": "slice", "slices": "slice",
	"clove": "clove", "cloves": "clove",
	"can": "can", "cans": "can", "tin": "can", "tins": "can",
	"bunch": "bunch", "bunches": "bunch",
	"sprig": "sprig", "sprigs": "sprig",
	"stalk": "stalk", "stalks": "stalk",
	"head": "head", "heads": "head",
	"pack": "pack", "packs": "pack", "package": "pack", "pkg": "pack",
	"bag": "bag", "bags": "bag",
	"bottle": "bottle", "bottles": "bottle",
	"jar": "jar", "jars": "jar",
}

// Units that are never merged: each usage becomes its own line.
var freeUnits = map[string]bool{
	"pinch": true, "pinches": true, "to_taste": true, "taste": true,
}

// Category display order for grouping.
var categoryOrder = []string{
	"produce", "dairy", "meat", "fish", "bakery", "grains",
	"frozen", "canned", "spices", "oils", "other",
}

func catRank(c string) int {
	for i, v := range categoryOrder {
		if v == c {
			return i
		}
	}
	return len(categoryOrder)
}

func canonUnit(u string) string {
	u = strings.ToLower(strings.TrimSpace(u))
	u = strings.TrimSuffix(u, ".")
	if _, ok := massUnits[u]; ok {
		return u
	}
	if _, ok := volumeUnits[u]; ok {
		return u
	}
	if v, ok := countAlias[u]; ok {
		return v
	}
	return u
}

// Generate merges all parts into list items, skipping pantry ingredients.
func Generate(parts []Part, pantry map[int64]bool) []Item {
	type key struct {
		id   int64
		unit string
	}
	merged := map[key]*Item{}
	var items []*Item

	addLoose := func(ln Line, qty float64, unit string) {
		id := ln.IngredientID
		it := &Item{IngredientID: &id, Name: ln.Name, Unit: unit, Category: catOr(ln.Category)}
		if qty > 0 {
			q := round1(qty)
			it.Quantity = &q
		}
		items = append(items, it)
	}

	for _, p := range parts {
		ratio := 1.0
		if p.RecipeServings > 0 {
			ratio = p.ItemServings / p.RecipeServings
		}
		for _, ln := range p.Lines {
			if pantry[ln.IngredientID] {
				continue
			}
			qty := ln.Quantity * ratio
			u := canonUnit(ln.Unit)
			if ln.Quantity <= 0 || freeUnits[u] {
				addLoose(ln, qty, u)
				continue
			}
			unit := u
			if f, ok := massUnits[u]; ok {
				qty *= f
				unit = "g"
			} else if f, ok := volumeUnits[u]; ok {
				qty *= f
				unit = "ml"
			}
			k := key{ln.IngredientID, unit}
			if it, ok := merged[k]; ok {
				*it.Quantity += qty
				continue
			}
			id := ln.IngredientID
			it := &Item{IngredientID: &id, Name: ln.Name, Unit: unit, Category: catOr(ln.Category), Quantity: &qty}
			merged[k] = it
			items = append(items, it)
		}
	}

	for _, it := range items {
		display(it)
	}
	sort.SliceStable(items, func(i, j int) bool {
		ci, cj := catRank(items[i].Category), catRank(items[j].Category)
		if ci != cj {
			return ci < cj
		}
		return items[i].Name < items[j].Name
	})
	out := make([]Item, len(items))
	for i, it := range items {
		it.Position = i
		out[i] = *it
	}
	return out
}

// display converts large base quantities to kg/l (1 decimal) for readability.
func display(it *Item) {
	if it.Quantity == nil {
		return
	}
	q := *it.Quantity
	switch it.Unit {
	case "g":
		if q >= 1000 {
			q = round1(q / 1000)
			it.Unit = "kg"
		}
	case "ml":
		if q >= 1000 {
			q = round1(q / 1000)
			it.Unit = "l"
		}
	}
	q = round1(q)
	it.Quantity = &q
}

func catOr(c string) string {
	if c == "" {
		return "other"
	}
	return c
}

func round1(f float64) float64 { return math.Round(f*10) / 10 }
