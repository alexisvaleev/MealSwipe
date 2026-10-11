# Architecture

## Overview

MealSwipe is a classic three-tier self-hosted web app, deliberately monolithic:

```
┌──────────┐   HTTP    ┌────────────┐   SQL    ┌────────────┐
│ Browser  │ ────────► │ Go backend │ ───────► │ PostgreSQL │
│ (React   │           │ (REST API, │          │            │
│  SPA)    │ ◄──────── │  sessions) │ ◄─────── │            │
└──────────┘   JSON    └────────────┘          └────────────┘
                              ▲
                     nginx (frontend container) serves the SPA
                     and proxies /api → backend inside compose
```

- **Frontend**: React 19 + TypeScript SPA (Vite). In production it is built to
  static files and served by an unprivileged nginx which also proxies `/api/*`
  to the backend — the app is always same-origin, so CORS is not required.
- **Backend**: single Go binary — REST API, auth/session handling, the menu
  planner and the shopping-list aggregator. SQL migrations and the seed recipe
  dataset are embedded into the binary (`go:embed`), so the container needs no
  mounted files to bootstrap.
- **PostgreSQL**: the only stateful component; a named Docker volume holds data.

## Backend layout (`backend/`)

| Package | Responsibility |
|---|---|
| `cmd/server` | entrypoint; subcommands `serve` (default), `migrate`, `seed`, `import-mealdb` |
| `internal/config` | env parsing & validation |
| `internal/db` | pgxpool + retry, embedded migration runner |
| `internal/httpapi` | router, handlers, middleware (auth, CSRF, rate-limit, headers, body cap) |
| `internal/store` | SQL access layer per entity (pgx, parameterized) |
| `internal/planner` | **pure** weekly-menu generator (hard filters + scoring) |
| `internal/shopping` | **pure** unit normalisation + ingredient aggregation |
| `internal/seed` | loads embedded `data/recipes.seed.json` idempotently |
| `migrations` | `NNNN_name.up.sql`/`.down.sql` + embed |
| `data` | embedded seed JSON |

`planner` and `shopping` contain no I/O — they are the business core and are
heavily unit-tested. HTTP handlers are thin: auth → validate → call store/pure
logic → encode JSON.

## Request flow (example: generate menu)

1. nginx proxy → `POST /api/v1/menus/generate`
2. middleware: body cap → security headers → rate limit → Origin check →
   session lookup (cookie → sha256 → sessions row) → CSRF token check
3. handler loads prefs, locked items, swipe/favorite state, candidate recipes
   with ingredients
4. `planner.Generate` picks per-slot winners honoring hard constraints
5. `store` replaces non-locked `menu_items` in one transaction
6. JSON `{menu, unfilled[]}` returned

## Data integrity

- Personal queries filter by session `user_id`; shared ones (favorites,
  menus, lists, pantry) by the user's `household_id` — IDOR-safe either way.
- Uniqueness: `menus(household_id, week_start)`, `menu_items(menu_id, day_index,
  meal_type)`, `swipes(user_id, recipe_id)`, `favorites(household_id, recipe_id)`,
  `shopping_lists(household_id, menu_id)`.
- Households: `users.household_id NOT NULL` — every user is in exactly one;
  a personal kitchen is created at registration and `join`/`leave` swap it in
  a transaction. An emptied household self-deletes, cascading its shared rows.
- Cascades: deleting a user removes sessions/prefs/swipes rows; deleting a
  household removes favorites/menus/lists/pantry; deleting a menu removes
  items and its list items via `ON DELETE CASCADE` (list itself gets
  `menu_id = NULL`).
- Menu regeneration and shopping-list rebuilds run in DB transactions.

## Scaling notes / non-goals

Single-process backend is intentional (self-hosted scale). `pgxpool` sizing is
default; no caching layer needed at this data size (hundreds of recipes).
See SPEC.md §13 for feature non-goals.
