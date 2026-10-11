# HTTP API

Base: `/api/v1` — JSON in/out. Auth is a session cookie
(`mealswipe_session`, HttpOnly, SameSite=Lax); mutations require the
`X-CSRF-Token` header with the token returned by auth endpoints.

Errors are always `{"error":{"code":"snake_code","message":"human text"}}`
with an appropriate HTTP status.

## Endpoints

| Method & path | Auth | Description |
|---|---|---|
| `GET /health` | — | Liveness: `{status:"ok",version}` |
| `POST /auth/register` | — | `{username,password,display_name}` → `201 {user,csrf}` + cookie |
| `POST /auth/login` | — | `{username,password}` → `{user,csrf}` + cookie |
| `POST /auth/logout` | ✓ | Destroys session → `204` |
| `GET /auth/me` | ✓ | → `{user,csrf}` (also used on app load) |
| `GET /preferences` | ✓ | → `{preferences}` (server defaults if unset) |
| `PUT /preferences` | ✓ | full update of onboarding fields → `{preferences}` |
| `GET /ingredients?q=` | ✓ | ingredient dictionary search (exclusion picker) |
| `GET /recipes?meal_type=&q=&page=&page_size=` | ✓ | catalog browse → `{recipes[],total}` |
| `GET /recipes/{id}` | ✓ | full recipe: `ingredients[]` with qty/unit, `instructions[]` |
| `GET /discover?limit=` | ✓ | swipe deck: unseen, constraint-filtered, shuffled |
| `PUT /recipes/{id}/swipe` | ✓ | `{direction:"like"\|"dislike"}` → `{swipe}` |
| `DELETE /recipes/{id}/swipe` | ✓ | removes that swipe (unlike/undislike), idempotent `204` |
| `POST /swipes/undo` | ✓ | removes the most recent swipe → `{undone:true}` or `404` |
| `GET /likes?category=&q=&sort=` | ✓ | my liked recipes (personal, not shared) |
| `PUT /recipes/{id}/favorite` | ✓ | add bookmark → `204` |
| `DELETE /recipes/{id}/favorite` | ✓ | remove bookmark → `204` |
| `GET /favorites?category=&q=&sort=` | ✓ | bookmarked recipes |
| `GET /menus/current?week=YYYY-MM-DD` | ✓ | menu for that week (Monday) or `404` |
| `POST /menus/generate` | ✓ | `{week_start}` → `{menu,unfilled[]}`; preserves `locked` items |
| `GET /menus/{id}` | ✓ | menu with embedded items |
| `POST /menus/{id}/items` | ✓ | `{day_index,meal_type,recipe_id,servings}` → `{item}` |
| `PATCH /menus/{id}/items/{item_id}` | ✓ | partial update (recipe/servings/slot/locked) → `{item}` |
| `DELETE /menus/{id}/items/{item_id}` | ✓ | remove from menu → `204` |
| `POST /menus/{id}/items/{item_id}/swap` | ✓ | server-picked replacement honoring constraints → `{item}` |
| `GET /menus/{id}/shopping-list` | ✓ | get-or-create list + generated items → `{list}` |
| `POST /shopping-lists/{id}/regenerate` | ✓ | rebuild auto items (keeps manual + checked) → `{list}` |
| `POST /shopping-lists/{id}/items` | ✓ | `{name,quantity?,unit?,category?}` manual item |
| `PATCH /shopping-lists/{id}/items/{item_id}` | ✓ | `{checked?,quantity?,name?}` |
| `DELETE /shopping-lists/{id}/items/{item_id}` | ✓ | remove line → `204` |
| `PUT /pantry/{ingredient_id}` | ✓ | mark "already have it" (excluded from lists) → `204` |
| `DELETE /pantry/{ingredient_id}` | ✓ | unmark → `204` |
| `GET /household` | ✓ | → `{household:{id,name,invite_code,members[]}}` |
| `PATCH /household` | ✓ | `{name}` rename the shared kitchen → `{household}` |
| `POST /household/join` | ✓ | `{code}` move into that household → `{household}` |
| `POST /household/leave` | ✓ | fresh personal household → `{household}` |
| `POST /household/invite/rotate` | ✓ | new invite code (old invalidated) → `{household}` |

## Conventions

- `week`/`week_start` are ISO dates of the week's **Monday**.
- `user` object: `{id,username,display_name,has_preferences}`.
- `menu.items[]` embed their recipe summary; `list.items[]` carry `category`
  for grouping.
- Scopes: preferences/swipes are per-user; favorites/menus/shopping/pantry
  are per-**household** (every user sits in exactly one household — personal
  until they join another). Foreign IDs return `404`, not `403`
  (no existence leak).
- Unauthenticated calls → `401`; CSRF/origin failures → `403`;
  validation → `400`/`422`; conflicts → `409`; rate limit → `429`.

## Menu generation semantics

`POST /menus/generate` fills all `7 × prefs.meal_types` slots greedily by
score (see [ARCHITECTURE.md](ARCHITECTURE.md) and `internal/planner` source for
weights). Hard constraints are never violated to fill a slot — unfillable
slots come back in `unfilled[]` with a `reason`. For multi-member households
the constraints merge across members (allergens/exclusions ∪, strictest diet,
tightest caps, any member's dislike = veto) and each member's like adds +100.
