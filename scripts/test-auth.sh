#!/usr/bin/env bash
# Test the Excavate auth API end to end.
# Usage:  ./scripts/test-auth.sh
# Auto-starts Postgres/Redis (docker) and the backend if they are not running.
# Exit code 0 = all tests passed, 1 = one or more failed.
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND="$ROOT/backend"
BIN_DIR="$BACKEND/bin"
BASE="http://localhost:8080"
COOKIE="$(mktemp -u /tmp/excavate_cookies_XXXXXX.txt)"

fail() { echo "ERROR: $1" >&2; exit 1; }

command -v curl >/dev/null || fail "curl not found"
command -v docker >/dev/null || fail "docker not found"

# ---- 1. Databases -----------------------------------------------------------
echo "== Starting Postgres + Redis =="
docker compose -f "$ROOT/docker-compose.yml" up -d postgres redis >/dev/null 2>&1 || fail "docker compose up"
ready=0
for i in $(seq 1 30); do
  sleep 2
  count=$(docker compose -f "$ROOT/docker-compose.yml" ps --format '{{.Status}}' 2>/dev/null | grep -c healthy || true)
  [ "$count" -ge 2 ] && { ready=1; break; }
done
[ "$ready" -eq 1 ] || fail "Postgres/Redis did not become healthy in 60s"
echo "Databases healthy."

# ---- 2. Backend (start only if nothing is on :8080) -------------------------
STARTED=0
if curl -sf -o /dev/null "$BASE/api/healthz"; then
  echo "Backend already listening on :8080 - reusing it."
else
  echo "Building backend..."
  (cd "$BACKEND" && mkdir -p "$BIN_DIR" && go build -o "$BIN_DIR/server" ./cmd/server) || fail "backend build failed"
  (cd "$BACKEND" && "$BIN_DIR/server") &
  SERVER_PID=$!
  STARTED=1
  up=0
  for i in $(seq 1 20); do
    sleep 1
    if curl -sf -o /dev/null "$BASE/api/healthz"; then up=1; break; fi
  done
  [ "$up" -eq 1 ] || fail "backend did not start on :8080"
  echo "Backend started (PID $SERVER_PID)."
fi

cleanup() {
  rm -f "$COOKIE" /tmp/excavate_resp.txt
  if [ "$STARTED" -eq 1 ]; then
    kill "$SERVER_PID" 2>/dev/null || true
    rm -rf "$BIN_DIR"
    echo "Stopped backend started by this script."
  fi
}
trap cleanup EXIT

# ---- 3. Helpers -------------------------------------------------------------
PASS=0
FAIL=0
failures=()

http() { # method path body use_cookie
  local method="$1" path="$2" body="${3:-}" use_cookie="${4:-0}"
  local args=(-s -o /tmp/excavate_resp.txt -w "%{http_code}" -X "$method" "$BASE$path")
  [ "$use_cookie" = "1" ] && args+=(-b "$COOKIE")
  [ -n "$body" ] && args+=(-H "Content-Type: application/json" -d "$body")
  local code
  code=$(curl "${args[@]}")
  echo "$code|$(cat /tmp/excavate_resp.txt)"
}

check() { # name expected_code expected_substring actual
  local name="$1" exp_code="$2" exp_sub="$3" result="$4"
  local code="${result%%|*}" body="${result#*|}"
  if [ "$code" = "$exp_code" ] && printf '%s' "$body" | grep -qF "$exp_sub"; then
    PASS=$((PASS+1))
    printf "PASS  %s\n" "$name"
  else
    FAIL=$((FAIL+1))
    printf "FAIL  %s (expected %s %q, got %s %s)\n" "$name" "$exp_code" "$exp_sub" "$code" "$body"
  fi
}

# ---- 4. Run the assertions --------------------------------------------------
EMAIL="script-$$@test.com"
PASSWORD="password123"

r=$(http POST "/api/auth/register" "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
check "register returns 201" "201" "\"email\"" "$r"

curl -s -c "$COOKIE" -o /dev/null -X POST -H "Content-Type: application/json" -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" "$BASE/api/auth/login"
r=$(http GET "/api/me" "" 1)
check "cookie session works (/api/me)" "200" "$EMAIL" "$r"

r=$(http POST "/api/auth/register" "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
check "duplicate email -> 422" "422" "email already registered" "$r"

r=$(http POST "/api/auth/register" "{\"email\":\"x-$$@test.com\",\"password\":\"short\"}")
check "short password -> 422" "422" "password must be at least 8 characters" "$r"

r=$(http POST "/api/auth/login" "{\"email\":\"$EMAIL\",\"password\":\"wrongpass1\"}")
check "wrong password login -> 401" "401" "invalid email or password" "$r"

r=$(http GET "/api/me")
check "no cookie /api/me -> 401" "401" "unauthorized" "$r"

r=$(http POST "/api/auth/login" "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}" 1)
check "login returns 200" "200" "$EMAIL" "$r"

r=$(http POST "/api/auth/logout" "" 1)
if [ "${r%%|*}" = "200" ]; then
  r2=$(http GET "/api/me" "" 1)
  check "logout destroys session (/api/me -> 401)" "401" "unauthorized" "$r2"
else
  check "logout destroys session (/api/me -> 401)" "200" "" "$r"
fi

# ---- 5. Report --------------------------------------------------------------
echo ""
echo "PASSED: $PASS/$((PASS+FAIL))"
[ "$FAIL" -eq 0 ] || exit 1
