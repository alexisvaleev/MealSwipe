# Архитектура

*[English version →](../en/ARCHITECTURE.md)*

## Обзор

Классическое трёхзвенное self-hosted приложение, сознательно монолитное:

```
Браузер (React SPA) ──HTTP──► Go backend ──SQL──► PostgreSQL
                             (REST, сессии)
```

- **Frontend**: React 19 + TypeScript (Vite). В production собирается в
  статику, которую раздаёт nginx; он же проксирует `/api` на backend —
  приложение всегда same-origin, CORS не нужен.
- **Backend**: один Go-бинарник — REST API, сессии, генератор меню и
  агрегатор списка покупок. Миграции и стартовый набор рецептов встроены в
  бинарник (`go:embed`) — контейнеру не нужны примонтированные файлы.
- **PostgreSQL**: единственный компонент с состоянием; данные в именованном
  Docker-томе.

## Структура backend

| Пакет | Ответственность |
|---|---|
| `cmd/server` | входная точка: `serve` (по умолчанию), `migrate`, `seed`, `import-mealdb` |
| `internal/config` | конфигурация через env |
| `internal/db` | pgxpool с ретраями + раннер миграций |
| `internal/httpapi` | роутер, хендлеры, middleware (auth, CSRF, rate-limit, заголовки, лимит тела) |
| `internal/store` | SQL-слой по сущностям (pgx, параметризованные запросы) |
| `internal/planner` | **чистая** логика генератора меню (жёсткие фильтры + скоринг) |
| `internal/shopping` | **чистая** нормализация единиц + агрегация ингредиентов |
| `internal/seed` | загрузка встроенного `data/recipes.seed.json` (идемпотентно) |
| `migrations` | `NNNN_name.up.sql`/`.down.sql` + embed |
| `data` | встроенный seed JSON |

`planner` и `shopping` не содержат I/O — это бизнес-ядро, покрытое
unit-тестами. Хендлеры тонкие: auth → валидация → store/чистая логика → JSON.

## Поток запроса (генерация меню)

1. `POST /api/v1/menus/generate` через nginx.
2. Middleware: лимит тела → security-заголовки → rate-limit → проверка Origin
   → сессия (cookie → sha256 → строка в `sessions`) → CSRF-токен.
3. Хендлер загружает предпочтения, закреплённые позиции, лайки/дизлайки и
   кандидатов с ингредиентами.
4. `planner.Generate` жадно заполняет слоты с учётом жёстких ограничений.
5. `store` заменяет незакреплённые `menu_items` в одной транзакции.
6. Ответ `{menu, unfilled[]}`.

## Целостность данных

- Все запросы по пользовательским данным фильтруются `user_id` из сессии.
- Уникальности: `menus(user_id, week_start)`, `menu_items(menu_id, day_index,
  meal_type)`, `swipes/favorites(user_id, recipe_id)`,
  `shopping_lists(user_id, menu_id)`.
- Каскадные удаления: пользователь → его сессии/настройки/свайпы/меню/списки;
  меню → позиции.
- Регенерация меню и пересборка списка покупок — в транзакциях.

## Ограничения масштабирования

Однопроцессный backend — осознанное решение для self-hosted сценария; при
каталоге в сотни рецептов кэш-слой не нужен.
