#!/bin/bash
# Run this on the prod server to set up monitoring cron + systemd
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "setting up monitoring for openpaths..."

mkdir -p "$SCRIPT_DIR/logs"
chmod +x "$SCRIPT_DIR"/*.sh

# Install playwright if needed
if ! command -v npx &>/dev/null; then
    echo "node/npx not found, skipping playwright setup"
else
    cd "$PROJECT_DIR"
    npx playwright install chromium 2>/dev/null || echo "playwright install skipped"
fi

# Setup cron job - every 30 min: site health + supervisor restart on hard down
CRON_CMD="cd $PROJECT_DIR && bash monitoring/autofix_agent.sh >> monitoring/logs/cron.log 2>&1"
if crontab -l 2>/dev/null | grep -q "autofix_agent"; then
    echo "autofix cron already configured"
else
    (crontab -l 2>/dev/null || true; echo "*/30 * * * * $CRON_CMD") | crontab -
    echo "autofix cron installed: every 30 min"
fi

# Setup error watcher cron - every 30 min, internally debounced to once / 12h
WATCHER_CMD="cd $PROJECT_DIR && bash monitoring/error_watcher.sh >> monitoring/logs/watcher_cron.log 2>&1"
if crontab -l 2>/dev/null | grep -q "error_watcher"; then
    echo "error_watcher cron already configured"
else
    (crontab -l 2>/dev/null || true; echo "*/30 * * * * $WATCHER_CMD") | crontab -
    echo "error_watcher cron installed: every 30 min (12h debounce on Codex spawn)"
fi

# Setup daily route monitor cron - daily at 7 AM
ROUTE_CRON="cd $PROJECT_DIR && bash monitoring/daily_route_monitor.sh >> monitoring/logs/daily.log 2>&1"
if crontab -l 2>/dev/null | grep -q "daily_route_monitor"; then
    echo "daily route monitor cron already configured"
else
    (crontab -l 2>/dev/null || true; echo "0 7 * * * $ROUTE_CRON") | crontab -
    echo "daily route monitor cron installed: 7 AM daily"
fi

# Optional: systemd timers (needs sudo)
if [ "$(id -u)" = "0" ] || sudo -n true 2>/dev/null; then
    sudo cp "$SCRIPT_DIR/systemd/openpaths-monitor.service" /etc/systemd/system/
    sudo cp "$SCRIPT_DIR/systemd/openpaths-monitor.timer" /etc/systemd/system/
    sudo cp "$SCRIPT_DIR/systemd/openpaths-route-monitor.service" /etc/systemd/system/
    sudo cp "$SCRIPT_DIR/systemd/openpaths-route-monitor.timer" /etc/systemd/system/
    sudo systemctl daemon-reload
    sudo systemctl enable --now openpaths-monitor.timer
    sudo systemctl enable --now openpaths-route-monitor.timer
    echo "systemd timers enabled (site monitor + daily route tests)"
else
    echo "skipping systemd setup (no sudo). using cron only."
fi

echo "monitoring setup complete"
