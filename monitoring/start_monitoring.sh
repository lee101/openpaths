#!/bin/bash
set -e
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
mkdir -p "$SCRIPT_DIR/logs"

if pgrep -f "monitor_site.py" > /dev/null 2>&1; then
    echo "monitoring already running ($(pgrep -f monitor_site.py))"
    exit 0
fi

nohup python3 "$SCRIPT_DIR/monitor_site.py" --interval 60 \
    >> "$SCRIPT_DIR/logs/monitor_stdout.log" 2>&1 &
echo $! > "$SCRIPT_DIR/logs/monitor.pid"
echo "monitoring started (pid $!)"
