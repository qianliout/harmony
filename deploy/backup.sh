#!/usr/bin/env bash
set -euo pipefail

CONFIG="${HARMONY_CONFIG:-/etc/harmony/config.yaml}"
BACKUP_DIR="${HARMONY_BACKUP_DIR:-/var/backups/harmony}"
OSS_BUCKET="${HARMONY_OSS_BUCKET:?set HARMONY_OSS_BUCKET}"

/opt/harmony/harmony backup -config "$CONFIG"
latest=$(ls -t "$BACKUP_DIR"/harmony-*.db | head -n 1)
gzip -kf "$latest"
ossutil cp -f "$latest.gz" "oss://$OSS_BUCKET/db/$(basename "$latest").gz"
ossutil sync /var/lib/harmony/uploads "oss://$OSS_BUCKET/uploads/" --update
rm -f "$latest.gz"
