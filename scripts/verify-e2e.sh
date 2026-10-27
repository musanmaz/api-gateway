#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
COMPOSE="docker compose -f deploy/compose/compose.yaml"
API="http://127.0.0.1:9080"
GW="http://127.0.0.1:10000"
N1="http://127.0.0.1:10010"
N2="http://127.0.0.1:10011"
TOKEN="$(cat .local/operator.token)"
VIEWER="$(cat .local/viewer.token)"
HOST="e2e-$(date +%s).local"
SIDE="side-${HOST}"
REPORT="docs/test-results.md"

auth() {
  token="$1"
  shift
  curl -sS -H "Authorization: Bearer $token" "$@"
}

wait_host() {
  target="$1"
  host="$2"
  path="$3"
  expect="$4"
  i=0
  until curl -sS --max-time 2 -H "Host: $host" "$target$path" | grep -q "$expect"; do
    i=$((i + 1))
    if [ "$i" -gt 40 ]; then
      echo "yanıt gelmedi: $target $host$path" >&2
      $COMPOSE logs --tail=60 agent agent-2 envoy envoy-2 edge api >&2 || true
      exit 1
    fi
    sleep 1
  done
}

echo "API bekleniyor"
i=0
until auth "$TOKEN" "$API/healthz" | grep -q '"ok"'; do
  i=$((i + 1))
  if [ "$i" -gt 60 ]; then
    echo "API hazır değil" >&2
    exit 1
  fi
  sleep 1
done

BOOT="$(auth "$TOKEN" "$API/api/v1/bootstrap")"
ENV_ID="$(printf '%s' "$BOOT" | python3 -c 'import json,sys; print(json.load(sys.stdin)["environmentId"])')"
TEAM_ID="$(printf '%s' "$BOOT" | python3 -c 'import json,sys; print(json.load(sys.stdin)["teamId"])')"
FLEET_ID="$(printf '%s' "$BOOT" | python3 -c 'import json,sys; print(json.load(sys.stdin)["fleetId"])')"
LISTENER_ID="$(printf '%s' "$BOOT" | python3 -c 'import json,sys; print(json.load(sys.stdin)["listener"]["id"])')"

body_for() {
  python3 - "$1" "$2" "$ENV_ID" "$TEAM_ID" "$FLEET_ID" "$LISTENER_ID" <<'PY'
import json, sys
name, host, env, team, fleet, listener = sys.argv[1:]
print(json.dumps({
  "name": name,
  "owner": "platform",
  "importance": "medium",
  "description": "uçtan uca doğrulama",
  "environmentId": env,
  "teamId": team,
  "fleetId": fleet,
  "listenerId": listener,
  "hostnames": [host],
  "routes": [
    {"path": "/a", "matchType": "prefix", "methods": ["GET"], "upstream": {"name": "backend-a", "algorithm": "round_robin", "endpoints": [{"address": "backend-a", "port": 8080, "weight": 1}]}},
    {"path": "/b", "matchType": "prefix", "methods": ["POST"], "upstream": {"name": "backend-b", "algorithm": "round_robin", "endpoints": [{"address": "backend-b", "port": 8080, "weight": 1}]}}
  ]
}))
PY
}

BODY="$(body_for "e2e${HOST%%.*}" "$HOST")"
CREATE="$(auth "$TOKEN" -H 'Content-Type: application/json' -d "$BODY" "$API/api/v1/services")"
ID="$(printf '%s' "$CREATE" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')"
VERSION="$(printf '%s' "$CREATE" | python3 -c 'import json,sys; print(json.load(sys.stdin)["version"])')"
echo "servis $ID"

DIFF="$(auth "$TOKEN" "$API/api/v1/services/$ID/diff")"
printf '%s' "$DIFF" | python3 -c 'import json,sys; data=json.load(sys.stdin); assert any(item["op"]=="add" and item["path"]=="/a" for item in data["changes"]), data'

VALID="$(auth "$TOKEN" -X POST "$API/api/v1/services/$ID/validate")"
printf '%s' "$VALID" | python3 -c 'import json,sys; data=json.load(sys.stdin); assert data["valid"], data'

