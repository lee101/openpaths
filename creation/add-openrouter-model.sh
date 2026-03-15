#!/bin/bash
# Template for investigating and adding a new OpenRouter model
# Usage: ./creation/add-openrouter-model.sh <model-id>
# Example: ./creation/add-openrouter-model.sh "meta-llama/llama-4-scout:free"

MODEL_ID="${1:?Usage: $0 <openrouter-model-id>}"

echo "=== Fetching model info for $MODEL_ID ==="
curl -s "https://openrouter.ai/api/v1/models" | python3 -c "
import json, sys
data = json.load(sys.stdin)
matches = [m for m in data.get('data', []) if m.get('id') == '$MODEL_ID']
if not matches:
    print('Model not found')
    sys.exit(1)
m = matches[0]
print(f'ID: {m[\"id\"]}')
print(f'Name: {m.get(\"name\", \"?\")}')
print(f'Context: {m.get(\"context_length\", \"?\")}')
print(f'Pricing: {m.get(\"pricing\", {})}')
print(f'Architecture: {m.get(\"architecture\", {})}')
print()

# Generate config.yaml entry
safe_id = m['id'].split('/')[-1].split(':')[0].replace('.', '-')
ctx = m.get('context_length', 8192)
pricing = m.get('pricing', {})
inp = float(pricing.get('prompt', '0')) * 1000000
out = float(pricing.get('completion', '0')) * 1000000

print('--- config.yaml entry ---')
print(f'  - id: \"or/{safe_id}\"')
print(f'    provider: openrouter')
print(f'    provider_model_id: \"{m[\"id\"]}\"')
print(f'    input_price_per_1m: {inp}')
print(f'    output_price_per_1m: {out}')
print(f'    context_window: {ctx}')
print(f'    max_output_tokens: 8192')
print(f'    supports_streaming: true')
print(f'    supports_tools: false')
print(f'    supports_vision: false')
"

echo ""
echo "=== Testing model ==="
curl -s https://openrouter.ai/api/v1/chat/completions \
  -H "Authorization: Bearer $OPENROUTER_API_KEY" \
  -H "Content-Type: application/json" \
  -H "HTTP-Referer: https://openpaths.io" \
  -d "{\"model\":\"$MODEL_ID\",\"messages\":[{\"role\":\"user\",\"content\":\"say hello in one word\"}],\"max_tokens\":20}" | python3 -m json.tool
