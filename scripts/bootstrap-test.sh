#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
sh scripts/bootstrap.sh
# Containers run as UID/GID 65532. Keep secrets group-readable, never world-readable.
if [ "$(id -u)" != "0" ]; then
  echo "Test deployment secret ownership requires root on the target host." >&2
  exit 1
fi
chgrp 65532 .local/operator.token .local/viewer.token .local/agent.token .local/keys/ed25519.key .local/certs/server.key .local/certs/passthrough.key
chmod 640 .local/operator.token .local/viewer.token .local/agent.token .local/keys/ed25519.key .local/certs/server.key .local/certs/passthrough.key
chmod 644 .local/keys/ed25519.pub
if [ ! -s .local/test.env ]; then
  umask 077
  python3 - <<'ENVGEN'
from pathlib import Path
import secrets
Path('.local/test.env').write_text('GATEWAY_DB_PASSWORD='+secrets.token_hex(32)+'\n')
ENVGEN
fi
chmod 600 .local/test.env
echo "Test identities and database configuration are ready. No secrets were printed."
