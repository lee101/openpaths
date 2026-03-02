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

# Setup cron job - every 30 min
CRON_CMD="cd $PROJECT_DIR && bash monitoring/autofix_agent.sh >> monitoring/logs/cron.log 2>&1"
if crontab -l 2>/dev/null | grep -q "autofix_agent"; then
    echo "cron already configured"
else
    (crontab -l 2>/dev/null || true; echo "*/30 * * * * $CRON_CMD") | crontab -
    echo "cron installed: every 30 min"
fi

# Optional: systemd timer (needs sudo)
if [ "$(id -u)" = "0" ] || sudo -n true 2>/dev/null; then
    sudo cp "$SCRIPT_DIR/systemd/openpaths-monitor.service" /etc/systemd/system/
    sudo cp "$SCRIPT_DIR/systemd/openpaths-monitor.timer" /etc/systemd/system/
    sudo systemctl daemon-reload
    sudo systemctl enable --now openpaths-monitor.timer
    echo "systemd timer enabled"
else
    echo "skipping systemd setup (no sudo). using cron only."
fi

echo "monitoring setup complete"
