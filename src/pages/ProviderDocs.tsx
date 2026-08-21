import React, { useEffect, useMemo, useState } from 'react';
import { Link, Navigate, useParams } from 'react-router-dom';
import { BookOpen, Copy, Check, ExternalLink, ArrowLeft, Play, KeyRound } from 'lucide-react';
import { providers, FALLBACK_LOGO } from '../data/providers';
import { models } from '../data/models';
import { getAPIBaseURL, getStoredAPIKey } from '../lib/session';
import { CodeBlock } from '../components/CodeBlock';
import { Seo } from '../components/Seo';

interface ProviderExample {
  description: string;
  endpoint: string;
  chatModel?: string;
  imageModel?: string;
  videoModel?: string;
  speechModel?: string;
  transcriptionModel?: string;
  embeddingModel?: string;
  realtimeModel?: string;
  realtimeURL?: string;
  realtimeAPIKeyEnv?: string;
  realtimeVoice?: string;
  provides?: Array<{
    title: string;
    description: string;
  }>;
  notes?: string[];
}

const EXAMPLES: Record<string, ProviderExample> = {
  manifoldgen: {
    description: 'ManifoldGen is the first-party K-Fold video generator: a creator-focused ManifoldGen API for cinematic H3 video generation, with text-to-video and real gallery outputs.',
    endpoint: '/v1',
    videoModel: 'kfold-video',
    provides: [
      {
        title: 'K-Fold video generation',
        description: 'Generate cinematic clips through the ManifoldGen API. The K-Fold route is backed by ManifoldGen H3 video generation and supports prompt, aspect ratio, duration, steps, audio, and output format controls.',
      },
    ],
    notes: [
      'Use kfold-video with /v1/videos/generations through OpenPaths.',
      'ManifoldGen gallery examples on the homepage are real K-Fold/H3 outputs hosted by ManifoldGen.',
      'For the provider-native creator workflow, visit manifoldgen.com.',
    ],
  },
  openai: {
    description: 'OpenAI GPT-5, GPT Realtime voice, o3/o4 reasoning, GPT Image 2, Sora 2 video, and transcription — routed through OpenPaths.',
    endpoint: '/v1',
    chatModel: 'openai-chat-latest',
    imageModel: 'gpt-image-2',
    videoModel: 'sora-2',
    transcriptionModel: 'gpt-transcribe',
    realtimeModel: 'gpt-realtime-2.1-mini',
    realtimeURL: 'wss://openpaths.io/v1/realtime',
    realtimeAPIKeyEnv: 'OPENPATHS_API_KEY',
    realtimeVoice: 'marin',
    provides: [
      {
        title: 'GPT Live voice',
        description: 'Live speech-to-speech through OpenPaths’ authenticated WebSocket relay. GPT Realtime 2.1 Mini uses token pricing: text is $0.60/$2.40 and audio is $10/$20 per 1M input/output tokens.',
      },
    ],
    notes: [
      'gpt-image-2 returns base64 PNGs by default — decode with base64.b64decode.',
      'sora-2 is async; OpenPaths polls for you and returns a signed content URL.',
      'Use openai-coding-latest alias for gpt-5-codex.',
      '`gpt-realtime-2.1-mini` is the default low-latency voice route; use `gpt-realtime-2.1` when you need the flagship model.',
      'If the direct OpenAI GPT Image 2 path goes unhealthy, OpenPaths can fail over to Fal-hosted GPT Image 2 using the model-level circuit breaker.',
    ],
  },
  anthropic: {
    description: 'Claude Opus 4.7, Sonnet 4.6, Haiku 4.5. Native /v1/messages and /v1/chat/completions supported.',
    endpoint: '/v1',
    chatModel: 'claude-sonnet-latest',
    notes: [
      'Anthropic endpoints accept the same Bearer header — no x-api-key needed.',
      'Prefill works: pass a trailing assistant message.',
    ],
  },
  google: {
    description: 'Gemini 3.7 Flash, 2.5 Pro, 2.5 Flash, Flash Lite, plus Gemini embedding models.',
    endpoint: '/v1',
    chatModel: 'gemini-3.7-flash',
    embeddingModel: 'gemini-embedding-2-preview',
    notes: [
      'Pass image URLs as content parts for vision queries.',
      'OpenPaths exposes Google embedding models through the standard `/v1/embeddings` text-input path.',
      '`gemini-embedding-001` follows Google’s published $0.15 / 1M text-token pricing. OpenPaths currently prices the text path for `gemini-embedding-2-preview` at $0.20 / 1M tokens, while multimodal upstream rates are higher for image/audio/video.',
    ],
  },
  xai: {
    description: 'Grok 4.6, Grok 4.5, Grok Build, Grok 4.3/4.20, Grok Imagine, plus xAI realtime Voice, Text to Speech, and Speech to Text APIs.',
    endpoint: '/v1',
    chatModel: 'grok-latest',
    imageModel: 'grok-imagine-image',
    speechModel: 'xai-tts',
    transcriptionModel: 'xai-stt',
    realtimeModel: 'grok-voice-latest',
    realtimeURL: 'wss://api.x.ai/v1/realtime',
    realtimeAPIKeyEnv: 'XAI_API_KEY',
    realtimeVoice: 'eve',
    provides: [
      {
        title: 'Voice Agent API',
        description: 'Realtime speech-to-speech over xAI’s `/v1/realtime` WebSocket API. Use `grok-voice-latest`, or pin Think Fast 1.0 ($3.00/hour) or 2.0 ($4.80/hour). Text-only input events are $0.004 each.',
      },
      {
        title: 'Text to Speech',
        description: 'Generate speech from text through `/v1/tts` or `/v1/audio/speech` with five expressive voices: eve, ara, rex, sal, and leo. Priced at $15.00 per 1M input characters.',
      },
      {
        title: 'Speech to Text',
        description: 'Transcribe audio through `/v1/stt` or `/v1/audio/transcriptions` at the REST rate of $0.10/hour. xAI’s provider-native streaming STT endpoint is $0.20/hour.',
      },
      {
        title: 'Grok Imagine Image',
        description: 'Standard outputs are $0.02 each with $0.002 inputs. Quality outputs cost $0.05 at 1K or $0.07 at 2K, with $0.01 input images.',
      },
      {
        title: 'Grok Imagine Video',
        description: 'Video output is resolution-priced: Grok Imagine costs $0.05/sec at 480p or $0.07/sec at 720p; Video 1.5 costs $0.08/$0.14/$0.25 per second at 480p/720p/1080p.',
      },
      {
        title: 'Server-side Tools',
        description: 'xAI bills tokens plus invocations: web/X search and code execution are $5/1K calls, attachments $10/1K, and collections/file search $2.50/1K. Image/video understanding and remote MCP tools are token-priced.',
      },
    ],
    notes: [
      '`grok-latest` resolves automatically to the highest configured xAI Grok text model.',
      '`/v1/tts` defaults to `xai-tts`; `/v1/stt` defaults to `xai-stt`.',
      '`grok-voice-latest` follows xAI’s newest voice model; pin a versioned Think Fast ID when production behavior and pricing must remain stable.',
      'Realtime voice uses xAI’s WebSocket endpoint: `wss://api.x.ai/v1/realtime?model=grok-voice-latest`.',
      'Grok text prices double for the entire request once the prompt reaches 200K tokens.',
    ],
  },
  deepseek: {
    description: 'DeepSeek V3 Chat and Reasoner — frontier-level performance, extremely cheap.',
    endpoint: '/v1',
    chatModel: 'deepseek-chat',
  },
  cursor: {
    description: 'Cursor Composer 2.5 and Cursor Grok 4.5/4.6 via the Cursor Cloud Agents API — agentic coding and knowledge work exposed as standard OpenAI-style chat completions through OpenPaths.',
    endpoint: '/v1',
    chatModel: 'composer-2.5',
    provides: [
      {
        title: 'Composer 2.5',
        description: 'Cursor’s agentic coding model with tool use. Use `composer-2.5` for the standard tier and `composer-2.5-fast` for the low-latency tier.',
      },
      {
        title: 'Cursor Grok',
        description: 'Use `cursor-grok-4.5` or `cursor-grok-4.6` for standard reasoning, or append `-fast` for the low-latency variants.',
      },
      {
        title: 'OpenAI-Compatible Surface',
        description: 'OpenPaths runs Composer and Cursor Grok through the Cursor Cloud Agents API and returns OpenAI-style chat completion responses, so existing SDKs work unchanged.',
      },
    ],
    notes: [
      'Use `composer-2.5`, `composer-2.5-fast`, `cursor-grok-4.5`, `cursor-grok-4.5-fast`, `cursor-grok-4.6`, or `cursor-grok-4.6-fast` as the model name.',
      'Cursor Grok supports `low`, `medium`, `high`, and (for 4.6) `xhigh` reasoning effort through the standard `reasoning_effort` field.',
      'Best for coding agents, refactors, knowledge work, and tool-driven workflows — pass your tools array as usual.',
    ],
  },
  mistral: {
    description: 'Mistral Large, Medium, Codestral, Pixtral, Magistral, Devstral, Ministral, embeddings.',
    endpoint: '/v1',
    chatModel: 'mistral-large-latest',
    embeddingModel: 'mistral-embed',
  },
  groq: {
    description: 'Ultra-fast LPU inference. Llama 3.3, Mixtral, plus Whisper turbo transcription.',
    endpoint: '/v1',
    chatModel: 'llama-3.3-70b-versatile',
    transcriptionModel: 'whisper-large-v3-turbo',
  },
  together: {
    description: 'Qwen 3.8 2.4T, Qwen 3.5, Kimi, GLM, MiniMax, DeepSeek hosted on Together, plus FLUX image models.',
    endpoint: '/v1',
    chatModel: 'qwen3.5-397b',
    imageModel: 'flux-schnell',
  },
  openrouter: {
    description: 'Fallback gateway to 600+ models. Use `or/` prefix to target OpenRouter explicitly.',
    endpoint: '/v1',
    chatModel: 'or/gpt-5.4',
  },
  netwrck: {
    description: 'Creative media platform centered on RA1 images, ZImage anime art, and video tooling such as RA2V and LTX. OpenPaths maps that into a clean OpenAI-style images/videos surface.',
    endpoint: '/v1',
    imageModel: 'ra1',
    videoModel: 'ra2v',
    provides: [
      {
        title: 'RA1 Art Generator',
        description: 'Netwrck positions RA1 as its flagship text-to-image system for high-quality creative work, marketing visuals, and prompt-driven image generation.',
      },
      {
        title: 'Video Generation Stack',
        description: 'The site also highlights RA2V smart video plus LTX image-to-video and text-to-video flows for turning stills or prompts into short motion pieces.',
      },
      {
        title: 'Image And Anime Workflows',
        description: 'Related tools on netwrck.com include ZImage anime art, Flux Kontext, background removal, and image upscaling around the core generation pipeline.',
      },
      {
        title: 'OpenPaths Mapping',
        description: 'Inside OpenPaths, Netwrck is the first-party media lane for `ra1` image generation and `ra2v` video generation through `/v1/images/generations` and `/v1/videos/generations`.',
      },
    ],
    notes: [
      'The Netwrck site emphasizes creator workflows beyond raw inference: generation, editing, cleanup, and image-to-video conversion.',
      'Use `ra1` for images and `ra2v` for video when you want the first-party Netwrck path through OpenPaths.',
    ],
  },
  fal: {
    description: 'FLUX image generation, Smart Resize image recomposition, and ByteDance Seedance 2.0 text-to-video/reference-to-video through OpenPaths.',
    endpoint: '/v1',
    imageModel: 'flux-pro',
    videoModel: 'seedance-2.0-fast-text-to-video',
    notes: [
      '`seedance-2.0-fast-text-to-video` accepts prompt, resolution, duration, aspect_ratio, generate_audio, seed, and end_user_id.',
      '`seedance-2.0-image-to-video` accepts image_url plus optional end_image_url for start/end frame control.',
      '`seedance-2.0-reference-to-video` and `seedance-2.0-fast-reference-to-video` also accept image_urls, video_urls, and audio_urls. Reference them in prompts as @Image1, @Video1, and @Audio1.',
      'OpenPaths exposes these through `/v1/videos/generations`; you do not call Fal directly or send a Fal key.',
    ],
  },
  alibaba: {
    description: 'Alibaba Happy Horse image-to-video exposed through OpenPaths with OpenPaths billing and Fal infrastructure under the hood.',
    endpoint: '/v1',
    videoModel: 'alibaba/happy-horse/image-to-video',
    provides: [
      {
        title: 'Happy Horse Image to Video',
        description: 'Animate a still image into a 720p or 1080p video using prompt-guided motion, native audio, and lip-sync capable generation.',
      },
      {
        title: 'OpenPaths Gateway',
        description: 'Call `/v1/videos/generations` with an OpenPaths key. OpenPaths signs and polls the underlying Fal queue request for you.',
      },
    ],
    notes: [
      '`alibaba/happy-horse/image-to-video` accepts image_url, prompt, resolution, duration, seed, and enable_safety_checker.',
      'Duration can be sent as a number or string from 3 to 15 seconds.',
      'The default OpenPaths demo uses a mirrored Alibaba sample input and the corresponding Happy Horse MP4 output.',
    ],
  },
  exa: {
    description: 'Exa Search API through OpenPaths. Search the web with fast search modes, result categories, domain and date filters, highlights, full webpage text, structured outputs, and livecrawl freshness controls.',
    endpoint: '/v1',
    provides: [
      {
        title: 'Search API',
        description: 'POST `/v1/search` with an Exa-compatible JSON body. OpenPaths forwards the request to Exa and returns the original Exa response shape.',
      },
      {
        title: 'Search Types',
        description: 'Use `instant`, `fast`, `auto`, or `deep`. Auto is the recommended default for quality and latency balance.',
      },
      {
        title: 'Result Controls',
        description: 'Tune `numResults`, `category`, `includeDomains`, `excludeDomains`, `startPublishedDate`, `endPublishedDate`, and `userLocation`.',
      },
      {
        title: 'Content Extraction',
        description: 'Request token-efficient `highlights`, full `text`, or structured outputs inside the `contents` object.',
      },
    ],
    notes: [
      'Set `EXA_API_KEY` from https://exa.ai/account for OpenPaths platform routing.',
      'OpenPaths pricing adds 10% to Exa public rates: $0.0077 per search request for 1-10 results, $0.0011 per additional result beyond 10, and $0.0011 per requested content page.',
      'Users can store a provider key named `exa` for BYOK search requests.',
      'The dedicated UI lives at `/search`.',
    ],
  },
  papers: {
    description: 'Papers by Applied AI NZ through OpenPaths. Search papers, methods, datasets, and GitHub code from papers.app.nz with agent-friendly markdown output.',
    endpoint: '/v1',
    provides: [
      {
        title: 'Research Search',
        description: 'POST `/v1/search` with `provider: "papers"` to route to papers.app.nz search over papers, methods, datasets, or GitHub code.',
      },
      {
        title: 'Markdown For Agents',
        description: 'Set `format: "markdown"` to receive compact markdown search results designed for LLM and agent context windows.',
      },
      {
        title: 'Papers API Keys',
        description: 'Create an app.nz API key from `https://app.nz/account`, then use it as `APP_API_KEY` or store a BYOK provider key named `papers`.',
      },
      {
        title: 'Search Credits',
        description: 'Papers API credits cost $1 per 1,000 searches. OpenPaths prices the routed provider at $0.001 per search request.',
      },
    ],
    notes: [
      'Set `APP_API_KEY` from https://app.nz/account for OpenPaths platform routing.',
      'Supported `type` values are `papers`, `methods`, `datasets`, and `github_code`.',
      'Optional request fields include `sort: "recent"`, `hasCode: true`, `includeGithubCode: true`, and `format: "markdown"`.',
    ],
  },
  minimax: {
    description: 'MiniMax M2.5 chat, Hailuo 2.3 video, and Speech 2.8 HD TTS.',
    endpoint: '/v1',
    chatModel: 'minimax-m2.5-direct',
    videoModel: 'hailuo-2.3',
  },
  zai: {
    description: 'GLM-5.2, GLM-5.1, GLM-5, GLM-4.7, GLM-4.6v vision, and GLM Image generation. BYOK GLM Coding Plan keys are routed to z.ai’s coding endpoint (api.z.ai/api/coding/paas/v4) and circuit-break down the GLM series (5.2 → 5.1 → 5) on failure.',
    endpoint: '/v1',
    chatModel: 'glm-5.2',
    imageModel: 'glm-image',
  },
  fireworks: {
    description: 'Fast inference for GPT-OSS 120B, GLM-5 and Whisper v3 Large.',
    endpoint: '/v1',
    chatModel: 'fireworks/gpt-oss-120b',
    transcriptionModel: 'whisper-v3-large-turbo',
  },
  nvidia: {
    description: 'NVIDIA NIM hosts frontier open models on accelerated infra. OpenPaths wires each one into a circuit-breaker fallback chain with the original provider so healthy capacity is used first and traffic cuts over on saturation.',
    endpoint: '/v1',
    chatModel: 'nvidia/deepseek-v4-pro',
    provides: [
      {
        title: 'MiniMax M2.7',
        description: 'Latest MiniMax chat model (1M context). Aliases `minimax-m2.7` / `mm-m2.7`. Paired with `minimax-m2.5-direct` and Together-hosted `minimax-m2.5` for balancing.',
      },
      {
        title: 'DeepSeek V3.2',
        description: 'DeepSeek V3.2 with reasoning_content streaming. Aliases `deepseek-v3.2` / `deepseek-3.2`. Circuit-broken with `deepseek-chat`, Together V3.1, and OpenRouter.',
      },
      {
        title: 'DeepSeek V4 Pro',
        description: 'Free DeepSeek V4 Pro endpoint. Use `nvidia/deepseek-v4-pro` or alias `deepseek-v4-pro-free`; OpenPaths sends NVIDIA `chat_template_kwargs` with thinking and high reasoning enabled.',
      },
      {
        title: 'Devstral 2 123B',
        description: 'Mistral Devstral 2 123B-instruct, tuned for code. Aliases `devstral-2` / `devstral-2-123b`. Balanced with Mistral `devstral-medium-latest` and `codestral-latest`.',
      },
    ],
    notes: [
      'Health-tracker circuit breakers mark each provider+model key unhealthy on 5xx/429 with a 30s–2min exponential cooldown; traffic automatically shifts to the next healthy candidate.',
      'Use the direct `nvidia/*` IDs to force NVIDIA routing; use the short aliases (e.g. `deepseek-v3.2`) when you want the full balanced pool.',
      'NVIDIA DeepSeek routes use `chat_template_kwargs={"thinking": true, "reasoning_effort": "high"}` for reasoning_content streaming when called directly.',
    ],
  },
  nous: {
    description: 'Hermes 4 70B and 405B — open reasoning models with tool use at low cost.',
    endpoint: '/v1',
    chatModel: 'hermes-4-405b',
  },
  'text-generator': {
    description: 'Text-Generator.io presents itself as a unified API for text, vision, and speech with privacy-first infrastructure. In OpenPaths today, the direct provider integration is its ModernBERT embedding capability.',
    endpoint: '/v1',
    embeddingModel: 'text-embedding',
    provides: [
      {
        title: 'ModernBERT Embeddings',
        description: 'OpenPaths uses Text-Generator.io for first-party embedding generation tuned for retrieval, semantic search, similarity scoring, and RAG pipelines.',
      },
      {
        title: 'Unified AI API',
        description: 'On text-generator.io, the broader product pitch is one API for text, speech, and vision workloads including chat, OCR-style analysis, summarization, and synthesis.',
      },
      {
        title: 'Privacy-First Positioning',
        description: 'The site repeatedly frames the platform as privacy-focused with lower operating cost and predictable infrastructure for production use cases.',
      },
      {
        title: 'Playground And Tooling',
        description: 'Text-Generator.io also ships a playground plus packaged tools like prompt optimization, image captioning, speech workflows, and domain/content helpers around the API.',
      },
    ],
    notes: [
      'OpenPaths currently exposes the Text-Generator.io integration as an embeddings provider rather than mirroring the full standalone site product surface.',
      'If you want a first-party embedding path for search or RAG, `text-embedding` is the model to start with.',
    ],
  },
  openpaths: {
    description: 'First-party OpenPaths Auto models (openpaths/auto and variants) plus our local gobed embedding router.',
    endpoint: '/v1',
    chatModel: 'auto',
    embeddingModel: 'openpaths-embed',
    notes: [
      '`openpaths-embed` is billed per request and optimized for cost-effective local embeddings.',
      '`reasoning_effort="auto"` can be used on any thinking-capable direct chat model to keep that model while auto-selecting none, low, medium, or high thinking.',
      'Use `long_text_mode=average_chunks` when you want full-text averaging on longer inputs.',
    ],
  },
};