KEY="e2e-$HOST"
PUBLISH="$(auth "$TOKEN" -H 'Content-Type: application/json' -H "Idempotency-Key: $KEY" -d "{\"expectedVersion\":$VERSION}" "$API/api/v1/services/$ID/publish")"
REV="$(printf '%s' "$PUBLISH" | python3 -c 'import json,sys; print(json.load(sys.stdin)["revision"]["id"])')"
REPLAY="$(auth "$TOKEN" -H 'Content-Type: application/json' -H "Idempotency-Key: $KEY" -d "{\"expectedVersion\":$VERSION}" "$API/api/v1/services/$ID/publish")"
printf '%s' "$REPLAY" | python3 -c 'import json,sys; data=json.load(sys.stdin); assert data["replayed"] and data["revision"]["id"]=='"$REV"

CODE="$(curl -sS -o /tmp/gw-viewer.txt -w '%{http_code}' -H "Authorization: Bearer $VIEWER" -H 'Content-Type: application/json' -d "{\"expectedVersion\":$VERSION}" "$API/api/v1/services/$ID/publish")"
test "$CODE" = "403"

echo "trafik bekleniyor: $HOST"
wait_host "$GW" "$HOST" "/a" "instance=backend-a"
wait_host "$N1" "$HOST" "/a" "instance=backend-a"
wait_host "$N2" "$HOST" "/a" "instance=backend-a"
curl -sS -H "Host: $HOST" -X POST "$GW/b" | grep -q 'instance=backend-b'
UNKNOWN="$(curl -sS -o /tmp/gw-unknown.txt -w '%{http_code}' -H 'Host: nope.example' "$GW/")"
test "$UNKNOWN" = "404"

wait_node() {
  name="$1"
  python_expr="$2"
  i=0
  until auth "$TOKEN" "$API/api/v1/overview" | python3 -c "$python_expr"; do
    i=$((i + 1))
    if [ "$i" -gt 40 ]; then
      echo "düğüm durumu gelmedi: $name" >&2
      auth "$TOKEN" "$API/api/v1/overview" >&2 || true
      exit 1
    fi
    sleep 1
  done
}

echo "ACK bekleniyor"
wait_node gw-local-1 'import json,sys; n=next(x for x in json.load(sys.stdin)["nodes"] if x["name"]=="gw-local-1"); raise SystemExit(0 if n.get("ackedRevision")=='"$REV"' else 1)'
wait_node gw-local-2 'import json,sys; n=next(x for x in json.load(sys.stdin)["nodes"] if x["name"]=="gw-local-2"); raise SystemExit(0 if n.get("ackedRevision")=='"$REV"' else 1)'

echo "eşzamanlı yayım"
CONC_HOST="conc-$HOST"
CONC_BODY="$(body_for "conc${HOST%%.*}" "$CONC_HOST")"
CONC_CREATE="$(auth "$TOKEN" -H 'Content-Type: application/json' -d "$CONC_BODY" "$API/api/v1/services")"
CONC_ID="$(printf '%s' "$CONC_CREATE" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')"
CONC_VERSION="$(printf '%s' "$CONC_CREATE" | python3 -c 'import json,sys; print(json.load(sys.stdin)["version"])')"
auth "$TOKEN" -H 'Content-Type: application/json' -H "Idempotency-Key: conc-a-$HOST" -d "{\"expectedVersion\":$CONC_VERSION}" -o /tmp/gw-conc-a.json -w '%{http_code}' "$API/api/v1/services/$CONC_ID/publish" >/tmp/gw-conc-a.code &
auth "$TOKEN" -H 'Content-Type: application/json' -H "Idempotency-Key: conc-b-$HOST" -d "{\"expectedVersion\":$CONC_VERSION}" -o /tmp/gw-conc-b.json -w '%{http_code}' "$API/api/v1/services/$CONC_ID/publish" >/tmp/gw-conc-b.code &
wait
CODE_A="$(cat /tmp/gw-conc-a.code)"
CODE_B="$(cat /tmp/gw-conc-b.code)"
python3 - "$CODE_A" "$CODE_B" <<'PY'
import sys
codes = sorted(sys.argv[1:])
if codes != ["200", "409"]:
    raise SystemExit(f"eşzamanlı yayım kodları {codes}")
