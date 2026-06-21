import React, { useMemo, useState } from 'react';
import { Link, Navigate, useNavigate, useParams } from 'react-router-dom';
import { ArrowLeft, BookOpen, Check, Copy, Image as ImageIcon, MessageSquare, Video } from 'lucide-react';
import { CodeBlock } from '../components/CodeBlock';
import { Seo } from '../components/Seo';
import { models, type Model } from '../data/models';
import { providersByName, getProviderLogo } from '../data/providers';
import { IMAGE_DEMOS, type ImageDemo } from '../data/imageDemos';
import { VIDEO_DEMOS, type VideoDemo } from '../data/videoDemos';
import { providerDocsPath, providerPath } from '../lib/paths';

const NON_CHAT_TAGS = ['art generation', 'video generation', 'audio', 'embedding'];
type VideoCodeLanguage = 'python' | 'javascript' | 'curl';

export function ModelPage() {
  const { modelId = '' } = useParams<{ modelId: string }>();
  const navigate = useNavigate();
  const decodedId = decodeURIComponent(modelId);
  const model = models.find(item => item.id === decodedId);

  if (!model) {
    return <Navigate to="/models" replace />;
  }

  const provider = providersByName[model.provider];
  const isImage = model.tags.includes('art generation');
  const isVideo = model.tags.includes('video generation');
  const canChat = isChatModel(model);
  const imageDemo = IMAGE_DEMOS[model.id];
  const videoDemo = VIDEO_DEMOS[model.id];
  const [videoCodeLang, setVideoCodeLang] = useState<VideoCodeLanguage>('python');
  const [copiedCode, setCopiedCode] = useState(false);
  const imageSnippet = useMemo(() => (
    imageDemo ? generateImageSnippet(imageDemo, videoCodeLang) : ''
  ), [imageDemo, videoCodeLang]);
  const videoSnippet = useMemo(() => (
    isVideo ? generateVideoSnippet(model.id, videoDemo, videoCodeLang) : ''
  ), [isVideo, model.id, videoDemo, videoCodeLang]);
  const title = `${model.name} API, Pricing, Context Window | OpenPaths`;
  const description = `${model.name} from ${model.provider}: ${model.description} Use model ID ${model.id} through the OpenPaths API.`;
  const canonical = `https://openpaths.io/models/${encodeURIComponent(model.id)}`;
  const jsonLd = [
    {
      '@context': 'https://schema.org',
      '@type': 'Product',
      name: model.name,
      description: model.description,
      sku: model.id,
      category: 'AI Model',
      brand: { '@type': 'Brand', name: model.provider },
      url: canonical,
      ...(model.priceInput > 0 ? {
        offers: {
          '@type': 'Offer',
          price: model.priceInput.toString(),
          priceCurrency: 'USD',
          availability: 'https://schema.org/InStock',
          url: canonical,
        },
      } : {}),
    },
    {
      '@context': 'https://schema.org',
      '@type': 'BreadcrumbList',
      itemListElement: [
        { '@type': 'ListItem', position: 1, name: 'Models', item: 'https://openpaths.io/models' },
        { '@type': 'ListItem', position: 2, name: model.name, item: canonical },
      ],
    },
  ];

  const copyVideoSnippet = async () => {
    await navigator.clipboard.writeText(isVideo ? videoSnippet : imageSnippet);
    setCopiedCode(true);
    window.setTimeout(() => setCopiedCode(false), 1600);
  };

  return (
    <>
      <Seo title={title} description={description} path={`/models/${encodeURIComponent(model.id)}`} jsonLd={jsonLd} />

      <section className="max-w-5xl mx-auto px-6 py-16">
        <div className="mb-8">
          <Link to="/models" className="inline-flex items-center gap-1.5 text-xs font-mono text-white/50 hover:text-white transition-colors">
            <ArrowLeft className="w-3.5 h-3.5" /> Model directory
          </Link>
        </div>

        <div className="mb-12">
          <div className="flex items-center gap-2 text-xs font-mono text-white/45 mb-4">
            <img src={getProviderLogo(model.provider)} alt={`${model.provider} logo`} className="w-5 h-5 rounded-sm object-contain" />
            {provider ? (
              <Link to={providerPath(provider.slug)} className="hover:text-white transition-colors underline underline-offset-4 decoration-white/20">
                {model.provider}
              </Link>
            ) : (
              model.provider
            )}
          </div>
          <h1 className="text-4xl md:text-6xl font-bold tracking-tight mb-5">{model.name}</h1>
          <p className="max-w-3xl text-lg leading-relaxed text-white/62 font-light">{model.description}</p>
        </div>

        <div className="grid gap-4 md:grid-cols-3 mb-10">
          <Fact label="Model ID" value={model.id} code />
          <Fact label="Context" value={model.contextLength} />
          <Fact label="Released" value={model.released} />
          <Fact label="Input price" value={formatPrice(model, model.priceInput, 'input')} />
          <Fact label="Output price" value={formatPrice(model, model.priceOutput, 'output')} />
          <Fact label="Provider" value={model.provider} />
        </div>

        <div className="rounded-2xl border border-white/10 bg-white/[0.02] p-6 mb-10">
          <h2 className="text-xl font-bold tracking-tight mb-4">Capabilities</h2>
          <div className="flex flex-wrap gap-2">
            {model.tags.map(tag => (
              <span key={tag} className="rounded-full border border-white/10 bg-white/[0.04] px-3 py-1.5 text-xs font-mono text-white/65">
                {tag}
              </span>
            ))}
          </div>
        </div>

        <div className="flex flex-col gap-3 sm:flex-row">
          {canChat && (
            <button
              onClick={() => navigate(`/playground?model=${encodeURIComponent(model.id)}`)}
              className="inline-flex items-center justify-center gap-2 rounded border border-white bg-white px-5 py-3 font-mono text-sm font-bold text-black hover:bg-white/90 transition-colors"
            >
              <MessageSquare className="w-4 h-4" /> Chat in playground
            </button>
          )}
          {isImage && (
            <button
              onClick={() => navigate(`/playground?model=${encodeURIComponent(model.id)}&mode=image`)}
              className="inline-flex items-center justify-center gap-2 rounded border border-white bg-white px-5 py-3 font-mono text-sm font-bold text-black hover:bg-white/90 transition-colors"
            >
              <ImageIcon className="w-4 h-4" /> Generate image
            </button>
          )}
          {isVideo && (
            <button
              onClick={() => navigate(`/playground?model=${encodeURIComponent(model.id)}&mode=video`)}
              className="inline-flex items-center justify-center gap-2 rounded border border-white bg-white px-5 py-3 font-mono text-sm font-bold text-black hover:bg-white/90 transition-colors"
            >
              <Video className="w-4 h-4" /> Generate video
            </button>
          )}
          {provider && (
            <Link
              to={providerDocsPath(provider.slug)}
              className="inline-flex items-center justify-center gap-2 rounded border border-white/15 bg-white/[0.03] px-5 py-3 font-mono text-sm text-white/70 hover:text-white hover:border-white/30 transition-colors"
            >
              <BookOpen className="w-4 h-4" /> {model.provider} docs
            </Link>
          )}
        </div>

        {isVideo && (
          <section className="mt-12 rounded-2xl border border-white/10 bg-white/[0.02] overflow-hidden">
            <div className="grid gap-0 lg:grid-cols-[minmax(0,1.05fr)_minmax(0,0.95fr)]">
              <div className="border-b border-white/10 lg:border-b-0 lg:border-r">
                {videoDemo ? (
                  <video src={videoDemo.outputUrl} controls muted loop playsInline className="aspect-video w-full bg-black object-contain" />
                ) : (
                  <div className="flex aspect-video items-center justify-center bg-white/[0.03] text-sm font-mono text-white/35">
                    Generate a sample in the playground
                  </div>
                )}
                {videoDemo && (
                  <div className="grid gap-4 border-t border-white/10 p-4 md:grid-cols-[120px_minmax(0,1fr)]">
                    {videoDemo.imageUrl || videoDemo.imageUrls?.[0] ? (
                      <div>
                        <div className="mb-2 text-[10px] font-mono uppercase tracking-[0.16em] text-white/35">Input image</div>
                        <img
                          src={videoDemo.imageUrl || videoDemo.imageUrls?.[0]}
                          alt={`${model.name} reference input`}
                          className="aspect-square w-full rounded-lg border border-white/10 bg-white/[0.03] object-contain p-2"
                        />
                      </div>
                    ) : null}
                    <div className="min-w-0">
                      <div className="mb-2 text-[10px] font-mono uppercase tracking-[0.16em] text-white/35">Prompt</div>
                      <p className="text-sm leading-relaxed text-white/58">{videoDemo.prompt}</p>
                      <div className="mt-4 flex flex-wrap gap-2 text-[10px] font-mono uppercase tracking-[0.14em] text-white/35">
                        <span className="rounded border border-white/10 px-2 py-1">{videoDemo.resolution}</span>
                        <span className="rounded border border-white/10 px-2 py-1">{videoDemo.duration}s</span>
                        <span className="rounded border border-white/10 px-2 py-1">{videoDemo.aspectRatio}</span>
                      </div>
                    </div>
                  </div>
                )}
              </div>

              <div className="p-5">
                <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  <div>
                    <h2 className="text-xl font-bold tracking-tight">Video API example</h2>
                    <p className="mt-1 text-sm text-white/45">Copy the request or open it with this model selected in the playground.</p>
                  </div>
                  <button
                    type="button"
                    onClick={copyVideoSnippet}
                    className="inline-flex items-center justify-center gap-2 rounded border border-white/12 px-3 py-2 font-mono text-xs text-white/60 transition-colors hover:border-white/30 hover:text-white"
                  >
                    {copiedCode ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
                    {copiedCode ? 'Copied' : 'Copy'}
                  </button>
                </div>
                <div className="mb-3 flex flex-wrap gap-2">
                  {(['python', 'javascript', 'curl'] as const).map(lang => (
                    <button
                      key={lang}
                      type="button"
                      onClick={() => setVideoCodeLang(lang)}
                      className={`rounded px-3 py-1.5 font-mono text-xs transition-colors ${videoCodeLang === lang ? 'bg-white text-black' : 'border border-white/10 text-white/45 hover:text-white'}`}
                    >
                      {lang === 'javascript' ? 'JavaScript' : lang === 'python' ? 'Python' : 'cURL'}
                    </button>
                  ))}
                </div>
                <CodeBlock
                  code={videoSnippet}
                  language={videoCodeLang === 'curl' ? 'bash' : videoCodeLang}
                  containerClassName="overflow-hidden rounded-lg border border-white/10 bg-black/60"
                  preClassName="max-h-[420px] text-xs"
                />
                <button
                  type="button"
                  onClick={() => navigate(`/playground?model=${encodeURIComponent(model.id)}&mode=video`)}
                  className="mt-4 inline-flex w-full items-center justify-center gap-2 rounded border border-white bg-white px-4 py-2.5 font-mono text-sm font-bold text-black transition-colors hover:bg-white/90"
                >
                  <Video className="h-4 w-4" /> Use this in playground
                </button>
              </div>
            </div>
          </section>
        )}

        {imageDemo && (
          <section className="mt-12 rounded-2xl border border-white/10 bg-white/[0.02] overflow-hidden">
            <div className="grid gap-0 lg:grid-cols-[minmax(0,1.05fr)_minmax(0,0.95fr)]">
              <div className="border-b border-white/10 lg:border-b-0 lg:border-r">
                <div className="grid grid-cols-2 gap-px bg-white/10">
                  {imageDemo.imageUrl && (
                    <div className="bg-black">
                      <div className="border-b border-white/10 px-4 py-2 text-[10px] font-mono uppercase tracking-[0.16em] text-white/35">Input</div>
                      <img src={imageDemo.imageUrl} alt={`${model.name} input`} className="aspect-square w-full object-contain p-3" />
                    </div>
                  )}
                  <div className="bg-black">
                    <div className="border-b border-white/10 px-4 py-2 text-[10px] font-mono uppercase tracking-[0.16em] text-white/35">Output</div>
                    <img src={imageDemo.outputUrl} alt={`${model.name} output`} className="aspect-square w-full object-contain p-3" />
                  </div>
                </div>
                <div className="border-t border-white/10 p-5">
                  <h2 className="text-xl font-bold tracking-tight">{imageDemo.title}</h2>
                  <p className="mt-2 text-sm leading-relaxed text-white/55">{imageDemo.description}</p>
                </div>
              </div>

              <div className="p-5">
                <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  <div>
                    <h2 className="text-xl font-bold tracking-tight">Image API example</h2>
                    <p className="mt-1 text-sm text-white/45">Copy the request or open it with this model selected in the playground.</p>
                  </div>
                  <button
                    type="button"
                    onClick={copyVideoSnippet}
                    className="inline-flex items-center justify-center gap-2 rounded border border-white/12 px-3 py-2 font-mono text-xs text-white/60 transition-colors hover:border-white/30 hover:text-white"
                  >
                    {copiedCode ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
                    {copiedCode ? 'Copied' : 'Copy'}
                  </button>
                </div>
                <div className="mb-3 flex flex-wrap gap-2">
                  {(['python', 'javascript', 'curl'] as const).map(lang => (
                    <button
                      key={lang}
                      type="button"
                      onClick={() => setVideoCodeLang(lang)}
                      className={`rounded px-3 py-1.5 font-mono text-xs transition-colors ${videoCodeLang === lang ? 'bg-white text-black' : 'border border-white/10 text-white/45 hover:text-white'}`}
                    >
                      {lang === 'javascript' ? 'JavaScript' : lang === 'python' ? 'Python' : 'cURL'}
                    </button>
                  ))}
                </div>
                <CodeBlock
                  code={imageSnippet}
                  language={videoCodeLang === 'curl' ? 'bash' : videoCodeLang}
                  containerClassName="overflow-hidden rounded-lg border border-white/10 bg-black/60"
                  preClassName="max-h-[420px] text-xs"
                />
                <button
                  type="button"
                  onClick={() => navigate(`/playground?model=${encodeURIComponent(model.id)}&mode=image`)}
                  className="mt-4 inline-flex w-full items-center justify-center gap-2 rounded border border-white bg-white px-4 py-2.5 font-mono text-sm font-bold text-black transition-colors hover:bg-white/90"
                >
                  <ImageIcon className="h-4 w-4" /> Use this in playground
                </button>
              </div>
            </div>
          </section>
        )}
      </section>
    </>
  );
}

function Fact({ label, value, code = false }: { label: string; value: string; code?: boolean }) {
  return (
    <div className="rounded-xl border border-white/10 bg-white/[0.02] p-5 min-w-0">
      <div className="text-xs font-mono uppercase tracking-[0.18em] text-white/35 mb-2">{label}</div>
      {code ? (
        <code className="block truncate text-sm text-white/80">{value}</code>
      ) : (
        <div className="truncate text-lg font-semibold tracking-tight">{value}</div>
      )}
    </div>
  );
}

function isChatModel(model: Model) {
  return !model.tags.every(tag => NON_CHAT_TAGS.includes(tag)) && model.contextLength !== 'N/A';
}

function formatPrice(model: Model, price: number, label: 'input' | 'output') {
  if (model.pricingType === 'request') {
    return label === 'input' ? `${formatCurrency(price)} / request` : 'N/A';
  }
  if (model.pricingType === 'chars') {
    return label === 'input' ? `${formatCurrency(price)} / 1M chars` : 'N/A';
  }
  if (model.pricingType === 'hour') {
    return label === 'input' ? `${formatCurrency(price)} / hour` : 'N/A';
  }
  if (model.pricingType === 'second') {
    return label === 'input' ? `${formatCurrency(price)} / second` : 'N/A';
  }
  if (model.pricingType === 'megapixel') {
    return label === 'input' ? `${formatCurrency(price)} / megapixel` : 'N/A';
  }
  return `${formatCurrency(price)} / 1M tokens`;
}

function formatCurrency(value: number) {
  if (value === 0) return '$0';
  if (value < 0.01) return `$${value.toFixed(3)}`;
  return `$${value.toFixed(2)}`;
}

function buildVideoPayload(modelId: string, demo?: VideoDemo) {
  const isHappyHorse = modelId === 'alibaba/happy-horse/image-to-video';
  const payload: Record<string, unknown> = {
    model: modelId,
    prompt: demo?.prompt || 'A cinematic product demo shot of a compact AI routing console, slow camera push-in, clean studio lighting, no readable text.',
    resolution: demo?.resolution || '720p',
    duration: isHappyHorse ? Number(demo?.duration || 5) : demo?.duration || '4',
    aspect_ratio: demo?.aspectRatio || '16:9',
  };
  if (isHappyHorse) {
    payload.enable_safety_checker = true;
  } else {
    payload.generate_audio = demo?.generateAudio ?? false;
  }
  if (demo?.imageUrl) payload.image_url = demo.imageUrl;
  if (demo?.endImageUrl) payload.end_image_url = demo.endImageUrl;
  if (demo?.imageUrls?.length) payload.image_urls = demo.imageUrls;
  if (demo?.videoUrls?.length) payload.video_urls = demo.videoUrls;
  if (demo?.audioUrls?.length) payload.audio_urls = demo.audioUrls;
  return payload;
}

function generateVideoSnippet(modelId: string, demo: VideoDemo | undefined, lang: VideoCodeLanguage) {
  const apiBase = 'https://openpaths.io/v1';
  const apiKey = 'op-...';
  const payload = buildVideoPayload(modelId, demo);
  const body = JSON.stringify(payload, null, 2);

  if (lang === 'python') {
    return `from openai import OpenAI

client = OpenAI(
    api_key="${apiKey}",
    base_url="${apiBase}",
)

result = client.post(
    "/videos/generations",
    body=${JSON.stringify(payload, null, 4)},
    cast_to=dict,
)

print(result["video_url"])`;
  }

  if (lang === 'javascript') {
    return `import OpenAI from "openai";

const client = new OpenAI({
  apiKey: "${apiKey}",
  baseURL: "${apiBase}",
});

const result = await client.post("/videos/generations", {
  body: ${body},
  cast_to: Object,
});

console.log(result.video_url);`;
  }

  return `curl "${apiBase}/videos/generations" \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${apiKey}" \\
  -d @- <<'JSON'
${body}
JSON`;
}

function generateImageSnippet(demo: ImageDemo, lang: VideoCodeLanguage) {
  const apiBase = 'https://openpaths.io/v1';
  const apiKey = 'op-...';
  const payload = demo.payload;
  const body = JSON.stringify(payload, null, 2);

  if (lang === 'python') {
    return `from openai import OpenAI

client = OpenAI(
    api_key="${apiKey}",
    base_url="${apiBase}",
)

result = client.post(
    "/images/generations",
    body=${JSON.stringify(payload, null, 4)},
    cast_to=dict,
)

print(result["data"][0]["url"])`;
  }

  if (lang === 'javascript') {
    return `import OpenAI from "openai";

const client = new OpenAI({
  apiKey: "${apiKey}",
  baseURL: "${apiBase}",
});

const result = await client.post("/images/generations", {
  body: ${body},
  cast_to: Object,
});

console.log(result.data[0].url);`;
  }

  return `curl "${apiBase}/images/generations" \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${apiKey}" \\
  -d @- <<'JSON'
${body}
JSON`;
}
