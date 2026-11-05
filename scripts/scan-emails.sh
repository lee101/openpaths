#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
gitleaks dir emails --no-banner --redact
if rg -n -i "[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[a-z]{2,}" emails | rg -v "noreply@|support@openpaths|@example\.com|@openpaths\.io|@netwrck\.com"; then
  echo "personal email addresses found in emails/" >&2
  exit 1
fi
echo "emails/ clean"
