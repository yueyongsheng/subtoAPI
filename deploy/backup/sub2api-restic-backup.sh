#!/usr/bin/env bash
set -euo pipefail

source /etc/sub2api-restic.env

export AWS_ACCESS_KEY_ID
export AWS_SECRET_ACCESS_KEY
export RESTIC_PASSWORD
export RESTIC_REPOSITORY

if ! restic snapshots --no-lock >/dev/null 2>&1; then
  restic init
fi

restic backup \
  --tag daily \
  --exclude '/opt/sub2api/compose/data/logs' \
  --exclude '/opt/sub2api/compose/postgres_data' \
  /opt/sub2api/compose \
  /opt/sub2api/caddy \
  /etc/caddy \
  /etc/systemd/system/sub2api-health-monitor.service \
  /etc/systemd/system/sub2api-health-monitor.timer \
  /usr/local/sbin/sub2api-health-monitor \
  /etc/sub2api-monitor.env

restic forget \
  --tag daily \
  --keep-daily 14 \
  --keep-weekly 8 \
  --keep-monthly 12 \
  --prune

restic check
install -d -m 700 /var/lib/sub2api-backup
date -u +%s > /var/lib/sub2api-backup/last-success
chmod 600 /var/lib/sub2api-backup/last-success
