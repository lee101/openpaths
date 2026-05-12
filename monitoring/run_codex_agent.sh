#!/bin/bash
# Run the local Codex CLI with the standard OpenPaths auto-fix settings.
set -euo pipefail

CODEX_BIN="${CODEX_LOCAL:-codex}"
CODEX_MODEL="${CODEX_MODEL:-gpt-5.5}"
CODEX_REASONING_EFFORT="${CODEX_REASONING_EFFORT:-medium}"

if [ "$#" -gt 0 ]; then
    PROMPT="$1"
else
    PROMPT="$(cat)"
fi

"$CODEX_BIN" --yolo3 -m "$CODEX_MODEL" --config "model_reasoning_effort=$CODEX_REASONING_EFFORT" exec "$PROMPT"
