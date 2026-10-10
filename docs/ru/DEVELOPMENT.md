# Локальная разработка

*[English version →](../en/DEVELOPMENT.md)*

## Требования

Go ≥ 1.27, Node.js ≥ 22, Docker + Compose v2, make (опционально).

## Быстрый старт

```bash
git clone https://github.com/alexisvaleev/MealSwipe.git && cd MealSwipe
cp .env.example .env

docker compose -f compose.dev.yaml up -d   # PostgreSQL на 127.0.0.1:5432

export DATABASE_URL="postgres://mealswipe:mealswipe_dev_password@127.0.0.1:5432/mealswipe?sslmode=disable"
cd backend && go run ./cmd/server serve    # миграции + сид → :8080

cd frontend && npm install && npm run dev  # http://localhost:5173
```

Vite проксирует `/api` → `http://localhost:8080` — куки сессии работают без
настройки CORS.

## Полезные команды

| Задача | Команда |
|---|---|
| Только БД | `make dev-db` |
| Backend | `make backend` |
| Frontend dev | `make frontend` |
| Миграции | `make migrate` |
| Сид рецептов | `make seed` |
| Импорт из TheMealDB | `make import-mealdb N=30` |
| Тесты Go | `make test-backend` |
| Интеграционные тесты | `TEST_DATABASE_URL=postgres://mealswipe:mealswipe_dev_password@127.0.0.1:5432/mealswipe?sslmode=disable go test ./...` (в `backend/`) |
| Тесты/типы/линт frontend | `npm test` / `npm run typecheck` / `npm run lint` |
| Весь стек в Docker | `make up` → http://localhost:8080 |
| E2E (Playwright) | `make e2e` (нужен запущенный стек) |

## Заметки

- Подкоманды backend: `serve` (по умолчанию), `migrate`, `seed`,
  `import-mealdb -n N`.
- Новая миграция = новый файл `backend/migrations/NNNN_name.up.sql` +
  `.down.sql`. Уже применённые миграции не редактируются.
- Строки интерфейса — в `src/i18n/ru.ts`; вёрстка хардкодом строк не допускается.

## Перед pull request

```bash
cd backend  && gofmt -l . && go vet ./... && go test ./...
cd frontend && npm run typecheck && npm run lint && npm test && npm run build
docker compose up -d --build && curl -fsS localhost:8080/api/v1/health
```
