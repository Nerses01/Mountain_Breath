#!/usr/bin/env bash
# Restore drill and real restore for the archives backup.sh writes.
#
#   bash restore.sh --drill [DUMP|latest]   prove a backup restores: into a
#                                           scratch database, dropped afterwards
#   bash restore.sh --real  [DUMP|latest]   disaster recovery: REPLACE the live
#                                           database and uploads with the backup
#
# Same environment knobs as backup.sh: MB_BACKUP_DIR, MB_COMPOSE_FILE,
# MB_UPLOADS_VOLUME. DUMP is the path of an mb_<stamp>.dump; the uploads
# archive is chosen automatically — the newest one not newer than the dump.
set -euo pipefail

MB_BACKUP_DIR="${MB_BACKUP_DIR:-/opt/backups}"
MB_COMPOSE_FILE="${MB_COMPOSE_FILE:-/opt/mountain-breath/deploy/docker-compose.prod.yml}"
MB_UPLOADS_VOLUME="${MB_UPLOADS_VOLUME:-mountain-breath_uploads_data}"
DRILL_DB="mb_restore_drill"
export MSYS_NO_PATHCONV=1   # Git Bash path rewriting off; see backup.sh

compose() { docker compose -f "$MB_COMPOSE_FILE" "$@"; }
log() { printf '%s restore: %s\n' "$(date '+%F %T')" "$*"; }
die() { log "ERROR: $*" >&2; exit 1; }
# Run a Postgres client tool inside the container as the server's own
# role. `sh -c '…"$@"' sh ARGS…`: the word after the script becomes $0
# and the rest arrive as "$@" — arguments cross into the container with
# their quoting intact, and $POSTGRES_USER expands in there, not here.
pgtool() { compose exec -T postgres sh -c 'tool=$1; shift; exec "$tool" -U "$POSTGRES_USER" "$@"' sh "$@"; }

MODE="${1:-}"; DUMP="${2:-latest}"
case "$MODE" in
    --drill|--real) ;;
    *) sed -n '2,11p' "$0"; exit 2 ;;
esac

if [[ "$DUMP" == "latest" ]]; then
    DUMP=$(ls -1t "$MB_BACKUP_DIR"/mb_*.dump 2>/dev/null | head -n 1 || true)
    [[ -n "$DUMP" ]] || die "no mb_*.dump in $MB_BACKUP_DIR"
