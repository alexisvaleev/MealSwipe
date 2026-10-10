# Backup & Restore

PostgreSQL is the only stateful component — back it up and you back up
everything (users, preferences, menus, shopping lists; the recipe catalog
itself is re-seedable).

## Backup

```bash
make backup          # or: bash scripts/backup.sh
```

Creates `backups/mealswipe-<UTC timestamp>.sql.gz` — a `pg_dump --clean
--if-exists --no-owner` piped through gzip, verified with `gzip -t`.

Options (env): `BACKUP_DIR` (default `backups/`), `RETENTION_DAYS`
(default `30`, older dumps are pruned automatically), `COMPOSE`.

Schedule example (crontab):

```cron
0 4 * * *  cd /opt/MealSwipe && RETENTION_DAYS=14 bash scripts/backup.sh >> backups/cron.log 2>&1
```

The `backups/` directory is gitignored — store copies off-box for real safety.

## Restore

```bash
bash scripts/restore.sh backups/mealswipe-20260101-040000.sql.gz
```

- Requires the `db` service running (`docker compose up -d db`).
- Asks for a typed confirmation — the dump is `--clean`, so **existing tables
  are dropped and replaced**.
- Afterwards: `docker compose restart backend` to drop stale sessions state.

## Verify a backup works

On a throwaway stack (recommended periodically):

```bash
# spin up a second compose project against a scratch volume
docker compose -p mswipe-verify up -d db
# restore into it using compose override or by piping into psql inside the container
```

A backup that was never restored is a rumor — test yours after the first
scheduled run.
