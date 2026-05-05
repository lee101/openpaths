# Speech-to-Text (STT) on OpenPaths

Last updated: 2026-04-04

---

## Current Integration

### Architecture

```
POST /v1/audio/transcriptions  (OpenAI-compatible multipart form)
    |
    v
TranscriptionHandler  (internal/handler/transcription.go)
    |
    v  model-aware routing (matches model -> provider, then fallback chain)
+-------+----------+------------+---------+
| Groq  | OpenAI   | Fireworks  | Fal     |
| (1st) | (2nd)    | (3rd)      | (4th)   |
+-------+----------+------------+---------+
```

**Model-aware routing**: If a specific model is requested (e.g. `gpt-4o-mini-transcribe`), the handler routes directly to the correct provider (OpenAI), with remaining providers as fallbacks. If no model or `auto` is specified, uses the default chain.

Registration order in `cmd/openpaths/main.go`:
- Groq is **prepended** to the list (fastest, cheapest -- always tried first)
- OpenAI is appended (reliable fallback)
- Fireworks is appended (good middle ground)
- Fal is appended last (accurate, chunk timestamps)

Health tracking via `router.HealthTracker` -- unhealthy providers are skipped, with automatic recovery.

### Implemented Providers

| Provider | File | Default Model | Our Cost | API Style |
|---|---|---|---|---|
| **Groq** | `internal/provider/groq/transcription.go` | `whisper-large-v3-turbo` | $0.04/hr | OpenAI-compatible multipart |
| **OpenAI** | `internal/provider/openai/transcription.go` | `gpt-4o-mini-transcribe` | $0.18/hr | Native OpenAI multipart |
| **Fireworks** | `internal/provider/fireworks/transcription.go` | `whisper-v3-large-turbo` | $0.054/hr | OpenAI-compatible multipart |
| **Fal** | `internal/provider/fal/fal.go` | `fal-ai/whisper` | varies | JSON + base64 data URL |

### Request Parameters

From `internal/model/transcription.go` and the handler:

| Param | Type | Required | Notes |
|---|---|---|---|
| `file` | multipart file | yes | Audio file (mp3, wav, ogg, m4a, webm, flac) |
| `model` | string | no | Model ID; defaults to provider default. `auto` = provider default. |
| `language` | string | no | ISO-639 language hint |
| `prompt` | string | no | Context/vocabulary hint |
| `response_format` | string | no | `json` (default), `text`, `srt`, `verbose_json`, `vtt` |

### Config Models (config.yaml)

All transcription models now have explicit entries with per-minute pricing:

| Model ID | Provider | $/min | $/hr | Aliases |
|---|---|---|---|---|
| `distil-whisper-large-v3-en` | groq | $0.00033 | $0.02 | distil-whisper, whisper-en |
| `whisper-large-v3-turbo` | groq | $0.00067 | $0.04 | whisper-turbo, groq-whisper |
| `whisper-v3-large-turbo` | fireworks | $0.0009 | $0.054 | fireworks-whisper-turbo |
| `whisper-v3-large` | fireworks | $0.0015 | $0.09 | fireworks-whisper |
| `whisper-large-v3` | groq | $0.00185 | $0.111 | whisper-v3 |
| `gpt-4o-mini-transcribe` | openai | $0.003 | $0.18 | openai-transcribe-mini |
| `gpt-4o-transcribe` | openai | $0.006 | $0.36 | openai-transcribe |
| `whisper-1` | openai | $0.006 | $0.36 | - |

Pricing uses `price_per_minute` field in `ModelConfig` (added alongside `price_per_image` and `price_per_video`). Billing method `CalculateTranscriptionCost` added to `PricingTable`.

### Blog Coverage

- `music-and-speech-models` blog post has full transcription section with all models, providers, and per-minute pricing
- `provider-groq` blog post mentions all 3 Whisper models with pricing
- Provider descriptions in `providers.ts` updated for Groq, Fireworks, and Fal