PY
VIEWER_CANARY="$(curl -sS -o /tmp/gw-viewer-canary.txt -w '%{http_code}' -H "Authorization: Bearer $VIEWER" -H 'Content-Type: application/json' -d "{\"expectedVersion\":$CONC_VERSION,\"node\":\"gw-local-1\"}" "$API/api/v1/services/$CONC_ID/publish")"
test "$VIEWER_CANARY" = "403"

echo "canary gw-local-1"
CANARY_HOST="canary-$HOST"
CANARY_BODY="$(body_for "canary${HOST%%.*}" "$CANARY_HOST")"
CANARY_CREATE="$(auth "$TOKEN" -H 'Content-Type: application/json' -d "$CANARY_BODY" "$API/api/v1/services")"
CANARY_ID="$(printf '%s' "$CANARY_CREATE" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')"
CANARY_VERSION="$(printf '%s' "$CANARY_CREATE" | python3 -c 'import json,sys; print(json.load(sys.stdin)["version"])')"
CANARY_PUBLISH="$(auth "$TOKEN" -H 'Content-Type: application/json' -H "Idempotency-Key: canary-$HOST" -d "{\"expectedVersion\":$CANARY_VERSION,\"node\":\"gw-local-1\"}" "$API/api/v1/services/$CANARY_ID/publish")"
CANARY_REV="$(printf '%s' "$CANARY_PUBLISH" | python3 -c 'import json,sys; print(json.load(sys.stdin)["revision"]["id"])')"
wait_host "$N1" "$CANARY_HOST" "/a" "instance=backend-a"
i=0
while [ "$i" -lt 8 ]; do
  N2_CODE="$(curl -sS -o /tmp/gw-canary-n2.txt -w '%{http_code}' --max-time 2 -H "Host: $CANARY_HOST" "$N2/a" || true)"
  if [ "$N2_CODE" = "200" ]; then
    echo "canary ikinci düğüme sızdı" >&2
    exit 1
  fi
  i=$((i + 1))
  sleep 1
done
wait_node gw-local-1 'import json,sys; n=next(x for x in json.load(sys.stdin)["nodes"] if x["name"]=="gw-local-1"); raise SystemExit(0 if n.get("ackedRevision")=='"$CANARY_REV"' else 1)'
N2_ACK="$(auth "$TOKEN" "$API/api/v1/overview" | python3 -c 'import json,sys; n=next(x for x in json.load(sys.stdin)["nodes"] if x["name"]=="gw-local-2"); print(n.get("desiredRevision"))')"
test "$N2_ACK" != "$CANARY_REV"

