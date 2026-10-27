#!/bin/sh
set -eu
mkdir -p /var/run/gateway /var/lib/gateway-agent
chown 65532:65532 /var/run/gateway /var/lib/gateway-agent
chmod 755 /var/run/gateway /var/lib/gateway-agent
exec setpriv --reuid=65532 --regid=65532 --clear-groups /usr/local/bin/agent
