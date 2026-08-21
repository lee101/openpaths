import React, { useMemo } from 'react';
import { Link, Navigate, useParams } from 'react-router-dom';
import { ArrowLeft, ArrowRight, BookOpen, ExternalLink, Search as SearchIcon } from 'lucide-react';
import { Seo } from '../components/Seo';
import { BflProviderShowcase } from '../components/BflProviderShowcase';
import { FALLBACK_LOGO, providers } from '../data/providers';
import { models } from '../data/models';
import { modelPath, providerDocsPath } from '../lib/paths';

export function ProviderPage() {
  const { slug = '' } = useParams<{ slug: string }>();
  const decodedSlug = decodeURIComponent(slug);
  const provider = providers.find(p => p.slug === decodedSlug);

  const providerModels = useMemo(
    () => provider ? models.filter(model => model.provider === provider.name) : [],
    [provider],
  );

  if (!provider) {
    return <Navigate to="/providers" replace />;
  }

  const isSearch = provider.kind === 'search';
  const title = isSearch
    ? `${provider.name} Search API And Docs | OpenPaths`
    : `${provider.name} Models And API Docs | OpenPaths`;
  const description = isSearch
    ? `${provider.description} Route ${provider.name} search through the OpenPaths /v1/search API.`
    : `${provider.description} Browse ${providerModels.length} ${provider.name} model${providerModels.length === 1 ? '' : 's'} available through the OpenPaths API.`;

  return (
    <>
      <Seo title={title} description={description} path={`/providers/${provider.slug}`} />

      <section className="max-w-6xl mx-auto px-6 py-16">
        <div className="mb-8">
          <Link to="/providers" className="inline-flex items-center gap-1.5 text-xs font-mono text-white/50 hover:text-white transition-colors">
            <ArrowLeft className="w-3.5 h-3.5" /> All providers
          </Link>
        </div>

        <div className="flex flex-col gap-8 md:flex-row md:items-start md:justify-between mb-12">
          <div className="flex items-start gap-5">
            <img
              src={provider.logo || FALLBACK_LOGO}
              srcSet={provider.logoSrcSet}
              sizes="56px"
              alt={`${provider.name} logo`}
              className={`w-14 h-14 rounded-xl border border-white/20 p-2 object-contain ${provider.slug === 'black-forest-labs' ? 'bg-white' : 'bg-white/[0.07]'}`}
            />
            <div>
              <div className="text-xs font-mono uppercase tracking-[0.22em] text-white/55 mb-3">Provider</div>
              <h1 className="text-4xl md:text-6xl font-bold tracking-tight mb-4">{provider.name}</h1>
              <p className="max-w-3xl text-white/62 leading-relaxed font-light">
                {provider.description}
              </p>
            </div>
          </div>

          <div className="flex flex-col sm:flex-row md:flex-col gap-3 md:min-w-44">
            <Link
              to={providerDocsPath(provider.slug)}
              className="inline-flex items-center justify-center gap-2 rounded border border-white bg-white px-4 py-3 text-sm font-mono font-bold text-black hover:bg-white/90 transition-colors"
            >
              <BookOpen className="w-4 h-4" /> API Docs
            </Link>
            {provider.url !== '/' && (
              <a
                href={provider.url}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center justify-center gap-2 rounded border border-white/15 bg-white/[0.06] px-4 py-3 text-sm font-mono text-white/70 hover:text-white hover:border-white/50 transition-colors"
              >
                <ExternalLink className="w-4 h-4" /> Website
              </a>
            )}
          </div>
        </div>

        {isSearch ? (
          <>
            <div className="grid gap-4 md:grid-cols-3 mb-12">
              <Stat label="Type" value="Search API" />
              <Stat label="Endpoint" value="/v1/search" />
              <Stat label="Provider" value={`provider: "${provider.slug}"`} />
            </div>

            <div className="rounded-xl border border-white/20 bg-white/[0.05] p-6">
              <h2 className="text-2xl font-bold tracking-tight mb-2">{provider.name} search</h2>
              <p className="text-sm text-white/58 leading-relaxed font-light max-w-3xl mb-6">
                {provider.name} is a search API, not an LLM. Route requests through the OpenPaths
                <code className="mx-1 rounded bg-white/10 px-1.5 py-0.5 text-xs text-white/70">/v1/search</code>
                endpoint with <code className="mx-1 rounded bg-white/10 px-1.5 py-0.5 text-xs text-white/70">provider: "{provider.slug}"</code>,
                or try it live in the search playground.
              </p>
              <div className="flex flex-wrap gap-3">
                <Link
                  to="/search"
                  className="inline-flex items-center gap-2 rounded border border-white bg-white px-4 py-2.5 text-sm font-mono font-bold text-black hover:bg-white/90 transition-colors"
                >
                  <SearchIcon className="w-4 h-4" /> Try search
                </Link>
                <Link
                  to={providerDocsPath(provider.slug)}
                  className="inline-flex items-center gap-2 rounded border border-white/15 bg-white/[0.06] px-4 py-2.5 text-sm font-mono text-white/70 hover:text-white hover:border-white/50 transition-colors"
                >
                  <BookOpen className="w-4 h-4" /> API docs
                </Link>
              </div>
            </div>
          </>
        ) : (
          <>
            <div className="grid gap-4 md:grid-cols-3 mb-12">
              <Stat label="Models" value={providerModels.length.toString()} />
              <Stat label="Lowest input price" value={formatLowestPrice(providerModels)} />
              <Stat label="Capabilities" value={capabilities(providerModels)} />
            </div>

            {provider.slug === 'black-forest-labs' ? (
              <BflProviderShowcase models={providerModels} />
            ) : (
              <>
            <div className="mb-5 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
              <div>
                <h2 className="text-2xl font-bold tracking-tight">Available {provider.name} models</h2>
                <p className="text-sm text-white/55 font-light mt-1">
                  Crawlable model pages for pricing, context window, tags, and example launch links.
                </p>
              </div>
              <Link to={`/models?q=${encodeURIComponent(provider.name)}`} className="text-xs font-mono text-white/50 hover:text-white transition-colors">
                Filter directory <ArrowRight className="inline w-3 h-3" />
              </Link>
            </div>

            <div className="grid gap-4 md:grid-cols-2">
              {providerModels.map(model => (
                <Link
                  key={model.id}
                  to={modelPath(model.id)}
                  className="rounded-xl border border-white/20 bg-white/[0.05] p-5 hover:border-white/45 hover:bg-white/[0.07] transition-colors"
                >
                  <div className="flex items-start justify-between gap-4 mb-3">
                    <div>
                      <h3 className="font-bold tracking-tight text-lg">{model.name}</h3>
                      <code className="mt-1 block text-xs text-white/55 truncate">{model.id}</code>
                    </div>
                    <span className="shrink-0 rounded bg-white/10 px-2 py-1 text-[10px] font-mono text-white/60">
                      {model.contextLength !== 'N/A' ? `${model.contextLength} ctx` : model.tags[0]}
                    </span>
                  </div>
                  <p className="text-sm text-white/58 leading-relaxed font-light line-clamp-2">{model.description}</p>
                </Link>
              ))}
            </div>
              </>
            )}
          </>
        )}
      </section>
    </>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-white/20 bg-white/[0.05] p-5">
      <div className="text-xs font-mono uppercase tracking-[0.18em] text-white/50 mb-2">{label}</div>
      <div className="text-2xl font-bold tracking-tight">{value}</div>
    </div>
  );
}

function formatLowestPrice(items: typeof models) {
  if (!items.length) return 'N/A';
  const min = Math.min(...items.map(model => model.priceInput));
  if (min === 0) return '$0';
  if (min < 0.01) return `$${min.toFixed(3)}`;
  return `$${min.toFixed(2)}`;
}

function capabilities(items: typeof models) {
  const tags = Array.from(new Set(items.flatMap(model => model.tags)));
  if (!tags.length) return 'API';
  return tags.slice(0, 2).join(', ');
}
