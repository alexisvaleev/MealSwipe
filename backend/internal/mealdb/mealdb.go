// Package mealdb implements the `import-mealdb` subcommand: it pulls random
// meals from TheMealDB public API and upserts them as recipes
// (source='themealdb', image_credit='TheMealDB').
package mealdb

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const apiBase = "https://www.themealdb.com/api/json/v1/"

// Import fetches n random meals and upserts them. Returns imported count.
func Import(ctx context.Context, pool *pgxpool.Pool, apiKey string, n int, log *slog.Logger) (int, error) {
	if apiKey == "" {
		apiKey = "1"
	}
	client := &http.Client{Timeout: 20 * time.Second}
	imported := 0
	for i := 0; i < n; i++ {
		id, err := randomID(ctx, client, apiKey)
		if err != nil {
			return imported, err
		}
		meal, err := lookup(ctx, client, apiKey, id)
		if err != nil {
			return imported, err
		}
		if err := upsertMeal(ctx, pool, meal); err != nil {
			return imported, err
		}
		imported++
		if log != nil {
			log.Info("mealdb imported", "id", id, "title", meal["strMeal"], "n", imported)
		}
		// Be polite to the free API.
		select {
		case <-ctx.Done():
			return imported, ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}
	return imported, nil
}

func get(ctx context.Context, client *http.Client, url string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mealdb: get %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mealdb: %s -> %d", url, resp.StatusCode)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("mealdb: decode %s: %w", url, err)
	}
	return out, nil
}

func randomID(ctx context.Context, client *http.Client, key string) (string, error) {
	out, err := get(ctx, client, apiBase+key+"/random.php")
	if err != nil {
		return "", err
	}
	return mealField(out, "idMeal")
}

func lookup(ctx context.Context, client *http.Client, key, id string) (map[string]any, error) {
	out, err := get(ctx, client, apiBase+key+"/lookup.php?i="+id)
	if err != nil {
		return nil, err
	}
	meals, _ := out["meals"].([]any)
	if len(meals) == 0 {
		return nil, fmt.Errorf("mealdb: meal %s not found", id)
	}
	m, _ := meals[0].(map[string]any)
	if m == nil {
		return nil, fmt.Errorf("mealdb: bad meal payload for %s", id)
	}
	return m, nil
}

func mealField(payload map[string]any, field string) (string, error) {
	meals, _ := payload["meals"].([]any)
	if len(meals) == 0 {
		return "", fmt.Errorf("mealdb: empty meals array")
	}
	m, _ := meals[0].(map[string]any)
	s, _ := m[field].(string)
	return s, nil
}

func str(m map[string]any, k string) string {
	s, _ := m[k].(string)
	return strings.TrimSpace(s)
}

// upsertMeal inserts/updates one recipe keyed by (source, external_id).
func upsertMeal(ctx context.Context, pool *pgxpool.Pool, m map[string]any) error {
	extID := str(m, "idMeal")
	title := str(m, "strMeal")
	if extID == "" || title == "" {
		return fmt.Errorf("mealdb: meal missing id/title")
	}
	thumb := str(m, "strMealThumb")
	var imageURL, imageCredit *string
	if thumb != "" {
		imageURL = &thumb
		credit := "TheMealDB"
		imageCredit = &credit
	}
	tags := splitCSV(str(m, "strTags"))
	mealTypes, dietFlags := classify(str(m, "strCategory"), tags)
	desc := str(m, "strArea")
	if cat := str(m, "strCategory"); cat != "" {
		if desc != "" {
			desc += " · "
		}
		desc += cat
	}
	instructions := splitSteps(str(m, "strInstructions"))
	slug := slugify(title) + "-" + extID

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var recipeID string
	err = tx.QueryRow(ctx,
		`INSERT INTO recipes
			(slug, title, description, instructions, servings, meal_types,
			 category, tags, diet_flags, image_url, image_credit, source, external_id)
		 VALUES ($1,$2,$3,$4,2,$5,$6,$7,$8,$9,$10,'themealdb',$11)
		 ON CONFLICT (source, external_id) DO UPDATE SET
			title=EXCLUDED.title, description=EXCLUDED.description,
			instructions=EXCLUDED.instructions, meal_types=EXCLUDED.meal_types,
			category=EXCLUDED.category, tags=EXCLUDED.tags, diet_flags=EXCLUDED.diet_flags,
			image_url=EXCLUDED.image_url, image_credit=EXCLUDED.image_credit
		 RETURNING id::text`,
		slug, title, desc, instructions,
		mealTypes, str(m, "strCategory"), tags, dietFlags, imageURL, imageCredit, extID,
	).Scan(&recipeID)
	if err != nil {
		return fmt.Errorf("mealdb: upsert %q: %w", title, err)
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM recipe_ingredients WHERE recipe_id = $1::uuid`, recipeID); err != nil {
		return err
	}
	for pos, ing := range mealIngredients(m) {
		ingID, err := ensureIngredient(ctx, tx, ing)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO recipe_ingredients (recipe_id, ingredient_id, quantity, unit, position)
			 VALUES ($1::uuid,$2,$3,$4,$5)`, recipeID, ingID, ing.qty, ing.unit, pos); err != nil {
			return fmt.Errorf("mealdb: link %q: %w", ing.name, err)
		}
	}
	return tx.Commit(ctx)
}

