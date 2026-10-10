# MealSwipe — Technical Specification

> Working contract for the codebase. Written before implementation; kept in sync
> with reality. Russian-language UI, English-language code/docs identifiers.

## 1. Product

MealSwipe is a self-hosted web app that helps a user decide what to cook for the
week. Swipe-based recipe discovery (Tinder-style), weekly menu generation, and a
consolidated shopping list. Mobile-first, Russian UI in v1.

App name is configurable: `APP_NAME` env (backend) / `VITE_APP_NAME` (frontend
build). Default: `MealSwipe`.

## 2. Stack

| Layer | Choice | Why |
|---|---|---|
| Backend | Go 1.27, stdlib `net/http` ServeMux (method+pattern routing) | No router dep needed since Go 1.22 |
| DB access | `github.com/jackc/pgx/v5` + pgxpool | Standard, sqlc codegen adds tooling friction |
| Migrations | Embedded SQL (`embed.FS`), tiny in-house runner | Zero external tooling; DB self-contained |
| Auth | Server sessions in DB, HttpOnly cookie, session-bound CSRF token | Self-hosted SPA; no JWT needed |
| Passwords | `golang.org/x/crypto/bcrypt` cost 12 | Standard |
| Rate limit | `golang.org/x/time/rate` per-IP | Tiny, stdlib-adjacent |
| Frontend | React 19, TypeScript, Vite, Tailwind CSS 4 (`@tailwindcss/vite`), framer-motion, react-router v7, @tanstack/react-query v5, vitest | Spec-prescribed |
| DB | PostgreSQL 17 | |
| Infra | Docker, Compose, nginx (frontend), distroless-less alpine backend | |

Recipe data: own curated seed dataset (`backend/data/recipes.seed.json`,
Russian, ~36 recipes, original text). Images: hotlinked TheMealDB CDN URLs
(`themealdb.com/images/media/meals/*.jpg`, free to use) where a matching meal
exists, otherwise `null` → UI shows a styled placeholder. Optional importer
subcommand can pull more recipes from TheMealDB public API (no key needed for
the test key `1`; paid key supported via env).

## 3. Repository layout

```
backend/
  cmd/server/main.go        # subcommands: serve (default), migrate, seed, import-mealdb
  migrations/               # NNNN_name.up.sql / .down.sql + migrations.go (embed)
  internal/
    config/                 # env parsing
    db/                     # pgxpool + migrate runner
    httpapi/                # router, handlers_*.go, middleware.go, json.go
    planner/                # weekly menu generator (pure, testable)
    shopping/               # unit normalisation + list aggregation (pure)
    store/                  # SQL per entity (pgx)
    seed/                   # embedded JSON loader
  data/recipes.seed.json
frontend/                   # Vite React TS app
deployments/docker/         # nginx conf
docs/{en,ru}
scripts/                    # backup.sh, restore.sh
tests/e2e/                  # playwright (optional, not in default CI)
```

Migrations live under `backend/` because `go:embed` cannot reach parent dirs —
embedding makes the binary self-contained.

## 4. Database schema

Authoritative SQL: `backend/migrations/0001_init.up.sql`.

Entities: `users`, `sessions`, `user_preferences`,
`user_excluded_ingredients`, `ingredients`, `recipes`, `recipe_ingredients`,
`swipes`, `favorites`, `menus`, `menu_items`, `shopping_lists`,
`shopping_list_items`, `pantry_items`.

Key rules:

- `sessions.token_hash` = SHA-256 of the opaque token (never store raw token).
- `swipes` PK `(user_id, recipe_id)`, `direction IN ('like','dislike')`,
  `updated_at` — re-swiping overwrites.
- `favorites` is separate from likes (like = algorithm signal, favorite =
  bookmark).
- `menus` UNIQUE `(user_id, week_start)`; `week_start` is a Monday `date`.
- `menu_items` UNIQUE `(menu_id, day_index, meal_type)` — one recipe per slot;
  `locked` protects an item from regeneration; `servings` numeric.
