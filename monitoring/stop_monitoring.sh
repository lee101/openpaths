#!/bin/bash
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PID_FILE="$SCRIPT_DIR/logs/monitor.pid"

if [ -f "$PID_FILE" ]; then
    kill "$(cat "$PID_FILE")" 2>/dev/null && echo "stopped" || echo "not running"
    rm -f "$PID_FILE"
else
    pkill -f "monitor_site.py" 2>/dev/null && echo "stopped" || echo "not running"
fi
