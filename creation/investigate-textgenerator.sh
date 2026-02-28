#!/bin/bash
# Investigate text-generator.io embedding API
# Usage: claude --dangerously-skip-permissions -p "$(cat creation/investigate-textgenerator.sh)"

echo "=== Test text-generator.io feature extraction ==="
curl -s 'https://api.text-generator.io/api/v1/feature-extraction' \
  -H 'content-type: application/json' \
  -H "secret: $TEXTGENERATOR_API_KEY" \
  -d '{"num_features":768,"text":"Test text here"}' | python3 -c "
import json, sys
data = json.load(sys.stdin)
if isinstance(data, list):
    print(f'Embedding dims: {len(data)}')
    print(f'First 5 values: {data[:5]}')
    print(f'Type: {type(data[0]).__name__}')
else:
    print(f'Unexpected response: {json.dumps(data, indent=2)}')
"

echo ""
echo "=== Test with different dimensions ==="
for dims in 256 512 768 1024 1536; do
  result=$(curl -s 'https://api.text-generator.io/api/v1/feature-extraction' \
    -H 'content-type: application/json' \
    -H "secret: $TEXTGENERATOR_API_KEY" \
    -d "{\"num_features\":$dims,\"text\":\"Test\"}" | python3 -c "
import json, sys
data = json.load(sys.stdin)
if isinstance(data, list): print(f'{len(data)} dims ok')
else: print(f'error: {data}')
" 2>/dev/null)
  echo "  $dims -> $result"
done