type parsedIngredient struct {
	name string
	qty  float64
	unit string
}

// mealIngredients pairs strIngredientN with strMeasureN (1..20).
func mealIngredients(m map[string]any) []parsedIngredient {
	var out []parsedIngredient
	for i := 1; i <= 20; i++ {
		name := strings.ToLower(str(m, fmt.Sprintf("strIngredient%d", i)))
		if name == "" {
			continue
		}
		qty, unit := parseMeasure(str(m, fmt.Sprintf("strMeasure%d", i)))
		out = append(out, parsedIngredient{name: name, qty: qty, unit: unit})
	}
	return out
}

var measureRe = regexp.MustCompile(`^([\d\s\.,/\-]+)?(.*)$`)

// parseMeasure turns "200g", "1/2 tsp", "to taste" into (qty, unit).
func parseMeasure(raw string) (float64, string) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return 1, "pcs"
	}
	for _, t := range []string{"to taste", "for serving", "for garnish", "garnish", "optional"} {
		if strings.Contains(raw, t) {
			return 0, "to_taste"
		}
	}
	mm := measureRe.FindStringSubmatch(raw)
	qty := parseQty(mm[1])
	unit := canonicalUnit(strings.TrimSpace(mm[2]))
	if qty <= 0 && unit == "pcs" && mm[1] == "" {
		qty = 1
	}
	return qty, unit
}

func parseQty(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "-") // "1-2" style ranges -> take the first number
	if s == "" {
		return 0
	}
	if i := strings.IndexAny(s, " -"); i >= 0 {
		s = s[:i]
	}
	if strings.Contains(s, "/") {
		parts := strings.SplitN(s, "/", 2)
		a, _ := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		b, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if b != 0 {
			return a / b
		}
		return 0
	}
	s = strings.ReplaceAll(s, ",", ".")
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// canonicalUnit maps free-form measures to the units the shopping
// normaliser understands. Unknown count units are kept as-is (they merge
// only with identical units per SPEC §7).
func canonicalUnit(u string) string {
	u = strings.ToLower(strings.TrimSpace(u))
	u = strings.TrimSuffix(u, ".")
	switch u {
	case "", "whole", "piece", "pieces":
		return "pcs"
	case "g", "gram", "grams":
		return "g"
	case "kg", "kilogram", "kilograms":
		return "kg"
	case "ml", "milliliters", "millilitres":
		return "ml"
	case "l", "liter", "litre", "liters", "litres":
		return "l"
	case "tbsp", "tablespoon", "tablespoons", "tbl", "tbs":
		return "tbsp"
	case "tsp", "teaspoon", "teaspoons":
		return "tsp"
	case "cup", "cups":
		return "cup"
	case "pinch", "pinches":
		return "pinch"
	case "clove", "cloves":
		return "clove"
	case "slice", "slices", "sliced":
		return "slice"
	case "can", "cans", "tin", "tins", "tinned", "canned":
		return "can"
	case "oz", "ounce", "ounces":
		return "oz"
	case "lb", "lbs", "pound", "pounds":
		return "lb"
	}
	return u
}

