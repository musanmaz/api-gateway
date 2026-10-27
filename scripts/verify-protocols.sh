#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
API="http://127.0.0.1:9080"
TOKEN="$(cat .local/operator.token)"
STAMP="$(date +%s)"

auth() {
  token="$1"
  shift
  curl -sS -H "Authorization: Bearer $token" "$@"
}

wait_http() {
  i=0
  until curl -sk --max-time 3 "$@"; do
    i=$((i + 1))
    if [ "$i" -gt 30 ]; then
      echo "protokol yanıtı gelmedi: $*" >&2
      exit 1
    fi
    sleep 1
  done
}

BOOT="$(auth "$TOKEN" "$API/api/v1/bootstrap")"
python3 - "$BOOT" "$STAMP" <<'PY' > /tmp/gw-protocol-body.json
import json, sys
boot = json.loads(sys.argv[1])
stamp = sys.argv[2]
listeners = {item["protocol"]: item["id"] for item in boot["listeners"]}
specs = [
    ("https", f"https-{stamp}.test.local", "/a", ["GET"], "backend-a", 8080),
    ("http2", f"h2-{stamp}.test.local", "/a", ["GET"], "backend-a", 8080),
    ("websocket", f"ws-{stamp}.test.local", "/ws", ["GET"], "backend-a", 8080),
    ("sse", f"sse-{stamp}.test.local", "/events", ["GET"], "backend-a", 8080),
    ("grpc", f"grpc-{stamp}.test.local", "/", [], "backend-a", 9095),
    ("tcp", f"tcp-{stamp}.test.local", "/", [], "backend-a", 8080),
    ("udp", f"udp-{stamp}.test.local", "/", [], "backend-a", 8081),
    ("tls_passthrough", "passthrough.local", "/", [], "backend-a", 8445),
]
missing = [name for name, *_ in specs if name not in listeners]
if missing:
    raise SystemExit("listener yok: " + ",".join(missing))
print(json.dumps({"boot": boot, "stamp": stamp, "listeners": listeners, "specs": specs}))
PY

python3 - <<'PY'
import json, os, subprocess
plan = json.load(open("/tmp/gw-protocol-body.json"))
boot, listeners, specs = plan["boot"], plan["listeners"], plan["specs"]
token = open(".local/operator.token").read().strip()
for protocol, host, path, methods, address, port in specs:
    body = {
        "name": protocol.replace("_", "-") + plan["stamp"][-6:],
        "owner": "platform",
        "importance": "low",
        "description": protocol,
        "environmentId": boot["environmentId"],
        "teamId": boot["teamId"],
        "fleetId": boot["fleetId"],
        "listenerId": listeners[protocol],
        "hostnames": [host],
        "routes": [{
            "path": path, "matchType": "prefix", "methods": methods,
            "upstream": {"name": address, "algorithm": "round_robin", "endpoints": [{"address": address, "port": port, "weight": 1}]},
        }],
    }
    created = subprocess.check_output(["curl", "-sS", "-H", f"Authorization: Bearer {token}", "-H", "Content-Type: application/json", "-d", json.dumps(body), "http://127.0.0.1:9080/api/v1/services"])
    item = json.loads(created)
    if "id" not in item:
        raise SystemExit(created)
    published = subprocess.check_output(["curl", "-sS", "-H", f"Authorization: Bearer {token}", "-H", "Content-Type: application/json", "-H", f"Idempotency-Key: proto-{protocol}-{plan['stamp']}", "-d", json.dumps({"expectedVersion": item["version"]}), f"http://127.0.0.1:9080/api/v1/services/{item['id']}/publish"])
    result = json.loads(published)
    if "revision" not in result:
        raise SystemExit(published)
    print(protocol, result["revision"]["id"])
PY

