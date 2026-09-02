import React, { useEffect, useMemo, useState } from 'react';
import { Copy, Download, ImageIcon, Loader2, Wand2 } from 'lucide-react';
import { ToolSeo } from '../components/ToolSeo';
import { CodeBlock } from '../components/CodeBlock';

const DEFAULT_PROMPT = 'A beige ceramic coffee mug on a wooden table, soft natural window light, editorial product photo';
const EXAMPLE_OUTPUT_URL = 'https://openpathsstatic.openpaths.io/static/uploads/text-to-image/mug-example.jpg';

const IMAGE_MODELS = [
  { id: 'openpaths/auto-image', label: 'Auto Image (routed)', price: 0.211 },
  { id: 'gpt-image-2', label: 'GPT Image 2', price: 0.04 },
  { id: 'ra1', label: 'RA1 Art Generator', price: 0.04 },
  { id: 'flux-pro', label: 'Flux Pro', price: 0.014 },
];

const SIZES = ['1024x1024', '768x1024', '1024x768', '1280x720', '720x1280'];

type ImageData = { url?: string; b64_json?: string; revised_prompt?: string };
type ImageResponse = { data?: ImageData[] };
type APIErrorBody = { error?: { message?: string } | string };

function getStoredAPIKey() {
  if (typeof window === 'undefined') return '';
  return localStorage.getItem('op_api_key') || '';
}

async function parseAPIResponse(resp: Response) {
  const text = await resp.text();
  if (!text) return null;
  try {
    return JSON.parse(text);
  } catch {
    if (!resp.ok) throw new Error(text);
    throw new Error('Invalid JSON response: ' + text.slice(0, 160));
  }
}

function getAPIErrorMessage(data: unknown, fallback: string) {
  const error = (data as APIErrorBody | null)?.error;
  if (typeof error === 'string') return error;
  return error?.message || fallback;
}

function imageSrc(d: ImageData) {
  if (d.url) return d.url;
  if (d.b64_json) return `data:image/png;base64,${d.b64_json}`;
  return '';
}