// classify derives meal_types and diet_flags from the MealDB category.
func classify(category string, tags []string) (mealTypes, dietFlags []string) {
	switch strings.ToLower(category) {
	case "breakfast":
		mealTypes = []string{"breakfast"}
	case "dessert":
		mealTypes = []string{"snack", "breakfast"}
	case "starter", "side", "snack":
		mealTypes = []string{"snack", "lunch"}
	default:
		mealTypes = []string{"lunch", "dinner"}
	}
	lower := strings.ToLower(category + " " + strings.Join(tags, " "))
	switch {
	case strings.Contains(lower, "vegan"):
		dietFlags = []string{"vegan", "vegetarian"}
	case strings.Contains(lower, "vegetarian"):
		dietFlags = []string{"vegetarian"}
	case strings.Contains(lower, "seafood") || strings.Contains(lower, "fish"):
		dietFlags = []string{"pescatarian"}
	}
	if dietFlags == nil {
		dietFlags = []string{}
	}
	return mealTypes, dietFlags
}

// --- best-guess ingredient metadata ---

var allergenGuesses = []struct {
	tag      string
	keywords []string
}{
	{"milk", []string{"milk", "cream", "cheese", "butter", "yogurt", "yoghurt", "ghee", "creme", "mozzarella", "parmesan", "cheddar", "feta", "ricotta", "mascarpone", "condensed milk", "evaporated milk", "sour cream"}},
	{"eggs", []string{"egg"}},
	{"gluten", []string{"flour", "bread", "pasta", "noodle", "spaghetti", "penne", "macaroni", "couscous", "tortilla", "baguette", "breadcrumbs", "pastry", "puff pastry", "pizza", "udon", "ramen", "vermicelli", "lasagne", "lasagna", "wonton", "semolina", "bulgur", "barley", "rye", "croissant", "bun", "roll", "pita"}},
	{"nuts", []string{"almond", "walnut", "pecan", "cashew", "hazelnut", "pine nut", "pistachio", "macadamia", "chestnut", "nut oil"}},
	{"peanuts", []string{"peanut"}},
	{"fish", []string{"salmon", "tuna", "cod", "haddock", "mackerel", "anchov", "trout", "sea bass", "tilapia", "pollock", "halibut", "sardine", "herring", "fish sauce", "bonito"}},
	{"shellfish", []string{"shrimp", "prawn", "crab", "lobster", "mussel", "scallop", "clam", "oyster", "squid", "octopus", "crayfish", "langoustine"}},
	{"soy", []string{"soy sauce", "soya", "tofu", "edamame", "miso", "tempeh"}},
	{"sesame", []string{"sesame", "tahini"}},
}

