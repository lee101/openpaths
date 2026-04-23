import React, { useEffect, useMemo, useState } from 'react';
import { Link, Navigate, useParams } from 'react-router-dom';
import { BookOpen, Copy, Check, ExternalLink, ArrowLeft } from 'lucide-react';
import { providers, FALLBACK_LOGO } from '../data/providers';
import { models } from '../data/models';
import { getAPIBaseURL, getStoredAPIKey } from '../lib/session';
import { CodeBlock } from '../components/CodeBlock';

interface ProviderExample {
  description: string;
  endpoint: string;
  chatModel?: string;
  imageModel?: string;
  videoModel?: string;
  transcriptionModel?: string;
  embeddingModel?: string;
  provides?: Array<{
    title: string;
    description: string;
  }>;
  notes?: string[];
}

const EXAMPLES: Record<string, ProviderExample> = {
  openai: {
    description: 'OpenAI GPT-5, GPT-4o, o3/o4 reasoning, GPT Image 2, Sora 2 video, and Whisper transcription — routed through OpenPaths.',
    endpoint: '/v1',
    chatModel: 'openai-chat-latest',
    imageModel: 'gpt-image-2',
    videoModel: 'sora-2',
    transcriptionModel: 'gpt-4o-transcribe',
    notes: [
      'gpt-image-2 returns base64 PNGs by default — decode with base64.b64decode.',
      'sora-2 is async; OpenPaths polls for you and returns a signed content URL.',
      'Use openai-coding-latest alias for gpt-5-codex.',
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
    description: 'Gemini 3.1 Pro, 2.5 Pro, 2.5 Flash, Flash Lite, plus Gemini embedding models.',
    endpoint: '/v1',
    chatModel: 'gemini-3.1-pro-preview',
    embeddingModel: 'gemini-embedding-2-preview',
    notes: [
      'Pass image URLs as content parts for vision queries.',
      'OpenPaths exposes Google embedding models through the standard `/v1/embeddings` text-input path.',
      '`gemini-embedding-001` follows Google’s published $0.15 / 1M text-token pricing. OpenPaths currently prices the text path for `gemini-embedding-2-preview` at $0.20 / 1M tokens, while multimodal upstream rates are higher for image/audio/video.',
    ],
  },
  xai: {
    description: 'Grok 4, Grok 4.1 Fast Reasoning (2M context), Grok 3 Mini.',
    endpoint: '/v1',
    chatModel: 'grok-4-0709',
  },
  deepseek: {
    description: 'DeepSeek V3 Chat and Reasoner — frontier-level performance, extremely cheap.',
    endpoint: '/v1',
    chatModel: 'deepseek-chat',
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
    description: 'Qwen, Kimi, GLM, MiniMax, DeepSeek hosted on Together, plus FLUX image models.',
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
    description: 'FLUX Klein 4B, FLUX Schnell, FLUX Dev, FLUX Pro — fast serverless image generation.',
    endpoint: '/v1',
    imageModel: 'flux-pro',
  },
  minimax: {
    description: 'MiniMax M2.5 chat, Hailuo 2.3 video, and Speech 2.8 HD TTS.',
    endpoint: '/v1',
    chatModel: 'minimax-m2.5-direct',
    videoModel: 'hailuo-2.3',
  },
  zai: {
    description: 'GLM-5, GLM-4.7, GLM-4.6v vision, and GLM Image generation.',
    endpoint: '/v1',
    chatModel: 'zai/glm-5.1',
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
    chatModel: 'nvidia/minimax-m2.7',
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
        title: 'Devstral 2 123B',
        description: 'Mistral Devstral 2 123B-instruct, tuned for code. Aliases `devstral-2` / `devstral-2-123b`. Balanced with Mistral `devstral-medium-latest` and `codestral-latest`.',
      },
    ],
    notes: [
      'Health-tracker circuit breakers mark each provider+model key unhealthy on 5xx/429 with a 30s–2min exponential cooldown; traffic automatically shifts to the next healthy candidate.',
      'Use the direct `nvidia/*` IDs to force NVIDIA routing; use the short aliases (e.g. `deepseek-v3.2`) when you want the full balanced pool.',
      'DeepSeek V3.2 supports `chat_template_kwargs={"thinking": true}` for reasoning_content streaming when called against nvidia/deepseek-v3.2 directly.',
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
    description: 'First-party auto-routing tiers (auto, auto-easy, auto-medium, auto-think) plus the gobed embedding model.',
    endpoint: '/v1',
    chatModel: 'auto',
    embeddingModel: 'openpaths-embed',
    notes: [
      '`openpaths-embed` is billed per request rather than per token.',
      'Long inputs default to truncation; set `long_text_mode=average_chunks` if you want OpenPaths to average chunk embeddings across the full text.',
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
              alt=""
              className="w-12 h-12 rounded-lg border border-white/10 bg-white/[0.04] p-2 object-contain"
            />
          )}
          <div>
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border border-white/10 bg-white/[0.04] text-xs font-mono text-white/60 mb-3">
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
            className="inline-flex items-center gap-1.5 px-3 py-2 text-xs font-mono text-white/60 border border-white/10 rounded-lg hover:text-white hover:border-white/30 transition-colors self-start"
          >
            <ExternalLink className="w-3.5 h-3.5" /> {providerName} website
          </a>
        )}
      </div>

      <div className="border border-white/10 bg-white/[0.02] rounded-2xl p-6 mb-8">
        <div className="text-xs font-mono text-white/40 mb-1">Base URL</div>
        <code className="text-sm text-white/80" data-testid="provider-docs-base-url">{apiBase}</code>
        {example?.notes && example.notes.length > 0 && (
          <div className="mt-4">
            <div className="text-xs font-mono text-white/40 mb-2">Notes</div>
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
              <div key={item.title} className="rounded-2xl border border-white/10 bg-white/[0.02] p-5">
                <h3 className="text-base font-semibold tracking-tight mb-2">{item.title}</h3>
                <p className="text-sm text-white/60 font-light leading-relaxed">{item.description}</p>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="space-y-8">
        {snippets.map(snip => (
          <div key={snip.title} className="border border-white/10 bg-white/[0.02] rounded-2xl p-6">
            <div className="flex items-center justify-between mb-3 gap-3 flex-wrap">
              <div>
                <h2 className="text-xl font-bold tracking-tight">{snip.title}</h2>
                <p className="text-sm text-white/60 font-light mt-1">{snip.description}</p>
              </div>
              <button
                onClick={() => copy(snip.title, snip.curl)}
                className="inline-flex items-center gap-2 border border-white/10 px-3 py-2 rounded-lg text-xs font-mono text-white/70 hover:text-white hover:border-white/20 transition-colors"
              >
                {copied === snip.title ? <Check className="w-3.5 h-3.5 text-green-400" /> : <Copy className="w-3.5 h-3.5" />}
                {copied === snip.title ? 'Copied' : 'Copy cURL'}
              </button>
            </div>
            <div className="grid gap-4 md:grid-cols-2">
              <div>
                <div className="text-xs font-mono text-white/40 mb-2">Python</div>
                <CodeBlock
                  code={snip.python}
                  language="python"
                  preClassName="rounded-xl border border-white/10 bg-black/60 p-4 text-xs leading-6"
                />
              </div>
              <div>
                <div className="text-xs font-mono text-white/40 mb-2">cURL</div>
                <CodeBlock
                  code={snip.curl}
                  language="bash"
                  preClassName="rounded-xl border border-white/10 bg-black/60 p-4 text-xs leading-6"
                />
              </div>
            </div>
          </div>
        ))}
      </div>

      {providerModels.length > 0 && (
        <div className="border border-white/10 bg-white/[0.02] rounded-2xl p-6 mt-10">
          <h2 className="text-xl font-bold tracking-tight mb-3">Available models on {providerName}</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {providerModels.map(m => (
              <Link
                key={m.id}
                to={`/playground?model=${encodeURIComponent(m.id)}`}
                className="rounded-xl border border-white/10 px-4 py-3 hover:border-white/30 transition-colors flex items-center justify-between gap-3"
              >
                <div className="min-w-0">
                  <div className="text-sm font-mono text-white truncate">{m.name}</div>
                  <div className="text-xs text-white/50 truncate">{m.description}</div>
                </div>
                <span className="text-[10px] font-mono text-white/40 shrink-0">{m.contextLength}</span>
              </Link>
            ))}
          </div>
        </div>
      )}
    </section>
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
import base64, pathlib

client = OpenAI(base_url="${apiBase}", api_key="${key}")

img = client.images.generate(
    model="${ex.imageModel}",
    prompt="A photo of a beige ceramic coffee mug on a wooden table",
    size="1024x1024",
    n=1,
)

pathlib.Path("out.png").write_bytes(base64.b64decode(img.data[0].b64_json))`,
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
  }
  if (ex?.videoModel) {
    out.push({
      title: 'Video Generation',
      description: `Default video model: ${ex.videoModel}`,
      python: `import httpx

resp = httpx.post(
    "${apiBase}/videos/generations",
    headers={"Authorization": "Bearer ${key}"},
    json={
        "model": "${ex.videoModel}",
        "prompt": "A cinematic fly-through of a neon city at night",
        "resolution": "1280x720",
        "num_frames": 48,
        "frames_per_second": 24,
    },
    timeout=900,
)

print(resp.json()["video_url"])`,
      curl: `curl ${apiBase}/videos/generations \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${key}" \\
  -d '{
    "model": "${ex.videoModel}",
    "prompt": "A cinematic fly-through of a neon city at night",
    "resolution": "1280x720",
    "num_frames": 48,
    "frames_per_second": 24
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
  return out;
}
