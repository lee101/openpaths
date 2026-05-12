import React from 'react';
import { Link, Navigate, useNavigate, useParams } from 'react-router-dom';
import { ArrowLeft, BookOpen, Image as ImageIcon, MessageSquare } from 'lucide-react';
import { Seo } from '../components/Seo';
import { models, type Model } from '../data/models';
import { providersByName, getProviderLogo } from '../data/providers';
import { providerDocsPath, providerPath } from '../lib/paths';

const NON_CHAT_TAGS = ['art generation', 'video generation', 'audio', 'embedding'];

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
  const canChat = isChatModel(model);
  const title = `${model.name} API, Pricing, Context Window | OpenPaths`;
  const description = `${model.name} from ${model.provider}: ${model.description} Use model ID ${model.id} through the OpenPaths API.`;

  return (
    <>
      <Seo title={title} description={description} path={`/models/${encodeURIComponent(model.id)}`} />

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
          {provider && (
            <Link
              to={providerDocsPath(provider.slug)}
              className="inline-flex items-center justify-center gap-2 rounded border border-white/15 bg-white/[0.03] px-5 py-3 font-mono text-sm text-white/70 hover:text-white hover:border-white/30 transition-colors"
            >
              <BookOpen className="w-4 h-4" /> {model.provider} docs
            </Link>
          )}
        </div>
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
