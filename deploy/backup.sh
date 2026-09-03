#!/usr/bin/env bash
# Nightly backup of everything the shop cannot recreate: the Postgres
# database AND the uploads volume (product photos and videos).
#
# Scheduled by deploy/systemd/mb-backup.timer (docs/DEPLOYMENT.md step 7).
#   by hand:    bash /opt/mountain-breath/deploy/backup.sh
#   proven by:  bash /opt/mountain-breath/deploy/restore.sh --drill latest
#
# Every setting is an environment variable with a production default, so
# the same script runs unchanged against the dev stack for a rehearsal
# (docs/DEPLOYMENT.md step 7 shows the knobs).
set -euo pipefail

MB_BACKUP_DIR="${MB_BACKUP_DIR:-/opt/backups}"
MB_COMPOSE_FILE="${MB_COMPOSE_FILE:-/opt/mountain-breath/deploy/docker-compose.prod.yml}"
# The named volume the api writes uploads to. Docker names it
# <compose project>_<volume>; the project is the `name:` line at the top
# of the compose file.
MB_UPLOADS_VOLUME="${MB_UPLOADS_VOLUME:-mountain-breath_uploads_data}"
# Local retention — by COUNT, not age (the prune step says why).
MB_BACKUP_KEEP="${MB_BACKUP_KEEP:-14}"
# Off-machine copy: an rclone remote such as r2:mountain-breath-backups
# (docs/DEPLOYMENT.md step 7b). Empty = this machine only.
MB_BACKUP_REMOTE="${MB_BACKUP_REMOTE:-}"
MB_REMOTE_KEEP_DAYS="${MB_REMOTE_KEEP_DAYS:-60}"
# Paging (decision #105): a failed run fires BackupFailed through
# Alertmanager, which reaches the phone and repeats until the next
# successful run resolves it. Empty = skip (a rehearsal with no
# Alertmanager); unreachable = one warning line, never a failed backup.
MB_ALERTMANAGER_URL="${MB_ALERTMANAGER_URL:-http://127.0.0.1:9093}"

# Git Bash on Windows rewrites arguments that look like POSIX paths
# ("/src" becomes "C:\Program Files\Git\src") before docker sees them.
# This switches that off for a rehearsal on a dev machine; Linux ignores it.
export MSYS_NO_PATHCONV=1

compose() { docker compose -f "$MB_COMPOSE_FILE" "$@"; }
log() { printf '%s backup: %s\n' "$(date '+%F %T')" "$*"; }
alert() { # fire | resolve — the same road every alert takes (alert.sh)
    [[ -n "$MB_ALERTMANAGER_URL" ]] || return 0
    MB_ALERTMANAGER_URL="$MB_ALERTMANAGER_URL" bash "$(dirname "$0")/alert.sh" "$1" BackupFailed \
        "nightly backup failed on $(hostname): journalctl -u mb-backup" > /dev/null \
        || log "WARNING: could not $1 BackupFailed at $MB_ALERTMANAGER_URL"
}
# set -e aborts silently; the ERR trap says where (journalctl shows this
# line) and pages before the exit.
trap 'log "FAILED (exit $?) at line $LINENO"; alert fire' ERR

