import React from 'react';
import { Link, Navigate, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { ArrowLeft, ArrowRight, BookOpen, Gift, Image as ImageIcon, MessageSquare, Video } from 'lucide-react';
import { ImageSpacePanel } from '../components/ImageSpacePanel';
import { VideoSpacePanel } from '../components/VideoSpacePanel';
import { Seo } from '../components/Seo';
import { isPromotionActive, models, type Model } from '../data/models';
import { providersByName, getProviderLogo } from '../data/providers';
import { IMAGE_DEMOS } from '../data/imageDemos';
import { VIDEO_DEMOS } from '../data/videoDemos';
import { modelPath, providerDocsPath, providerPath } from '../lib/paths';

const NON_CHAT_TAGS = ['art generation', 'video generation', 'audio', 'embedding'];

export function ModelPage() {
  const { modelId = '' } = useParams<{ modelId: string }>();
  const [searchParams] = useSearchParams();
  const initialPrompt = searchParams.get('prompt') || undefined;
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
  const relatedModels = getRelatedModels(model);
  const promotionActive = isPromotionActive(model);
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

  const scrollToWorkspace = () => {
    document.getElementById('model-workspace')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
  };

  return (
    <>
      <Seo title={title} description={description} path={`/models/${encodeURIComponent(model.id)}`} image={model.ogImage} jsonLd={jsonLd} />

      <section className="max-w-5xl mx-auto px-6 py-16">
        <div className="mb-8">
          <Link to="/models" className="inline-flex items-center gap-1.5 text-xs font-mono text-white/50 hover:text-white transition-colors">
            <ArrowLeft className="w-3.5 h-3.5" /> Model directory
          </Link>
        </div>

        <div className="mb-12">
          <div className="flex items-center gap-2 text-xs font-mono text-white/45 mb-4">
            <img src={getProviderLogo(model.provider)} alt={`${model.provider} logo`} className={`w-5 h-5 rounded-sm object-contain ${model.provider === 'Black Forest Labs' ? 'bg-white p-px' : ''}`} />
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

        {promotionActive && model.promotion && (
          <div className="mb-10 rounded-2xl border border-emerald-300/20 bg-emerald-300/[0.06] p-6">
            <div className="flex items-start gap-3">
              <Gift className="mt-0.5 h-5 w-5 shrink-0 text-emerald-200" />
              <div>
                <h2 className="font-semibold text-emerald-50">OpenRouter-exclusive launch discount: 50% off through August 27</h2>
                <p className="mt-2 text-sm leading-relaxed text-emerald-50/65">
                  OpenPaths is routing {model.name} through OpenRouter and passing the full savings on to customers: {formatCurrency(model.priceInput)} / 1M input tokens and {formatCurrency(model.priceOutput)} / 1M output tokens. Standard {formatCurrency(model.promotion.standardPriceInput)} / {formatCurrency(model.promotion.standardPriceOutput)} rates and direct Google routing resume August 28.
                </p>
              </div>
            </div>
          </div>
        )}

        <div className="grid gap-4 md:grid-cols-3 mb-10">
          <Fact label="Model ID" value={model.id} code />
          <Fact label="Context" value={model.contextLength} />
          <Fact label="Released" value={model.released} />
          <Fact label="Input price" value={formatPrice(model, model.priceInput, 'input')} />
          <Fact label="Output price" value={formatPrice(model, model.priceOutput, 'output')} />
          <Fact label="Provider" value={model.provider} />
        </div>

        <div className="rounded-2xl border border-white/20 bg-white/[0.05] p-6 mb-10">
          <h2 className="text-xl font-bold tracking-tight mb-4">Capabilities</h2>
          <div className="flex flex-wrap gap-2">
            {model.tags.map(tag => (
              <span key={tag} className="rounded-full border border-white/20 bg-white/[0.07] px-3 py-1.5 text-xs font-mono text-white/65">
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
              onClick={scrollToWorkspace}
              className="inline-flex items-center justify-center gap-2 rounded border border-white bg-white px-5 py-3 font-mono text-sm font-bold text-black hover:bg-white/90 transition-colors"
            >
              <ImageIcon className="w-4 h-4" /> Open image workspace
            </button>
          )}
          {isVideo && (
            <button
              onClick={scrollToWorkspace}
              className="inline-flex items-center justify-center gap-2 rounded border border-white bg-white px-5 py-3 font-mono text-sm font-bold text-black hover:bg-white/90 transition-colors"
            >
              <Video className="w-4 h-4" /> Open video workspace
            </button>
          )}
          {provider && (
            <Link
              to={providerDocsPath(provider.slug)}
              className="inline-flex items-center justify-center gap-2 rounded border border-white/15 bg-white/[0.06] px-5 py-3 font-mono text-sm text-white/70 hover:text-white hover:border-white/50 transition-colors"
            >
              <BookOpen className="w-4 h-4" /> {model.provider} docs
            </Link>
          )}
        </div>

        {isVideo && (
          <section id="model-workspace" className="mt-12 scroll-mt-24 overflow-hidden rounded-2xl border border-white/20 bg-white/[0.05]">
            <VideoSpacePanel modelId={model.id} modelName={model.name} demo={videoDemo} initialPrompt={initialPrompt} />
          </section>
        )}

        {isImage && (
          <section id={isVideo ? undefined : 'model-workspace'} className="mt-12 scroll-mt-24 rounded-2xl border border-white/20 bg-white/[0.05] overflow-hidden">
            <ImageSpacePanel
              modelId={model.id}
              modelName={model.name}
              imageToImage={model.tags.includes('image-to-image')}
              demo={imageDemo}
              initialPrompt={initialPrompt}
            />
          </section>
        )}

        <RelatedModels current={model} related={relatedModels} />
      </section>
    </>
  );
}

function Fact({ label, value, code = false }: { label: string; value: string; code?: boolean }) {
  return (
    <div className="rounded-xl border border-white/20 bg-white/[0.05] p-5 min-w-0">
      <div className="text-xs font-mono uppercase tracking-[0.18em] text-white/50 mb-2">{label}</div>
      {code ? (
        <code className="block truncate text-sm text-white/80">{value}</code>
      ) : (
        <div className="truncate text-lg font-semibold tracking-tight">{value}</div>
      )}
    </div>
  );
}

function isChatModel(model: Model) {
  return !model.tags.some(tag => NON_CHAT_TAGS.includes(tag)) && model.contextLength !== 'N/A';
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
  const precision = value < 0.01 || Math.abs(value * 100 - Math.round(value * 100)) > 1e-9 ? 3 : 2;
  return `$${value.toFixed(precision)}`;
}

function mediaTask(model: Model) {
  const id = model.id.toLowerCase();
  if (id.includes('image-to-video')) return 'image-to-video';
  if (id.includes('reference-to-video')) return 'reference-to-video';
  if (id.includes('text-to-video')) return 'text-to-video';
  if (model.tags.includes('outpainting')) return 'outpainting';
  if (model.tags.includes('image-to-image')) return 'image-to-image';
  if (model.tags.includes('text-to-image') || model.tags.includes('art generation')) return 'text-to-image';
  return model.tags[0] || 'model';
}

function getRelatedModels(current: Model) {
  const task = mediaTask(current);
  return models
    .filter(candidate => candidate.id !== current.id)
    .map(candidate => {
      let score = 0;
      if (candidate.provider === current.provider) score += 6;
      if (mediaTask(candidate) === task) score += 8;
      score += candidate.tags.filter(tag => current.tags.includes(tag)).length;
      if (candidate.id.split('/').slice(0, -2).join('/') === current.id.split('/').slice(0, -2).join('/')) score += 3;
      return { candidate, score };
    })
    .filter(item => item.score >= 4)
    .sort((a, b) => b.score - a.score || b.candidate.popularity - a.candidate.popularity)
    .slice(0, 6)
    .map(item => item.candidate);
}

function RelatedModels({ current, related }: { current: Model; related: Model[] }) {
  const task = mediaTask(current);
  if (!related.length) return null;
  return (
    <section className="mt-12 border-t border-white/20 pt-10" aria-labelledby="related-models-heading">
      <div className="mb-5 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h2 id="related-models-heading" className="text-2xl font-bold tracking-tight">Related {task} models</h2>
          <p className="mt-1 text-sm text-white/45">Compare similar APIs without losing the model-specific workflow.</p>
        </div>
        <Link to={`/models?q=${encodeURIComponent(task.replaceAll('-', ' '))}`} className="font-mono text-xs text-white/50 transition-colors hover:text-white">
          Browse all {task} models <ArrowRight className="inline h-3.5 w-3.5" />
        </Link>
      </div>
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {related.map(model => (
          <Link key={model.id} to={modelPath(model.id)} className="group rounded-xl border border-white/20 bg-white/[0.05] p-4 transition-colors hover:border-white/45 hover:bg-white/[0.07]">
            <div className="mb-2 flex items-start justify-between gap-3">
              <h3 className="font-semibold tracking-tight text-white/85 group-hover:text-white">{model.name}</h3>
              <ArrowRight className="mt-1 h-3.5 w-3.5 shrink-0 text-white/40 group-hover:text-white/65" />
            </div>
            <code className="block truncate text-[11px] text-white/50">{model.id}</code>
            <p className="mt-3 line-clamp-2 text-xs leading-relaxed text-white/48">{model.description}</p>
          </Link>
        ))}
      </div>
      <div className="mt-5 flex flex-wrap gap-x-5 gap-y-2 font-mono text-xs text-white/45">
        <Link to={providerPath(providersByName[current.provider]?.slug || current.provider.toLowerCase())} className="hover:text-white">More from {current.provider} →</Link>
        {current.tags.includes('video generation') && <Link to="/blog/video-model-tips-image-to-video-encoding" className="hover:text-white">Image-to-video guide →</Link>}
        {current.tags.includes('art generation') && <Link to="/image-evals" className="hover:text-white">Image model benchmarks →</Link>}
      </div>
    </section>
  );
}
