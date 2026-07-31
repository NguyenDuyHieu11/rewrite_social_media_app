#!/usr/bin/env bash
# Smoke: login → create post → SSE subscribe → comment (+ optional reaction).
# Requires: dispatcher :8080, gateway :8081, migrations applied.
set -euo pipefail

DISP="${DISP:-http://localhost:8080}"
GATE="${GATE:-http://localhost:8081}"
USER="smoke_$(date +%s)"
PASS="smoke-pass-12345"

echo "== register =="
curl -fsS -X POST "$DISP/auth/register" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"$USER\",\"email\":\"$USER@example.com\",\"password\":\"$PASS\"}" >/dev/null

echo "== login =="
LOGIN=$(curl -fsS -X POST "$DISP/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"$USER\",\"password\":\"$PASS\"}")
TOKEN=$(printf '%s' "$LOGIN" | python3 -c 'import sys,json; print(json.load(sys.stdin)["access_token"])')

echo "== create post =="
POST=$(curl -fsS -X POST "$DISP/posts/" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"body":"smoke post"}')
POST_ID=$(printf '%s' "$POST" | python3 -c 'import sys,json; print(json.load(sys.stdin)["id"])')
echo "post_id=$POST_ID"

SSE_OUT=$(mktemp)
trap 'rm -f "$SSE_OUT"; kill "$SSE_PID" 2>/dev/null || true' EXIT

echo "== open SSE =="
curl -fsS -N -X PUT \
  "$GATE/sse/posts/$POST_ID/subscriptions?access_token=$TOKEN" \
  -H 'Accept: text/event-stream' >"$SSE_OUT" &
SSE_PID=$!
sleep 1

echo "== create comment =="
curl -fsS -X POST "$DISP/posts/$POST_ID/comments" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"body":"smoke comment"}' >/dev/null

echo "== upsert reaction =="
curl -fsS -X PUT "$DISP/posts/$POST_ID/reactions" \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"type":"like"}' >/dev/null

sleep 1
echo "== SSE buffer =="
cat "$SSE_OUT"
echo

grep -q 'event: comment' "$SSE_OUT" || { echo "FAIL: missing comment event"; exit 1; }
grep -q 'event: reaction' "$SSE_OUT" || { echo "FAIL: missing reaction event"; exit 1; }
echo "OK: comment + reaction delivered over SSE"