echo "NACK gw-local-1"
HTTPS_LISTENER="$(printf '%s' "$BOOT" | python3 -c 'import json,sys; print(next(x["id"] for x in json.load(sys.stdin)["listeners"] if x["protocol"]=="https"))')"
NACK_HOST="nack-$HOST"
NACK_BODY="$(python3 - "$HTTPS_LISTENER" "$NACK_HOST" "$ENV_ID" "$TEAM_ID" "$FLEET_ID" <<'PY'
import json, sys
listener, host, env, team, fleet = sys.argv[1:]
print(json.dumps({
  "name": "nack" + host.split(".", 1)[0].replace("-", "")[:20],
  "owner": "platform",
  "importance": "medium",
  "description": "envoy nack",
  "environmentId": env,
  "teamId": team,
  "fleetId": fleet,
  "listenerId": listener,
  "hostnames": [host],
  "routes": [
    {"path": "/a", "matchType": "prefix", "methods": ["GET"], "upstream": {"name": "backend-a", "algorithm": "round_robin", "endpoints": [{"address": "backend-a", "port": 8080, "weight": 1}]}}
  ]
}))
PY
)"
cp .local/certs/server.crt .local/certs/server.crt.e2e-backup
restore_cert() {
  if [ -f .local/certs/server.crt.e2e-backup ]; then
    # Aynı inode üzerine yazılır. mv yeni dosya yaratır; Docker bağladığı dosyayı kaybeder.
    cat .local/certs/server.crt.e2e-backup > .local/certs/server.crt
    rm -f .local/certs/server.crt.e2e-backup
  fi
}
trap restore_cert EXIT
printf 'not-a-certificate\n' > .local/certs/server.crt
NACK_CREATE="$(auth "$TOKEN" -H 'Content-Type: application/json' -d "$NACK_BODY" "$API/api/v1/services")"
NACK_ID="$(printf '%s' "$NACK_CREATE" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')"
NACK_VERSION="$(printf '%s' "$NACK_CREATE" | python3 -c 'import json,sys; print(json.load(sys.stdin)["version"])')"
NACK_PUBLISH="$(auth "$TOKEN" -H 'Content-Type: application/json' -H "Idempotency-Key: nack-$HOST" -d "{\"expectedVersion\":$NACK_VERSION,\"node\":\"gw-local-1\"}" "$API/api/v1/services/$NACK_ID/publish")"
NACK_REV="$(printf '%s' "$NACK_PUBLISH" | python3 -c 'import json,sys; print(json.load(sys.stdin)["revision"]["id"])')"
wait_node gw-local-1 'import json,sys; n=next(x for x in json.load(sys.stdin)["nodes"] if x["name"]=="gw-local-1"); raise SystemExit(0 if n.get("nackRevision")=='"$NACK_REV"' and n.get("ackedRevision")=='"$CANARY_REV"' else 1)'
wait_host "$N1" "$HOST" "/a" "instance=backend-a"
DISK_REV="$(docker run --rm --user 0 --entrypoint /bin/sh -v gateway-control_agentstate:/var/lib/gateway-agent gateway-control-agent -c 'tr -d "\n" < /var/lib/gateway-agent/current')"
test "$DISK_REV" != "$NACK_REV"
restore_cert
trap - EXIT

SIDE_BODY="$(body_for "side${HOST%%.*}" "$SIDE")"
SIDE_CREATE="$(auth "$TOKEN" -H 'Content-Type: application/json' -d "$SIDE_BODY" "$API/api/v1/services")"
SIDE_ID="$(printf '%s' "$SIDE_CREATE" | python3 -c 'import json,sys; print(json.load(sys.stdin)["id"])')"
SIDE_VERSION="$(printf '%s' "$SIDE_CREATE" | python3 -c 'import json,sys; print(json.load(sys.stdin)["version"])')"
auth "$TOKEN" -H 'Content-Type: application/json' -H "Idempotency-Key: side-$SIDE" -d "{\"expectedVersion\":$SIDE_VERSION}" "$API/api/v1/services/$SIDE_ID/publish" >/dev/null
wait_host "$GW" "$SIDE" "/a" "instance=backend-a"

ROLLBACK="$(auth "$TOKEN" -X POST "$API/api/v1/revisions/$REV/rollback")"
ROLLBACK_ID="$(printf '%s' "$ROLLBACK" | python3 -c 'import json,sys; print(json.load(sys.stdin)["revision"]["id"])')"
CURRENT="$(curl -sS -o /tmp/gw-rollback.txt -w '%{http_code}' -H "Authorization: Bearer $TOKEN" -X POST "$API/api/v1/revisions/$ROLLBACK_ID/rollback")"
test "$CURRENT" = "409"
VIEWER_ROLLBACK="$(curl -sS -o /tmp/gw-viewer-rollback.txt -w '%{http_code}' -H "Authorization: Bearer $VIEWER" -X POST "$API/api/v1/revisions/$REV/rollback")"
test "$VIEWER_ROLLBACK" = "403"
wait_host "$GW" "$HOST" "/a" "instance=backend-a"
i=0
until test "$(curl -sS -o /tmp/gw-side.txt -w '%{http_code}' --max-time 2 -H "Host: $SIDE" "$GW/a" || true)" != "200"; do
  i=$((i + 1))
  if [ "$i" -gt 40 ]; then
    echo "geri alma ikinci hostname'i kaldırmadı" >&2
    exit 1
  fi
  sleep 1