---

## Completed Improvements

1. ~~OpenAI default model is stale~~ -- Updated to `gpt-4o-mini-transcribe`
2. ~~No transcription models in config.yaml~~ -- Added 8 models with per-minute pricing
3. ~~Groq has 3 models but we only use 1~~ -- All 3 now accessible via model parameter
4. ~~Fireworks not integrated for STT~~ -- Added `fireworks/transcription.go`, registered in main.go
5. ~~No model-aware routing~~ -- Handler now routes to correct provider based on model name
6. ~~Blog pricing is vague~~ -- Full pricing table with all models and providers
7. Added `PricePerMinute` field to `ModelConfig` and `CalculateTranscriptionCost` to billing
8. Updated discovery to classify `whisper`/`transcribe` models as `stt` type (was `audio`)
9. Updated provider descriptions in `providers.ts` for Groq, Fireworks, Fal

## Future Candidates

| Provider | Why | Effort | Notes |
|---|---|---|---|
| **Deepgram** | $200 free credit. Same price streaming/batch. Best WER (5.26%). | Medium -- different REST API | Best accuracy option, good for streaming use case |
| **AssemblyAI** | $50 free credit. Universal-2 at $0.15/hr. 99 languages. | Medium -- different API format | Good multilingual fallback |

### Not Worth Integrating (for us)

- **Google Cloud / Azure / AWS** -- expensive, complex auth (IAM/service accounts), enterprise-oriented
- **IBM Watson** -- expensive, dated
- **Rev.ai** -- good but no unique advantage over what we have
- **Speechmatics / Gladia / Soniox / ElevenLabs** -- niche, smaller ecosystems

---

## Market Research: All STT Providers & Pricing (April 2026)

### Quick Comparison (sorted by cheapest batch price)

| Provider | Model | Batch $/min | Batch $/hr | Stream $/min | Stream $/hr | Free Tier |
|---|---|---|---|---|---|---|
| Groq | Distil-Whisper (EN) | $0.00033 | $0.02 | N/A | N/A | limited |
| Fireworks AI | Whisper v3 Large Turbo (batch) | $0.0005 | $0.03 | - | - | limited |
| Groq | Whisper Large v3 Turbo | $0.00067 | $0.04 | N/A | N/A | limited |
| Fireworks AI | Whisper v3 Large Turbo | $0.0009 | $0.054 | $0.0032 | $0.192 | limited |
| Fireworks AI | Whisper v3 Large | $0.0015 | $0.09 | $0.0032 | $0.192 | limited |
| Soniox | Token-based | $0.0017 | $0.10 | $0.002 | $0.12 | - |
| Rev.ai | Reverb Turbo | $0.0017 | $0.10 | - | - | 5 hrs |
| Groq | Whisper Large v3 | $0.00185 | $0.111 | N/A | N/A | limited |
| AssemblyAI | Universal-2 | $0.0025 | $0.15 | $0.0025 | $0.15 | $50 credit |
| OpenAI | gpt-4o-mini-transcribe | $0.003 | $0.18 | - | - | $5 credit |
| Azure | Standard Batch | $0.003 | $0.18 | $0.0167 | $1.00 | 5 hrs/mo |
| Gladia | Growth Async | $0.0033 | $0.20 | $0.0042 | $0.25 | 10 hrs/mo |
| Rev.ai | Reverb | $0.0033 | $0.20 | - | - | 5 hrs |
| AssemblyAI | Universal-3 Pro | $0.0035 | $0.21 | $0.0075 | $0.45 | $50 credit |
| ElevenLabs | Scribe v2 (Business) | $0.0037 | $0.22 | $0.0047 | $0.28 | plan-based |
| Speechmatics | Standard/Enhanced | $0.004 | $0.24 | $0.004 | $0.24 | 480 min/mo |
| Google Cloud | Dynamic Batch | $0.004 | $0.24 | - | - | 60 min/mo |
| Deepgram | Nova-3 (mono) | $0.0043 | $0.258 | $0.0043 | $0.258 | $200 credit |
| Rev.ai | Whisper Fusion | $0.005 | $0.30 | - | - | 5 hrs |
| OpenAI | gpt-4o-transcribe | $0.006 | $0.36 | - | - | $5 credit |
| OpenAI | Whisper-1 | $0.006 | $0.36 | - | - | $5 credit |
| ElevenLabs | Scribe v2 (PAYG) | $0.0067 | $0.40 | $0.0065 | $0.39 | plan-based |
| Gladia | Starter Async | $0.0102 | $0.61 | $0.0125 | $0.75 | 10 hrs/mo |
| Azure | Fast Transcription | $0.011 | $0.66 | - | - | 5 hrs/mo |
| Google Cloud | Chirp 2/3 (standard) | $0.016 | $0.96 | $0.016 | $0.96 | 60 min/mo |
| IBM Watson | Plus | $0.02 | $1.20 | $0.02 | $1.20 | 500 min/mo |
| AWS Transcribe | Standard | $0.024 | $1.44 | $0.024 | $1.44 | 60 min/mo (12 mo) |

