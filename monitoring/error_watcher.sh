#!/bin/bash
# error_watcher.sh — main monitoring entrypoint.
#
# Captures fresh frontend (JS scan + browser-reported) + backend errors, and if
# anything new is found AND we haven't already done so in the last 12h,
# spawns Codex to diagnose, fix, redeploy, and verify.
#
# Flags:
#   --check-only   capture errors, print summary, do not spawn Codex
#   --force        ignore the 12h debounce
#   --dry-run      no Codex spawn, no debounce state write
#
# Env overrides:
#   OPENPATHS_SITE      default https://openpaths.io
#   OPENPATHS_HOST      default administrator@93.127.141.100
#   OPENPATHS_SSH_PASS  default baked-in
#   DEBOUNCE_SECONDS    default 43200 (12h)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
LOG_DIR="$SCRIPT_DIR/logs"
STATE_DIR="$SCRIPT_DIR/state"
ERR_DIR="$SCRIPT_DIR/errors"
mkdir -p "$LOG_DIR" "$STATE_DIR" "$ERR_DIR"

TS=$(date -u +%Y%m%dT%H%M%SZ)
LOG_FILE="$LOG_DIR/watcher_$TS.log"
DEBOUNCE_FILE="$STATE_DIR/last_codex_fix_at"
DEBOUNCE_SECONDS="${DEBOUNCE_SECONDS:-43200}"
SITE="${OPENPATHS_SITE:-https://openpaths.io}"
HOST="${OPENPATHS_HOST:-administrator@93.127.141.100}"
SSH_PASS="${OPENPATHS_SSH_PASS:-ka3iMI4OSNvgFcREuDaQyLguFxuP}"

CHECK_ONLY=false
FORCE=false
DRY_RUN=false
for a in "$@"; do
    case "$a" in
        --check-only) CHECK_ONLY=true ;;
        --force)      FORCE=true ;;
        --dry-run)    DRY_RUN=true ;;
    esac
done

