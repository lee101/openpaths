#!/bin/bash
# Cron-friendly auto-fix: check site, fix if down, verify JS, redeploy if needed.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
LOG_DIR="$SCRIPT_DIR/logs"
LOG_FILE="$LOG_DIR/autofix_$(date +%Y%m%d_%H%M%S).log"
STATE_DIR="$SCRIPT_DIR/state"
DEBOUNCE_FILE="$STATE_DIR/last_codex_fix_at"
DEBOUNCE_SECONDS="${DEBOUNCE_SECONDS:-43200}"
SSH_PASS="ka3iMI4OSNvgFcREuDaQyLguFxuP"
HOST="administrator@93.127.141.100"
SITE="https://openpaths.io"

mkdir -p "$LOG_DIR" "$STATE_DIR"
exec > >(tee -a "$LOG_FILE") 2>&1

log() { echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*"; }

can_spawn_codex() {
    local now last elapsed remain
    now=$(date -u +%s)
    last=0
    [ -f "$DEBOUNCE_FILE" ] && last=$(tr -d '[:space:]' < "$DEBOUNCE_FILE")
    [ -z "$last" ] && last=0
    elapsed=$((now - last))
    if [ "$last" -gt 0 ] && [ "$elapsed" -lt "$DEBOUNCE_SECONDS" ]; then
        remain=$((DEBOUNCE_SECONDS - elapsed))
        log "Codex spawn debounced (${elapsed}s since last run, ${remain}s remaining)"
        return 1
    fi
    echo "$now" > "$DEBOUNCE_FILE"
    return 0
}

ssh_cmd() {
    sshpass -p "$SSH_PASS" ssh -o StrictHostKeyChecking=no "$HOST" "$@"
}

check_site() {
    local status
    status=$(curl -s -o /dev/null -w '%{http_code}' --max-time 15 "$SITE" 2>/dev/null || echo "000")
    [ "$status" -lt 400 ] && [ "$status" != "000" ]
}

check_api() {
    local status
    status=$(curl -s -o /dev/null -w '%{http_code}' --max-time 15 "$SITE/v1/models" 2>/dev/null || echo "000")
    [ "$status" -lt 400 ] && [ "$status" != "000" ]
}

check_js() {
    cd "$PROJECT_DIR"
    node monitoring/check_js.mjs "$SITE" 2>/dev/null
}

# Phase 1: Site health check
log "checking $SITE..."
SITE_OK=true
if ! check_site; then
    log "SITE DOWN - attempting restart"
    ssh_cmd 'sudo supervisorctl restart openpaths' || true
    sleep 5
    if ! check_site; then
        log "SITE STILL DOWN after restart - spawning Codex agent"
        SITE_OK=false
        cd "$PROJECT_DIR"
        if can_spawn_codex; then
            bash "$SCRIPT_DIR/run_codex_agent.sh" "
Site $SITE is DOWN even after supervisor restart. Fully diagnose and fix.

Server: $HOST (ssh pass: $SSH_PASS)
Check logs: sshpass -p '$SSH_PASS' ssh -o StrictHostKeyChecking=no $HOST 'tail -100 /var/log/supervisor/openpaths-error.log'
Check status: sshpass -p '$SSH_PASS' ssh -o StrictHostKeyChecking=no $HOST 'sudo supervisorctl status openpaths'
Project: $PROJECT_DIR
Deploy cmd: bash deploy.sh api
Config: config.yaml, .env

Investigate the error logs, fix the root cause, rebuild and redeploy if needed.
Verify the site comes back: curl -s $SITE/v1/models
" >> "$LOG_DIR/codex_fix.log" 2>&1
        fi
        sleep 10
        if check_site; then
            log "Codex fixed it - site is back up"
            SITE_OK=true
        else
            log "Codex could not fix it - manual intervention needed"
        fi
    else
        log "site recovered after restart"
    fi
else
    log "site UP"
fi

# Phase 2: API health check
if $SITE_OK; then
    if ! check_api; then
        log "API /v1/models not responding"
    else
        log "API OK"
    fi
fi

# Phase 3: JS error check
if $SITE_OK; then
    log "checking for JS errors..."
    if ! check_js; then
        log "JS ERRORS detected - spawning Codex agent"
        cd "$PROJECT_DIR"
        JS_ERRORS=$(node monitoring/check_js.mjs "$SITE" 2>&1 || true)
        if can_spawn_codex; then
            bash "$SCRIPT_DIR/run_codex_agent.sh" "
JS errors detected on $SITE homepage:

$JS_ERRORS

Fix the frontend JS errors. The frontend is built with Vite+React in src/.
After fixing, rebuild and redeploy:
  npm run build
  bash deploy.sh site

Verify no JS errors remain.
" >> "$LOG_DIR/codex_fix.log" 2>&1
        fi
    else
        log "JS check clean"
    fi
fi

log "done"
