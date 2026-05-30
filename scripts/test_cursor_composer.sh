#!/bin/bash
# Smoke test Cursor Composer 2.5 Fast via @cursor/sdk (local agent runtime).
# Usage: ./scripts/test_cursor_composer.sh

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

KEY="${CURSOR_API_KEY:-}"
if [ -z "$KEY" ] && [ -f .env ]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
  KEY="${CURSOR_API_KEY:-}"
fi
if [ -z "$KEY" ]; then
  echo "CURSOR_API_KEY not set (add to .env or export it)"
  exit 1
fi

DEPS_DIR="$ROOT/scripts/.cursor-sdk-deps"
if [ ! -d "$DEPS_DIR/node_modules/@cursor/sdk" ]; then
  echo "Installing @cursor/sdk into $DEPS_DIR ..."
  mkdir -p "$DEPS_DIR"
  npm install --prefix "$DEPS_DIR" @cursor/sdk >/dev/null
fi

echo "=== Cursor API key ==="
curl -sS -u "$KEY:" https://api.cursor.com/v1/me | python3 -m json.tool

echo
echo "=== Composer 2.5 model entry ==="
curl -sS -u "$KEY:" https://api.cursor.com/v1/models \
  | python3 -c "import json,sys; items=json.load(sys.stdin)['items']; m=next(x for x in items if x['id']=='composer-2.5'); print(json.dumps(m, indent=2))"

echo
echo "=== Composer 2.5 Fast prompt test ==="
CURSOR_API_KEY="$KEY" CURSOR_TEST_CWD="$ROOT" node "$ROOT/scripts/test_cursor_composer.mjs"
