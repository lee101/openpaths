# OpenPath

[openpaths.io](https://openpaths.io) -- AI model gateway -- unified API for chat, image, video, music, speech, embedding, and transcription across 15+ providers.

## Quick Start

```bash
# 1. Set up postgres
sudo -u postgres psql -c "CREATE USER openpath WITH PASSWORD 'openpath';"
sudo -u postgres psql -c "CREATE DATABASE openpath OWNER openpath;"

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
  -o bin/openpath-gpu ./cmd/openpaths/
./bin/openpath-gpu
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `PORT` | Server port (default: 8080) |
| `DATABASE_URL` | Postgres connection string |
| `JWT_SECRET` | Required. Secret for JWT tokens |
| `OPENAI_API_KEY` | OpenAI API key |
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
| `Z_API_KEY` | Z.AI API key |
| `TEXTGENERATOR_API_KEY` | Text-Generator.io API key |
| `MISTRAL_API_KEY` | Mistral API key |

## API Endpoints

- `POST /v1/chat/completions` -- OpenAI-compatible chat
- `GET /v1/models` -- List available models
- `POST /v1/images/generations` -- Image generation
- `POST /v1/videos/generations` -- Video generation
- `POST /v1/music/generations` -- Music generation
- `POST /v1/audio/speech` -- Text-to-speech
- `POST /v1/audio/transcriptions` -- Audio transcription
- `POST /v1/embeddings` -- Text embeddings
- `POST /auth/register` -- Register user
- `POST /auth/login` -- Login
- `GET /health` -- Health check

## Frontend

```bash
npm install
npm run dev
```

The web UI includes API docs at `/docs`.
