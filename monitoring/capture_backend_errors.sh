#!/bin/bash
# Pull fresh backend errors from prod supervisor log into monitoring/errors/.
#
# Strategy:
#   - Read remote file size, compare against monitoring/state/last_log_offset
#   - Stream only the new bytes
#   - Grep for error/panic/SIGILL/SIGSEGV/runtime patterns
#   - On hits: write JSON to monitoring/errors/backend_<ts>.json and print path
#   - Always update offset to current remote size
#
# Exit 1 if fresh errors detected (drives error_watcher.sh), 0 otherwise.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STATE_DIR="$SCRIPT_DIR/state"
ERR_DIR="$SCRIPT_DIR/errors"
OFFSET_FILE="$STATE_DIR/last_log_offset"
mkdir -p "$STATE_DIR" "$ERR_DIR"

SSH_PASS="${OPENPATHS_SSH_PASS:-ka3iMI4OSNvgFcREuDaQyLguFxuP}"
HOST="${OPENPATHS_HOST:-administrator@93.127.141.100}"
REMOTE_LOG="${OPENPATHS_REMOTE_LOG:-/var/log/supervisor/openpaths-error.log}"

ssh_cmd() {
    sshpass -p "$SSH_PASS" ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 "$HOST" "$@"
}

current_size() {
    ssh_cmd "stat -c %s '$REMOTE_LOG' 2>/dev/null || echo 0"
}

last_offset() {
    [ -f "$OFFSET_FILE" ] && cat "$OFFSET_FILE" || echo 0
}

CUR=$(current_size | tr -d '[:space:]')
PREV=$(last_offset | tr -d '[:space:]')
[ -z "$CUR" ] && CUR=0
[ -z "$PREV" ] && PREV=0

# If log was rotated/truncated, reset offset
if [ "$CUR" -lt "$PREV" ]; then
    PREV=0
fi

NEW_BYTES=$((CUR - PREV))
echo "[backend] remote_log=$REMOTE_LOG size=$CUR prev=$PREV new=$NEW_BYTES" >&2

if [ "$NEW_BYTES" -le 0 ]; then
    echo "$CUR" > "$OFFSET_FILE"
    exit 0
fi

# Cap how much we pull at once (16 MiB) to avoid runaway during spam
MAX_BYTES=16777216
if [ "$NEW_BYTES" -gt "$MAX_BYTES" ]; then
    NEW_BYTES=$MAX_BYTES
fi

NEW_LINES=$(ssh_cmd "tail -c $NEW_BYTES '$REMOTE_LOG' 2>/dev/null" || true)

# Always advance offset so we don't re-read on next run
echo "$CUR" > "$OFFSET_FILE"

# Look for error indicators
PATTERN='panic:|SIGILL|SIGSEGV|fatal error|runtime error|level=error|goroutine [0-9]+ \[running\]:'
MATCHES=$(printf '%s\n' "$NEW_LINES" | grep -E -A 5 "$PATTERN" || true)

if [ -z "$MATCHES" ]; then
    echo "[backend] no fresh errors" >&2
    exit 0
fi

TS=$(date -u +%Y%m%dT%H%M%SZ)
OUT="$ERR_DIR/backend_$TS.json"
# Build JSON safely with python (always available)
python3 - "$OUT" "$REMOTE_LOG" "$MATCHES" <<'PY'
import json, sys, datetime
out, log_path, matches = sys.argv[1], sys.argv[2], sys.argv[3]
data = {
    "source": "backend",
    "remote_log": log_path,
    "detected_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
    "matches": matches,
}
with open(out, "w") as f:
    json.dump(data, f, indent=2)
PY

echo "$OUT"
exit 1
