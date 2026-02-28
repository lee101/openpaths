#!/bin/bash
# Test all Together AI model IDs to verify they respond correctly.
# Usage: TOGETHER_API_KEY=... ./scripts/test_together_models.sh

set -euo pipefail

KEY="${TOGETHER_API_KEY:-}"
if [ -z "$KEY" ]; then
  if [ -f .env ]; then
    KEY=$(grep TOGETHER_API_KEY .env | head -1 | sed 's/.*="\(.*\)"/\1/' | sed "s/.*='\(.*\)'/\1/")
  fi
fi
if [ -z "$KEY" ]; then
  echo "TOGETHER_API_KEY not set"
  exit 1
fi

CHAT_MODELS=(
  "zai-org/GLM-5"
  "Qwen/Qwen3.5-397B-A17B"
  "MiniMaxAI/MiniMax-M2.5"
  "moonshotai/Kimi-K2.5"
  "zai-org/GLM-4.7"
  "Qwen/Qwen3-Coder-Next-FP8"
  "deepseek-ai/DeepSeek-R1"
  "deepseek-ai/DeepSeek-V3.1"
)

IMAGE_MODELS=(
  "black-forest-labs/FLUX.1-schnell"
)

PASS=0
FAIL=0

echo "=== Testing Together AI Chat Models ==="
for model in "${CHAT_MODELS[@]}"; do
  resp=$(curl -s --max-time 60 -X POST "https://api.together.xyz/v1/chat/completions" \
    -H "Authorization: Bearer $KEY" -H "Content-Type: application/json" \
    -d "{\"model\":\"$model\",\"messages\":[{\"role\":\"user\",\"content\":\"say no nothing else\"}],\"max_tokens\":20}" 2>&1)

  has_choices=$(echo "$resp" | python3 -c "import json,sys;d=json.load(sys.stdin);print('yes' if d.get('choices') else 'no')" 2>/dev/null || echo "no")

  if [ "$has_choices" = "yes" ]; then
    content=$(echo "$resp" | python3 -c "import json,sys;d=json.load(sys.stdin);c=d['choices'][0]['message'];print(c.get('content','') or c.get('reasoning','')[:30])" 2>/dev/null)
    echo "PASS  $model -> $content"
    ((PASS++))
  else
    err=$(echo "$resp" | python3 -c "import json,sys;d=json.load(sys.stdin);print(d.get('error',{}).get('message','unknown')[:80])" 2>/dev/null || echo "$resp" | head -c 100)
    echo "FAIL  $model -> $err"
    ((FAIL++))
  fi
done

echo ""
echo "=== Testing Together AI Image Models ==="
for model in "${IMAGE_MODELS[@]}"; do
  resp=$(curl -s --max-time 120 -X POST "https://api.together.xyz/v1/images/generations" \
    -H "Authorization: Bearer $KEY" -H "Content-Type: application/json" \
    -d "{\"model\":\"$model\",\"prompt\":\"a red circle on white background\",\"n\":1}" 2>&1)

  has_data=$(echo "$resp" | python3 -c "import json,sys;d=json.load(sys.stdin);print('yes' if d.get('data') else 'no')" 2>/dev/null || echo "no")

  if [ "$has_data" = "yes" ]; then
    echo "PASS  $model -> got image data"
    ((PASS++))
  else
    err=$(echo "$resp" | python3 -c "import json,sys;d=json.load(sys.stdin);print(d.get('error',{}).get('message','unknown')[:80])" 2>/dev/null || echo "$resp" | head -c 100)
    echo "FAIL  $model -> $err"
    ((FAIL++))
  fi
done

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
exit $FAIL