mkdir -p "$MB_BACKUP_DIR"
rm -f "$MB_BACKUP_DIR"/*.part      # leftovers of a run that died halfway
STAMP=$(date +%F_%H%M%S)
DB_OUT="$MB_BACKUP_DIR/mb_$STAMP.dump"

# ── 1. Database ─────────────────────────────────────────────────────────
# pg_dump runs INSIDE the postgres container: same binary version as the
# server, nothing to install on the host, and the role and database name
# come from the container's own environment — the single source of truth
# rather than a copy of it in this script.
#
# --format=custom: a compressed archive that pg_restore can replay whole,
# in part, or in a different order. A plain SQL script can only be fed to
# psql from top to bottom.
#
# The dump lands in a .part file and is renamed only after pg_dump exited
# 0 (set -e + pipefail see to that): a run killed halfway never leaves a
# truncated file behind wearing a valid name. mv within one directory is
# a rename — atomic, no window where the name exists but the bytes don't.
log "dumping database -> $DB_OUT"
compose exec -T postgres sh -c 'pg_dump -U "$POSTGRES_USER" --format=custom "$POSTGRES_DB"' > "$DB_OUT.part"
mv "$DB_OUT.part" "$DB_OUT"

# Prove the archive parses. --list prints the table of contents without
# touching any database; a corrupt or empty file fails HERE, tonight,
# instead of on the day the restore is needed.
compose exec -T postgres pg_restore --list < "$DB_OUT" > /dev/null
log "database archive ok ($(du -h "$DB_OUT" | cut -f1))"

# ── 2. Uploads volume ───────────────────────────────────────────────────
# Photos and videos are files, not rows: a database-only backup restores
# a catalog pointing at images that no longer exist. But uploads change
# rarely and weigh far more than the database, so a fresh tarball every
# night would be fourteen copies of the same bytes. Instead the volume's
# file list (path, size, mtime) is fingerprinted, and a new archive is
# written only when the fingerprint differs from the last archive's.
#
# Guard first: `docker run -v name:/path` silently CREATES a volume that
# does not exist — a typo here would back up an empty directory every
# night and report success. inspect fails loudly instead.
docker volume inspect "$MB_UPLOADS_VOLUME" > /dev/null
FINGERPRINT_FILE="$MB_BACKUP_DIR/uploads.fingerprint"
FINGERPRINT=$(docker run --rm -v "$MB_UPLOADS_VOLUME:/src:ro" alpine:3.20 \
    sh -c 'cd /src && find . -type f -exec stat -c "%n %s %Y" {} + | sort | sha256sum | cut -d" " -f1')
LAST_FINGERPRINT=""; LAST_ARCHIVE=""
if [[ -f "$FINGERPRINT_FILE" ]]; then
    read -r LAST_FINGERPRINT LAST_ARCHIVE < "$FINGERPRINT_FILE" || true
fi
if [[ "$FINGERPRINT" == "$LAST_FINGERPRINT" && -f "$MB_BACKUP_DIR/$LAST_ARCHIVE" ]]; then
    log "uploads unchanged since $LAST_ARCHIVE — no new archive"
else
    UPLOADS_OUT="$MB_BACKUP_DIR/uploads_$STAMP.tar.gz"
    log "archiving uploads volume $MB_UPLOADS_VOLUME -> $UPLOADS_OUT"
    # The volume is mounted read-only into a throwaway container whose tar
    # streams to stdout; the file is created on the host by THIS shell, so
    # it belongs to the backup user, not to root inside the container.
    docker run --rm -v "$MB_UPLOADS_VOLUME:/src:ro" alpine:3.20 \
        tar -czf - -C /src . > "$UPLOADS_OUT.part"
    mv "$UPLOADS_OUT.part" "$UPLOADS_OUT"
    printf '%s %s\n' "$FINGERPRINT" "$(basename "$UPLOADS_OUT")" > "$FINGERPRINT_FILE"
    log "uploads archive ok ($(du -h "$UPLOADS_OUT" | cut -f1))"
fi

# ── 3. Off-machine copy (optional) ──────────────────────────────────────
# Two copies on one laptop are one copy. `rclone copy` uploads whatever
# the remote lacks — so a night the upload failed is caught up by the
# next successful one — and only THEN (set -e ordering) are old remote
# dumps pruned by age. Uploads archives are never pruned remotely: each
# is a distinct state of the volume, and they are only written on change.
if [[ -n "$MB_BACKUP_REMOTE" ]]; then
    log "copying to $MB_BACKUP_REMOTE"
    rclone copy "$MB_BACKUP_DIR" "$MB_BACKUP_REMOTE" --include 'mb_*.dump' --include 'uploads_*.tar.gz'
    rclone delete "$MB_BACKUP_REMOTE" --include 'mb_*.dump' --min-age "${MB_REMOTE_KEEP_DAYS}d"
    log "remote copy ok"
fi

# ── 4. Local retention ──────────────────────────────────────────────────
# Newest N by count, and only after tonight's files exist. Pruning by AGE
# has a famous failure mode: anything that deletes "older than 14 days"
# independently of a successful backup (a separate cron, a storage
# lifecycle rule) keeps deleting after backups quietly stop, until the
# directory is empty. Keeping the newest N can never empty it, however
# long ago N was.
prune() { ls -1t "$MB_BACKUP_DIR"/$1 2>/dev/null | tail -n +"$((MB_BACKUP_KEEP + 1))" | xargs -r rm -- ; }
prune 'mb_*.dump'
prune 'uploads_*.tar.gz'
alert resolve # a BackupFailed still firing from an earlier night gets its all-clear
log "done — newest $MB_BACKUP_KEEP of each kept in $MB_BACKUP_DIR"
