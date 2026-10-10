#!/usr/bin/env bash
# MealSwipe PostgreSQL restore.
# Usage: bash scripts/restore.sh backups/mealswipe-YYYYmmdd-HHMMSS.sql.gz
# WARNING: the dump was created with --clean --if-exists and will DROP existing
# tables before restoring. All current data in the target DB will be replaced.
set -euo pipefail

cd "$(dirname "$0")/.."
FILE="${1:?usage: restore.sh <backup.sql.gz>}"
COMPOSE="${COMPOSE:-docker compose}"

[[ -f "$FILE" ]] || { echo "error: file not found: $FILE" >&2; exit 1; }
gzip -t "$FILE" || { echo "error: not a valid gzip: $FILE" >&2; exit 1; }

if ! $COMPOSE ps --status running --services 2>/dev/null | grep -qx db; then
  echo "error: 'db' service is not running" >&2
  exit 1
fi

echo "WARNING: this will REPLACE all data in the running database."
read -r -p "Type 'restore' to continue: " confirm
[[ "$confirm" == "restore" ]] || { echo "aborted"; exit 1; }

echo ">> restoring $FILE"
gunzip -c "$FILE" | $COMPOSE exec -T db sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1'

echo ">> restore complete. Restart backend to refresh sessions:"
echo "   docker compose restart backend"
