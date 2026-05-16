import React, { useEffect, useMemo, useState } from 'react';
import { BookOpen, Copy, Check, KeyRound } from 'lucide-react';
import { Link } from 'react-router-dom';
import { getAPIBaseURL, getStoredAPIKey } from '../lib/session';
import { CodeBlock } from '../components/CodeBlock';
import { Seo } from '../components/Seo';

const ENDPOINTS = [
  { method: 'POST', path: '/v1/chat/completions', description: 'OpenAI-compatible chat completions (streaming + tools).' },
  { method: 'POST', path: '/v1/messages', description: 'Anthropic-compatible Messages API for Claude SDKs and agent SDKs.' },
  { method: 'GET', path: '/v1/models', description: 'List all available models and capabilities.' },
  { method: 'POST', path: '/v1/images/generations', description: 'Generate images (GPT Image 2, Flux, RA1, Klein, and more).' },
  { method: 'POST', path: '/v1/images/edits', description: 'Edit images and image-to-image workflows (GPT Image 2, Grok Imagine Image).' },
  { method: 'POST', path: '/v1/videos/generations', description: 'Generate videos (Sora 2, Hailuo, Wan, LTX).' },
  { method: 'POST', path: '/v1/audio/transcriptions', description: 'Transcribe speech to text (Whisper, GPT-4o Transcribe).' },
  { method: 'POST', path: '/v1/audio/speech', description: 'Text-to-speech (xAI TTS, MiniMax Speech 2.8 HD).' },
  { method: 'POST', path: '/v1/stt', description: 'xAI Speech to Text shortcut; defaults to xai-stt.' },
  { method: 'POST', path: '/v1/tts', description: 'xAI Text to Speech shortcut; defaults to xai-tts.' },
  { method: 'POST', path: '/v1/embeddings', description: 'Generate vector embeddings (OpenPaths, Google Gemini, Mistral, Nemotron).' },
];

type Tab = 'chat' | 'images' | 'videos' | 'transcription';

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
        <p className="text-white/60 max-w-3xl font-light leading-relaxed">
          Use the same base URL pattern for chat, images, video, music, speech, and models from either SDK. If you are signed in on this device,
          your API key is injected into the examples automatically. For framework-specific setup, see the <Link to="/integrations" className="text-white underline underline-offset-4">integrations guide</Link>.
        </p>
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
            {(['chat', 'images', 'videos', 'transcription'] as const).map(t => (
              <button
                key={t}
                onClick={() => setTab(t)}
                data-testid={`docs-tab-${t}`}
                className={`px-3 py-2 text-xs font-mono capitalize transition-colors border-b-2 -mb-px ${
                  tab === t ? 'text-white border-white' : 'text-white/40 hover:text-white/70 border-transparent'
                }`}
              >
                {t}
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
            <div className="text-xs font-mono text-white/40 mb-2">Latest aliases</div>
            <div className="space-y-1 text-sm text-white/70 font-mono" data-testid="docs-latest-aliases">
              <div><code>openai-chat-latest</code>{' -> '}<code>gpt-5-chat-latest</code></div>
              <div><code>openai-coding-latest</code>{' -> '}<code>gpt-5-codex</code></div>
              <div><code>openai-image-latest</code>{' -> '}<code>gpt-image-2</code></div>
              <div><code>anthropic-opus-latest</code>{' -> '}<code>claude-opus-latest</code></div>
              <div><code>openai-video</code>{' -> '}<code>sora-2</code></div>
            </div>
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
          Each provider has its own endpoints and model list. Use these shortcuts to browse per-provider docs and examples.
        </p>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-3">
          {['openai', 'anthropic', 'google', 'xai', 'deepseek', 'mistral', 'groq', 'together', 'openrouter', 'netwrck', 'text-generator', 'fal', 'minimax'].map(slug => (
            <Link
              key={slug}
              to={`/${slug}/docs`}
              className="rounded-xl border border-white/10 px-4 py-3 text-sm font-mono text-white/70 hover:text-white hover:border-white/30 transition-colors capitalize"
              data-testid={`docs-provider-${slug}`}
            >
              /{slug}/docs
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

# Creates a Sora job and waits for it to complete, then returns a signed URL.
result = client.post(
    "/videos/generations",
    body={
        "model": "sora-2",
        "prompt": "A cinematic fly-through of a neon-lit cyberpunk street at night.",
        "resolution": "1280x720",
        "num_frames": 48,
        "frames_per_second": 24,
    },
    cast_to=dict,
)

print(result["video_url"])`,
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
  }),
});

const result = await response.json();
console.log(result.video_url);`,
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
