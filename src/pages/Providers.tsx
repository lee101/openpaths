import React from 'react';
import { Link } from 'react-router-dom';
import { providers, FALLBACK_LOGO, Provider } from '../data/providers';
import { models } from '../data/models';
import { ExternalLink, ArrowRight, Star, Search as SearchIcon } from 'lucide-react';
import { motion } from 'motion/react';
import { Seo } from '../components/Seo';
import { providerDocsPath, providerPath } from '../lib/paths';

function modelCountFor(providerName: string) {
  return models.filter(m => m.provider === providerName).length;
}

function ProviderBadge({ provider }: { provider: Provider }) {
  if (provider.kind === 'search') {
    return (
      <span className="inline-flex items-center gap-1 px-2 py-0.5 bg-white/10 rounded text-[10px] font-mono text-white/60">
        <SearchIcon className="w-2.5 h-2.5" /> Search API
      </span>
    );
  }
  const count = modelCountFor(provider.name);
  return (
    <span className="px-2 py-0.5 bg-white/10 rounded text-[10px] font-mono text-white/60">
      {count} model{count !== 1 ? 's' : ''}
    </span>
  );
}

export function Providers() {
  const featured = providers.filter(p => p.featured);
  const others = providers.filter(p => !p.featured);

  return (
    <>
      <Seo
        title="AI Providers | OpenPaths"
        description={`Browse ${providers.length} AI providers available through OpenPaths, including OpenAI, Anthropic, Google, xAI, Netwrck, Text-Generator.io, Fal, and more.`}
        path="/providers"
      />

      <div className="max-w-7xl mx-auto px-6 py-12">
      <div className="mb-12">
        <h1 className="text-4xl font-bold tracking-tight mb-4">Providers</h1>
        <p className="text-white/60 text-lg max-w-2xl font-light">
          {providers.length} providers powering {models.length}+ models. Direct API access to leading AI labs and first-party partners.
        </p>
      </div>

      <div className="mb-12">
        <div className="mb-4 text-xs font-mono uppercase tracking-[0.22em] text-white/55">Jump to provider</div>
        <div className="flex flex-wrap gap-2">
          {providers.map(provider => (
            <a
              key={provider.slug}
              href={`#${provider.slug}`}
              className="rounded-full border border-white/20 bg-white/[0.06] px-3 py-1.5 font-mono text-xs text-white/65 hover:border-white/50 hover:bg-white/[0.09] hover:text-white transition-colors"
            >
              {provider.name}
            </a>
          ))}
        </div>
      </div>

      {/* Featured / First-Party */}
      <div className="mb-12">
        <div className="flex items-center gap-2 mb-6">
          <Star className="w-4 h-4 text-white/60" />
          <h2 className="text-sm font-mono text-white/60 uppercase tracking-wider">First-Party Partners</h2>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          {featured.map((provider, idx) => (
            <motion.div
              key={provider.slug}
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3, delay: idx * 0.05 }}
              id={provider.slug}
              className="scroll-mt-24 border border-white/20 bg-white/[0.07] rounded-xl p-6 hover:bg-white/[0.09] hover:border-white/50 transition-all flex flex-col"
            >
              <div className="flex justify-between items-start mb-3">
                <div className="flex items-center gap-3">
                  <img src={provider.logoSmall || provider.logo || FALLBACK_LOGO} srcSet={provider.logoSrcSet} sizes="32px" alt="" className={`w-8 h-8 rounded object-contain ${provider.slug === 'black-forest-labs' ? 'bg-white p-0.5' : ''}`} />
                  <Link to={providerPath(provider.slug)} className="text-xl font-bold tracking-tight hover:underline underline-offset-4">
                    {provider.name}
                  </Link>
                </div>
                <ProviderBadge provider={provider} />
              </div>
              <p className="text-sm text-white/60 font-light leading-relaxed mb-6 flex-1">
                {provider.description}
              </p>
              <div className="flex gap-3 mt-auto">
                {provider.url !== '/' && (
                  <a
                    href={provider.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono text-white/60 border border-white/20 rounded hover:text-white hover:border-white/50 transition-colors"
                  >
                    <ExternalLink className="w-3 h-3" /> Website
                  </a>
                )}
                <Link
                  to={providerDocsPath(provider.slug)}
                  className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono text-white/60 border border-white/20 rounded hover:text-white hover:border-white/50 transition-colors"
                >
                  <ArrowRight className="w-3 h-3" /> Docs
                </Link>
                <Link
                  to={providerPath(provider.slug)}
                  className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono text-white/60 border border-white/20 rounded hover:text-white hover:border-white/50 transition-colors"
                >
                  <ArrowRight className="w-3 h-3" /> Provider Page
                </Link>
                {provider.kind === 'search' ? (
                  <Link
                    to="/search"
                    className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono text-white/60 border border-white/20 rounded hover:text-white hover:border-white/50 transition-colors"
                  >
                    <ArrowRight className="w-3 h-3" /> Try Search
                  </Link>
                ) : (
                  <Link
                    to={`/models?q=${encodeURIComponent(provider.name)}`}
                    className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono text-white/60 border border-white/20 rounded hover:text-white hover:border-white/50 transition-colors"
                  >
                    <ArrowRight className="w-3 h-3" /> View Models
                  </Link>
                )}
              </div>
            </motion.div>
          ))}
        </div>
      </div>

      {/* All Providers */}
      <div>
        <h2 className="text-sm font-mono text-white/60 uppercase tracking-wider mb-6">All Providers</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {others.map((provider, idx) => (
            <motion.div
              key={provider.slug}
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3, delay: idx * 0.05 }}
              id={provider.slug}
              className="scroll-mt-24 border border-white/20 bg-white/[0.05] rounded-xl p-6 hover:bg-white/[0.07] hover:border-white/40 transition-all flex flex-col"
            >
              <div className="flex justify-between items-start mb-3">
                <div className="flex items-center gap-3">
                  <img src={provider.logoSmall || provider.logo || FALLBACK_LOGO} srcSet={provider.logoSrcSet} sizes="28px" alt="" className={`w-7 h-7 rounded object-contain ${provider.slug === 'black-forest-labs' ? 'bg-white p-0.5' : ''}`} />
                  <Link to={providerPath(provider.slug)} className="text-lg font-bold tracking-tight hover:underline underline-offset-4">
                    {provider.name}
                  </Link>
                </div>
                <ProviderBadge provider={provider} />
              </div>
              <p className="text-sm text-white/60 font-light leading-relaxed mb-6 flex-1">
                {provider.description}
              </p>
              <div className="flex gap-3 mt-auto">
                <a
                  href={provider.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono text-white/60 border border-white/20 rounded hover:text-white hover:border-white/50 transition-colors"
                >
                  <ExternalLink className="w-3 h-3" /> Website
                </a>
                <Link
                  to={providerDocsPath(provider.slug)}
                  className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono text-white/60 border border-white/20 rounded hover:text-white hover:border-white/50 transition-colors"
                >
                  <ArrowRight className="w-3 h-3" /> Docs
                </Link>
                <Link
                  to={providerPath(provider.slug)}
                  className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono text-white/60 border border-white/20 rounded hover:text-white hover:border-white/50 transition-colors"
                >
                  <ArrowRight className="w-3 h-3" /> Provider Page
                </Link>
                {provider.kind === 'search' ? (
                  <Link
                    to="/search"
                    className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono text-white/60 border border-white/20 rounded hover:text-white hover:border-white/50 transition-colors"
                  >
                    <ArrowRight className="w-3 h-3" /> Try Search
                  </Link>
                ) : (
                  <Link
                    to={`/models?q=${encodeURIComponent(provider.name)}`}
                    className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono text-white/60 border border-white/20 rounded hover:text-white hover:border-white/50 transition-colors"
                  >
                    <ArrowRight className="w-3 h-3" /> View Models
                  </Link>
                )}
              </div>
            </motion.div>
          ))}
        </div>
      </div>
      </div>
    </>
  );
}
