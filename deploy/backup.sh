#!/usr/bin/env bash
# Nightly Postgres backup. Installed on the server via cron (docs/DEPLOYMENT.md):
#   0 3 * * * /opt/mountain-breath/deploy/backup.sh
set -euo pipefail

BACKUP_DIR=/opt/backups
COMPOSE="docker compose -f /opt/mountain-breath/deploy/docker-compose.prod.yml"

mkdir -p "$BACKUP_DIR"
STAMP=$(date +%F_%H%M)

# pg_dump produces a plain SQL script that recreates the whole database.
$COMPOSE exec -T postgres pg_dump -U "${POSTGRES_USER:-mb}" "${POSTGRES_DB:-mountain_breath}" \
    | gzip > "$BACKUP_DIR/mb_$STAMP.sql.gz"

# Keep the newest 14 backups, delete the rest.
ls -1t "$BACKUP_DIR"/mb_*.sql.gz | tail -n +15 | xargs -r rm --

echo "backup written: $BACKUP_DIR/mb_$STAMP.sql.gz"
