import React, { useEffect, useMemo, useState } from 'react';
import { BookOpen, Copy, Check, KeyRound, Play, ArrowRight } from 'lucide-react';
import { Link } from 'react-router-dom';
import { getAPIBaseURL, getStoredAPIKey } from '../lib/session';
import { CodeBlock } from '../components/CodeBlock';
import { Seo } from '../components/Seo';
import { providers, FALLBACK_LOGO } from '../data/providers';

const ENDPOINTS = [
  { method: 'POST', path: '/v1/chat/completions', description: 'OpenAI-compatible chat completions (streaming + tools).' },
  { method: 'POST', path: '/v1/messages', description: 'Anthropic-compatible Messages API for Claude SDKs and agent SDKs.' },
  { method: 'GET', path: '/v1/models', description: 'List all available models and capabilities.' },
  { method: 'POST', path: '/v1/images/generations', description: 'Generate images (GPT Image 2, Flux, RA1, Klein, and more).' },
  { method: 'POST', path: '/v1/images/edits', description: 'Edit images and image-to-image workflows (GPT Image 2, Grok Imagine Image).' },
  { method: 'POST', path: '/v1/3d/generations', description: 'Generate textured GLB models from image URLs with Pixal3D.' },
  { method: 'POST', path: '/v1/3d/text-generations', description: 'Generate a GLB straight from a text prompt (auto image then Pixal3D).' },
  { method: 'POST', path: '/v1/3d/rigging', description: 'Auto-rig a humanoid GLB into a rigged character (GLB + FBX) with Meshy.' },
  { method: 'POST', path: '/v1/3d/generations', description: 'Also retextures a mesh: pass model trellis-2-retexture with mesh_url + image_url.' },
  { method: 'POST', path: '/v1/videos/generations', description: 'Generate videos (Sora 2, Hailuo, Wan, LTX).' },
  { method: 'POST', path: '/v1/audio/transcriptions', description: 'Transcribe speech to text (Whisper, GPT-4o Transcribe).' },
  { method: 'POST', path: '/v1/audio/speech', description: 'Text-to-speech (xAI TTS, MiniMax Speech 2.8 HD).' },
  { method: 'POST', path: '/v1/stt', description: 'xAI Speech to Text shortcut; defaults to xai-stt.' },
  { method: 'POST', path: '/v1/tts', description: 'xAI Text to Speech shortcut; defaults to xai-tts.' },
  { method: 'POST', path: '/v1/embeddings', description: 'Generate vector embeddings (OpenPaths, Google Gemini, Mistral, Nemotron).' },
];

type Tab = 'chat' | 'images' | '3d' | 'text-to-3d' | 'videos' | 'transcription';

const TAB_LABELS: Record<Tab, string> = {
  chat: 'Chat',
  images: 'Images',
  '3d': '3D',
  'text-to-3d': 'Text to 3D',
  videos: 'Videos',
  transcription: 'Transcription',
};

