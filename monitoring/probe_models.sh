#!/usr/bin/env bash
# Run chat "say hi" probes against all token-priced models and print timings.
# Requires OPENPATHS_PROBE_API_KEY (or OPENPATHS_API_KEY) with balance for probes.
#
# Usage:
#   ./monitoring/probe_models.sh
#   ./monitoring/probe_models.sh --base=https://openpaths.io
#   ./monitoring/probe_models.sh --wait   # poll /stats/model-probes until run completes

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BASE="${OPENPATHS_PROBE_BASE_URL:-https://openpaths.io}"
WAIT=false
for arg in "$@"; do
  case "$arg" in
    --base=*) BASE="${arg#*=}" ;;
    --wait) WAIT=true ;;
  esac
done
BASE="${BASE%/}"

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

KEY="${OPENPATHS_PROBE_API_KEY:-${OPENPATHS_API_KEY:-}}"
if [[ -z "$KEY" ]]; then
  echo "OPENPATHS_PROBE_API_KEY or OPENPATHS_API_KEY required"
  exit 1
fi

if [[ -x ./openpaths-api ]] || command -v go >/dev/null 2>&1; then
  echo "Running probe-models (writes model_probe_results + usage_logs via API)..."
  OPENPATHS_PROBE_BASE_URL="$BASE" go run ./cmd/probe-models -config config.yaml
elif $WAIT; then
  echo "Waiting for server cron probe results (poll /stats/model-probes every 30s)..."
  for _ in $(seq 1 480); do
    body=$(curl -fsS "$BASE/stats/model-probes" 2>/dev/null || echo '{}')
    total=$(echo "$body" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('summary',{}).get('total',0))" 2>/dev/null || echo 0)
    if [[ "$total" -ge 50 ]]; then
      break
    fi
    sleep 30
  done
else
  echo "Install go or build openpaths-api to run probes locally; use --wait to poll remote cron."
fi

echo
curl -fsS "$BASE/stats/model-probes" | python3 - <<'PY'
import json, sys
d = json.load(sys.stdin)
probes = d.get("probes") or []
summary = d.get("summary") or {}
latest = d.get("latest_probed_at")
print(f"latest_probed_at: {latest}")
print(f"summary: {summary}")
print()
print(f"{'model':<42} {'provider':<12} {'ms':>8}  ok  preview/error")
print("-" * 100)
for p in sorted(probes, key=lambda x: (not x.get("ok"), x.get("latency_ms") or 0)):
    ok = "ok" if p.get("ok") else "FAIL"
    preview = (p.get("response_preview") or p.get("error") or "")[:48]
    print(f"{p.get('model',''):<42} {p.get('provider',''):<12} {p.get('latency_ms',0):>8}  {ok:<4} {preview}")
PY
