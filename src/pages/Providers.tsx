import React from 'react';
import { Link } from 'react-router-dom';
import { providers } from '../data/providers';
import { models } from '../data/models';
import { ExternalLink, ArrowRight, Star } from 'lucide-react';
import { motion } from 'motion/react';

function modelCountFor(providerName: string) {
  return models.filter(m => m.provider === providerName).length;
}

export function Providers() {
  const featured = providers.filter(p => p.featured);
  const others = providers.filter(p => !p.featured);

  return (
    <div className="max-w-7xl mx-auto px-6 py-12">
      <div className="mb-12">
        <h1 className="text-4xl font-bold tracking-tight mb-4">Providers</h1>
        <p className="text-white/60 text-lg max-w-2xl font-light">
          {providers.length} providers powering {models.length}+ models. Direct API access to leading AI labs and first-party partners.
        </p>
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
              className="border border-white/20 bg-white/[0.04] rounded-xl p-6 hover:bg-white/[0.06] hover:border-white/30 transition-all flex flex-col"
            >
              <div className="flex justify-between items-start mb-3">
                <h3 className="text-xl font-bold tracking-tight">{provider.name}</h3>
                <span className="px-2 py-0.5 bg-white/10 rounded text-[10px] font-mono text-white/60">
                  {modelCountFor(provider.name)} model{modelCountFor(provider.name) !== 1 ? 's' : ''}
                </span>
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
                    className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono text-white/60 border border-white/10 rounded hover:text-white hover:border-white/30 transition-colors"
                  >
                    <ExternalLink className="w-3 h-3" /> Website
                  </a>
                )}
                <Link
                  to={`/models?q=${encodeURIComponent(provider.name)}`}
                  className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono text-white/60 border border-white/10 rounded hover:text-white hover:border-white/30 transition-colors"
                >
                  <ArrowRight className="w-3 h-3" /> View Models
                </Link>
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
              className="border border-white/10 bg-white/[0.02] rounded-xl p-6 hover:bg-white/[0.04] hover:border-white/20 transition-all flex flex-col"
            >
              <div className="flex justify-between items-start mb-3">
                <h3 className="text-lg font-bold tracking-tight">{provider.name}</h3>
                <span className="px-2 py-0.5 bg-white/10 rounded text-[10px] font-mono text-white/60">
                  {modelCountFor(provider.name)} model{modelCountFor(provider.name) !== 1 ? 's' : ''}
                </span>
              </div>
              <p className="text-sm text-white/60 font-light leading-relaxed mb-6 flex-1">
                {provider.description}
              </p>
              <div className="flex gap-3 mt-auto">
                <a
                  href={provider.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono text-white/60 border border-white/10 rounded hover:text-white hover:border-white/30 transition-colors"
                >
                  <ExternalLink className="w-3 h-3" /> Website
                </a>
                <Link
                  to={`/models?q=${encodeURIComponent(provider.name)}`}
                  className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono text-white/60 border border-white/10 rounded hover:text-white hover:border-white/30 transition-colors"
                >
                  <ArrowRight className="w-3 h-3" /> View Models
                </Link>
              </div>
            </motion.div>
          ))}
        </div>
      </div>
    </div>
  );
}