export function TextToImage() {
  const [apiKey, setApiKey] = useState(getStoredAPIKey);
  const [prompt, setPrompt] = useState(DEFAULT_PROMPT);
  const [model, setModel] = useState('openpaths/auto-image');
  const [size, setSize] = useState('1024x1024');
  const [activeSnippet, setActiveSnippet] = useState<'python' | 'javascript' | 'curl'>('python');
  const [images, setImages] = useState<ImageData[]>([]);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (apiKey) localStorage.setItem('op_api_key', apiKey);
  }, [apiKey]);

  const price = IMAGE_MODELS.find(m => m.id === model)?.price ?? 0.04;

  const payload = useMemo(() => ({ model, prompt, size, n: 1 }), [model, prompt, size]);

  const snippets = useMemo(() => {
    const bearer = apiKey || 'op-...';
    const body = JSON.stringify(payload, null, 2);
    return {
      python: `from openai import OpenAI

client = OpenAI(base_url="https://openpaths.io/v1", api_key="${bearer}")

result = client.images.generate(
    model="${model}",
    prompt=${JSON.stringify(prompt)},
    size="${size}",
    n=1,
)

print(result.data[0].url or result.data[0].b64_json[:32])`,
      javascript: `const resp = await fetch("https://openpaths.io/v1/images/generations", {
  method: "POST",
  headers: {
    "Authorization": "Bearer ${bearer}",
    "Content-Type": "application/json",
  },
  body: JSON.stringify(${body.replace(/^/gm, '  ').trim()}),
});

if (!resp.ok) throw new Error(await resp.text());

const data = await resp.json();
console.log(data.data[0].url);`,
      curl: `curl https://openpaths.io/v1/images/generations \\
  -H "Authorization: Bearer ${bearer}" \\
  -H "Content-Type: application/json" \\
  -d '${body}'`,
    };
  }, [apiKey, model, payload, prompt, size]);

  async function generate() {
    if (!apiKey) {
      setError('Set an OpenPaths API key before generating.');
      return;
    }
    if (!prompt.trim()) {
      setError('Enter a prompt to generate from.');
      return;
    }
    setLoading(true);
    setError('');
    try {
      const resp = await fetch('/v1/images/generations', {
        method: 'POST',
        headers: { Authorization: `Bearer ${apiKey}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      const data = await parseAPIResponse(resp) as ImageResponse;
      if (!resp.ok) throw new Error(getAPIErrorMessage(data, 'Image generation failed'));
      setImages(data?.data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Image generation failed');
    } finally {
      setLoading(false);
    }
  }

  async function copySnippet() {
    await navigator.clipboard.writeText(snippets[activeSnippet]);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1400);
  }

  return (
    <>
      <ToolSeo slug="text-to-image" />

      <div className="px-6 py-10">
        <div className="mx-auto max-w-7xl">
          <div className="mb-8 flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
            <div>
              <div className="mb-3 inline-flex items-center gap-2 rounded border border-white/20 bg-white/[0.06] px-3 py-1 text-xs font-mono text-white/45">
                <ImageIcon className="h-3.5 w-3.5" /> Auto Image endpoint
              </div>
              <h1 className="max-w-3xl text-4xl font-bold tracking-tight md:text-6xl">Text to Image</h1>
              <p className="mt-4 max-w-2xl text-base leading-relaxed text-white/55">
                One prompt, one image. The auto image route picks the best model, or pin a specific one. Same endpoint the rest of the API uses.
              </p>
            </div>
            <div className="grid grid-cols-2 gap-2 text-xs font-mono text-white/45">
              <span className="rounded border border-white/20 px-3 py-2">{model}</span>
              <span className="rounded border border-white/20 px-3 py-2">~${price.toFixed(3)}/image</span>
            </div>
          </div>

          <div className="grid gap-6 lg:grid-cols-[420px_minmax(0,1fr)]">
            <section className="rounded-lg border border-white/20 bg-white/[0.05] p-5">
              <h2 className="mb-4 font-mono text-sm font-bold uppercase tracking-wider text-white/70">Generate</h2>

              <label className="mb-3 block">
                <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">OpenPaths API key</span>
                <input
                  value={apiKey}
                  onChange={e => setApiKey(e.target.value)}
                  placeholder="op-..."
                  className="w-full rounded border border-white/20 bg-black px-3 py-2 text-sm font-mono text-white placeholder:text-white/45 focus:border-white/50 focus:outline-none"
                />
              </label>

              <label className="mb-3 block">
                <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">Prompt</span>
                <textarea
                  value={prompt}
                  onChange={e => setPrompt(e.target.value)}
                  rows={5}
                  className="w-full resize-y rounded border border-white/20 bg-black px-3 py-2 text-sm text-white placeholder:text-white/45 focus:border-white/50 focus:outline-none"
                />
              </label>

              <div className="grid grid-cols-2 gap-3">
                <label>
                  <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">Model</span>
                  <select value={model} onChange={e => setModel(e.target.value)} className="w-full rounded border border-white/20 bg-black px-2 py-2 text-xs font-mono text-white focus:border-white/50 focus:outline-none">
                    {IMAGE_MODELS.map(m => <option key={m.id} value={m.id}>{m.label}</option>)}
                  </select>
                </label>
                <label>
                  <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">Size</span>
                  <select value={size} onChange={e => setSize(e.target.value)} className="w-full rounded border border-white/20 bg-black px-2 py-2 text-xs font-mono text-white focus:border-white/50 focus:outline-none">
                    {SIZES.map(s => <option key={s} value={s}>{s}</option>)}
                  </select>
                </label>
              </div>

              <button
                type="button"
                onClick={generate}
                disabled={loading}
                className="mt-4 inline-flex w-full items-center justify-center gap-2 rounded bg-white px-4 py-3 text-sm font-mono font-bold text-black transition-colors hover:bg-white/90 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Wand2 className="h-4 w-4" />}
                {loading ? 'Generating...' : `Generate (~$${price.toFixed(3)})`}
              </button>

              {error && <p className="mt-3 rounded border border-red-400/20 bg-red-400/10 px-3 py-2 text-xs font-mono text-red-200">{error}</p>}
            </section>

            <section className="overflow-hidden rounded-lg border border-white/20 bg-white/[0.05]">
              <div className="border-b border-white/20 px-4 py-3">
                <h2 className="font-mono text-sm font-bold uppercase tracking-wider text-white/70">Result</h2>
              </div>
              <div className="flex min-h-[420px] items-center justify-center p-5">
                {images.length > 0 ? (
                  <div className="grid w-full gap-4 sm:grid-cols-2">
                    {images.map((d, i) => {
                      const src = imageSrc(d);
                      return (
                        <div key={i} className="overflow-hidden rounded border border-white/20 bg-black">
                          {src && <img src={src} alt={`Generated ${i + 1}`} className="h-auto w-full" />}
                          {src && (
                            <a href={src} download={`openpaths-${i + 1}.png`} className="flex items-center justify-center gap-2 border-t border-white/20 px-3 py-2 text-xs font-mono text-white/50 hover:text-white">
                              <Download className="h-3.5 w-3.5" /> Download
                            </a>
                          )}
                        </div>
                      );
                    })}
                  </div>
                ) : loading ? (
                  <p className="text-sm font-mono text-white/45">Generating image...</p>
                ) : (
                  <div className="w-full max-w-md overflow-hidden rounded border border-white/20 bg-black">
                    <img src={EXAMPLE_OUTPUT_URL} alt="Example generated image" className="h-auto w-full" />
                    <div className="flex items-center justify-between gap-3 border-t border-white/20 px-3 py-2 text-xs font-mono text-white/50">
                      <span>Example - hit Generate on the default prompt</span>
                      <a href={EXAMPLE_OUTPUT_URL} download="openpaths-example.png" className="flex items-center gap-2 hover:text-white"><Download className="h-3.5 w-3.5" /> Download</a>
                    </div>
                  </div>
                )}
              </div>
            </section>
          </div>

          <section className="mt-6 min-w-0 overflow-hidden rounded-lg border border-white/20 bg-white/[0.05] p-5">
            <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
              <h2 className="font-mono text-sm font-bold uppercase tracking-wider text-white/70">API code</h2>
              <div className="flex items-center gap-2">
                {(['python', 'javascript', 'curl'] as const).map(snippet => (
                  <button
                    key={snippet}
                    type="button"
                    onClick={() => setActiveSnippet(snippet)}
                    className={`rounded border px-2.5 py-1.5 text-[10px] font-mono uppercase tracking-[0.14em] transition-colors ${activeSnippet === snippet ? 'border-white/30 bg-white/10 text-white' : 'border-white/20 text-white/50 hover:text-white'}`}
                  >
                    {snippet === 'javascript' ? 'JS' : snippet}
                  </button>
                ))}
                <button type="button" onClick={copySnippet} className="inline-flex items-center gap-1.5 rounded border border-white/20 px-2.5 py-1.5 text-[10px] font-mono uppercase tracking-[0.14em] text-white/45 hover:text-white">
                  <Copy className="h-3 w-3" /> {copied ? 'Copied' : 'Copy'}
                </button>
              </div>
            </div>
            <CodeBlock
              language={activeSnippet === 'javascript' ? 'javascript' : activeSnippet === 'curl' ? 'bash' : 'python'}
              label={activeSnippet === 'javascript' ? 'JavaScript' : activeSnippet === 'curl' ? 'cURL' : 'Python'}
              code={snippets[activeSnippet]}
              containerClassName="border border-white/20 rounded-lg overflow-hidden bg-black/40"
              headerClassName="bg-white/[0.06]"
              preClassName="text-xs"
            />
          </section>
        </div>
      </div>
    </>
  );
}