- `ingredients.allergen_tags text[]` ⊂ `{milk,eggs,gluten,nuts,peanuts,fish,
  shellfish,soy,sesame}`; `ingredients.category` ∈
  `{produce,dairy,meat,fish,bakery,grains,frozen,canned,spices,oils,other}`;
  `ingredients.unit_family` ∈ `{mass,volume,count}`.
- `recipes.meal_types` ⊂ `{breakfast,lunch,dinner,snack}`;
  `diet_flags` ⊂ `{vegetarian,vegan,pescatarian,gluten_free,dairy_free}`;
  `cost_per_serving`, `kcal_per_serving` nullable + `nutrition_estimate bool`.
- `shopping_list_items.ingredient_id` nullable (manual items), `is_manual`,
  `checked`, `category`, `position`.
- `pantry_items(user_id, ingredient_id)` — stuff the user already has; excluded
  from generated shopping list.

## 5. API contract — `/api/v1`, JSON, session cookie auth

Cookie `mealswipe_session`: HttpOnly, SameSite=Lax, Path=/, Secure from
`COOKIE_SECURE`. Auth responses return `csrf` token; every mutating request must
send `X-CSRF-Token`. If `Origin` header present it must match Host or
`ALLOWED_ORIGINS`. JSON bodies limited to 1 MiB. Errors:
`{"error":{"code":"snake_code","message":"human text"}}`.

| Method | Path | Auth | Body → Response |
|---|---|---|---|
| GET | /health | no | → `{"status":"ok","version":..}` |
| POST | /auth/register | no | `{email,password,display_name}` → 201 `{user,csrf}` |
| POST | /auth/login | no | `{email,password}` → `{user,csrf}` |
| POST | /auth/logout | yes | → 204 |
| GET | /auth/me | yes | → `{user,csrf}` |
| GET | /preferences | yes | → `{preferences}` (defaults if unset) |
| PUT | /preferences | yes | `{household_size,meals_per_day,meal_types[],weekly_budget?,max_cook_minutes?,diet_type,allergens[],excluded_ingredient_ids[],goal}` → `{preferences}` |
| GET | /ingredients?q= | yes | → `{ingredients[]}` (for exclusion picker) |
| GET | /recipes?meal_type=&q=&page=&page_size= | yes | → `{recipes[],total}` (card summaries) |
| GET | /recipes/{id} | yes | → `{recipe}` full incl. `ingredients[]`, `instructions[]` |
| GET | /discover?limit= | yes | → `{recipes[]}` swipe deck: unseen, hard-filtered, shuffled |
| PUT | /recipes/{id}/swipe | yes | `{direction:"like"|"dislike"}` → `{swipe}` |
| POST | /swipes/undo | yes | → `{undone:true}` \| 404 (pops latest swipe) |
| PUT | /recipes/{id}/favorite | yes | → 204; DELETE same → 204 |
| GET | /favorites?category=&q=&sort= | yes | → `{recipes[]}` each with `favorited_at` |
| GET | /menus/current?week=YYYY-MM-DD | yes | → `{menu}` (items embedded) or 404 |
| POST | /menus/generate | yes | `{week_start}` → `{menu}` + `{unfilled:[{day_index,meal_type,reason}]}`; replaces non-locked items of existing menu for that week |
| GET | /menus/{id} | yes | → `{menu}` |
| POST | /menus/{id}/items | yes | `{day_index,meal_type,recipe_id,servings}` → `{item}` |
| PATCH | /menus/{id}/items/{item_id} | yes | `{recipe_id?,servings?,day_index?,meal_type?,locked?}` → `{item}` |
| DELETE | /menus/{id}/items/{item_id} | yes | → 204 |
| POST | /menus/{id}/items/{item_id}/swap | yes | → `{item}` (server picks replacement honoring constraints) |
| GET | /menus/{id}/shopping-list | yes | → `{list}` (auto-creates list bound to menu, generates items) |
| POST | /shopping-lists/{id}/regenerate | yes | → `{list}` (rebuilds non-manual items from menu, keeps checked/manual) |
| POST | /shopping-lists/{id}/items | yes | `{name,quantity?,unit?,category?}` → `{item}` |
| PATCH | /shopping-lists/{id}/items/{item_id} | yes | `{checked?,quantity?,name?}` → `{item}` |
| DELETE | /shopping-lists/{id}/items/{item_id} | yes | → 204 |

