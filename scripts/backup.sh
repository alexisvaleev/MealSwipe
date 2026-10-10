#!/usr/bin/env bash
# MealSwipe PostgreSQL backup. Creates a compressed pg_dump in ./backups/.
# Usage: bash scripts/backup.sh  (optionally: RETENTION_DAYS=14)
set -euo pipefail

cd "$(dirname "$0")/.."
BACKUP_DIR="${BACKUP_DIR:-backups}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"
COMPOSE="${COMPOSE:-docker compose}"
TS="$(date +%Y%m%d-%H%M%S)"
OUT="${BACKUP_DIR}/mealswipe-${TS}.sql.gz"

mkdir -p "$BACKUP_DIR"

if ! $COMPOSE ps --status running --services 2>/dev/null | grep -qx db; then
  echo "error: 'db' service is not running (start the stack first)" >&2
  exit 1
fi

echo ">> dumping database to ${OUT}"
$COMPOSE exec -T db sh -c 'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" --no-owner --clean --if-exists' \
  | gzip -9 > "$OUT"

# verify the dump is a non-empty valid gzip
if [[ ! -s "$OUT" ]] || ! gzip -t "$OUT" 2>/dev/null; then
  echo "error: backup file is empty or corrupt: $OUT" >&2
  exit 1
fi

# prune old backups
find "$BACKUP_DIR" -name 'mealswipe-*.sql.gz' -mtime "+${RETENTION_DAYS}" -delete 2>/dev/null || true

echo ">> backup OK: $OUT ($(du -h "$OUT" | cut -f1))"
