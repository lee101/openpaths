#!/bin/bash
# Investigate OpenRouter models and integration
# Usage: "$CODEX_LOCAL" --yolo3 -m gpt-5.5 --config model_reasoning_effort=medium exec "$(cat creation/investigate-openrouter.sh)"

# Fetch current free models from OpenRouter
echo "=== Fetching OpenRouter free models ==="
curl -s https://openrouter.ai/api/v1/models | python3 -c "
import json, sys
data = json.load(sys.stdin)
free = [m for m in data.get('data', []) if ':free' in m.get('id', '')]
for m in free:
    print(f\"  {m['id']}  ctx={m.get('context_length', '?')}  pricing={m.get('pricing', {})}\")
print(f'\nTotal free models: {len(free)}')
"

echo ""
echo "=== Test embedding endpoint ==="
curl -s https://openrouter.ai/api/v1/embeddings \
  -H "Authorization: Bearer $OPENROUTER_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"nvidia/llama-nemotron-embed-vl-1b-v2:free","input":"test embedding"}' | python3 -m json.tool

echo ""
echo "=== Test free chat model ==="
curl -s https://openrouter.ai/api/v1/chat/completions \
  -H "Authorization: Bearer $OPENROUTER_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"nvidia/nemotron-3-nano-30b-a3b:free","messages":[{"role":"user","content":"say hello"}],"max_tokens":50}' | python3 -m json.tool
