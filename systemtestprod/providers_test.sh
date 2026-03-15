#!/usr/bin/env bash
set -uo pipefail

BASE="${BASE_URL:-https://openpaths.io}"
API_KEY="${API_KEY:-op-620d62e78837c2a0e242f4ce7efdbdbe3c55ddde2e5872ba19e2c6a758799b00}"
PASS=0
FAIL=0
SKIP=0
TIMEOUT=120

ok() { echo "  PASS: $1"; ((PASS++)); }
fail() { echo "  FAIL: $1 -> $2"; ((FAIL++)); }
skip() { echo "  SKIP: $1 -> $2"; ((SKIP++)); }

test_chat() {
  local label="$1" model="$2" format="${3:-openai}"
  local resp
  if [[ "$format" == "anthropic" ]]; then
    resp=$(curl -s --max-time "$TIMEOUT" "$BASE/v1/messages" \
      -H "Authorization: Bearer $API_KEY" \
      -H "Content-Type: application/json" \
      -d "{\"model\":\"$model\",\"messages\":[{\"role\":\"user\",\"content\":\"Reply with only the word yes.\"}],\"max_tokens\":10,\"temperature\":0}" 2>&1) || { fail "$label" "curl error"; return; }
    local content
    content=$(echo "$resp" | python3 -c "import sys,json; print(json.load(sys.stdin)['content'][0]['text'])" 2>/dev/null) || { fail "$label" "parse error: $resp"; return; }
  else
    resp=$(curl -s --max-time "$TIMEOUT" "$BASE/v1/chat/completions" \
      -H "Authorization: Bearer $API_KEY" \
      -H "Content-Type: application/json" \
      -d "{\"model\":\"$model\",\"messages\":[{\"role\":\"user\",\"content\":\"Reply with only the word yes.\"}],\"max_tokens\":10,\"temperature\":0}" 2>&1) || { fail "$label" "curl error"; return; }
    local content
    content=$(echo "$resp" | python3 -c "import sys,json; print(json.load(sys.stdin)['choices'][0]['message']['content'])" 2>/dev/null) || { fail "$label" "parse error: $resp"; return; }
  fi
  local lower=$(echo "$content" | tr '[:upper:]' '[:lower:]' | xargs)
  if [[ "$lower" == *"yes"* ]]; then
    ok "$label ($content)"
  else
    fail "$label" "unexpected: $content"
  fi
}

test_embedding() {
  local label="$1" model="$2"
  local resp
  resp=$(curl -s --max-time "$TIMEOUT" "$BASE/v1/embeddings" \
    -H "Authorization: Bearer $API_KEY" \
    -H "Content-Type: application/json" \
    -d "{\"model\":\"$model\",\"input\":\"test embedding\"}" 2>&1) || { fail "$label" "curl error"; return; }
  local dim
  dim=$(echo "$resp" | python3 -c "import sys,json; d=json.load(sys.stdin); print(len(d['data'][0]['embedding']))" 2>/dev/null) || { fail "$label" "parse error: $resp"; return; }
  if [[ "$dim" -gt 0 ]]; then
    ok "$label (dim=$dim)"
  else
    fail "$label" "zero dimensions"
  fi
}

test_image() {
  local label="$1" model="$2"
  local resp
  resp=$(curl -s --max-time "$TIMEOUT" "$BASE/v1/images/generations" \
    -H "Authorization: Bearer $API_KEY" \
    -H "Content-Type: application/json" \
    -d "{\"model\":\"$model\",\"prompt\":\"A solid blue square\",\"size\":\"512x512\"}" 2>&1) || { fail "$label" "curl error"; return; }
  local has_data
  has_data=$(echo "$resp" | python3 -c "
import sys,json
d=json.load(sys.stdin)
item=d['data'][0]
print('ok' if item.get('url') or item.get('b64_json') else 'no')
" 2>/dev/null) || { fail "$label" "parse error: $resp"; return; }
  if [[ "$has_data" == "ok" ]]; then
    ok "$label"
  else
    fail "$label" "no image data"
  fi
}

echo "============================================"
echo " OpenPaths Production Provider Test"
echo " Target: $BASE"
echo " $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
echo "============================================"
echo ""

# --- OpenAI ---
echo "=== OpenAI ==="
test_chat "GPT-4o Mini" "gpt-4o-mini"
echo ""

# --- Anthropic ---
echo "=== Anthropic ==="
test_chat "Claude Haiku 4.5" "claude-haiku-4-5-20251001" "anthropic"
echo ""

# --- Google ---
echo "=== Google ==="
test_chat "Gemini Flash Lite" "gemini-flash-lite"
echo ""

# --- xAI ---
echo "=== xAI ==="
test_chat "Grok 3 Mini" "grok-3-mini"
echo ""

# --- DeepSeek ---
echo "=== DeepSeek ==="
test_chat "DeepSeek V3.2" "deepseek-chat"
echo ""

# --- Mistral ---
echo "=== Mistral ==="
test_chat "Mistral Nemo" "open-mistral-nemo"
echo ""

# --- Groq ---
echo "=== Groq ==="
test_chat "Llama 3.1 8B" "llama-3.1-8b-instant"
echo ""

# --- Together AI ---
echo "=== Together AI ==="
test_chat "MiniMax M2.5 (Together)" "minimax-m2.5"
echo ""

# --- MiniMax ---
echo "=== MiniMax ==="
test_chat "MiniMax M2" "minimax-m2"
echo ""

# --- Z.AI ---
echo "=== Z.AI ==="
test_chat "GLM-4.6v Flash" "glm-4.6v-flash"
echo ""

# --- OpenRouter ---
echo "=== OpenRouter ==="
test_chat "StepFun Flash" "or/stepfun-flash"
echo ""

# --- OpenPaths Auto ---
echo "=== OpenPaths Auto ==="
test_chat "Auto Easy Task" "auto-easy-task"
test_chat "Auto Medium Task" "auto-medium-task"
echo ""

# --- Text-Generator.io (Embedding) ---
echo "=== Text-Generator.io ==="
test_embedding "ModernBERT Embedding" "text-embedding"
echo ""

# --- Netwrck (Image) ---
echo "=== Netwrck ==="
test_image "RA1 Art Generator" "ra1"
echo ""

# --- Fal (Image) ---
echo "=== Fal ==="
test_image "FLUX Klein 4B" "klein"
echo ""

# --- Together (Image) ---
echo "=== Together AI (Image) ==="
test_image "FLUX Schnell" "flux-schnell"
echo ""

echo "============================================"
echo " Results: $PASS passed, $FAIL failed, $SKIP skipped"
echo "============================================"

if [[ $FAIL -gt 0 ]]; then
  exit 1
fi