export function Docs() {
  const [apiKey, setApiKey] = useState(() => getStoredAPIKey());
  const [copied, setCopied] = useState(false);
  const [tab, setTab] = useState<Tab>('chat');
  const apiBase = getAPIBaseURL();
  const exampleKey = apiKey || 'op-...';

  useEffect(() => {
    const sync = () => setApiKey(getStoredAPIKey());
    window.addEventListener('storage', sync);
    return () => window.removeEventListener('storage', sync);
  }, []);

  const samples = useMemo(() => buildSamples(apiBase, exampleKey), [apiBase, exampleKey]);
  const active = samples[tab];

  const copyExample = async () => {
    await navigator.clipboard.writeText(active.curl);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1500);
  };

  return (
    <>
      <Seo
        title="OpenPaths API Docs | OpenAI-Compatible AI Gateway"
        description="OpenPaths API docs for chat completions, images, videos, speech, transcription, embeddings, provider routing, and OpenAI-compatible SDK setup."
        path="/docs"
      />

      <section className="max-w-6xl mx-auto px-6 py-16">
      <div className="mb-12">
        <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border border-white/10 bg-white/[0.04] text-xs font-mono text-white/60 mb-6">
          <BookOpen className="w-3.5 h-3.5" />
          API Docs
        </div>
        <h1 className="text-4xl md:text-5xl font-bold tracking-tight mb-4">OpenAI And Anthropic SDK Compatible Docs</h1>
        <p className="text-white/60 max-w-3xl font-light leading-relaxed mb-6">
          Use the same base URL pattern for chat, images, video, music, speech, and models from either SDK. If you are signed in on this device,
          your API key is injected into the examples automatically. For framework-specific setup, see the <Link to="/integrations" className="text-white underline underline-offset-4">integrations guide</Link>.
        </p>
        <div className="flex flex-wrap items-center gap-3">
          <Link
            to="/playground"
            className="inline-flex items-center gap-2 bg-white text-black px-4 py-2.5 rounded-lg text-sm font-mono font-bold hover:bg-white/90 transition-colors"
            data-testid="docs-open-playground"
          >
            <Play className="w-4 h-4" /> Try it in the Playground
          </Link>
          {!apiKey && (
            <Link
              to="/account"
              className="inline-flex items-center gap-2 border border-white/15 px-4 py-2.5 rounded-lg text-sm font-mono text-white/80 hover:text-white hover:border-white/30 transition-colors"
              data-testid="docs-get-key"
            >
              <KeyRound className="w-4 h-4" /> Sign up for a free API key
            </Link>
          )}
          <Link
            to="/models"
            className="inline-flex items-center gap-2 text-sm font-mono text-white/60 hover:text-white transition-colors"
          >
            Browse all models <ArrowRight className="w-4 h-4" />
          </Link>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-[1.2fr_0.8fr] gap-6 mb-12">
        <div className="border border-white/10 bg-white/[0.02] rounded-2xl p-6" data-testid="docs-code-card">
          <div className="flex items-center justify-between mb-4 gap-3 flex-wrap">
            <div>
              <div className="text-xs font-mono text-white/40 mb-1">Base URL</div>
              <code className="text-sm text-white/80" data-testid="docs-base-url">{apiBase}</code>
            </div>
            <button
              onClick={copyExample}
              className="inline-flex items-center gap-2 border border-white/10 px-3 py-2 rounded-lg text-xs font-mono text-white/70 hover:text-white hover:border-white/20 transition-colors"
              data-testid="docs-copy-curl"
            >
              {copied ? <Check className="w-3.5 h-3.5 text-green-400" /> : <Copy className="w-3.5 h-3.5" />}
              {copied ? 'Copied' : 'Copy cURL'}
            </button>
          </div>

          <div className="flex gap-1 mb-4 border-b border-white/10">
            {(['chat', 'images', '3d', 'text-to-3d', 'videos', 'transcription'] as const).map(t => (
              <button
                key={t}
                onClick={() => setTab(t)}
                data-testid={`docs-tab-${t}`}
                className={`px-3 py-2 text-xs font-mono transition-colors border-b-2 -mb-px ${
                  tab === t ? 'text-white border-white' : 'text-white/40 hover:text-white/70 border-transparent'
                }`}
              >
                {TAB_LABELS[t]}
              </button>
            ))}
          </div>

          <div className="mb-2 text-xs font-mono text-white/40">{active.description}</div>

          <div className="grid gap-4 md:grid-cols-2">
            {[
              { label: 'Python', language: 'python', code: active.python, testId: 'docs-python' },
              { label: 'JavaScript', language: 'javascript', code: active.javascript, testId: 'docs-javascript' },
              { label: 'Go', language: 'go', code: active.go, testId: 'docs-go' },
              { label: 'cURL', language: 'bash', code: active.curl, testId: 'docs-curl' },
            ].map(sample => (
              <div key={sample.label}>
                <div className="text-xs font-mono text-white/40 mb-2">{sample.label}</div>
                <CodeBlock
                  code={sample.code}
                  language={sample.language}
                  preClassName="rounded-xl border border-white/10 bg-black/60 p-4 text-xs leading-6"
                  testId={sample.testId}
                />
              </div>
            ))}
          </div>
        </div>

        <div className="border border-white/10 bg-white/[0.02] rounded-2xl p-6">
          <div className="flex items-start gap-3 mb-4">
            <KeyRound className="w-5 h-5 text-white/50 mt-0.5" />
            <div>
              <h2 className="text-xl font-bold tracking-tight mb-1">Authentication</h2>
              <p className="text-sm text-white/60 font-light">
                Send your API key in the `Authorization` header as `Bearer op-...`.
              </p>
            </div>
          </div>

          {apiKey ? (
            <div className="rounded-xl border border-green-500/20 bg-green-500/5 p-4 mb-5">
              <div className="text-xs font-mono text-green-400 mb-2">Using this device key</div>
              <code className="break-all text-sm text-white/80" data-testid="docs-api-key">{apiKey}</code>
            </div>
          ) : (
            <div className="rounded-xl border border-white/10 bg-black/40 p-4 mb-5 text-sm text-white/60">
              Sign in on the <Link to="/account" className="text-white underline underline-offset-4">account page</Link> to auto-generate an API key on this device.
            </div>
          )}

          <div className="rounded-xl border border-white/10 bg-black/40 p-4 mb-5">
            <div className="text-xs font-mono text-white/40 mb-2">OpenPaths Auto (embedding-routed)</div>
            <div className="space-y-1 text-sm text-white/70 font-mono mb-4" data-testid="docs-auto-models">
              <div><code>openpaths/auto</code> — default chat; legacy <code>auto</code></div>
              <div><code>openpaths/auto-code</code> — agents, refactors, bug fixes</div>
              <div><code>openpaths/auto-fast</code> — low-latency chat (DeepSeek Flash)</div>
              <div><code>openpaths/auto-cheap</code> — Nano / Flash Lite classifiers</div>
              <div><code>openpaths/auto-reasoning</code> — planning, math, auto thinking depth</div>
              <div><code>openpaths/auto-vision</code> — image understanding</div>
              <div><code>openpaths/auto-image</code> — GPT Image 2, RA1 fallback</div>
            </div>
            <div className="text-xs font-mono text-white/40 mb-2">Latest aliases</div>
            <div className="space-y-1 text-sm text-white/70 font-mono" data-testid="docs-latest-aliases">
              <div><code>openai-chat-latest</code>{' -> '}<code>gpt-5-chat-latest</code></div>
              <div><code>openai-coding-latest</code>{' -> '}<code>gpt-5-codex</code></div>
              <div><code>openai-image-latest</code>{' -> '}<code>gpt-image-2</code></div>
              <div><code>anthropic-opus-latest</code>{' -> '}<code>claude-opus-latest</code></div>
              <div><code>openai-video</code>{' -> '}<code>sora-2</code></div>
            </div>
          </div>

          <div className="rounded-xl border border-white/10 bg-black/40 p-4 mb-5">
            <div className="text-xs font-mono text-white/40 mb-2">Opt-in app attribution</div>
            <p className="text-sm text-white/60 mb-3">
              Send `HTTP-Referer` with your app URL and `X-OpenRouter-Title` or `X-Title` with your app name. OpenPaths records app-level model and token stats only when those headers are present.
            </p>
            <CodeBlock
              language="bash"
              code={`-H "HTTP-Referer: https://your-app.example"\n-H "X-OpenRouter-Title: Your App"\n-H "X-OpenRouter-Categories: cli-agent,programming-app"`}
              preClassName="rounded-lg border border-white/10 bg-black/60 p-3 text-xs leading-5"
            />
          </div>

          <div className="space-y-3">
            {ENDPOINTS.map(endpoint => (
              <div key={endpoint.path} className="rounded-xl border border-white/10 p-4">
                <div className="flex items-center gap-3 mb-1">
                  <span className="font-mono text-[11px] px-2 py-1 rounded bg-white/10 text-white/80">{endpoint.method}</span>
                  <code className="text-sm text-white/90">{endpoint.path}</code>
                </div>
                <p className="text-sm text-white/50">{endpoint.description}</p>
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="border border-white/10 bg-white/[0.02] rounded-2xl p-6">
        <h2 className="text-xl font-bold tracking-tight mb-2">Provider guides</h2>
        <p className="text-sm text-white/60 font-light mb-5">
          Each provider has its own endpoints, default models, and copy-paste examples. Open a guide to see Python and cURL snippets,
          then jump straight into the Playground to test any model.
        </p>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          {providers.map(p => (
            <Link
              key={p.slug}
              to={`/${p.slug}/docs`}
              className="group rounded-xl border border-white/10 bg-white/[0.01] p-4 hover:border-white/30 hover:bg-white/[0.04] transition-colors flex items-start gap-3"
              data-testid={`docs-provider-${p.slug}`}
            >
              <img
                src={p.logo || FALLBACK_LOGO}
                alt=""
                className="w-9 h-9 rounded-lg border border-white/10 bg-white/[0.04] p-1.5 object-contain shrink-0"
              />
              <div className="min-w-0">
                <div className="flex items-center gap-1.5">
                  <span className="text-sm font-bold tracking-tight truncate">{p.name}</span>
                  <ArrowRight className="w-3.5 h-3.5 text-white/30 group-hover:text-white/70 transition-colors shrink-0" />
                </div>
                <div className="text-[11px] font-mono text-white/40 mb-1">/{p.slug}/docs</div>
                <p className="text-xs text-white/50 leading-relaxed line-clamp-2">{p.description}</p>
              </div>
            </Link>
          ))}
        </div>
      </div>
      </section>
    </>
  );
}

function buildSamples(apiBase: string, exampleKey: string) {
  const python = {
    chat: `from openai import OpenAI

client = OpenAI(
    base_url="${apiBase}",
    api_key="${exampleKey}",
)

response = client.chat.completions.create(
    model="openai-chat-latest",
    messages=[{"role": "user", "content": "Write a tiny SSE server in Go."}],
    reasoning_effort="auto",  # none | low | medium | high | auto
)

print(response.choices[0].message.content)`,
    images: `from openai import OpenAI
import base64, pathlib

client = OpenAI(base_url="${apiBase}", api_key="${exampleKey}")

result = client.images.generate(
    model="gpt-image-2",
    prompt="A beige coffee mug on a wooden table, natural light, editorial photo",
    size="1024x1024",
    quality="high",
    n=1,
)

image_b64 = result.data[0].b64_json
pathlib.Path("mug.png").write_bytes(base64.b64decode(image_b64))
print("wrote mug.png")`,
    videos: `from openai import OpenAI

client = OpenAI(base_url="${apiBase}", api_key="${exampleKey}")

result = client.post(
    "/videos/generations",
    body={
        "model": "sora-2",
        "prompt": "A cinematic fly-through of a neon-lit cyberpunk street at night.",
        "resolution": "1280x720",
        "num_frames": 48,
        "frames_per_second": 24,
        "async": True,
    },
    cast_to=dict,
)

while result["status"] not in ("completed", "failed"):
    result = client.get(f"/videos/generations/{result['id']}", cast_to=dict)

print(result["result"]["video_url"])`,
    '3d': `from openai import OpenAI

client = OpenAI(base_url="${apiBase}", api_key="${exampleKey}")

result = client.post(
    "/3d/generations",
    body={
        "model": "pixal3d-image-to-3d",
        "image_url": "https://openpathsstatic.openpaths.io/static/uploads/image-to-3d/sword-reference.jpg",
        "resolution": 1024,
        "texture_size": 1024,
        "remesh": True,
    },
    cast_to=dict,
)

print(result["model_glb"]["url"])`,
    'text-to-3d': `from openai import OpenAI

client = OpenAI(base_url="${apiBase}", api_key="${exampleKey}")

result = client.post(
    "/3d/text-generations",
    body={
        "prompt": "A single ornate medieval longsword on a white background, 3D game asset",
        "image_model": "openpaths/auto-image",
        "texture_size": 1024,
        "remesh": True,
    },
    cast_to=dict,
)

print(result["model_glb"]["url"])  # billed image price + Pixal3D price`,
    transcription: `from openai import OpenAI

client = OpenAI(base_url="${apiBase}", api_key="${exampleKey}")

with open("meeting.mp3", "rb") as f:
    transcript = client.audio.transcriptions.create(
        model="gpt-4o-transcribe",
        file=f,
    )

print(transcript.text)`,
  };

  const curl = {
    chat: `curl ${apiBase}/chat/completions \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${exampleKey}" \\
  -d '{
    "model": "openai-chat-latest",
    "reasoning_effort": "auto",
    "messages": [
      {"role": "user", "content": "Write a tiny SSE server in Go."}
    ]
  }'`,
    images: `curl ${apiBase}/images/generations \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${exampleKey}" \\
  -d '{
    "model": "gpt-image-2",
    "prompt": "A beige coffee mug on a wooden table",
    "size": "1024x1024",
    "quality": "high",
    "n": 1
  }'`,
    videos: `curl ${apiBase}/videos/generations \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${exampleKey}" \\
  -d '{
    "model": "sora-2",
    "prompt": "A cinematic fly-through of a neon-lit cyberpunk street at night.",
    "resolution": "1280x720",
    "num_frames": 48,
    "frames_per_second": 24
  }'`,
    '3d': `curl ${apiBase}/3d/generations \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${exampleKey}" \\
  -d '{
    "model": "pixal3d-image-to-3d",
    "image_url": "https://openpathsstatic.openpaths.io/static/uploads/image-to-3d/sword-reference.jpg",
    "resolution": 1024,
    "texture_size": 1024,
    "remesh": true
  }'`,
    'text-to-3d': `curl ${apiBase}/3d/text-generations \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${exampleKey}" \\
  -d '{
    "prompt": "A single ornate medieval longsword on a white background, 3D game asset",
    "image_model": "openpaths/auto-image",
    "texture_size": 1024,
    "remesh": true
  }'`,
    transcription: `curl ${apiBase}/audio/transcriptions \\
  -H "Authorization: Bearer ${exampleKey}" \\
  -F model=gpt-4o-transcribe \\
  -F file=@meeting.mp3`,
  };

  const javascript = {
    chat: `import OpenAI from "openai";

const client = new OpenAI({
  baseURL: "${apiBase}",
  apiKey: "${exampleKey}",
});

const response = await client.chat.completions.create({
  model: "openai-chat-latest",
  messages: [{ role: "user", content: "Write a tiny SSE server in Go." }],
  reasoning_effort: "auto", // none | low | medium | high | auto
});

console.log(response.choices[0].message.content);`,
    images: `import OpenAI from "openai";
import { writeFile } from "node:fs/promises";

const client = new OpenAI({ baseURL: "${apiBase}", apiKey: "${exampleKey}" });

const result = await client.images.generate({
  model: "gpt-image-2",
  prompt: "A beige coffee mug on a wooden table, natural light, editorial photo",
  size: "1024x1024",
  quality: "high",
  n: 1,
});

await writeFile("mug.png", Buffer.from(result.data[0].b64_json, "base64"));
console.log("wrote mug.png");`,
    videos: `const response = await fetch("${apiBase}/videos/generations", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    Authorization: "Bearer ${exampleKey}",
  },
  body: JSON.stringify({
    model: "sora-2",
    prompt: "A cinematic fly-through of a neon-lit cyberpunk street at night.",
    resolution: "1280x720",
    num_frames: 48,
    frames_per_second: 24,
    async: true,
  }),
});

let result = await response.json();

while (result.status && result.status !== "completed" && result.status !== "failed") {
  await new Promise((resolve) => setTimeout(resolve, 2000));
  const poll = await fetch("${apiBase}/videos/generations/" + result.id, {
    headers: { Authorization: "Bearer ${exampleKey}" },
  });
  result = await poll.json();
}

console.log((result.result ?? result).video_url);`,
    '3d': `const response = await fetch("${apiBase}/3d/generations", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    Authorization: "Bearer ${exampleKey}",
  },
  body: JSON.stringify({
    model: "pixal3d-image-to-3d",
    image_url: "https://openpathsstatic.openpaths.io/static/uploads/image-to-3d/sword-reference.jpg",
    resolution: 1024,
    texture_size: 1024,
    remesh: true,
  }),
});

const result = await response.json();
console.log(result.model_glb.url);`,
    'text-to-3d': `const response = await fetch("${apiBase}/3d/text-generations", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    Authorization: "Bearer ${exampleKey}",
  },
  body: JSON.stringify({
    prompt: "A single ornate medieval longsword on a white background, 3D game asset",
    image_model: "openpaths/auto-image",
    texture_size: 1024,
    remesh: true,
  }),
});

const result = await response.json();
console.log(result.model_glb.url); // billed image price + Pixal3D price`,
    transcription: `import OpenAI from "openai";
import fs from "node:fs";

const client = new OpenAI({ baseURL: "${apiBase}", apiKey: "${exampleKey}" });

const transcript = await client.audio.transcriptions.create({
  model: "gpt-4o-transcribe",
  file: fs.createReadStream("meeting.mp3"),
});

console.log(transcript.text);`,
  };

  const go = {
    chat: `package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

func main() {
	body := []byte(\`{
  "model": "openai-chat-latest",
  "reasoning_effort": "auto",
  "messages": [{"role": "user", "content": "Write a tiny SSE server in Go."}]
}\`)

	req, _ := http.NewRequest("POST", "${apiBase}/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer ${exampleKey}")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	out, _ := io.ReadAll(resp.Body)
	fmt.Println(string(out))
}`,
    images: `package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

func main() {
	body := []byte(\`{
  "model": "gpt-image-2",
  "prompt": "A beige coffee mug on a wooden table",
  "size": "1024x1024",
  "quality": "high",
  "n": 1
}\`)

	req, _ := http.NewRequest("POST", "${apiBase}/images/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer ${exampleKey}")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	out, _ := io.ReadAll(resp.Body)
	fmt.Println(string(out))
}`,
    videos: `package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

func main() {
	body := []byte(\`{
  "model": "sora-2",
  "prompt": "A cinematic fly-through of a neon-lit cyberpunk street at night.",
  "resolution": "1280x720",
  "num_frames": 48,
  "frames_per_second": 24
}\`)

	req, _ := http.NewRequest("POST", "${apiBase}/videos/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer ${exampleKey}")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	out, _ := io.ReadAll(resp.Body)
	fmt.Println(string(out))
}`,
    '3d': `package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

func main() {
	body := []byte(\`{
  "model": "pixal3d-image-to-3d",
  "image_url": "https://openpathsstatic.openpaths.io/static/uploads/image-to-3d/sword-reference.jpg",
  "resolution": 1024,
  "texture_size": 1024,
  "remesh": true
}\`)

	req, _ := http.NewRequest("POST", "${apiBase}/3d/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer ${exampleKey}")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	out, _ := io.ReadAll(resp.Body)
	fmt.Println(string(out))
}`,
    'text-to-3d': `package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

func main() {
	body := []byte(\`{
  "prompt": "A single ornate medieval longsword on a white background, 3D game asset",
  "image_model": "openpaths/auto-image",
  "texture_size": 1024,
  "remesh": true
}\`)

	req, _ := http.NewRequest("POST", "${apiBase}/3d/text-generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer ${exampleKey}")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	out, _ := io.ReadAll(resp.Body)
	fmt.Println(string(out))
}`,
    transcription: `package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
)

func main() {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	writer.WriteField("model", "gpt-4o-transcribe")
	file, _ := os.Open("meeting.mp3")
	defer file.Close()
	part, _ := writer.CreateFormFile("file", "meeting.mp3")
	io.Copy(part, file)
	writer.Close()

	req, _ := http.NewRequest("POST", "${apiBase}/audio/transcriptions", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer ${exampleKey}")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	out, _ := io.ReadAll(resp.Body)
	fmt.Println(string(out))
}`,
  };

  return {
    chat: {
      description: 'Send messages, receive streamed tokens. OpenAI SDK drop-in.',
      python: python.chat,
      javascript: javascript.chat,
      go: go.chat,
      curl: curl.chat,
    },
    images: {
      description: 'Generate images with GPT Image 2, Flux, Netwrck RA1, and more.',
      python: python.images,
      javascript: javascript.images,
      go: go.images,
      curl: curl.images,
    },
    '3d': {
      description: 'Generate textured GLB models from a public object image.',
      python: python['3d'],
      javascript: javascript['3d'],
      go: go['3d'],
      curl: curl['3d'],
    },
    'text-to-3d': {
      description: 'Prompt → auto image → textured GLB in one call. Billed image price + Pixal3D price.',
      python: python['text-to-3d'],
      javascript: javascript['text-to-3d'],
      go: go['text-to-3d'],
      curl: curl['text-to-3d'],
    },
    videos: {
      description: 'Generate video clips with Sora 2, Hailuo, Wan, and LTX.',
      python: python.videos,
      javascript: javascript.videos,
      go: go.videos,
      curl: curl.videos,
    },
    transcription: {
      description: 'Upload an audio file and receive a JSON transcript.',
      python: python.transcription,
      javascript: javascript.transcription,
      go: go.transcription,
      curl: curl.transcription,
    },
  } as const;
}
