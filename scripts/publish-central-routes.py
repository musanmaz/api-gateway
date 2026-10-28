#!/usr/bin/env python3
"""Activate central-gateway route compilation without logging operator credentials."""
import json
import time
import uuid
import urllib.request
from pathlib import Path

root = Path("/opt/api-gateway")
bearer = (root / ".local/operator.token").read_text().strip()
base = "http://127.0.0.1:9080"
def request(path, payload=None):
    headers = {"Authorization": "Bearer " + bearer, "Content-Type": "application/json"}
    if payload is not None: headers["Idempotency-Key"] = str(uuid.uuid4())
    data = json.dumps(payload).encode() if payload is not None else None
    with urllib.request.urlopen(urllib.request.Request(base + path, headers=headers, data=data), timeout=12) as response:
        return json.load(response)

page = request("/api/v1/services?limit=100")
items = page.get("items", [])
names = sorted(s["name"] for s in items if s["status"] == "active")
if names != ["catalog-api", "orders-api"]:
    raise SystemExit("Unexpected active services: " + ", ".join(names) + ". Review before public exposure.")
service = next(s for s in items if s["name"] == "catalog-api")
result = request("/api/v1/services/" + service["id"] + "/publish", {"expectedVersion": service["version"]})
revision = result["revision"]["id"]
print("Published central route config revision:", revision)
for i in range(45):
    overview = request("/api/v1/overview")
    nodes = overview["nodes"]
    if len(nodes) == 2 and all(n.get("heartbeat") == "fresh" and n.get("ackedRevision") == n.get("desiredRevision") == revision for n in nodes):
        print("All two Envoy nodes acknowledged revision:", revision)
        break
    time.sleep(1)
else:
    raise SystemExit("Node ACKs not converged; inspect /api/v1/overview before Caddy reload")