var categoryGuesses = []struct {
	cat      string
	keywords []string
}{
	{"meat", []string{"chicken", "beef", "pork", "lamb", "bacon", "sausage", "mince", "minced", "veal", "turkey", "ham", "chorizo", "salami", "prosciutto", "steak", "goat", "duck", "liver", "meatball"}},
	{"fish", []string{"salmon", "tuna", "cod", "haddock", "mackerel", "anchov", "trout", "sea bass", "tilapia", "pollock", "halibut", "sardine", "herring", "shrimp", "prawn", "crab", "lobster", "mussel", "scallop", "clam", "squid", "octopus", "fish"}},
	{"dairy", []string{"milk", "cream", "cheese", "butter", "yogurt", "yoghurt", "egg", "ghee", "creme", "mozzarella", "parmesan", "cheddar", "feta", "ricotta", "mascarpone"}},
	{"produce", []string{"onion", "garlic", "tomato", "potato", "carrot", "pepper", "chilli", "chili", "lemon", "lime", "lettuce", "spinach", "cucumber", "celery", "mushroom", "ginger", "scallion", "spring onion", "shallot", "zucchini", "courgette", "aubergine", "eggplant", "avocado", "corn", "peas", "bean sprout", "cabbage", "broccoli", "cauliflower", "kale", "leek", "radish", "beet", "turnip", "squash", "pumpkin", "apple", "banana", "orange", "mango", "pineapple", "pear", "peach", "plum", "apricot", "cherry", "berry", "berries", "strawberry", "raspberry", "blueberry", "grape", "raisin", "sultana", "date", "fig", "melon", "watermelon", "kiwi", "pomegranate", "coconut", "herb", "parsley", "cilantro", "coriander leaves", "basil", "mint", "dill", "thyme fresh", "rosemary fresh", "sage", "chive", "arugula", "rocket", "watercress", "fennel", "artichoke", "asparagus", "okra", "jalapeno", "habanero"}},
	{"bakery", []string{"bread", "bun", "roll", "tortilla", "pita", "baguette", "croissant", "pastry", "puff pastry", "pizza dough", "pie crust", "phyllo", "filo", "brioche", "naan", "crumpet", "muffin", "bagel"}},
	{"grains", []string{"rice", "pasta", "noodle", "spaghetti", "penne", "macaroni", "fusilli", "lasagne", "lasagna", "couscous", "quinoa", "oats", "flour", "cornmeal", "polenta", "bulgur", "buckwheat", "barley", "semolina", "breadcrumbs", "udon", "ramen", "vermicelli", "gnocchi", "tortellini", "risotto", "farro", "millet", "tapioca", "starch", "cornstarch", "cornflour", "cereal", "granola", "taco shells"}},
	{"frozen", []string{"frozen", "ice cream", "chips", "fries"}},
	{"canned", []string{"chopped tomatoes", "canned", "tinned", "coconut milk", "coconut cream", "kidney beans", "black beans", "chickpeas", "baked beans", "sweetcorn", "corn kernels", "tomato puree", "tomato paste", "passata", "beans", "lentils", "peas", "olives", "capers", "artichoke hearts", "bamboo shoots", "water chestnut", "jackfruit", "pineapple chunks"}},
	{"spices", []string{"salt", "pepper", "cumin", "paprika", "oregano", "dried", "ground", "powder", "cinnamon", "turmeric", "curry", "thyme", "rosemary", "bay", "saffron", "nutmeg", "clove", "allspice", "cardamom", "coriander seed", "fennel seed", "mustard seed", "cayenne", "chili powder", "chilli powder", "smoked paprika", "garam masala", "five spice", "sumac", "zaatar", "marjoram", "anise", "star anise", "vanilla", "bouillon", "stock cube", "seasoning", "herbs de provence"}},
	{"oils", []string{"oil", "olive oil", "sunflower oil", "vegetable oil", "sesame oil", "coconut oil", "rapeseed", "canola", "vinegar", "vinaigrette"}},
}

func guessAllergens(name string) []string {
	out := []string{}
	for _, g := range allergenGuesses {
		for _, kw := range g.keywords {
			if strings.Contains(name, kw) {
				out = append(out, g.tag)
				break
			}
		}
	}
	return out
}

func guessCategory(name string) string {
	for _, g := range categoryGuesses {
		for _, kw := range g.keywords {
			if strings.Contains(name, kw) {
				return g.cat
			}
		}
	}
	return "other"
}

func guessUnitFamily(unit string) string {
	switch unit {
	case "g", "kg":
		return "mass"
	case "ml", "l", "tbsp", "tsp", "cup":
		return "volume"
	default:
		return "count"
	}
}

func ensureIngredient(ctx context.Context, tx pgx.Tx, ing parsedIngredient) (int64, error) {
	name := strings.ToLower(strings.TrimSpace(ing.name))
	var id int64
	cat, fam := guessCategory(name), guessUnitFamily(ing.unit)
	err := tx.QueryRow(ctx,
		`INSERT INTO ingredients (name, category, unit_family, allergen_tags)
		 VALUES ($1,$2,$3,$4)
		 ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
		 RETURNING id`, name, cat, fam, guessAllergens(name)).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("mealdb: ingredient %q: %w", name, err)
	}
	return id, nil
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if r < 128 {
				b.WriteRune(r)
			}
		} else {
			b.WriteByte('-')
		}
	}
	slug := nonSlug.ReplaceAllString(b.String(), "-")
	return strings.Trim(slug, "-")
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if out == nil {
		out = []string{}
	}
	return out
}

func splitSteps(s string) []string {
	var out []string
	for _, line := range strings.FieldsFunc(s, func(r rune) bool { return r == '\n' || r == '\r' }) {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	if out == nil {
		out = []string{}
	}
	return out
}
