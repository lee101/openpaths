#!/usr/bin/env bash
# pricing_guard.sh - recurring check of the openpaths pricing rule:
#   what we bill for every model >= what the upstream provider charges us, and
#   what the site advertises == what we bill.
#
# It runs `go run ./cmd/pricing-guard -live` against the checkout that serves
# prod (its config.yaml is what actually bills), which re-reads upstream prices
# from public endpoints (OpenRouter's catalog and per-provider endpoint
# listings; no API key needed), writes a markdown report, and:
#   - clean            -> report only (email too with PRICING_GUARD_EMAIL_ON_CLEAN=1)
#   - critical finding -> emails the table to leepenkman@gmail.com via SES SMTP
#                         (notify.sh), then with PRICING_GUARD_AUTOFIX=1 runs
#                         agent_fix.sh (codex -> claude -> op/DeepSeek), which
#                         raises prices on a branch and opens a PR; never deploys
#   - guard error      -> emails the error
#
# Schedule: systemd timer fires weekly (Persistent=true catches missed runs);
# this script skips unless PRICING_GUARD_MIN_DAYS (20) have passed since the
# last completed run, so it effectively runs every 3 weeks. --force skips that.
#
# Install on prod (leaf-gpu, user administrator), once cmd/pricing-guard is
# merged and pulled into /nvme0n1-disk/code/openpaths:
#   cd /nvme0n1-disk/code/openpaths
#   sudo install -m 644 scripts/pricing-guard/systemd/openpaths-pricing-guard.service \
#                       scripts/pricing-guard/systemd/openpaths-pricing-guard.timer /etc/systemd/system/
#   sudo systemctl daemon-reload
#   sudo systemctl enable --now openpaths-pricing-guard.timer
#   systemctl list-timers openpaths-pricing-guard.timer
#   # first run now (dry: no email, no agent):
#   PRICING_GUARD_DRY_RUN=1 scripts/pricing-guard/pricing_guard.sh --force
#   # real run through systemd:
#   sudo systemctl start openpaths-pricing-guard.service; journalctl -u openpaths-pricing-guard -n 50
#
# Env knobs: PRICING_GUARD_REPO (checkout whose config/site files are checked,
# default /nvme0n1-disk/code/openpaths), PRICING_GUARD_STATE_DIR (default
# ~/.local/state/openpaths-pricing-guard), PRICING_GUARD_MIN_DAYS (20),
# PRICING_GUARD_AUTOFIX (0|1, default 1), PRICING_GUARD_DRY_RUN (1: no email,
# no agent), PRICING_GUARD_EMAIL_ON_CLEAN (0|1), ALERT_EMAIL, ALERT_FROM,
# ALERT_ENV_FILES (see notify.sh; SMTP creds come from env files, never printed).
set -uo pipefail

SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO="${PRICING_GUARD_REPO:-/nvme0n1-disk/code/openpaths}"
STATE_DIR="${PRICING_GUARD_STATE_DIR:-$HOME/.local/state/openpaths-pricing-guard}"
MIN_DAYS="${PRICING_GUARD_MIN_DAYS:-20}"
AUTOFIX="${PRICING_GUARD_AUTOFIX:-1}"
DRY_RUN="${PRICING_GUARD_DRY_RUN:-0}"
export PATH="$HOME/.local/monitor-bin:$HOME/.local/bin:/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin:$PATH"

FORCE=0
[ "${1:-}" = "--force" ] && FORCE=1

mkdir -p "$STATE_DIR"
LOG="$STATE_DIR/guard.log"
log() { echo "[$(date -u +%FT%TZ)] $*" | tee -a "$LOG" >&2; }

LAST="$STATE_DIR/last_run"
if [ "$FORCE" = 0 ] && [ -f "$LAST" ]; then
    age_days=$(( ($(date +%s) - $(cat "$LAST")) / 86400 ))
    if [ "$age_days" -lt "$MIN_DAYS" ]; then
        log "last run ${age_days}d ago (< ${MIN_DAYS}d); skipping"
        exit 0
    fi
