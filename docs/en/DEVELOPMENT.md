# Local development

## Prerequisites

| Tool | Version |
|---|---|
| Go | ≥ 1.27 |
| Node.js | ≥ 22 (LTS recommended) |
| Docker + Compose v2 | any recent |
| make | optional convenience |

## Quick start (hybrid: DB in Docker, apps native)

```bash
git clone https://github.com/alexisvaleev/MealSwipe.git && cd MealSwipe
cp .env.example .env

docker compose -f compose.dev.yaml up -d        # postgres on 127.0.0.1:5432

export DATABASE_URL="postgres://mealswipe:mealswipe_dev_password@127.0.0.1:5432/mealswipe?sslmode=disable"
cd backend && go run ./cmd/server serve         # migrates + seeds on start → :8080

cd frontend && npm install && npm run dev       # http://localhost:5173
```

Vite proxies `/api` → `http://localhost:8080`, so the SPA and API share origin
and session cookies just work. Register a user, skip or complete onboarding,
start swiping.

## Useful commands

| Task | Command |
|---|---|
| Start DB only | `make dev-db` |
| Backend | `make backend` |
| Frontend dev | `make frontend` |
| Re-run migrations | `make migrate` |
| Re-seed recipes | `make seed` |
| Import extra recipes (TheMealDB) | `make import-mealdb N=30` |
| Go tests | `make test-backend` |
| Integration tests | `TEST_DATABASE_URL=postgres://mealswipe:mealswipe_dev_password@127.0.0.1:5432/mealswipe?sslmode=disable go test ./...` (from `backend/`) |
| Frontend tests / typecheck / lint | `npm test` / `npm run typecheck` / `npm run lint` |
| Full stack in Docker | `make up` → http://localhost:8080 |
| E2E (Playwright) | `make e2e` (stack must be running) |

## Backend notes

- Subcommands: `serve` (default), `migrate`, `seed`, `import-mealdb -n N`.
- Migrations: add `backend/migrations/NNNN_name.up.sql` + `.down.sql`; never
  edit applied ones. They run automatically on `serve` start.
- Config via env only — see `.env.example` / README table.

## Frontend notes

- `VITE_APP_NAME` controls branding at build time.
- All UI strings live in `src/i18n/ru.ts` — add keys there, don't hardcode.
- API access goes through `src/lib/api.ts` + React Query hooks.

## Testing checklist before a PR

```bash
cd backend  && gofmt -l . && go vet ./... && go test ./...
cd frontend && npm run typecheck && npm run lint && npm test && npm run build
docker compose up -d --build && curl -fsS localhost:8080/api/v1/health
```