---

### Detailed Provider Breakdown

#### 1. Groq (LPU inference) -- INTEGRATED

**Models:**
- `whisper-large-v3` -- $0.111/hr ($0.00185/min) -- 217x real-time, best multilingual quality
- `whisper-large-v3-turbo` -- $0.04/hr ($0.00067/min) -- 228x real-time, best bang for buck
- `distil-whisper-large-v3-en` -- $0.02/hr ($0.00033/min) -- English only, absolute cheapest

**Details:**
- Batch API gets 50% discount on top
- No streaming support
- 10-second minimum billing per request
- 100 MB max file size
- OpenAI-compatible `/v1/audio/transcriptions` endpoint

---

#### 2. OpenAI -- INTEGRATED

**Models:**
- `gpt-4o-transcribe` -- $0.36/hr ($0.006/min) -- best accuracy from OpenAI
- `gpt-4o-mini-transcribe` -- $0.18/hr ($0.003/min) -- recommended for cost/quality
- `whisper-1` -- $0.36/hr ($0.006/min) -- legacy

**Details:**
- Batch only (no streaming)
- 50+ languages, auto language detection
- 25 MB max file size
- Formats: mp3, mp4, mpeg, mpga, m4a, wav, webm
- $5 free credit for new accounts

---

#### 3. Fal -- INTEGRATED

**Models:**
- `fal-ai/whisper` -- pricing varies (serverless)

**Details:**
- Uses base64 data URL (not multipart)
- Returns chunk-level timestamps
- Supports mp3, wav, ogg, m4a, webm, flac
- Good for accuracy; slower than Groq

---

#### 4. Fireworks AI -- NOT INTEGRATED (easy win)

**Models:**
- `whisper-v3-large` -- $0.09/hr ($0.0015/min)
- `whisper-v3-large-turbo` -- $0.054/hr ($0.0009/min)

**Details:**
- Streaming: $0.192/hr ($0.0032/min)
- Batch discount: 40% off (turbo drops to ~$0.03/hr)
- Diarization: +40% surcharge
- OpenAI-compatible API -- trivial to integrate
- Already have API key configured for chat

---

#### 5. Deepgram -- NOT INTEGRATED (recommended)

**Models:**
- `Nova-3` (mono) -- $0.258/hr PAYG ($0.0043/min)
- `Nova-3` (multilingual) -- $0.0092/min PAYG
- `Nova-2` -- $0.0058/min PAYG
- `Flux` (conversational) -- $0.0077/min PAYG
- `Nova-3 Medical` -- specialized

