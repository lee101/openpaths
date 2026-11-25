#!/usr/bin/env bash
# agent_fix.sh REPORT - autofix agent for pricing-guard findings.
# Started by pricing_guard.sh when the live check finds a model billed below
# upstream cost or advertised at a different price than billed.
# Agent chain: codex (ChatGPT plan, via ~/.local/monitor-bin shim) -> claude -> op + DeepSeek.
# The agent works in a fresh git worktree of origin/main, raises prices on a
# branch and opens a PR. It never deploys and never touches the prod checkout.
# Afterwards this script re-runs the guard on the branch and emails the result.
set -uo pipefail

REPORT_IN="${1:?usage: agent_fix.sh REPORT}"
PROD_DIR="${PRICING_GUARD_REPO:-/nvme0n1-disk/code/openpaths}"
STATE_DIR="${PRICING_GUARD_STATE_DIR:-$HOME/.local/state/openpaths-pricing-guard}"
AGENT_TIMEOUT="${PRICING_GUARD_AGENT_TIMEOUT:-5400}"
CODEX_MODEL="${PRICING_GUARD_CODEX_MODEL:-gpt-6.1-terra}"
OP_BIN="${OP_BIN:-$HOME/.local/bin/op}"
FALLBACK_MODEL="${PRICING_GUARD_FALLBACK_MODEL:-deepseek-v4-flash}"
SELF_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export PATH="$HOME/.local/monitor-bin:$HOME/.local/bin:$HOME/.bun/bin:/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin:$PATH"

TS=$(date -u +%Y%m%dT%H%M%SZ)
BRANCH="pricing-guard/autofix-$TS"
WORK="$STATE_DIR/worktree-$TS"
AGENT_REPORT="$STATE_DIR/agent-report-$TS.md"
TRANSCRIPT="$STATE_DIR/agent-transcript-$TS.log"
LOG="$STATE_DIR/guard.log"
log() { echo "[$(date -u +%FT%TZ)] agent: $*" >> "$LOG"; }

git -C "$PROD_DIR" fetch -q origin main || log "git fetch failed"
if ! git -C "$PROD_DIR" worktree add -q -b "$BRANCH" "$WORK" origin/main; then
    log "could not create worktree $WORK"
    exit 1
fi

read -r -d '' PROMPT <<EOF
You are fixing openpaths.io model pricing. Owner rule: openpaths always charges at cost or above
(what we bill a user for a model >= what the upstream provider charges us, including fees), and the
prices the site advertises must equal what we actually bill. A scheduled check found violations.
Work end to end without asking; nobody is watching.

Where things live (you are in a fresh git worktree of origin/main on branch $BRANCH: $WORK)
- config.yaml: what the gateway bills. Per-1M-token fields input_price_per_1m,
  input_cache_hit_price_per_1m, output_price_per_1m, *_long + long_context_threshold,
  scheduled_token_pricing (off-peak/peak tariffs). Cost follows provider + provider_model_id.
- Upstream costs: "# openrouter: in=.. out=.. cache=.. per 1M, +5.5% fee" comments above OpenRouter
  entries (we must bill >= rate * 1.055), and scripts/pricing-guard/upstream-costs.json for direct
  providers (key "<provider>/<provider_model_id>", no fee).
- Advertised: src/data/models.ts (priceInput/priceOutput; for a scheduled model the off-peak rate in
  force), src/data/pricingConfig.generated.json (regenerate with: python3 scripts/export-pricing.py,
  needs PyYAML), FREE_MODEL_IDS in src/data/capabilities.ts (exactly the \$0 models).
