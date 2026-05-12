#!/bin/bash
# Pull fresh browser-reported frontend errors from the prod backend JSONL file.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STATE_DIR="$SCRIPT_DIR/state"
ERR_DIR="$SCRIPT_DIR/errors"
OFFSET_FILE="$STATE_DIR/last_frontend_client_log_offset"
mkdir -p "$STATE_DIR" "$ERR_DIR"

SSH_PASS="${OPENPATHS_SSH_PASS:-ka3iMI4OSNvgFcREuDaQyLguFxuP}"
HOST="${OPENPATHS_HOST:-administrator@93.127.141.100}"
REMOTE_LOG="${OPENPATHS_FRONTEND_REMOTE_LOG:-/nvme0n1-disk/code/openpaths/monitoring/errors/frontend_client.jsonl}"

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

if [ "$CUR" -lt "$PREV" ]; then
    PREV=0
fi

NEW_BYTES=$((CUR - PREV))
echo "[frontend-client] remote_log=$REMOTE_LOG size=$CUR prev=$PREV new=$NEW_BYTES" >&2

if [ "$NEW_BYTES" -le 0 ]; then
    echo "$CUR" > "$OFFSET_FILE"
    exit 0
fi

MAX_BYTES=16777216
if [ "$NEW_BYTES" -gt "$MAX_BYTES" ]; then
    NEW_BYTES=$MAX_BYTES
fi

NEW_LINES=$(ssh_cmd "tail -c $NEW_BYTES '$REMOTE_LOG' 2>/dev/null" || true)
echo "$CUR" > "$OFFSET_FILE"

if [ -z "$NEW_LINES" ]; then
    echo "[frontend-client] no fresh errors" >&2
    exit 0
fi

TS=$(date -u +%Y%m%dT%H%M%SZ)
OUT="$ERR_DIR/frontend_client_$TS.json"
python3 - "$OUT" "$REMOTE_LOG" "$NEW_LINES" <<'PY'
import datetime
import json
import sys

out, log_path, lines = sys.argv[1], sys.argv[2], sys.argv[3]
events = []
for line in lines.splitlines():
    if not line.strip():
        continue
    try:
        events.append(json.loads(line))
    except json.JSONDecodeError:
        events.append({"raw": line})

data = {
    "source": "frontend-client",
    "remote_log": log_path,
    "detected_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
    "event_count": len(events),
    "events": events,
}
with open(out, "w") as f:
    json.dump(data, f, indent=2)
PY

echo "$OUT"
exit 1