Auth code: `user` = `{id,email,display_name,has_preferences}`.
Every menu item response embeds its recipe summary.
All `/api/v1/*` except `/health` and `/auth/register|login` require a valid session.
All user-scoped queries filter by `user_id` from the session — never trust ids.

## 6. Menu planner (`internal/planner`, pure functions)

`Generate(input) (items, unfilled)` — deterministic given the same `randSeed`.

Hard filters (per slot, before scoring): recipe has the slot's `meal_type`;
diet_type compatibility (`vegetarian`→flags∩{vegetarian,vegan}≠∅;
`vegan`→vegan; `pescatarian`→flags∩{vegetarian,vegan,pescatarian}≠∅;
`any`→no check); no ingredient with allergen_tags ∩ prefs.allergens ≠ ∅;
no excluded ingredient; not disliked; `prep+cook ≤ max_cook_minutes` when set;
recipe not already used in the menu; not in `excludeRecipeIDs`.

Score (higher wins):

| Signal | Points |
|---|---|
| liked | +100 |
| favorited | +60 |
| shared ingredient with already-picked recipes (reuse) | +8 each, cap +25 |
| category not yet present in menu (variety) | +10 |
| same category repeated | −15 each occurrence, cap −45 |
| prep+cook ≤ 30 min | +10 |
| cost fits remaining budget | +15; projected overrun scaled 0…−30; unknown cost = 0 |
| in previous week's menu | −40 |
| jitter from `randSeed` | ±0…5 |

Greedy fill: iterate slots (day 0..6 × configured meal_types in fixed order),
pick argmax score, remove from pool. If no candidate → record
`unfilled{day_index,meal_type,reason}` and continue — never violate hard
constraints to fill a slot. Swap = same pipeline for one slot with the current
recipe excluded. Callers pass `randSeed` (time-based in prod) so regeneration
varies; tests pass a fixed seed.

## 7. Shopping list (`internal/shopping`, pure)

For each menu item: `effective_qty = ingredient.qty × (item.servings /
recipe.servings)`. Normalise unit → base: mass→g (kg×1000), volume→ml
(l×1000, tbsp=15, tsp=5, cup=240), count family units keep their own unit and
merge only with identical unit (pcs≠slice≠clove). `pinch`,`to_taste`,qty=0 →
separate unmerged line. Merge rule: same `ingredient_id` AND same normalised
unit. Display: ≥1000 g→kg (1 decimal), ≥1000 ml→l. Subtract nothing for pantry
but skip items whose ingredient is in `pantry_items`. Group output by
ingredient.category. Regeneration: keep manual items and `checked` state keyed
by `(ingredient_id, unit, name)`.

## 8. Seed data format (`backend/data/recipes.seed.json`)

```json
{"recipes":[{
  "slug":"ovsyanka-banana","title":"Овсянка с бананом и орехами",
  "description":"...","instructions":["шаг 1","..."],
  "prep_minutes":5,"cook_minutes":10,"servings":1,
  "meal_types":["breakfast"],"category":"Каша","tags":["быстро"],
  "diet_flags":["vegetarian"],"image_url":"https://www.themealdb.com/images/media/meals/xyz.jpg"|null,
  "image_credit":"TheMealDB"|null,"cost_per_serving":80,"kcal_per_serving":380,
  "ingredients":[{"name":"овсяные хлопья","quantity":50,"unit":"g","note":""},
                 {"name":"банан","quantity":1,"unit":"pcs"}],
  "ingredient_meta":{"овсяные хлопья":{"category":"grains","unit_family":"mass","allergen_tags":["gluten"]}}
}]}
```

