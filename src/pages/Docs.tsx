import React, { useEffect, useState } from 'react';
import { BookOpen, Copy, Check, KeyRound } from 'lucide-react';
import { Link } from 'react-router-dom';
import { getAPIBaseURL, getStoredAPIKey } from '../lib/session';
import { CodeBlock } from '../components/CodeBlock';

const ENDPOINTS = [
  { method: 'POST', path: '/v1/chat/completions', description: 'OpenAI-compatible chat completions.' },
  { method: 'GET', path: '/v1/models', description: 'List all available models and capabilities.' },
  { method: 'POST', path: '/v1/images/generations', description: 'Generate images with first-party and routed models.' },
  { method: 'POST', path: '/v1/videos/generations', description: 'Generate videos through the unified API.' },
];

export function Docs() {
  const [apiKey, setApiKey] = useState(() => getStoredAPIKey());
  const [copied, setCopied] = useState(false);
  const apiBase = getAPIBaseURL();
  const exampleKey = apiKey || 'op-...';

  useEffect(() => {
    const sync = () => setApiKey(getStoredAPIKey());
    window.addEventListener('storage', sync);
    return () => window.removeEventListener('storage', sync);
  }, []);

  const pythonExample = `from openai import OpenAI

client = OpenAI(
    base_url="${apiBase}",
    api_key="${exampleKey}",
)

response = client.chat.completions.create(
    model="openai-chat-latest",
    messages=[{"role": "user", "content": "Write a tiny SSE server in Go."}],
)

print(response.choices[0].message.content)`;

  const curlExample = `curl ${apiBase}/chat/completions \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${exampleKey}" \\
  -d '{
    "model": "openai-chat-latest",
    "messages": [
      {"role": "user", "content": "Write a tiny SSE server in Go."}
    ]
  }'`;

  const copyExample = async () => {
    await navigator.clipboard.writeText(curlExample);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1500);
  };

  return (
    <section className="max-w-6xl mx-auto px-6 py-16">
      <div className="mb-12">
        <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border border-white/10 bg-white/[0.04] text-xs font-mono text-white/60 mb-6">
          <BookOpen className="w-3.5 h-3.5" />
          API Docs
        </div>
        <h1 className="text-4xl md:text-5xl font-bold tracking-tight mb-4">OpenAI And Anthropic SDK Compatible Docs</h1>
        <p className="text-white/60 max-w-3xl font-light leading-relaxed">
          Use the same base URL pattern for chat, images, video, music, speech, and models from either SDK. If you are signed in on this device,
          your API key is injected into the examples automatically.
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-[1.2fr_0.8fr] gap-6 mb-12">
        <div className="border border-white/10 bg-white/[0.02] rounded-2xl p-6" data-testid="docs-code-card">
          <div className="flex items-center justify-between mb-4">
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

          <div className="grid gap-4 md:grid-cols-2">
            <div>
              <div className="text-xs font-mono text-white/40 mb-2">Python</div>
              <CodeBlock
                code={pythonExample}
                language="python"
                preClassName="rounded-xl border border-white/10 bg-black/60 p-4 text-xs leading-6"
                testId="docs-python"
              />
            </div>
            <div>
              <div className="text-xs font-mono text-white/40 mb-2">cURL</div>
              <CodeBlock
                code={curlExample}
                language="bash"
                preClassName="rounded-xl border border-white/10 bg-black/60 p-4 text-xs leading-6"
                testId="docs-curl"
              />
            </div>
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
                <div><code>anthropic-opus-latest</code>{' -> '}<code>claude-opus-latest</code></div>
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
    </section>
  );
}