done
echo "geri alma revision $ROLLBACK_ID"

echo "bir düğüm durduruluyor"
START="$(python3 -c 'import time; print(time.time())')"
docker stop gateway-control-envoy-1 >/dev/null
i=0
until curl -sS --max-time 1 -H "Host: $HOST" "$GW/a" | grep -q 'instance=backend-a'; do
  i=$((i + 1))
  if [ "$i" -gt 40 ]; then
    echo "kalan düğüm yeni bağlantı kabul etmedi" >&2
    exit 1
  fi
done
RECOVERY_MS="$(python3 -c "import time; print(int((time.time()-float('$START'))*1000))")"
echo "yeni bağlantı ${RECOVERY_MS} ms"
docker start gateway-control-envoy-1 >/dev/null
wait_host "$N1" "$HOST" "/a" "instance=backend-a"

echo "API durduruluyor"
$COMPOSE stop api
curl -sS -H "Host: $HOST" "$GW/a" | grep -q 'instance=backend-a'
echo "API kapalıyken trafik sürdü"

echo "merkez kapalıyken iki düğüm yeniden başlatılıyor"
$COMPOSE restart agent agent-2 envoy envoy-2
wait_host "$GW" "$HOST" "/a" "instance=backend-a"
wait_host "$N1" "$HOST" "/a" "instance=backend-a"
wait_host "$N2" "$HOST" "/a" "instance=backend-a"
echo "yeniden başlatma sonrası trafik sürdü"

echo "bozuk yerel önbellek, merkez hâlâ kapalı"
$COMPOSE stop api agent agent-2 envoy envoy-2
docker run --rm --user 0 --entrypoint /bin/sh -v gateway-control_agentstate:/var/lib/gateway-agent gateway-control-agent -c 'for f in /var/lib/gateway-agent/current /var/lib/gateway-agent/accepted/*.json; do [ -f "$f" ] && printf bad > "$f"; done'
docker run --rm --user 0 --entrypoint /bin/sh -v gateway-control_agentstate2:/var/lib/gateway-agent gateway-control-agent -c 'for f in /var/lib/gateway-agent/current /var/lib/gateway-agent/accepted/*.json; do [ -f "$f" ] && printf bad > "$f"; done'
docker start gateway-control-agent-1 gateway-control-agent-2-1 >/dev/null
sleep 2
docker start gateway-control-envoy-1 gateway-control-envoy-2-1 >/dev/null
i=0
while [ "$i" -lt 4 ]; do
  BAD="$(curl -sS -o /tmp/gw-bad.txt -w '%{http_code}' --max-time 1 -H "Host: $HOST" "$GW/a" || true)"
  if [ "$BAD" = "200" ]; then
    echo "bozuk önbellek trafik vermeye devam etti" >&2
    exit 1
  fi
  i=$((i + 1))
  sleep 1
done
echo "bozuk önbellek trafik vermedi: ${BAD:-bağlantı yok}"

echo "API geri açılıyor"
docker start gateway-control-api-1 >/dev/null
wait_host "$GW" "$HOST" "/a" "instance=backend-a"
wait_host "$N2" "$HOST" "/a" "instance=backend-a"

mkdir -p docs
cat > "$REPORT" <<EOF
# Test sonuçları

Tarih: $(date -u +%Y-%m-%dT%H:%M:%SZ)
Ortam: Docker Compose, tek makine, iki gateway düğümü ve yerel edge. Bu sonuç rack, veri merkezi veya üretim HA kanıtı değildir.

