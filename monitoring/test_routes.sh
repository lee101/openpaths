#!/bin/bash
#
# OpenPaths Route Tester
#
# Tests all major API routes against local or production server.
# Uses a real API key to verify auth, billing, models, chat, and account routes.
#
# Usage:
#   ./monitoring/test_routes.sh                    # Test production
#   ./monitoring/test_routes.sh --local            # Test local (localhost:8090)
#   ./monitoring/test_routes.sh --base=URL          # Test custom base URL
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
LOG_DIR="$SCRIPT_DIR/logs"
mkdir -p "$LOG_DIR"

API_KEY="op-00464b407889e976e971f6036843bb72218904f0f50b2393fae4ee15fab07326"
BASE="https://openpaths.io"
CHAT_MODEL="gemini-flash-lite"
VERBOSE=false

for arg in "$@"; do
    case $arg in
        --local) BASE="http://localhost:8090" ;;
        --base=*) BASE="${arg#*=}" ;;
        --verbose) VERBOSE=true ;;
    esac
done

PASSED=0
FAILED=0
ERRORS=""

log() { echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*"; }

pass() {
    log "PASS: $1"
    ((PASSED++)) || true
}

fail() {
    log "FAIL: $1 — $2"
    ERRORS="$ERRORS\n  FAIL: $1 — $2"
    ((FAILED++)) || true
}

# Expect HTTP status in range
check_status() {
    local name="$1" method="$2" url="$3" expect_status="$4"
    shift 4
    local status body
    body=$(curl -s -w '\n%{http_code}' --max-time 30 "$@" -X "$method" "$url" 2>/dev/null)
    status=$(echo "$body" | tail -1)
    body=$(echo "$body" | head -n -1)

    if [ "$status" = "$expect_status" ]; then
        pass "$name (HTTP $status)"
        if $VERBOSE; then echo "  $body" | head -c 200; echo; fi
    else
        fail "$name" "expected $expect_status, got $status: $(echo "$body" | head -c 200)"
    fi
}

# Expect JSON field in response
check_json_field() {
    local name="$1" method="$2" url="$3" field="$4"
    shift 4
    local body status
    body=$(curl -s -w '\n%{http_code}' --max-time 30 "$@" -X "$method" "$url" 2>/dev/null)
    status=$(echo "$body" | tail -1)
    body=$(echo "$body" | head -n -1)

    if [ "$status" -ge 200 ] && [ "$status" -lt 400 ]; then
        if echo "$body" | python3 -c "import sys,json; d=json.load(sys.stdin); assert '$field' in d or '$field' in str(d)" 2>/dev/null; then
            pass "$name"
            if $VERBOSE; then echo "  $body" | head -c 200; echo; fi
        else
            fail "$name" "field '$field' not found in response: $(echo "$body" | head -c 200)"
        fi
    else
        fail "$name" "HTTP $status: $(echo "$body" | head -c 200)"
    fi
}

log "=========================================="
log "OpenPaths Route Tests"
log "Base: $BASE"
log "=========================================="

# --- Public routes ---
log "--- Public Routes ---"
check_json_field "GET /health" GET "$BASE/health" "status"
check_status "GET /stats/models" GET "$BASE/stats/models" 200
check_status "GET /stats/providers" GET "$BASE/stats/providers" 200

# --- Auth-required: Account routes ---
log "--- Account Routes ---"
check_json_field "GET /account/balance" GET "$BASE/account/balance" "balance_usd" \
    -H "Authorization: Bearer $API_KEY"

check_json_field "GET /account/keys" GET "$BASE/account/keys" "keys" \
    -H "Authorization: Bearer $API_KEY"

check_json_field "GET /account/transactions" GET "$BASE/account/transactions" "transactions" \
    -H "Authorization: Bearer $API_KEY"

# --- Auth: 401 for bad key ---
log "--- Auth Rejection ---"
check_status "401 for invalid key" GET "$BASE/account/balance" 401 \
    -H "Authorization: Bearer op-invalidkey"

check_status "401 for no auth" GET "$BASE/account/balance" 401

# --- Models routes ---
log "--- Model Routes ---"
check_json_field "GET /v1/models" GET "$BASE/v1/models" "data" \
    -H "Authorization: Bearer $API_KEY"

check_json_field "GET /v1/models/deepseek-chat" GET "$BASE/v1/models/deepseek-chat" "id" \
    -H "Authorization: Bearer $API_KEY"

# --- Chat completions ---
log "--- Chat Completions ---"
CHAT_BODY='{"model":"'"$CHAT_MODEL"'","messages":[{"role":"user","content":"Say ok"}],"max_tokens":5}'
check_json_field "POST /v1/chat/completions" POST "$BASE/v1/chat/completions" "choices" \
    -H "Authorization: Bearer $API_KEY" \
    -H "Content-Type: application/json" \
    -d "$CHAT_BODY"

# --- Anthropic messages ---
log "--- Anthropic Messages ---"
MSG_BODY='{"model":"'"$CHAT_MODEL"'","messages":[{"role":"user","content":"Say ok"}],"max_tokens":5}'
check_json_field "POST /v1/messages" POST "$BASE/v1/messages" "content" \
    -H "Authorization: Bearer $API_KEY" \
    -H "Content-Type: application/json" \
    -H "anthropic-version: 2023-06-01" \
    -d "$MSG_BODY"

# --- Balance after usage (should have been deducted) ---
log "--- Post-usage Balance ---"
BALANCE=$(curl -s "$BASE/account/balance" -H "Authorization: Bearer $API_KEY" 2>/dev/null)
BAL_USD=$(echo "$BALANCE" | python3 -c "import sys,json; print(json.load(sys.stdin).get('balance_usd',0))" 2>/dev/null || echo "0")
if python3 -c "assert float('$BAL_USD') > 0" 2>/dev/null; then
    pass "Balance still positive (\$$BAL_USD)"
else
    fail "Balance check" "Balance is $BAL_USD — may need more credits"
fi

# --- Summary ---
log "=========================================="
log "RESULTS: $PASSED passed, $FAILED failed"
if [ $FAILED -gt 0 ]; then
    log "Failures:$ERRORS"
fi
log "=========================================="

exit $FAILED
