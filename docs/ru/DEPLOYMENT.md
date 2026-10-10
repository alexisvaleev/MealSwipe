# Развёртывание

*[English version →](../en/DEPLOYMENT.md)*

Приложение рассчитано на один VPS/домашний сервер с Docker + Compose v2.
Наружу смотрит только порт frontend; PostgreSQL остаётся во внутренней сети.

## 1. Требования

- Linux-хост с Docker Engine и compose-плагином.
- Свободный TCP-порт для приложения (по умолчанию `8080`, задаётся `APP_PORT`).
- Для HTTPS — домен и reverse proxy, терминирующий TLS (Caddy или
  существующий nginx; примеры в `deployments/`).

## 2. Установка

```bash
git clone https://github.com/alexisvaleev/MealSwipe.git && cd MealSwipe
cp .env.example .env
# обязательно: сильный POSTGRES_PASSWORD (например `openssl rand -hex 24`)

docker compose up -d --build
docker compose ps                        # все сервисы healthy
curl -fsS localhost:8080/api/v1/health   # {"status":"ok",...}
```

Откройте `http://<сервер>:8080` и зарегистрируйте первого пользователя.
Миграции и стартовый набор рецептов применяются при запуске backend
(`SEED_ON_START`).

## 3. Обновление

```bash
git pull
docker compose up -d --build
```

Откат: `git checkout <предыдущий коммит>` + сборка. Перед обновлениями —
`make backup` (см. [BACKUP_AND_RESTORE.md](../en/BACKUP_AND_RESTORE.md)).

## 4. HTTPS и reverse proxy

Контейнер слушает `APP_PORT` (8080). TLS терминируется на внешнем прокси,
запросы проксируются на приложение. После включения HTTPS установите
`COOKIE_SECURE=true` в `.env`.

- **Caddy** (автоматический HTTPS): `deployments/caddy/Caddyfile.example`.
- **Существующий nginx**: `deployments/nginx/mealswipe.conf.example`.

Не занимайте порты 80/443 самим приложением, если они уже используются
другими сервисами сервера.

## 5. Чеклист production

- [ ] `POSTGRES_PASSWORD` — стойкий и уникальный
- [ ] `COOKIE_SECURE=true` при HTTPS
- [ ] `APP_PORT` свободен
- [ ] `SEED_ON_START=false` после первого запуска (опционально)
- [ ] cron на `scripts/backup.sh`

## 6. Эксплуатация

- Логи: `docker compose logs -f backend` (JSON).
- Health: `GET /api/v1/health`; healthcheck'и на всех сервисах.
- Данные в volume `pgdata` переживают `docker compose down`; `down -v`
  **удаляет** их.
- Догрузить рецепты: `docker compose exec backend /app/server import-mealdb -n 30`.
