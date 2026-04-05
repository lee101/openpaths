#!/bin/bash
#
# Daily Route Monitor with Auto-Fix
#
# Tests all API routes on local and production. On failure, spawns Claude
# to diagnose, fix, retest locally, redeploy, and retest production.
#
# Usage:
#   ./monitoring/daily_route_monitor.sh              # Test prod + local
#   ./monitoring/daily_route_monitor.sh --prod-only   # Test prod only
#   ./monitoring/daily_route_monitor.sh --local-only  # Test local only
#   ./monitoring/daily_route_monitor.sh --no-fix      # Don't auto-fix
#   ./monitoring/daily_route_monitor.sh --dry-run     # Just show what would run
#
# Cron (daily at 7 AM):
#   0 7 * * * cd /nvme0n1-disk/code/openpaths && bash monitoring/daily_route_monitor.sh >> monitoring/logs/daily.log 2>&1
#
# Systemd timer: see monitoring/systemd/
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
LOG_DIR="$SCRIPT_DIR/logs"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_FILE="$LOG_DIR/daily_routes_$TIMESTAMP.log"

mkdir -p "$LOG_DIR"

AUTO_FIX=true
TEST_LOCAL=true
TEST_PROD=true
DRY_RUN=false

for arg in "$@"; do
    case $arg in
        --no-fix)     AUTO_FIX=false ;;
        --prod-only)  TEST_LOCAL=false ;;
        --local-only) TEST_PROD=false ;;
        --dry-run)    DRY_RUN=true ;;
    esac
done

log() { echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*" | tee -a "$LOG_FILE"; }

# Make sure ANTHROPIC_API_KEY is unset so Claude uses its default auth
unset ANTHROPIC_API_KEY

log "=========================================="
log "Daily Route Monitor"
log "Auto-fix: $AUTO_FIX"
log "Test local: $TEST_LOCAL"
log "Test prod: $TEST_PROD"
log "=========================================="

LOCAL_FAILED=false
PROD_FAILED=false

# --- Phase 1: Test local ---
if $TEST_LOCAL; then
    log "=== Phase 1: Local Route Tests ==="

    # Check if local server is running
    if curl -s --max-time 5 http://localhost:8090/health >/dev/null 2>&1; then
        if $DRY_RUN; then
            log "[DRY RUN] Would run: bash $SCRIPT_DIR/test_routes.sh --local"
        elif bash "$SCRIPT_DIR/test_routes.sh" --local 2>&1 | tee -a "$LOG_FILE"; then
            log "Local tests: PASSED"
        else
            log "Local tests: FAILED"
            LOCAL_FAILED=true
        fi
    else
        log "Local server not running at :8090 — skipping local tests"
    fi
fi

# --- Phase 2: Test production ---
if $TEST_PROD; then
    log "=== Phase 2: Production Route Tests ==="

    if $DRY_RUN; then
        log "[DRY RUN] Would run: bash $SCRIPT_DIR/test_routes.sh"
    elif bash "$SCRIPT_DIR/test_routes.sh" 2>&1 | tee -a "$LOG_FILE"; then
        log "Production tests: PASSED"
    else
        log "Production tests: FAILED"
        PROD_FAILED=true
    fi
fi

# --- Phase 3: Auto-fix if failures ---
if ($LOCAL_FAILED || $PROD_FAILED) && $AUTO_FIX && ! $DRY_RUN; then
    log "=== Phase 3: Auto-Fix with Claude ==="

    FAIL_CONTEXT=""
    if $LOCAL_FAILED; then
        FAIL_CONTEXT="$FAIL_CONTEXT\nLocal test failures detected."
    fi
    if $PROD_FAILED; then
        FAIL_CONTEXT="$FAIL_CONTEXT\nProduction test failures detected."
    fi

    FIX_PROMPT="You are a coding agent. OpenPaths API route tests have failed.
$FAIL_CONTEXT

Recent test log:
$(tail -80 "$LOG_FILE")

Project: $PROJECT_DIR
Server code: internal/server/server.go, internal/handler/
Config: config.yaml

Steps to fix:
1. Read the test failures above and diagnose the root cause
2. Fix the issue in the Go server code
3. Rebuild: cd $PROJECT_DIR && go build -o dist/openpaths-api ./cmd/openpaths/
4. Retest locally: bash monitoring/test_routes.sh --local
5. If local passes, deploy to prod: bash deploy.sh api
6. Retest production: bash monitoring/test_routes.sh
7. Verify all tests pass

Key files:
- internal/server/server.go (routes)
- internal/handler/ (handlers)
- internal/middleware/ (auth, billing)
- internal/billing/ (credit system)
- config.yaml (model config)

Fix the issue and verify everything works end to end."

    log "Spawning Claude auto-fix agent..."
    cd "$PROJECT_DIR"
    if claude --dangerously-skip-permissions -p "$FIX_PROMPT" 2>&1 | tee -a "$LOG_FILE"; then
        log "Auto-fix agent: Completed"

        # Re-run tests after fix
        log "=== Phase 4: Re-testing after fix ==="
        STILL_FAILING=false

        if $TEST_LOCAL; then
            if curl -s --max-time 5 http://localhost:8090/health >/dev/null 2>&1; then
                if ! bash "$SCRIPT_DIR/test_routes.sh" --local 2>&1 | tee -a "$LOG_FILE"; then
                    STILL_FAILING=true
                fi
            fi
        fi

        if $TEST_PROD; then
            if ! bash "$SCRIPT_DIR/test_routes.sh" 2>&1 | tee -a "$LOG_FILE"; then
                STILL_FAILING=true
            fi
        fi

        if $STILL_FAILING; then
            log "FINAL STATUS: FAILED (auto-fix did not resolve all issues)"
        else
            log "FINAL STATUS: PASSED (after auto-fix)"
        fi
    else
        log "Auto-fix agent: Failed to run"
        log "FINAL STATUS: FAILED"
    fi
elif ($LOCAL_FAILED || $PROD_FAILED); then
    log "FINAL STATUS: FAILED (auto-fix disabled)"
else
    log "FINAL STATUS: ALL PASSED"
fi

log "Log: $LOG_FILE"
