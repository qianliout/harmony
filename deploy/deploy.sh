#!/usr/bin/env bash
set -euo pipefail

HOST="${HOST:?set HOST=root@server}"
APP_DIR=/opt/harmony
ENV_FILE=/etc/harmony/harmony.env
CONFIG=/etc/harmony/config.yaml

scp backend/bin/harmony-linux-amd64 "$HOST:$APP_DIR/harmony.new"
ssh "$HOST" bash -s <<REMOTE
set -euo pipefail
cd $APP_DIR
[ -x harmony ] && $APP_DIR/harmony backup -config $CONFIG || true
$APP_DIR/harmony.new migrate -config $CONFIG
[ -x harmony ] && mv -f harmony harmony.prev || true
mv -f harmony.new harmony
chmod 0755 harmony
systemctl restart harmony
for i in \$(seq 1 10); do
  if curl -fsS http://127.0.0.1:8080/api/v1/health >/dev/null; then
    exit 0
  fi
  sleep 1
done
echo "health check failed, rolling back" >&2
[ -x harmony.prev ] && mv -f harmony.prev harmony && systemctl restart harmony
exit 1
REMOTE
