#!/bin/bash
# Cron-friendly auto-fix: check site, fix if down, verify JS, redeploy if needed
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
LOG_DIR="$SCRIPT_DIR/logs"
LOG_FILE="$LOG_DIR/autofix_$(date +%Y%m%d_%H%M%S).log"
SSH_PASS="ka3iMI4OSNvgFcREuDaQyLguFxuP"
HOST="administrator@93.127.141.100"
SITE="https://openpaths.io"

mkdir -p "$LOG_DIR"
exec > >(tee -a "$LOG_FILE") 2>&1

log() { echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*"; }

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
        log "SITE STILL DOWN after restart - spawning claude agent"
        SITE_OK=false
        cd "$PROJECT_DIR"
        claude --dangerously-skip-permissions -p "
Site $SITE is DOWN even after supervisor restart. Fully diagnose and fix.

Server: $HOST (ssh pass: $SSH_PASS)
Check logs: sshpass -p '$SSH_PASS' ssh -o StrictHostKeyChecking=no $HOST 'tail -100 /var/log/supervisor/openpaths-error.log'
Check status: sshpass -p '$SSH_PASS' ssh -o StrictHostKeyChecking=no $HOST 'sudo supervisorctl status openpaths'
Project: $PROJECT_DIR
Deploy cmd: bash deploy.sh api
Config: config.yaml, .env

Investigate the error logs, fix the root cause, rebuild and redeploy if needed.
Verify the site comes back: curl -s $SITE/v1/models
" >> "$LOG_DIR/claude_fix.log" 2>&1
        sleep 10
        if check_site; then
            log "claude fixed it - site is back up"
            SITE_OK=true
        else
            log "claude could not fix it - manual intervention needed"
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
        log "JS ERRORS detected - spawning claude agent"
        cd "$PROJECT_DIR"
        JS_ERRORS=$(node monitoring/check_js.mjs "$SITE" 2>&1 || true)
        claude --dangerously-skip-permissions -p "
JS errors detected on $SITE homepage:

$JS_ERRORS

Fix the frontend JS errors. The frontend is built with Vite+React in src/.
After fixing, rebuild and redeploy:
  npm run build
  bash deploy.sh site

Verify no JS errors remain.
" >> "$LOG_DIR/claude_fix.log" 2>&1
    else
        log "JS check clean"
    fi
fi

log "done"
