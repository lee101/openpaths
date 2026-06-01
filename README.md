# OpenPaths

[openpaths.io](https://openpaths.io) -- AI model gateway -- unified API for chat, image, video, music, speech, embedding, and transcription across 15+ providers.

## Quick Start

```bash
# 1. Set up postgres
sudo -u postgres psql -c "CREATE USER openpaths WITH PASSWORD 'openpaths';"
sudo -u postgres psql -c "CREATE DATABASE openpaths OWNER openpaths;"

# 2. Copy and edit .env
cp .env.example .env
# Edit .env with your API keys and JWT_SECRET

# 3. Build and run
go build -o bin/openpaths ./cmd/openpaths/
GOMAXPROCS=3 ./bin/openpaths
```

## GPU Build (CUDA + gobed)

```bash
# Requires CUDA 12.x and gobed GPU libs
CUDA_PATH=/usr/local/cuda-12.9 \
GOBED_GPU=/home/lee/code/gobed/gpu \
./scripts/build-gpu.sh

# Run with GPU
./scripts/run-gpu.sh
```

Or manually:

```bash
export LD_LIBRARY_PATH="/home/lee/code/gobed/gpu:/usr/local/cuda-12.9/lib64:$LD_LIBRARY_PATH"
GOMAXPROCS=3 go build -tags="gpu cuda" \
  -ldflags="-extldflags '-Wl,-rpath,/home/lee/code/gobed/gpu -Wl,-rpath,/usr/local/cuda-12.9/lib64'" \
  -o bin/openpaths-gpu ./cmd/openpaths/
./bin/openpaths-gpu
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `PORT` | Server port (default: 8080) |
| `DATABASE_URL` | Postgres connection string |
| `JWT_SECRET` | Required. Secret for JWT tokens |
| `OPENAI_API_KEY` | OpenAI API key |
| `OPENAI_ADMIN_API_KEY` | Optional. OpenAI admin key used by `rotation/rotate_provider_key.py` to create a replacement service account key |
| `OPENAI_PROJECT_ID` | Optional. OpenAI project ID used by `rotation/rotate_provider_key.py` |
| `ANTHROPIC_API_KEY` | Anthropic API key |
| `GEMINI_API_KEY` | Google Gemini API key |
| `GROQ_API_KEY` | Groq API key |
| `XAI_API_KEY` | xAI/Grok API key |
| `DEEPSEEK_API_KEY` | DeepSeek API key |
| `OPENROUTER_API_KEY` | OpenRouter API key |
| `TOGETHER_API_KEY` | Together AI API key |
| `MINIMAX_API_KEY` | MiniMax API key |
| `NETWRCK_API_KEY` | Netwrck API key |
| `FAL_API_KEY` | Fal API key |
| `FAL_ADMIN_API_KEY` | Optional. fal admin key used by `rotation/rotate_provider_key.py`; falls back to `FAL_API_KEY` if unset |
| `Z_API_KEY` | Z.AI API key |
| `TEXTGENERATOR_API_KEY` | Text-Generator.io API key |
| `MISTRAL_API_KEY` | Mistral API key |
| `NVIDIA_API_KEY` | NVIDIA API key |
| `EXA_API_KEY` | Exa Search API key. Create or copy one from `https://exa.ai/account` |
| `APP_API_KEY` | app.nz Papers API key for `papers.app.nz`. Create one at `https://app.nz/account` |

Provider key creation scripts live in `rotation/`. They create a new supported provider key and update the matching `.env` value, but intentionally do not revoke the old key.

## API Endpoints

- `POST /v1/chat/completions` -- OpenAI-compatible chat
- `GET /v1/models` -- List available models
- `POST /v1/images/generations` -- Image generation
- `POST /v1/videos/generations` -- Video generation
- `POST /v1/music/generations` -- Music generation
- `POST /v1/audio/speech` -- Text-to-speech
- `POST /v1/audio/transcriptions` -- Audio transcription
- `POST /v1/embeddings` -- Text embeddings
- `POST /v1/search` -- Search API for Exa and Papers providers
- `POST /auth/register` -- Register user
- `POST /auth/login` -- Login
- `GET /health` -- Health check

## Chat Parameters

`POST /v1/chat/completions` supports the standard OpenAI-style fields we route internally across providers:

| Parameter | Status | Notes |
|----------|--------|-------|
| `model` | Full support | Hero: `openpaths/auto`. Variants: `openpaths/auto-code`, `openpaths/auto-fast`, `openpaths/auto-cheap`, `openpaths/auto-reasoning`, `openpaths/auto-vision`, `openpaths/auto-image`. Legacy aliases (`auto`, `auto-easy-task`, `auto-think`, …) still work. |
| `messages` | Full support | OpenAI chat message format |
| `temperature`, `top_p`, `stop` | Full support | Passed through where supported |
| `max_tokens`, `max_completion_tokens` | Full support | Normalized per provider/model family |
| `stream` | Full support | Streaming SSE responses |
| `tools`, `tool_choice` | Full support | Function/tool calling |
| `response_format` | Full support | Structured outputs / JSON mode where supported |
| `reasoning_effort` | Full support | Supported values: `none`, `low`, `medium`, `high`, `auto` |
| `thinking` | Provider-specific | Passed through for direct DeepSeek V4 models; mapped for Anthropic-compatible requests |

`reasoning_effort` can be set directly on compatible OpenAI-format requests:

```json
{
  "model": "openpaths/auto-reasoning",
  "messages": [
    {"role": "user", "content": "Design a mesh simplification algorithm."}
  ],
  "reasoning_effort": "high",
  "max_tokens": 2048
}
```

Use `openpaths/auto` as the default — OpenPaths picks the backend from your prompt. Use `openpaths/auto-code`, `openpaths/auto-fast`, `openpaths/auto-cheap`, `openpaths/auto-reasoning`, `openpaths/auto-vision`, or `openpaths/auto-image` when you want a modality bias. Set `reasoning_effort: "auto"` on direct thinking models or use `openpaths/auto-reasoning` to route reasoning depth automatically.

Set `reasoning_effort: "auto"` on any thinking-capable direct model to keep that model while letting OpenPaths choose `none`, `low`, `medium`, or `high` from the same embedding table used by `auto-think`:

```json
{
  "model": "nvidia/deepseek-v4-pro",
  "messages": [
    {"role": "user", "content": "Make a 3D simulation of cogs in a clock."}
  ],
  "reasoning_effort": "auto"
}
```

The Anthropic-compatible `POST /v1/messages` endpoint also accepts `thinking`, which we map onto the same internal reasoning controls.
Direct DeepSeek models `deepseek-v4-flash` and `deepseek-v4-pro` use the DeepSeek API and support `thinking: {"type":"enabled"}` or `{"type":"disabled"}` on chat completions.
The free NVIDIA-hosted DeepSeek Pro route is available as `nvidia/deepseek-v4-pro`; OpenPaths sends NVIDIA `chat_template_kwargs` for high-reasoning thinking mode automatically.

## Frontend

```bash
npm install
npm run dev
```

The web UI includes API docs at `/docs`.