export function ProviderDocs() {
  const { slug = '' } = useParams<{ slug: string }>();
  const [apiKey, setApiKey] = useState(() => getStoredAPIKey());
  const [copied, setCopied] = useState<string | null>(null);

  useEffect(() => {
    const sync = () => setApiKey(getStoredAPIKey());
    window.addEventListener('storage', sync);
    return () => window.removeEventListener('storage', sync);
  }, []);

  const provider = providers.find(p => p.slug === slug);
  const example = EXAMPLES[slug];
  const apiBase = getAPIBaseURL();
  const exampleKey = apiKey || 'op-...';
  const providerName = provider?.name || slug;
  const providerModels = useMemo(
    () => models.filter(m => m.provider === providerName),
    [providerName],
  );

  // Best model to prefill the Playground with: the documented default, else the
  // provider's first listed model. Search providers (exa/papers) have no chat model.
  const playgroundModel =
    example?.chatModel ||
    example?.imageModel ||
    example?.videoModel ||
    providerModels[0]?.id ||
    '';

  const snippets = useMemo(
    () => buildSnippets(apiBase, exampleKey, example),
    [apiBase, exampleKey, example],
  );

  if (!provider && !example) {
    return <Navigate to="/docs" replace />;
  }

  const copy = async (key: string, text: string) => {
    await navigator.clipboard.writeText(text);
    setCopied(key);
    window.setTimeout(() => setCopied(null), 1500);
  };

  return (
    <>
      <Seo
        title={`${providerName} API Docs | OpenPaths`}
        description={example?.description || provider?.description || `${providerName} API examples and model documentation for OpenPaths.`}
        path={`/${slug}/docs`}
      />

      <section className="max-w-6xl mx-auto px-6 py-16">
      <div className="mb-6">
        <Link to="/docs" className="inline-flex items-center gap-1.5 text-xs font-mono text-white/50 hover:text-white transition-colors">
          <ArrowLeft className="w-3.5 h-3.5" /> All docs
        </Link>
      </div>

      <div className="mb-10 flex flex-col md:flex-row md:items-start md:justify-between gap-6">
        <div className="flex items-start gap-4">
          {provider && (
            <img
              src={provider.logo || FALLBACK_LOGO}
              srcSet={provider.logoSrcSet}
              sizes="48px"
              alt=""
              className={`w-12 h-12 rounded-lg border border-white/20 p-2 object-contain ${provider.slug === 'black-forest-labs' ? 'bg-white' : 'bg-white/[0.07]'}`}
            />
          )}
          <div>
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border border-white/20 bg-white/[0.07] text-xs font-mono text-white/60 mb-3">
              <BookOpen className="w-3.5 h-3.5" />
              {providerName} Docs
            </div>
            <h1 className="text-4xl md:text-5xl font-bold tracking-tight mb-3 capitalize">{providerName} via OpenPaths</h1>
            <p className="text-white/60 max-w-3xl font-light leading-relaxed">
              {example?.description || (provider?.description ?? 'Provider docs.')}
            </p>
          </div>
        </div>
        {provider && provider.url !== '/' && (
          <a
            href={provider.url}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-1.5 px-3 py-2 text-xs font-mono text-white/60 border border-white/20 rounded-lg hover:text-white hover:border-white/50 transition-colors self-start"
          >
            <ExternalLink className="w-3.5 h-3.5" /> {providerName} website
          </a>
        )}
      </div>

      <div className="flex flex-wrap items-center gap-3 mb-8">
        {slug === 'exa' || slug === 'papers' ? (
          <Link
            to="/search"
            className="inline-flex items-center gap-2 bg-white text-black px-4 py-2.5 rounded-lg text-sm font-mono font-bold hover:bg-white/90 transition-colors"
            data-testid="provider-docs-try"
          >
            <Play className="w-4 h-4" /> Try {providerName} Search
          </Link>
        ) : playgroundModel ? (
          <Link
            to={`/playground?model=${encodeURIComponent(playgroundModel)}`}
            className="inline-flex items-center gap-2 bg-white text-black px-4 py-2.5 rounded-lg text-sm font-mono font-bold hover:bg-white/90 transition-colors"
            data-testid="provider-docs-try"
          >
            <Play className="w-4 h-4" /> Test {providerName} in the Playground
          </Link>
        ) : (
          <Link
            to="/playground"
            className="inline-flex items-center gap-2 bg-white text-black px-4 py-2.5 rounded-lg text-sm font-mono font-bold hover:bg-white/90 transition-colors"
            data-testid="provider-docs-try"
          >
            <Play className="w-4 h-4" /> Open the Playground
          </Link>
        )}
        {!apiKey && (
          <Link
            to="/account"
            className="inline-flex items-center gap-2 border border-white/15 px-4 py-2.5 rounded-lg text-sm font-mono text-white/80 hover:text-white hover:border-white/50 transition-colors"
            data-testid="provider-docs-get-key"
          >
            <KeyRound className="w-4 h-4" /> Sign up for a free API key
          </Link>
        )}
      </div>

      <div className="border border-white/20 bg-white/[0.05] rounded-2xl p-6 mb-8">
        <div className="text-xs font-mono text-white/55 mb-1">Base URL</div>
        <code className="text-sm text-white/80" data-testid="provider-docs-base-url">{apiBase}</code>
        {example?.notes && example.notes.length > 0 && (
          <div className="mt-4">
            <div className="text-xs font-mono text-white/55 mb-2">Notes</div>
            <ul className="list-disc list-inside space-y-1 text-sm text-white/70">
              {example.notes.map((n, i) => <li key={i}>{n}</li>)}
            </ul>
          </div>
        )}
      </div>

      {example?.provides && example.provides.length > 0 && (
        <div className="mb-8">
          <div className="mb-4">
            <h2 className="text-xl font-bold tracking-tight mb-1">What {providerName} provides</h2>
            <p className="text-sm text-white/55 font-light">
              Summary based on the provider site plus the OpenPaths integration surface.
            </p>
          </div>
          <div className="grid gap-4 md:grid-cols-2">
            {example.provides.map((item) => (
              <div key={item.title} className="rounded-2xl border border-white/20 bg-white/[0.05] p-5">
                <h3 className="text-base font-semibold tracking-tight mb-2">{item.title}</h3>
                <p className="text-sm text-white/60 font-light leading-relaxed">{item.description}</p>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="space-y-8">
        {snippets.map(snip => (
          <div key={snip.title} className="border border-white/20 bg-white/[0.05] rounded-2xl p-6">
            <div className="flex items-center justify-between mb-3 gap-3 flex-wrap">
              <div>
                <h2 className="text-xl font-bold tracking-tight">{snip.title}</h2>
                <p className="text-sm text-white/60 font-light mt-1">{snip.description}</p>
              </div>
              <button
                onClick={() => copy(snip.title, snip.curl)}
                className="inline-flex items-center gap-2 border border-white/20 px-3 py-2 rounded-lg text-xs font-mono text-white/70 hover:text-white hover:border-white/40 transition-colors"
              >
                {copied === snip.title ? <Check className="w-3.5 h-3.5 text-green-400" /> : <Copy className="w-3.5 h-3.5" />}
                {copied === snip.title ? 'Copied' : 'Copy cURL'}
              </button>
            </div>
            <div className="grid gap-4 md:grid-cols-2">
              <div>
                <div className="text-xs font-mono text-white/55 mb-2">Python</div>
                <CodeBlock
                  code={snip.python}
                  language="python"
                  preClassName="rounded-xl border border-white/20 bg-black/60 p-4 text-xs leading-6"
                />
              </div>
              <div>
                <div className="text-xs font-mono text-white/55 mb-2">cURL</div>
                <CodeBlock
                  code={snip.curl}
                  language="bash"
                  preClassName="rounded-xl border border-white/20 bg-black/60 p-4 text-xs leading-6"
                />
              </div>
            </div>
          </div>
        ))}
      </div>

      {providerModels.length > 0 && (
        <div className="border border-white/20 bg-white/[0.05] rounded-2xl p-6 mt-10">
          <h2 className="text-xl font-bold tracking-tight mb-3">Available models on {providerName}</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {providerModels.map(m => (
              <Link
                key={m.id}
                to={`/playground?model=${encodeURIComponent(m.id)}`}
                className="rounded-xl border border-white/20 px-4 py-3 hover:border-white/50 transition-colors flex items-center justify-between gap-3"
              >
                <div className="min-w-0">
                  <div className="text-sm font-mono text-white truncate">{m.name}</div>
                  <div className="text-xs text-white/50 truncate">{m.description}</div>
                </div>
                <span className="text-[10px] font-mono text-white/55 shrink-0">{m.contextLength}</span>
              </Link>
            ))}
          </div>
        </div>
      )}
      </section>
    </>
  );
}

interface Snippet {
  title: string;
  description: string;
  python: string;
  curl: string;
}

function buildSnippets(apiBase: string, key: string, ex?: ProviderExample): Snippet[] {
  const out: Snippet[] = [];
  if (ex?.chatModel) {
    out.push({
      title: 'Chat Completion',
      description: `Default chat model: ${ex.chatModel}`,
      python: `from openai import OpenAI

client = OpenAI(base_url="${apiBase}", api_key="${key}")

completion = client.chat.completions.create(
    model="${ex.chatModel}",
    messages=[{"role": "user", "content": "Hello!"}],
)

print(completion.choices[0].message.content)`,
      curl: `curl ${apiBase}/chat/completions \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${key}" \\
  -d '{
    "model": "${ex.chatModel}",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'`,
    });
  }
  if (ex?.imageModel) {
    out.push({
      title: 'Image Generation',
      description: `Default image model: ${ex.imageModel}`,
      python: `from openai import OpenAI
import base64, pathlib, requests

client = OpenAI(base_url="${apiBase}", api_key="${key}")

img = client.images.generate(
    model="${ex.imageModel}",
    prompt="A photo of a beige ceramic coffee mug on a wooden table",
    size="1024x1024",
    n=1,
)

image = img.data[0]
if image.b64_json:
    pathlib.Path("out.png").write_bytes(base64.b64decode(image.b64_json))
else:
    pathlib.Path("out.png").write_bytes(requests.get(image.url).content)`,
      curl: `curl ${apiBase}/images/generations \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${key}" \\
  -d '{
    "model": "${ex.imageModel}",
    "prompt": "A photo of a beige ceramic coffee mug on a wooden table",
    "size": "1024x1024",
    "n": 1
  }'`,
    });
    if (ex.imageModel === 'grok-imagine-image') {
      out.push({
        title: 'Image Edit',
        description: 'Edit one or more source images with Grok Imagine.',
        python: `import requests

resp = requests.post(
    "${apiBase}/images/edits",
    headers={
        "Authorization": "Bearer ${key}",
        "Content-Type": "application/json",
    },
    json={
        "model": "grok-imagine-image",
        "prompt": "Show all subjects sitting together on the grass in a sunny park.",
        "images": [
            {"type": "image_url", "url": "https://docs.x.ai/assets/api-examples/images/image-merge/woman.jpg"},
            {"type": "image_url", "url": "https://docs.x.ai/assets/api-examples/images/image-merge/man.jpg"},
        ],
        "aspect_ratio": "3:2",
    },
)
resp.raise_for_status()
print(resp.json()["data"][0]["url"])`,
        curl: `curl ${apiBase}/images/edits \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${key}" \\
  -d '{
    "model": "grok-imagine-image",
    "prompt": "Show all subjects sitting together on the grass in a sunny park.",
    "images": [
      { "type": "image_url", "url": "https://docs.x.ai/assets/api-examples/images/image-merge/woman.jpg" },
      { "type": "image_url", "url": "https://docs.x.ai/assets/api-examples/images/image-merge/man.jpg" }
    ],
    "aspect_ratio": "3:2"
  }'`,
      });
    }
  }
  if (ex?.videoModel) {
    const isHappyHorse = ex.videoModel === 'alibaba/happy-horse/image-to-video';
    const videoPayload = isHappyHorse
      ? `{
        "model": "${ex.videoModel}",
        "image_url": "https://openpathsstatic.openpaths.io/static/uploads/playground/happy-horse/rap.png",
        "prompt": "Bring the scene in the image to life.",
        "resolution": "1080p",
        "duration": 5,
        "enable_safety_checker": true,
    }`
      : `{
        "model": "${ex.videoModel}",
        "prompt": "A cinematic fly-through of a neon city at night",
        "resolution": "720p",
        "duration": "5",
        "aspect_ratio": "16:9",
    }`;
    out.push({
      title: 'Video Generation',
      description: `Default video model: ${ex.videoModel}`,
      python: `import json
import httpx

resp = httpx.post(
    "${apiBase}/videos/generations",
    headers={"Authorization": "Bearer ${key}"},
    json=json.loads(r'''${videoPayload}'''),
    timeout=900,
)

print(resp.json()["video_url"])`,
      curl: `curl ${apiBase}/videos/generations \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${key}" \\
  -d '${videoPayload.replaceAll('\n    ', '\n  ')}'`,
    });
  }
  if (ex?.speechModel) {
    out.push({
      title: 'Text To Speech',
      description: `Default speech model: ${ex.speechModel}`,
      python: `import base64
import requests

resp = requests.post(
    "${apiBase}/tts",
    headers={
        "Authorization": "Bearer ${key}",
        "Content-Type": "application/json",
    },
    json={
        "text": "Hello from xAI text to speech.",
        "voice_id": "eve",
        "language": "en",
    },
)
resp.raise_for_status()

audio = resp.json()["audio"]
open("hello.mp3", "wb").write(base64.b64decode(audio))`,
      curl: `curl ${apiBase}/tts \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${key}" \\
  -d '{
    "text": "Hello from xAI text to speech.",
    "voice_id": "eve",
    "language": "en"
  }'`,
    });
  }
  if (ex?.transcriptionModel) {
    out.push({
      title: 'Transcription',
      description: `Default transcription model: ${ex.transcriptionModel}`,
      python: `from openai import OpenAI

client = OpenAI(base_url="${apiBase}", api_key="${key}")

with open("meeting.mp3", "rb") as f:
    transcript = client.audio.transcriptions.create(
        model="${ex.transcriptionModel}",
        file=f,
    )

print(transcript.text)`,
      curl: `curl ${apiBase}/audio/transcriptions \\
  -H "Authorization: Bearer ${key}" \\
  -F model=${ex.transcriptionModel} \\
  -F file=@meeting.mp3`,
    });
  }
  if (ex?.realtimeModel) {
    const realtimeURL = ex.realtimeURL || 'wss://api.x.ai/v1/realtime';
    const realtimeAPIKeyEnv = ex.realtimeAPIKeyEnv || 'XAI_API_KEY';
    const realtimeVoice = ex.realtimeVoice || 'eve';
    const realtimeSession = ex.realtimeModel.startsWith('gpt-realtime')
      ? `{
            "type": "session.update",
            "session": {
                "type": "realtime",
                "model": "${ex.realtimeModel}",
                "output_modalities": ["audio"],
                "instructions": "You are a concise voice assistant.",
                "audio": {
                    "input": {
                        "format": {"type": "audio/pcm", "rate": 24000},
                        "turn_detection": {"type": "semantic_vad"},
                    },
                    "output": {
                        "format": {"type": "audio/pcm"},
                        "voice": "${realtimeVoice}",
                    },
                },
            },
        }`
      : `{
            "type": "session.update",
            "session": {
                "voice": "${realtimeVoice}",
                "instructions": "You are a concise voice assistant.",
                "turn_detection": {"type": "server_vad"},
            },
        }`;
    out.push({
      title: 'Realtime Voice',
      description: `Default realtime model: ${ex.realtimeModel}`,
      python: `import asyncio
import json
import os
import websockets

async def main():
    async with websockets.connect(
        "${realtimeURL}?model=${ex.realtimeModel}",
        additional_headers={"Authorization": f"Bearer {os.environ['${realtimeAPIKeyEnv}']}"},
    ) as ws:
        await ws.send(json.dumps(${realtimeSession}))
        await ws.send(json.dumps({
            "type": "conversation.item.create",
            "item": {
                "type": "message",
                "role": "user",
                "content": [{"type": "input_text", "text": "Hello!"}],
            },
        }))
        await ws.send(json.dumps({"type": "response.create"}))
        async for message in ws:
            print(json.loads(message)["type"])

asyncio.run(main())`,
      curl: `# Realtime voice is a WebSocket endpoint.
# Use an OpenPaths API key with a WebSocket client and connect to:
${realtimeURL}?model=${ex.realtimeModel}`,
    });
  }
  if (ex?.embeddingModel) {
    out.push({
      title: 'Embeddings',
      description: `Default embedding model: ${ex.embeddingModel}`,
      python: `from openai import OpenAI

client = OpenAI(base_url="${apiBase}", api_key="${key}")

resp = client.embeddings.create(
    model="${ex.embeddingModel}",
    input="the quick brown fox",
)

print(len(resp.data[0].embedding))`,
      curl: `curl ${apiBase}/embeddings \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${key}" \\
  -d '{
    "model": "${ex.embeddingModel}",
    "input": "the quick brown fox"
  }'`,
    });
  }
  if (ex && ex === EXAMPLES.exa) {
    out.push({
      title: 'Search',
      description: 'Run an Exa web search through OpenPaths.',
      python: `import requests

resp = requests.post(
    "${apiBase}/search",
    headers={
        "Authorization": "Bearer ${key}",
        "Content-Type": "application/json",
    },
    json={
        "query": "Latest news on Nvidia",
        "numResults": 10,
        "type": "auto",
        "contents": {
            "highlights": True
        },
    },
)
resp.raise_for_status()
print(resp.json()["results"][0]["title"])`,
      curl: `curl ${apiBase}/search \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${key}" \\
  -d '{
    "query": "Latest news on Nvidia",
    "numResults": 10,
    "type": "auto",
    "contents": {
      "highlights": true
    }
  }'`,
    });
  }
  if (ex && ex === EXAMPLES.papers) {
    out.push({
      title: 'Research Search',
      description: 'Search papers.app.nz through OpenPaths.',
      python: `import requests

resp = requests.post(
    "${apiBase}/search",
    headers={
        "Authorization": "Bearer ${key}",
        "Content-Type": "application/json",
    },
    json={
        "provider": "papers",
        "query": "diffusion transformers",
        "numResults": 5,
        "type": "papers",
        "format": "markdown",
        "hasCode": True,
    },
)
resp.raise_for_status()
print(resp.text)`,
      curl: `curl ${apiBase}/search \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${key}" \\
  -d '{
    "provider": "papers",
    "query": "diffusion transformers",
    "numResults": 5,
    "type": "papers",
    "format": "markdown",
    "hasCode": true
  }'`,
    });
  }
  return out;
}