fi

exec 9>"$STATE_DIR/lock"
if command -v flock >/dev/null; then
    flock -n 9 || { log "another run holds the lock"; exit 0; }
fi

TS=$(date -u +%Y%m%dT%H%M%SZ)
REPORT="$STATE_DIR/report-$TS.md"
ERR="$STATE_DIR/error-$TS.log"
ln -sfn "$REPORT" "$STATE_DIR/latest.md"

notify() { # notify SUBJECT < body
    if [ "$DRY_RUN" = 1 ]; then log "dry run: would email: $1"; cat >/dev/null; return 0; fi
    "$SELF_DIR/notify.sh" "$1" - >>"$LOG" 2>&1 && log "emailed: $1" || log "email FAILED: $1"
}

log "checking $REPO (report $REPORT)"
# The guard binary is built from this script's checkout so a newer guard can
# check an older prod tree; the checked files come from $REPO.
# Build first: `go run` reports any non-zero program exit as 1, which would turn a
# guard failure (2) into "findings" (1) and email an empty table.
GUARD_BIN="$STATE_DIR/pricing-guard"
LEDGER="$REPO/scripts/pricing-guard/upstream-costs.json"
[ -f "$LEDGER" ] || LEDGER="$SELF_DIR/upstream-costs.json"   # prod tree may predate the ledger
if (cd "$SELF_DIR/../.." && go build -o "$GUARD_BIN" ./cmd/pricing-guard) 2>"$ERR"; then
    "$GUARD_BIN" -repo "$REPO" -ledger "$LEDGER" -live -out "$REPORT" 2>>"$ERR"
    rc=$?
else
    rc=2
fi
# A failed check is retried at the next weekly tick instead of in 3 weeks.
[ "$rc" -le 1 ] && date +%s > "$LAST"

case $rc in
0)
    log "clean"
    rm -f "$ERR"
    if [ "${PRICING_GUARD_EMAIL_ON_CLEAN:-0}" = 1 ]; then
        notify "[OK] openpaths pricing guard: all models at or above cost" < "$REPORT"
    fi
    ;;
1)
    crit=$(grep -c '^| CRITICAL ' "$REPORT")
    log "$crit critical finding(s)"
    {
        echo "The openpaths pricing guard found $crit model price(s) that break the rule"
        echo "\"bill >= upstream cost, and advertise exactly what we bill\"."
        echo
        echo "Checked tree: $REPO   Host: $(hostname)   Report: $REPORT"
        if [ "$AUTOFIX" = 1 ] && [ "$DRY_RUN" != 1 ]; then
            echo "An autofix agent is starting; it opens a PR (no deploy) and emails the result."
        else
            echo "Autofix is off (PRICING_GUARD_AUTOFIX=$AUTOFIX); fix by hand following the brief in scripts/pricing-guard/agent_fix.sh."
        fi
        echo
        echo "Critical findings (prices per 1M tokens; 'floor' = upstream cost incl. fees):"
        echo
        grep -E '^\| (Severity|---|CRITICAL) ' "$REPORT"
        echo
        echo "== Full report"
        cat "$REPORT"
    } | notify "[PRICING] openpaths: $crit model price(s) below cost or mismatched"
    if [ "$AUTOFIX" = 1 ] && [ "$DRY_RUN" != 1 ]; then
        log "starting autofix agent"
        "$SELF_DIR/agent_fix.sh" "$REPORT" >>"$LOG" 2>&1 || log "agent_fix exited rc=$?"
    fi
    ;;
*)
    log "guard failed rc=$rc"
    { echo "pricing-guard failed (rc=$rc) on $(hostname) checking $REPO."; echo; tail -n 40 "$ERR"; } \
        | grep -viE '(api[_-]?key|token|secret|passw)[^ ]*[=:] *[^ ]{8,}' \
        | notify "[ERROR] openpaths pricing guard could not run"
    ;;
esac
exit 0