`ingredient_meta` is read per unique ingredient name; seed loader upserts
`ingredients` rows (name → category/unit_family/allergen_tags) and inserts
recipes idempotently by `slug`.

## 9. Frontend contract (`frontend/`)

Vite dev server proxies `/api` → `http://localhost:8080`. All UI strings via
`src/i18n/ru.ts` dictionary `t(key)`. `VITE_APP_NAME` for branding.

Routes: `/` landing, `/login` `/register`, `/onboarding` (skippable),
`/swipe` (main screen), `/favorites`, `/menu`, `/shopping`, `/settings`,
recipe detail as modal/route `/recipes/{id}`. Guarded by auth (redirect
`/login`), onboarding nudge if `!has_preferences`.

Swipe deck: framer-motion drag (x-threshold ±120px or velocity), like right /
dislike left / buttons / keyboard (←→, `u` undo, `i` info). Prefetch next batch
when <3 remain. End-of-deck state offers "reset dislikes" / "adjust filters".

Menu screen: 7-day grid × configured meal types; per-slot actions: swap,
change servings, lock, remove, open recipe. Regenerate button confirms
replacement (locked kept). Unfilled slots shown honestly.

Shopping: grouped by category, check-off persists, qty edit, manual add,
delete, "regenerate" after menu changes, pantry-ish "already have it" → adds
ingredient to pantry and removes line.

States everywhere: loading skeleton, empty state, network error, 401 → login.

## 10. Config / env

| Var | Default | Notes |
|---|---|---|
| `APP_NAME` | MealSwipe | branding |
| `DATABASE_URL` | — | `postgres://user:pass@host:5432/db?sslmode=disable` |
| `POSTGRES_USER/PASSWORD/DB` | mealswipe | compose builds DATABASE_URL |
| `HTTP_ADDR` | `:8080` | backend listen |
| `APP_PORT` | `8080` | host port → frontend:8080 in compose |
| `COOKIE_SECURE` | `false` | `true` behind HTTPS |
| `SESSION_TTL_HOURS` | `720` | 30 days |
| `SEED_ON_START` | `true` | seed if recipes table empty |
| `ALLOWED_ORIGINS` | `` | extra CORS/Origin allowlist, comma-sep |
| `LOG_LEVEL` | `info` | slog |
| `MEALDB_API_KEY` | `1` | only for `import-mealdb` |
| `VITE_APP_NAME` | MealSwipe | build-time |

## 11. Security requirements (must-implement)

bcrypt(12); sessions hashed at rest; CSRF header check on mutations; Origin
check; per-IP rate limit on `/auth/*` (20/min, burst 10) and global (600/min);
1 MiB body cap; security headers (`X-Content-Type-Options`,`X-Frame-Options:
DENY`,`Referrer-Policy`,`Content-Security-Policy` with `img-src 'self' https:
data:`); `slog` JSON logs without secrets; all queries parameterised (pgx
args); every user-scoped query filtered by session user_id; `Expires` session
cleanup; containers non-root; Postgres port never published; no secrets in
repo — `.env` gitignored, `.env.example` placeholders only.

## 12. Testing bar

- `go test ./...`: planner (allergen/dislike/no-repeat/slot-fill/budget),
  shopping (normalise+merge), pure helpers — no DB needed.
- Integration (`store`/auth/menu/shopping) guarded by `TEST_DATABASE_URL`,
  skipped otherwise; CI runs them against a postgres service.
- Frontend: `tsc --noEmit`, `eslint`, `vitest run` (api client CSRF header,
  i18n, a couple of component/logic tests), `vite build`.
- e2e (playwright) optional via `make e2e`, not default CI.

## 13. Non-goals (documented honestly)

No email verification/password reset (self-hosted MVP, roadmap item). No LLM in
menu generation. Nutrition/cost are rough estimates, flagged `≈` in UI. Recipe
catalog is a curated seed; TheMealDB import optional. No multi-user social
features.