fi
[[ -f "$DUMP" ]] || die "no such dump: $DUMP"
STAMP=$(basename "$DUMP" .dump); STAMP=${STAMP#mb_}

# Stamps are %F_%H%M%S, so they order correctly as plain strings: the
# newest uploads archive whose stamp is <= the dump's holds the volume as
# it was when the dump was taken (archives are only written on change).
UPLOADS=""
for f in "$MB_BACKUP_DIR"/uploads_*.tar.gz; do
    [[ -f "$f" ]] || continue
    s=$(basename "$f" .tar.gz); s=${s#uploads_}
    if [[ "$s" < "$STAMP" || "$s" == "$STAMP" ]]; then UPLOADS="$f"; fi
done

DB=$(compose exec -T postgres sh -c 'printf %s "$POSTGRES_DB"'); DB=${DB//$'\r'/}
[[ -n "$DB" ]] || die "could not read POSTGRES_DB from the postgres container"
log "dump:    $DUMP ($(date -r "$DUMP" '+%F %T'), $(du -h "$DUMP" | cut -f1))"
log "uploads: ${UPLOADS:-none found}"

# Table of contents first: does the archive even parse? No database is
# touched. TOC entries are the lines that start with a number.
ENTRIES=$(pgtool pg_restore --list < "$DUMP" | grep -c '^[0-9]' || true)
[[ "$ENTRIES" -gt 0 ]] || die "dump has an empty table of contents"
log "dump parses: $ENTRIES archive entries"

# The questions the shop asks of its data, asked of a database by name.
row_counts() {
    pgtool psql -Atq -v ON_ERROR_STOP=1 -d "$1" -c \
        "select (select count(*) from products) || ' products, ' ||
                (select count(*) from orders)   || ' orders, '   ||
                (select count(*) from users)    || ' users, schema version ' ||
                (select version from schema_migrations)"
}

drill() {
    # Scope guard, the bash way: whatever happens below — an error, a
    # Ctrl-C — the scratch database is dropped on the way out. This is
    # what a destructor or scope_exit does in C++; bash spells it as a
    # trap on EXIT.
    trap 'log "dropping $DRILL_DB"; pgtool dropdb --if-exists "$DRILL_DB" || true' EXIT
    pgtool dropdb --if-exists "$DRILL_DB"      # a previous drill that died
    pgtool createdb "$DRILL_DB"

    log "restoring into scratch database $DRILL_DB"
    # --exit-on-error: the first failed statement aborts with a non-zero
    # exit, instead of pg_restore's default of carrying on and summarising
    # "N errors ignored" at the end — a drill must not pass with caveats.
    # --no-owner: objects belong to whoever restores; the archive's owner
    # name does not have to exist on the restoring machine.
    pgtool pg_restore --dbname="$DRILL_DB" --no-owner --exit-on-error < "$DUMP"
    log "restored: $(row_counts "$DRILL_DB")"
    log "live:     $(row_counts "$DB")"

    if [[ -n "$UPLOADS" ]]; then
        # Listing a gzip archive reads every byte of it — that IS the
        # integrity check, and a truncated file fails here; set -e makes
        # that failure the drill's. The archive arrives on stdin rather
        # than as a path: GNU tar parses "C:/…" as "host C" (the old
        # remote-tape syntax), which would fail a rehearsal on Windows
        # for the wrong reason.
        LIST=$(tar -tzf - < "$UPLOADS")
        N=$(printf '%s\n' "$LIST" | grep -c -v '/$' || true)
        LIVE_N=$(docker run --rm -v "$MB_UPLOADS_VOLUME:/src:ro" alpine:3.20 sh -c 'find /src -type f | wc -l')
        log "uploads archive reads back: $N files (live volume now: ${LIVE_N//$'\r'/})"
    else
        log "WARNING: no uploads archive — photos and videos would not be restorable"
    fi
    log "DRILL PASSED — $DUMP restores"
}

real() {
    [[ -t 0 ]] || die "--real is interactive: it needs a terminal to confirm on"
    echo
    echo "  This REPLACES the live database '$DB' and the uploads volume"
    echo "  '$MB_UPLOADS_VOLUME' with:"
    echo "      $DUMP"
    echo "      ${UPLOADS:-(no uploads archive — the volume is left as it is)}"
    echo
    read -r -p "  Type the database name to continue: " ANSWER
    [[ "$ANSWER" == "$DB" ]] || die "aborted — nothing was changed"

    log "stopping the api so nothing writes during the restore"
    compose stop api
    # --clean --create: connect to the maintenance database (postgres),
    # DROP the shop's database, CREATE it again, restore into the fresh
    # one. Open connections would block the DROP — hence the stop above.
    # --if-exists lets the same command work on an empty volume (a new
    # machine), where there is nothing to drop yet.
    log "recreating $DB from $DUMP"
    pgtool pg_restore --dbname=postgres --clean --create --if-exists --no-owner --exit-on-error < "$DUMP"

    if [[ -n "$UPLOADS" ]]; then
        log "replacing the uploads volume from $UPLOADS"
        docker run --rm -i -v "$MB_UPLOADS_VOLUME:/dst" alpine:3.20 \
            sh -c 'rm -rf /dst/* /dst/.[!.]* 2>/dev/null; tar -xzf - -C /dst' < "$UPLOADS"
    fi

    # Bring the stack back. The migrate job runs `up` before the api
    # starts — if the images are newer than the archive, the schema is
    # carried forward to what the code expects.
    log "starting the stack"
    compose up -d
    log "RESTORED — check: docker compose -f $MB_COMPOSE_FILE ps"
}

case "$MODE" in
    --drill) drill ;;
    --real)  real ;;
esac