exec > >(tee -a "$LOG_FILE") 2>&1
log() { echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*"; }

log "=== error_watcher start (check_only=$CHECK_ONLY force=$FORCE dry_run=$DRY_RUN) ==="

# --- Frontend route scan ---
FE_JSON_PATH="$ERR_DIR/frontend_$TS.json"
FE_HAS_ERRORS=false
if command -v node >/dev/null 2>&1; then
    log "frontend: scanning $SITE routes via check_js.mjs"
    if node "$SCRIPT_DIR/check_js.mjs" "$SITE" --json > "$FE_JSON_PATH" 2>>"$LOG_FILE"; then
        if grep -q '"hasErrors": true' "$FE_JSON_PATH"; then
            FE_HAS_ERRORS=true
            log "frontend: errors detected -> $FE_JSON_PATH"
        else
            log "frontend: clean"
            rm -f "$FE_JSON_PATH"
        fi
    else
        log "frontend: check_js.mjs failed to run"
        rm -f "$FE_JSON_PATH"
    fi
else
    log "frontend: node not installed, skipping JS scan"
fi

# --- Frontend browser-reported errors ---
FE_CLIENT_JSON_PATH=""
FE_CLIENT_HAS_ERRORS=false
FE_CLIENT_CAPTURE_FAILED=false
log "frontend-client: tailing browser error log on $HOST"
set +e
FE_CLIENT_OUT=$(OPENPATHS_HOST="$HOST" OPENPATHS_SSH_PASS="$SSH_PASS" bash "$SCRIPT_DIR/capture_frontend_client_errors.sh")
FE_CLIENT_RC=$?
set -e
if [ $FE_CLIENT_RC -eq 1 ] && [ -n "$FE_CLIENT_OUT" ]; then
    FE_CLIENT_HAS_ERRORS=true
    FE_CLIENT_JSON_PATH="$FE_CLIENT_OUT"
    log "frontend-client: errors detected -> $FE_CLIENT_JSON_PATH"
elif [ $FE_CLIENT_RC -eq 0 ]; then
    log "frontend-client: clean"
else
    FE_CLIENT_CAPTURE_FAILED=true
    log "frontend-client: capture script failed rc=$FE_CLIENT_RC out=$FE_CLIENT_OUT"
fi

# --- Backend scan ---
BE_JSON_PATH=""
BE_HAS_ERRORS=false
BE_CAPTURE_FAILED=false
log "backend: tailing supervisor log on $HOST"
set +e
BE_OUT=$(OPENPATHS_HOST="$HOST" OPENPATHS_SSH_PASS="$SSH_PASS" bash "$SCRIPT_DIR/capture_backend_errors.sh")
BE_RC=$?
set -e
if [ $BE_RC -eq 1 ] && [ -n "$BE_OUT" ]; then
    BE_HAS_ERRORS=true
    BE_JSON_PATH="$BE_OUT"
    log "backend: errors detected -> $BE_JSON_PATH"
elif [ $BE_RC -eq 0 ]; then
    log "backend: clean"
else
    BE_CAPTURE_FAILED=true
    log "backend: capture script failed rc=$BE_RC out=$BE_OUT"
fi

# --- Decide ---
if ! $FE_HAS_ERRORS && ! $FE_CLIENT_HAS_ERRORS && ! $FE_CLIENT_CAPTURE_FAILED && ! $BE_HAS_ERRORS && ! $BE_CAPTURE_FAILED; then
    log "no errors detected, exiting clean"
    exit 0
fi

if $CHECK_ONLY; then
    log "check-only mode: errors present (frontend_scan=$FE_HAS_ERRORS frontend_client=$FE_CLIENT_HAS_ERRORS frontend_client_capture_failed=$FE_CLIENT_CAPTURE_FAILED backend=$BE_HAS_ERRORS backend_capture_failed=$BE_CAPTURE_FAILED)"
    exit 1
fi

# --- Debounce ---
NOW=$(date -u +%s)
LAST=0
[ -f "$DEBOUNCE_FILE" ] && LAST=$(cat "$DEBOUNCE_FILE" | tr -d '[:space:]')
[ -z "$LAST" ] && LAST=0
ELAPSED=$((NOW - LAST))

if ! $FORCE && [ "$LAST" -gt 0 ] && [ "$ELAPSED" -lt "$DEBOUNCE_SECONDS" ]; then
    REMAIN=$((DEBOUNCE_SECONDS - ELAPSED))
    log "debounced: last Codex fix was ${ELAPSED}s ago (window ${DEBOUNCE_SECONDS}s, ${REMAIN}s remaining). skipping spawn."
    exit 0
fi

if $DRY_RUN; then
    log "dry-run: would spawn Codex now"
    exit 0
fi

# --- Build prompt with attached error JSON ---
FE_BLOCK=""
FE_CLIENT_BLOCK=""
BE_BLOCK=""
if $FE_HAS_ERRORS; then
    FE_BLOCK=$'\n=== FRONTEND ERRORS ('"$FE_JSON_PATH"$') ===\n'"$(cat "$FE_JSON_PATH")"
fi
if $FE_CLIENT_HAS_ERRORS; then
    FE_CLIENT_BLOCK=$'\n=== FRONTEND CLIENT ERRORS ('"$FE_CLIENT_JSON_PATH"$') ===\n'"$(cat "$FE_CLIENT_JSON_PATH")"
fi
if $BE_HAS_ERRORS; then
    BE_BLOCK=$'\n=== BACKEND ERRORS ('"$BE_JSON_PATH"$') ===\n'"$(cat "$BE_JSON_PATH")"
fi
if $FE_CLIENT_CAPTURE_FAILED || $BE_CAPTURE_FAILED; then
    BE_BLOCK="$BE_BLOCK
=== MONITORING CAPTURE FAILURE ===
frontend_client_capture_failed=$FE_CLIENT_CAPTURE_FAILED
backend_capture_failed=$BE_CAPTURE_FAILED
Host: $HOST
Check SSH credentials and log paths before treating production logs as clean."
fi

CODEX_LOG="$LOG_DIR/codex_fix_$TS.log"

PROMPT="You are an autonomous coding agent for the OpenPaths project.

The error_watcher just detected fresh production errors. Diagnose root cause,
fix the code, rebuild, redeploy, and verify the fix took.

Project: $PROJECT_DIR
Site: $SITE
Server: $HOST (sshpass: $SSH_PASS)

Useful commands:
  - Site logs:     sshpass -p '$SSH_PASS' ssh -o StrictHostKeyChecking=no $HOST 'tail -200 /var/log/supervisor/openpaths-error.log'
  - Supervisor:    sshpass -p '$SSH_PASS' ssh -o StrictHostKeyChecking=no $HOST 'sudo supervisorctl status openpaths'
  - Restart api:   sshpass -p '$SSH_PASS' ssh -o StrictHostKeyChecking=no $HOST 'sudo supervisorctl restart openpaths'
  - Local test:    bash monitoring/test_routes.sh --local
  - Prod test:     bash monitoring/test_routes.sh
  - Build api:     go build -o dist/openpaths-api ./cmd/openpaths/
  - Build site:    npm run build
  - Deploy api:    bash deploy.sh api
  - Deploy site:   bash deploy.sh site
  - Re-verify:     bash monitoring/error_watcher.sh --check-only

Errors detected:
$FE_BLOCK
$FE_CLIENT_BLOCK
$BE_BLOCK

Your task:
1. Read the JSON error files above to understand what's broken.
2. Investigate the relevant code (Go server in internal/, React site in src/).
3. Fix the root cause — do not just paper over symptoms.
4. Rebuild and redeploy whichever side(s) changed (api vs site).
5. Run 'bash monitoring/error_watcher.sh --check-only' and confirm it exits clean.
6. Stop only when verification passes.

Do not edit monitoring/state/ or monitoring/errors/."

log "spawning Codex auto-fix agent; output -> $CODEX_LOG"
echo "$NOW" > "$DEBOUNCE_FILE"

cd "$PROJECT_DIR"
if bash "$SCRIPT_DIR/run_codex_agent.sh" "$PROMPT" >> "$CODEX_LOG" 2>&1; then
    log "Codex agent finished"
else
    log "Codex agent exited non-zero (see $CODEX_LOG)"
fi

# Re-verify
log "post-fix re-scan"
if bash "$SCRIPT_DIR/error_watcher.sh" --check-only; then
    log "post-fix scan clean"
else
    log "post-fix scan still flagged errors (see latest $ERR_DIR/*)"
fi
