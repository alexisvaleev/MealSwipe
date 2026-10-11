# MealSwipe

**Swipe-based meal planner** — discover recipes Tinder-style, generate a weekly
menu, and get a consolidated shopping list. Self-hosted, Russian UI.

*[Документация на русском →](docs/ru/README.md)*

## Screenshots

| Discover (swipe deck) | Weekly menu | Shopping list |
|---|---|---|
| ![Swipe deck](docs/screenshots/swipe.png) | ![Weekly menu](docs/screenshots/menu.png) | ![Shopping list](docs/screenshots/shopping.png) |

## Features

- **Swipe-based recipe discovery** — pointer, buttons and keyboard controls,
  undo, detail cards
- **Taste profile** — household size, meals/day, budget, cook-time limit, diet
  type, allergens, excluded ingredients (skippable onboarding)
- **Hard dietary constraints** — disliked recipes, allergens and excluded
  ingredients are *filtered out*, never merely down-scored
- **Weekly menu generator** — deterministic scoring (likes, variety,
  ingredient reuse, budget, time) with honest "unfilled slot" reporting
- **Menu management** — per-slot swap, servings, lock against regeneration
- **Favorites** — bookmarked recipes with search/filter/sort
- **Shopping list** — serving-scaled aggregation, unit normalisation
  (g⇄kg, ml⇄tbsp…), category grouping, check-off, manual items,
  "already have it" pantry exclusions
- **Shared kitchen** — invite your partner by code and plan together: common
  favorites, menu, shopping list and pantry; everyone's dislikes veto,
  everyone's likes boost
- **Accounts** — username + password (no e-mail — it's self-hosted),
  session-cookie auth with CSRF protection
- **Self-hosting** — one `docker compose up`; PostgreSQL, migrations and a
  curated Russian recipe seed included; optional TheMealDB import

## Tech stack

| Part | Tech |
|---|---|
| Frontend | React 19, TypeScript, Vite, Tailwind CSS 4, framer-motion, TanStack Query |
| Backend | Go 1.27, net/http, pgx/v5, embedded migrations + seed |
| Data | PostgreSQL 17 |
| Infra | Docker, Compose, nginx (unprivileged), distroless-style alpine backend |

## Requirements

- **Docker path**: Docker + Compose v2 — nothing else.
- **Dev path**: Go ≥ 1.27, Node ≥ 22, Docker (for Postgres).
- API keys: **none required** (optional `MEALDB_API_KEY` only for the
  recipe importer).
- Production: a domain + TLS-terminating reverse proxy (examples provided).

## Quick start (Docker)

```bash
git clone https://github.com/alexisvaleev/MealSwipe.git
cd MealSwipe
cp .env.example .env          # set POSTGRES_PASSWORD
docker compose up -d --build
curl -fsS localhost:8080/api/v1/health
```

Open **http://localhost:8080**, register, swipe. Migrations and the recipe
seed run automatically on first start.

## Local development

```bash
docker compose -f compose.dev.yaml up -d   # PostgreSQL on :5432
export DATABASE_URL="postgres://mealswipe:mealswipe_dev_password@127.0.0.1:5432/mealswipe?sslmode=disable"
cd backend  && go run ./cmd/server serve   # :8080, migrates + seeds
cd frontend && npm install && npm run dev  # :5173 (Vite proxies /api)
```

Tests & checks:

```bash
cd backend  && go test ./... && go vet ./...
cd frontend && npm run typecheck && npm run lint && npm test && npm run build
```

See [docs/en/DEVELOPMENT.md](docs/en/DEVELOPMENT.md).

## Configuration

| Variable | Required | Description | Example |
|---|---|---|---|
| `POSTGRES_PASSWORD` | ✅ | DB password | `openssl rand -hex 24` output |
| `POSTGRES_USER` / `POSTGRES_DB` | | DB name/user | `mealswipe` |
| `APP_PORT` | | host port → app | `8080` |
| `APP_NAME` / `VITE_APP_NAME` | | product branding | `MealSwipe` |
| `DATABASE_URL` | | full DSN (native runs) | `postgres://…@db:5432/mealswipe` |
| `COOKIE_SECURE` | | `Secure` cookie flag (HTTPS) | `true` |
| `SESSION_TTL_HOURS` | | session lifetime | `720` |
| `SEED_ON_START` | | load bundled recipes when empty | `true` |
| `ALLOWED_ORIGINS` | | extra CORS origins | `https://example.com` |
| `MEALDB_API_KEY` | | TheMealDB importer only | `1` (free test key) |
| `LOG_LEVEL` | | slog level | `info` |

## Documentation

- [Architecture](docs/en/ARCHITECTURE.md) · [Архитектура](docs/ru/ARCHITECTURE.md)
- [API reference](docs/en/API.md)
- [Database schema](docs/en/DATABASE.md)
- [Development](docs/en/DEVELOPMENT.md) · [Разработка](docs/ru/DEVELOPMENT.md)
- [Deployment](docs/en/DEPLOYMENT.md) · [Развёртывание](docs/ru/DEPLOYMENT.md)
- [Backup & restore](docs/en/BACKUP_AND_RESTORE.md)
- [Security model](docs/en/SECURITY.md) · [SPEC.md](SPEC.md) (design contract)

## Deployment

Ships with a production compose stack (Postgres + backend + nginx frontend);
only `APP_PORT` is published. For HTTPS put it behind Caddy or nginx —
ready-made configs in `deployments/`. Full guide:
[docs/en/DEPLOYMENT.md](docs/en/DEPLOYMENT.md).

## Contributing

Issues and PRs welcome — see [CONTRIBUTING.md](CONTRIBUTING.md).
Please read the [Code of Conduct](CODE_OF_CONDUCT.md).

## Security

Report vulnerabilities privately — see [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE) — © 2026 Alexey Valeev (alexisvaleev).
Recipe photos are served from TheMealDB CDN where credited; seed recipe texts
are original. See [NOTICE](NOTICE) if present.

## Project status

**MVP** — core flows implemented and tested; see TODO/roadmap notes in
`TODO.md` and open issues for what’s next (e-mail verification, password
reset, i18n EN UI, more recipe sources). Nutrition and cost values shown in
the app are estimates, not verified data.
