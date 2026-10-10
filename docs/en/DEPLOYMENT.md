# Deployment

MealSwipe is designed to run on a single VPS (or a home server) with Docker +
Compose v2. Only the frontend port is exposed; PostgreSQL never leaves the
internal network.

## 1. Prerequisites

- Linux host with Docker Engine and Compose plugin.
- A TCP port for the app (default `8080`; set `APP_PORT` in `.env`).
- For HTTPS: a domain pointing at the host and a reverse proxy that terminates
  TLS (Caddy or an existing nginx) — examples in `deployments/`.

## 2. Install

```bash
git clone https://github.com/alexisvaleev/MealSwipe.git && cd MealSwipe
cp .env.example .env
# REQUIRED: set a strong POSTGRES_PASSWORD (e.g. `openssl rand -hex 24`)

docker compose up -d --build
docker compose ps                      # all services healthy
curl -fsS localhost:8080/api/v1/health # {"status":"ok",...}
```

Open `http://<host>:8080` — register the first user. Migrations and the seed
recipe dataset are applied automatically on backend start (`SEED_ON_START`).

## 3. Updating

```bash
git pull
docker compose up -d --build   # rebuilds changed images; migrations auto-apply
```

Rollback: `git checkout <previous tag/commit>` + rebuild. DB backups:
`make backup` before upgrades (see [BACKUP_AND_RESTORE.md](BACKUP_AND_RESTORE.md)).

## 4. HTTPS / reverse proxy

The app container listens on `APP_PORT` (default 8080). Terminate TLS on an
outer proxy and forward to it, then set `COOKIE_SECURE=true` in `.env` so
session cookies are `Secure`.

- **Caddy** (simplest, auto-HTTPS): `deployments/caddy/Caddyfile.example` —
  set your domain and `caddy run`.
- **Existing nginx**: `deployments/nginx/mealswipe.conf.example`.

Do not bind port 80/443 directly to the app container unless you know they
are free — on shared hosts they are usually already taken.

## 5. Environment reference

See `.env.example` and the README configuration table. Production checklist:

- [ ] `POSTGRES_PASSWORD` — strong, unique
- [ ] `COOKIE_SECURE=true` (when HTTPS is on)
- [ ] `APP_PORT` — free on the host
- [ ] `SEED_ON_START=false` optional after first boot
- [ ] backups scheduled (`scripts/backup.sh` via cron)

## 6. Operational notes

- Logs: `docker compose logs -f backend` (JSON, `slog`).
- Health: `GET /api/v1/health`; compose healthchecks on all services.
- Data survives `docker compose down` in the `pgdata` volume;
  `down -v` **destroys** it.
- Extra recipes: `docker compose exec backend /app/server import-mealdb -n 30`.