| Test | Sonuç | Not |
| --- | --- | --- |
| Arayüz/API servisi iki backend'e gider | PASS | \`$HOST\` için \`/a\` backend-a, POST \`/b\` backend-b. Bilinmeyen host 404. İki düğüm de yanıtladı. |
| Viewer yayım yapamaz | PASS | HTTP 403. |
| Aynı idempotency anahtarı ikinci yayım açmaz | PASS | Revision $REV tekrarlandı. |
| Yayım farkı | PASS | Taslakta \`/a\` eklenecek olarak göründü. |
| Elle geri alma | PASS | Revision $REV paketinden revision $ROLLBACK_ID üretildi. İkinci hostname kalktı. Güncel pakete tekrar dönüş 409. Viewer 403. |
| Yönetim API durunca mevcut trafik sürer | PASS | API konteyneri durduruldu. |
| Merkez kapalıyken iki düğüm yeniden başlar | PASS | Doğrulanmış diskten iki düğüm ve edge trafik verdi. |
| Bozuk yerel önbellek ve merkez kapalı | PASS | İki düğümün dosyaları bozulunca HTTP 200 alınmadı. API dönünce trafik toparlandı. |
| Bir düğüm durunca yeni bağlantı | PASS | gw-local-1 durdurulunca edge ${RECOVERY_MS} ms içinde gw-local-2 üzerinden yanıt verdi. Ortak üretim VIP'i yok. |
| İmza ve çakışma birim testleri | PASS | \`go test ./...\` |
| TLS, yanlış SAN, sertifika yenileme | NOT RUN | Public TLS yok. Sağlayıcı bilinmiyor. |
| NACK, canary, eşzamanlı yazıcı | PASS | Aynı expectedVersion ve iki idempotency anahtarı 200 ve 409 verdi. Canary revision $CANARY_REV yalnız gw-local-1. NACK revision $NACK_REV ACK'i $CANARY_REV bıraktı; disk current bu revision olmadı. Edge kanıtı değildir. |
| WebSocket, SSE, gRPC, büyük yükleme | PASS | 2026-10-23 düğüm 1 ölçümü. Bu koşu protokolleri yeniden ölçmedi. Büyük yükleme ayrıca NOT RUN. |
| CSRF, sahte istemci IP, SSRF uçtan uca | NOT RUN | Loopback ve metadata adresi birim testinde reddedilir. |
| OIDC, kota ve WAF | NOT RUN | Kimlik sağlayıcı ve bu bağımlılıklar yok. |
| Telemetri diski dolu | NOT RUN | Compose log sınırı 10m x 3 tanımlı, doluluk deneyi yapılmadı. |
| PostgreSQL primary ve quorum | NOT RUN | Tek PostgreSQL var. |
| Yük testi ve eklenen gecikme | NOT RUN | Ölçüm yok. |
| Yedekten geri dönüş | NOT RUN | Yedek alınmadı. |
| Lokasyon kaybı | NOT RUN | İkinci lokasyon yok. |

## Protokoller, 2026-10-23

Düğüm 1 üzerinde, yayım 16 ve sonrası. Host portları \`127.0.0.1\` üzerindedir. Bu tablo bu koşuda yeniden ölçülmedi.

| Protokol | Sonuç | Not |
| --- | --- | --- |
| HTTPS | PASS | \`https-1792682249.test.local:18443\`, yerel CA, HTTP/2, \`instance=backend-a\`. |
| HTTP/2 | PASS | Port 10012, prior knowledge. |
| WebSocket | PASS | Port 10013, 101 ve \`instance=backend-a\`. |
| SSE | PASS | Port 10015, \`data: instance=backend-a\`. |
| gRPC | PASS | Port 10014, health \`serving\`. |
| TCP | PASS | Port 10020, \`/healthz\` gövdesi \`ok\`. |
| UDP | PASS | \`envoy:10021\` docker ağında \`pong\`. Hosttaki \`127.0.0.1:10021\` bu Docker Desktop oturumunda yanıt vermedi. |
| TLS geçişi | PASS | Port 18444 sertifikası backend sertifikasıyla aynı. Gövde \`passthrough instance=backend-a\`. |

EOF

echo "doğrulama tamam"