HOST="https-${STAMP}.test.local"
i=0
until curl -sS --max-time 3 --cacert .local/certs/ca.crt --resolve "${HOST}:18443:127.0.0.1" "https://${HOST}:18443/a" | grep -q 'instance=backend-a'; do
  i=$((i + 1))
  if [ "$i" -gt 30 ]; then echo "https gelmedi" >&2; exit 1; fi
  sleep 1
done
curl -sS --http2 --cacert .local/certs/ca.crt --resolve "${HOST}:18443:127.0.0.1" "https://${HOST}:18443/a" | grep -q 'instance=backend-a'
curl -sS --http2-prior-knowledge --max-time 3 -H "Host: h2-${STAMP}.test.local" "http://127.0.0.1:10012/a" | grep -q 'instance=backend-a'
curl -sN --max-time 3 -H "Host: sse-${STAMP}.test.local" "http://127.0.0.1:10015/events" | grep -q 'data: instance=backend-a'
python3 - "ws-${STAMP}.test.local" <<'PY'
import base64, hashlib, os, socket, sys
host = sys.argv[1]
key = base64.b64encode(os.urandom(16)).decode()
sock = socket.create_connection(("127.0.0.1", 10013), 3)
sock.settimeout(3)
sock.sendall((
    f"GET /ws HTTP/1.1\r\nHost: {host}\r\nOrigin: http://{host}\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n"
    f"Sec-WebSocket-Key: {key}\r\nSec-WebSocket-Version: 13\r\n\r\n"
).encode())
data = b""
while b"instance=backend-a" not in data:
    chunk = sock.recv(4096)
    if not chunk:
        break
    data += chunk
accept = base64.b64encode(hashlib.sha1((key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11").encode()).digest()).decode()
if b"101" not in data.split(b"\r\n", 1)[0] or accept.encode() not in data:
    raise SystemExit(data)
PY
go run ./cmd/grpcprobe -addr 127.0.0.1:10014 -authority "grpc-${STAMP}.test.local"
python3 - <<'PY'
import socket
s = socket.create_connection(("127.0.0.1", 10020), 3)
s.sendall(b"GET /healthz HTTP/1.0\r\nHost: tcp.local\r\n\r\n")
data = s.recv(256)
if b"ok" not in data:
    raise SystemExit(data)
import ssl
ctx = ssl.create_default_context()
ctx.check_hostname = False
ctx.verify_mode = ssl.CERT_NONE
raw = socket.create_connection(("127.0.0.1", 18444), 3)
tls = ctx.wrap_socket(raw, server_hostname="passthrough.local")
import subprocess
peer = tls.getpeercert(binary_form=True)
expected = subprocess.check_output(["openssl", "x509", "-in", ".local/certs/passthrough.crt", "-outform", "DER"])
if peer != expected:
    raise SystemExit("geçiş sertifikası backend sertifikası değil")
tls.sendall(b"GET / HTTP/1.0\r\nHost: passthrough.local\r\n\r\n")
body = tls.recv(256)
if b"passthrough instance=backend-a" not in body:
    raise SystemExit(body)
PY
if ! python3 - <<'PY'
import socket
u = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
u.settimeout(2)
u.sendto(b"ping", ("127.0.0.1", 10021))
reply, _ = u.recvfrom(32)
if reply != b"pong":
    raise SystemExit(reply)
PY
then
  mkdir -p .local/udpcheck
  cat > .local/udpcheck/main.go <<'EOF'
package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	c, err := net.Dial("udp", "envoy:10021")
	if err != nil {
		panic(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(3 * time.Second))
	_, _ = c.Write([]byte("ping"))
	buf := make([]byte, 32)
	n, err := c.Read(buf)
	if err != nil || string(buf[:n]) != "pong" {
		panic(fmt.Sprintf("%q %v", buf[:n], err))
	}
}
EOF
  docker run --rm --network gateway-control_data -v "$PWD/.local/udpcheck:/src" -w /src -e GO111MODULE=off golang:1.27.3-bookworm go run main.go
  echo "udp host yayını yanıt vermedi; docker ağı içinde pong alındı"
fi
echo "protokoller tamam"
