# Database schema

PostgreSQL 17. Authoritative definition:
[`backend/migrations/0001_init.up.sql`](../../backend/migrations/0001_init.up.sql)
(migrations are embedded in the backend binary and applied at startup).

## Entity-relationship overview

```
users ──┬── sessions
        ├── user_preferences ──┐
        ├── user_excluded_ingredients ──► ingredients ◄── recipe_ingredients ──► recipes
        ├── swipes ──────────────────────────────────────────────────────────► │
        ├── favorites ───────────────────────────────────────────────────────► │
        ├── menus ── menu_items ─────────────────────────────────────────────► │
        │     └── shopping_lists ── shopping_list_items ──► ingredients        │
        └── pantry_items ──► ingredients                                       │
```

## Tables

| Table | Purpose | Key constraints |
|---|---|---|
| `users` | accounts | `email citext UNIQUE` |
| `sessions` | server sessions | PK = SHA-256 hash of the opaque cookie token; `expires_at` |
| `user_preferences` | onboarding prefs per user | PK = user_id; `diet_type`, `allergens[]`, `meal_types[]` |
| `user_excluded_ingredients` | "never show me this" | PK (user_id, ingredient_id) |
| `pantry_items` | "already have it" — skipped in shopping lists | PK (user_id, ingredient_id) |
| `ingredients` | canonical ingredient dictionary | `name UNIQUE`; `category`, `unit_family`, `allergen_tags[]` |
| `recipes` | recipe catalog | `slug UNIQUE`; `meal_types[]`, `diet_flags[]`, `instructions text[]`; nullable cost/kcal + `nutrition_estimate` |
| `recipe_ingredients` | recipe composition | PK (recipe_id, ingredient_id); `quantity`, `unit`, `note` |
| `swipes` | like/dislike signals | PK (user_id, recipe_id); `direction ∈ {like,dislike}`; re-swiping overwrites |
| `favorites` | bookmarks (separate from likes) | PK (user_id, recipe_id) |
| `menus` | weekly menu header | UNIQUE (user_id, week_start); `status` |
| `menu_items` | one recipe per (day, meal_type) slot | UNIQUE (menu_id, day_index, meal_type); `servings`, `locked` |
| `shopping_lists` | list header, bound to a menu | UNIQUE (user_id, menu_id) |
| `shopping_list_items` | generated + manual lines | nullable `ingredient_id` for manual rows; `checked`, `is_manual` |

## Controlled vocabularies

- `ingredients.category`: produce, dairy, meat, fish, bakery, grains, frozen,
  canned, spices, oils, other
- `ingredients.unit_family`: mass, volume, count
- `ingredients.allergen_tags`: milk, eggs, gluten, nuts, peanuts, fish,
  shellfish, soy, sesame
- `recipes.meal_types`: breakfast, lunch, dinner, snack
- `recipes.diet_flags`: vegetarian, vegan, pescatarian, gluten_free, dairy_free
- `user_preferences.diet_type`: any, vegetarian, vegan, pescatarian
- `swipes.direction`: like, dislike

## Design notes

- **Sessions store hashes, not tokens** — a DB leak does not leak live sessions.
- **No JSONB for core entities** — normalized relations; the only array columns
  are bounded vocab lists (tags/flags/allergens) and ordered instruction steps.
- **One recipe per meal slot** keeps the UI grid and shopping math simple;
  `locked` lets regeneration preserve hand-picked items.
- **One shopping list per menu** (`UNIQUE(user_id, menu_id)`); regeneration is
  idempotent and preserves manual items and check marks.
- Indexes: `recipes_meal_types_idx` (GIN), FK/lookup indexes on swipes,
  menu_items, shopping_list_items, sessions(expires_at).
