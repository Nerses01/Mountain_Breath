#!/usr/bin/env bash
# Fire or resolve an alert in Alertmanager by hand — the door for things
# Prometheus cannot see (a failed backup, a maintenance note) and the way to
# test the notification channel without waiting for an outage.
#
#   bash deploy/alert.sh fire    NAME "what happened" [severity]   # pages; repeats until resolved
#   bash deploy/alert.sh resolve NAME                              # the all-clear
#
# Everything that pages goes through Alertmanager — one channel, one place
# for silences, one token holder — rather than each script owning a call
# to Telegram (decision #105).
set -euo pipefail

MB_ALERTMANAGER_URL="${MB_ALERTMANAGER_URL:-http://127.0.0.1:9093}"

MODE="${1:-}"; NAME="${2:-}"; SUMMARY="${3:-$NAME}"; SEVERITY="${4:-critical}"
if [[ -z "$NAME" ]] || [[ "$MODE" != "fire" && "$MODE" != "resolve" ]]; then
    sed -n '2,11p' "$0"; exit 2
fi

# Alertmanager's API takes an alert as a time span. "fire" opens one that
# lasts a day — plenty for repeat_interval to nag until someone resolves it
# — and "resolve" closes it now. Same labels both times: that is what makes
# the two calls describe the same alert.
now=$(date -u +%FT%TZ)
case "$MODE" in
    fire)    ends=$(date -u -d '+24 hours' +%FT%TZ) ;;
    resolve) ends=$now ;;
esac
curl -fsS -X POST "$MB_ALERTMANAGER_URL/api/v2/alerts" -H 'Content-Type: application/json' \
    -d "[{\"labels\":{\"alertname\":\"$NAME\",\"severity\":\"$SEVERITY\",\"source\":\"manual\"},\"annotations\":{\"summary\":\"$SUMMARY\"},\"startsAt\":\"$now\",\"endsAt\":\"$ends\"}]"
echo "$MODE $NAME -> $MB_ALERTMANAGER_URL"