**Details:**
- Same price for streaming and batch (unique)
- Best-in-class WER: 5.26% on general English
- Speaker diarization, smart formatting, language detection
- End-of-turn detection for voice agents
- $200 free credit (no expiration)

---

#### 6. AssemblyAI -- NOT INTEGRATED

**Pre-recorded:**
- `Universal-3 Pro` -- $0.21/hr -- best accuracy, prompt-based
- `Universal-2` -- $0.15/hr -- 99 languages
- `Slam-1` -- $0.27/hr

**Streaming:**
- `Universal-3 Pro Streaming` -- $0.45/hr
- `Universal-Streaming` -- $0.15/hr -- English only, fastest
- `Whisper-Streaming` -- $0.30/hr -- 99+ languages

**Add-ons:** Diarization +$0.02/hr (batch), Prompting +$0.05/hr, Medical +$0.15/hr

$50 free credit.

---

#### 7. Google Cloud Speech-to-Text

**Models:** Chirp 3, Chirp 2, standard models (short, long, phone_call, video, medical)

**Pricing:**
- Standard: $0.96/hr ($0.016/min)
- Dynamic Batch: $0.24/hr ($0.004/min) -- 24hr wait
- 100+ languages, streaming + batch, diarization, word timestamps
- 60 min/month free + $300 new account credit

---

#### 8. AWS Transcribe

- Standard: $1.44/hr ($0.024/min), drops to $0.47/hr at 5M+ min
- Medical: $4.50/hr ($0.075/min)
- 100+ languages, streaming + batch, diarization, content redaction
- 60 min/month free for 12 months

---

#### 9. Azure Speech Services

- Real-time: $1.00/hr, Batch: $0.18/hr, Fast: $0.66/hr
- 140+ languages, custom model training, on-prem deployment
- 5 hrs/month free

---

#### 10. Rev.ai

- Reverb Turbo: $0.10/hr (English, fastest)
- Reverb: $0.20/hr (English)
- Foreign Language: $0.30/hr (57 languages)
- 5 hrs free

---

#### 11. Speechmatics

- Standard/Enhanced: $0.24/hr ($0.004/min)
- 55+ languages, streaming, strong British accent handling
- 480 min/month free

---

#### 12. Gladia

- Solaria-1: $0.61/hr starter, $0.20/hr growth (async)
- All features bundled (no surcharges)
- 100+ languages, 103ms streaming latency
- 10 hrs/month free

---

#### 13. ElevenLabs

- Scribe v2: $0.22-0.40/hr depending on plan
- 90+ languages, 32-speaker diarization
- Plan-dependent free hours

---

#### 14. Soniox

- Token-based: ~$0.10/hr async, ~$0.12/hr streaming
- All features included
- 2-7x cheaper than Google Cloud

---

#### 15. IBM Watson

- Plus: $1.20/hr ($0.02/min)
- Custom model training, on-prem option
- 500 min/month free (Lite)

---

## Best For Each Use Case

| Use Case | Recommended | Price |
|---|---|---|
| Cheapest batch (English) | Groq Distil-Whisper | $0.02/hr |
| Cheapest batch (multilingual) | Groq Whisper v3 Turbo | $0.04/hr |
| Cheapest streaming | Soniox or AssemblyAI Universal-Streaming | $0.12-0.15/hr |
| Best accuracy | Deepgram Nova-3 (5.26% WER) | $0.258/hr |
| Fastest inference | Groq (228x real-time) | $0.04/hr |
| Most languages | Azure (140+), Gladia (100+), AssemblyAI (99) | varies |
| Best free tier | Deepgram ($200 credit), Gladia (10 hrs/mo) | free |
| Medical | AWS Medical, Deepgram Medical, AssemblyAI Medical | varies |
| Voice agents (low latency) | Deepgram Flux, AssemblyAI Streaming | $0.15-0.26/hr |
| All features bundled | Gladia, Soniox | $0.10-0.61/hr |
