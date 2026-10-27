#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
mkdir -p .local/keys
umask 077
if [ ! -s .local/keys/ed25519.key ]; then
  openssl genpkey -algorithm ED25519 -out .local/keys/ed25519.key
  openssl pkey -in .local/keys/ed25519.key -pubout -out .local/keys/ed25519.pub
fi
if [ ! -s .local/operator.token ]; then openssl rand -hex 32 > .local/operator.token; fi
if [ ! -s .local/viewer.token ]; then openssl rand -hex 32 > .local/viewer.token; fi
if [ ! -s .local/agent.token ]; then openssl rand -hex 32 > .local/agent.token; fi
chmod 600 .local/operator.token .local/viewer.token .local/agent.token .local/keys/ed25519.key
mkdir -p .local/certs
if [ ! -s .local/certs/ca.crt ]; then
  openssl req -x509 -newkey rsa:2048 -nodes -keyout .local/certs/ca.key -out .local/certs/ca.crt -days 825 -subj "/CN=gateway-local-ca"
fi
issue_cert() {
  name="$1"
  san="$2"
  if [ -s ".local/certs/${name}.crt" ]; then
    return
  fi
  openssl req -newkey rsa:2048 -nodes -keyout ".local/certs/${name}.key" -out ".local/certs/${name}.csr" -subj "/CN=${name}.local" -addext "subjectAltName=${san}"
  openssl x509 -req -in ".local/certs/${name}.csr" -CA .local/certs/ca.crt -CAkey .local/certs/ca.key -CAcreateserial -out ".local/certs/${name}.crt" -days 825 -copy_extensions copy
  rm -f ".local/certs/${name}.csr"
}
issue_cert server "DNS:*.test.local,DNS:localhost,IP:127.0.0.1"
issue_cert passthrough "DNS:passthrough.local"
chmod 644 .local/certs/*.crt
chmod 600 .local/certs/*.key
echo "Yerel kimlikler .local altında. Operatör tokenı: make token"