- Tools: \`go run ./cmd/pricing-guard -live\` (the check that fired; exit 1 = violations),
  \`go run ./cmd/pricing-guard -live -update-ledger\` (refresh recorded direct costs),
  \`make sync-openrouter\` (re-syncs OpenRouter entries + site data from the live catalog; it only
  raises hand-written prices to rate+fee and regenerates the generated block).

What to do
1. Read the report below, then re-run \`go run ./cmd/pricing-guard -live\` here to see what still applies
   on origin/main (prod's checkout may carry uncommitted config the report was taken from).
2. For OpenRouter models run \`make sync-openrouter\` first. If it refuses (e.g. a paid id withdrawn from
   the catalog), remove that id from scripts/openrouter-sync.json "paid" and re-run.
3. For every remaining BELOW_COST: raise the billed price to at least the floor shown (never lower any
   price below cost). For FREE_ROUTE_PAID: price the route at rate+fee or repoint it to a :free slug.
   Update the advertised price to match the new billed price everywhere.
4. For every AD_MISMATCH: if billed >= cost, change the advertisement to the billed price; if billed
   is below cost, raise billed and advertisement together.
5. Refresh the ledger (-update-ledger), regenerate pricingConfig.generated.json, and update any test
   that pins an old price (internal/config/*_test.go) with a comment naming the source.
6. Verify: \`go run ./cmd/pricing-guard -live\` exits 0, then \`make test && make test-frontend\` pass.
7. Commit only your changes on $BRANCH (message explains each price change: old -> new, upstream cost),
   \`git push -u origin $BRANCH\`, and open a PR with \`gh pr create --base main\` whose body has a table
   of every price changed (model, field, old, new, upstream cost, source). If prod's uncommitted
   config.yaml at $PROD_DIR also has the problem, say so in the PR body (do not edit prod).

Rules
- Do NOT deploy, restart services, run deploy.sh, or touch $PROD_DIR (read-only for you).
- Never push to main, never force-push. No secrets in commits, PR text, or the report.
- Prices only move up to reach cost or to make the site match billing; do not make other changes.

Report: create $AGENT_REPORT immediately and keep it updated. First line exactly \`STATUS: PR OPENED\`
or \`STATUS: NOT FIXED\`. Sections: Summary, Price changes (table), Tests, PR URL, Follow-ups.

== Pricing guard report
$(cat "$REPORT_IN")
EOF

run_codex() {
    command -v codex >/dev/null || return 127
    printf '%s' "$PROMPT" | timeout --kill-after=60 "$AGENT_TIMEOUT" codex exec \
        --dangerously-bypass-approvals-and-sandbox -C "$WORK" -m "$CODEX_MODEL" \
        --config model_reasoning_effort=high
}
run_claude() {
    command -v claude >/dev/null || return 127
    (cd "$WORK" && timeout --kill-after=60 "$AGENT_TIMEOUT" claude -p "$PROMPT" --model sonnet \
        --effort high --permission-mode bypassPermissions --output-format text)
}
run_op() {
    [ -x "$OP_BIN" ] || return 127
    if [ -z "${DEEPSEEK_API_KEY:-}" ]; then
        local f v
        for f in "$HOME/.secretbashrc" "$PROD_DIR/.env" /nvme0n1-disk/code/netwrck/.env; do
            v=$(grep -m1 -E '^(export )?DEEPSEEK_API_KEY=' "$f" 2>/dev/null | cut -d= -f2- | tr -d "\"'")
            [ -n "$v" ] && { export DEEPSEEK_API_KEY="$v"; break; }
        done
    fi
    [ -n "${DEEPSEEK_API_KEY:-}" ] || return 127
    local out; out=$(mktemp)
    (cd "$WORK" && printf '%s' "$PROMPT" | timeout --kill-after=60 "$AGENT_TIMEOUT" "$OP_BIN" -p \
        --no-session --auto-approve --model "$FALLBACK_MODEL" --thinking high) | tee "$out"
    local rc=${PIPESTATUS[0]}
    [ "$rc" -eq 0 ] && [ -s "$out" ]; local ok=$?   # op can exit 0 after an auth failure
    rm -f "$out"; return $ok
}

pr_url() { (cd "$WORK" && gh pr view "$BRANCH" --json url -q .url 2>/dev/null); }

log "starting in $WORK on $BRANCH (report $AGENT_REPORT)"
USED=""
for agent in codex claude op; do
    log "trying $agent"
    if "run_$agent" >> "$TRANSCRIPT" 2>&1; then USED="$agent"; log "$agent finished"; else log "$agent failed rc=$?"; continue; fi
    [ -n "$(pr_url)" ] && break
done

PR=$(pr_url)
CHECK=$( (cd "$WORK" && go run ./cmd/pricing-guard -live 2>&1); echo "exit=$?")
if [ -n "$PR" ] && printf '%s' "$CHECK" | grep -q 'exit=0$'; then VERDICT="PR OPENED"
elif [ -n "$PR" ]; then VERDICT="PR OPENED, GUARD STILL FAILING"
else VERDICT="NOT FIXED"; fi
log "verdict: $VERDICT (agent: ${USED:-none succeeded}) ${PR:-}"

{
    echo "openpaths pricing autofix: $VERDICT"
    echo "Agent: ${USED:-none succeeded (codex/claude/op all failed or unavailable)}"
    echo "Branch: $BRANCH   PR: ${PR:-none}"
    echo "Nothing was deployed. Review and merge the PR, then deploy as usual."
    echo
    echo "== Pricing guard on the branch after the agent"
    printf '%s\n' "$CHECK" | head -n 80
    echo
    if [ -s "$AGENT_REPORT" ]; then echo "== Agent report"; cat "$AGENT_REPORT"
    else echo "== No agent report written. Transcript tail:"; tail -n 60 "$TRANSCRIPT" 2>/dev/null; fi
} | grep -viE '(api[_-]?key|token|secret|passw)[^ ]*[=:] *[^ ]{8,}' \
  | "$SELF_DIR/notify.sh" "[$VERDICT] openpaths pricing autofix (${USED:-no agent})" - >> "$LOG" 2>&1 \
  && log "summary email sent" || log "summary email FAILED"

# Keep the worktree when nothing was pushed so a human can inspect it.
[ -n "$PR" ] && git -C "$PROD_DIR" worktree remove --force "$WORK" 2>/dev/null
exit 0
