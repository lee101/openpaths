import React, { useMemo } from 'react';
import { Link } from 'react-router-dom';
import { motion } from 'motion/react';
import { ArrowRight, CheckCircle2, Layers, Route, WalletCards, Zap } from 'lucide-react';
import { posts } from '../data/blog';
import { Seo } from '../components/Seo';

const comparisonHighlights = [
  {
    icon: <WalletCards className="h-5 w-5" />,
    title: 'One Credit Pool',
    copy: 'Fund one OpenPaths balance and spend across OpenAI, Anthropic, Google, xAI, DeepSeek, Mistral, Fal, Netwrck, Text-Generator.io, and more.',
  },
  {
    icon: <Route className="h-5 w-5" />,
    title: 'Task-Based Routing',
    copy: 'Use stable model names like auto-medium-task or auto-think while the router chooses stronger, cheaper, or healthier backends.',
  },
  {
    icon: <Layers className="h-5 w-5" />,
    title: 'More Than Chat',
    copy: 'Route chat, images, video, audio, embeddings, search, and image-to-3D from one OpenAI-compatible platform.',
  },
  {
    icon: <Zap className="h-5 w-5" />,
    title: 'Production Fallbacks',
    copy: 'Keep direct provider lanes, partner models, and OpenRouter-style fallback coverage without forcing every app feature to own routing logic.',
  },
];

function competitorName(postTitle: string) {
  return postTitle.split(' Alternative:')[0] || postTitle;
}

export function Alternatives() {
  const alternativePosts = useMemo(
    () => posts.filter(post => post.alternativePath),
    []
  );

  return (
    <>
      <Seo
        title="OpenPaths Alternatives | OpenRouter, OpenAI, Anthropic, Together AI"
        description="Compare OpenPaths against OpenRouter, OpenAI API, Anthropic API, Together AI, and other AI model routing alternatives."
        path="/alternatives"
      />

      <div className="mx-auto max-w-7xl px-6 py-14">
        <section className="mb-14 max-w-4xl">
          <div className="mb-4 inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/[0.03] px-3 py-1 font-mono text-xs uppercase tracking-[0.18em] text-white/45">
            Alternatives
          </div>
          <h1 className="text-4xl font-semibold tracking-tight md:text-6xl">
            Compare OpenPaths with other AI gateways.
          </h1>
          <p className="mt-5 max-w-3xl text-lg leading-relaxed text-white/58">
            OpenRouter, OpenAI, Anthropic, and Together AI are useful tools. OpenPaths is built for teams that want one API key, one credit pool, task-aware routing, and multimodal provider coverage without turning model selection into an operations project.
          </p>
          <div className="mt-8 flex flex-wrap gap-3">
            <Link to="/account" className="inline-flex items-center gap-2 rounded bg-white px-4 py-2 font-mono text-sm font-bold text-black transition-colors hover:bg-white/90">
              Start routing <ArrowRight className="h-4 w-4" />
            </Link>
            <Link to="/pricing" className="inline-flex items-center gap-2 rounded border border-white/12 px-4 py-2 font-mono text-sm text-white/70 transition-colors hover:border-white/30 hover:text-white">
              View pricing
            </Link>
          </div>
        </section>

        <section className="mb-14 grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          {comparisonHighlights.map((item) => (
            <div key={item.title} className="rounded-lg border border-white/10 bg-white/[0.02] p-5">
              <div className="mb-4 flex h-10 w-10 items-center justify-center rounded border border-white/10 bg-white/[0.04] text-white/70">
                {item.icon}
              </div>
              <h2 className="mb-2 text-lg font-semibold tracking-tight">{item.title}</h2>
              <p className="text-sm leading-relaxed text-white/48">{item.copy}</p>
            </div>
          ))}
        </section>

        <section className="mb-16 overflow-hidden rounded-lg border border-white/10 bg-white/[0.02]">
          <div className="grid border-b border-white/10 md:grid-cols-3">
            <div className="p-5">
              <div className="font-mono text-xs uppercase tracking-[0.18em] text-white/40">Compare</div>
              <div className="mt-2 text-xl font-semibold">OpenPaths vs the usual options</div>
            </div>
            <div className="border-t border-white/10 p-5 md:border-l md:border-t-0">
              <div className="font-mono text-xs uppercase tracking-[0.18em] text-white/40">Typical gateway</div>
              <p className="mt-2 text-sm leading-relaxed text-white/50">One catalog, but often marketplace-scoped billing, manual model selection, and uneven coverage outside chat.</p>
            </div>
            <div className="border-t border-white/10 p-5 md:border-l md:border-t-0">
              <div className="font-mono text-xs uppercase tracking-[0.18em] text-white/40">OpenPaths</div>
              <p className="mt-2 text-sm leading-relaxed text-white/50">One OpenAI-compatible API for routed text, media, embeddings, search, and 3D generation with pooled credits and fallbacks.</p>
            </div>
          </div>
          <div className="grid gap-0 md:grid-cols-3">
            {['OpenAI SDK shape', 'Cross-provider pooled credits', 'Auto-thinking routes', 'Image, video, audio, embeddings', 'Direct providers plus fallbacks', 'Open-source router code'].map((feature) => (
              <div key={feature} className="flex items-center gap-3 border-b border-white/8 px-5 py-4 text-sm text-white/58 md:border-r">
                <CheckCircle2 className="h-4 w-4 flex-none text-emerald-300" />
                {feature}
              </div>
            ))}
          </div>
        </section>

        <section>
          <div className="mb-6 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <div className="font-mono text-xs uppercase tracking-[0.18em] text-white/40">Alternative Guides</div>
              <h2 className="mt-2 text-3xl font-semibold tracking-tight">Pick the comparison that matches your stack.</h2>
            </div>
            <Link to="/blog" className="font-mono text-sm text-white/45 transition-colors hover:text-white">
              All posts
            </Link>
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            {alternativePosts.map((post, index) => (
              <motion.div
                key={post.slug}
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.25, delay: index * 0.04 }}
              >
                <Link
                  to={post.alternativePath || `/blog/${post.slug}`}
                  className="group block h-full rounded-lg border border-white/10 bg-white/[0.02] p-6 transition-all hover:border-white/24 hover:bg-white/[0.045]"
                >
                  <div className="mb-4 flex flex-wrap items-center gap-2">
                    <span className="rounded border border-white/10 bg-white/[0.04] px-2 py-1 font-mono text-[10px] uppercase tracking-[0.14em] text-white/45">
                      {competitorName(post.title)}
                    </span>
                    <span className="font-mono text-xs text-white/28">{post.readTime}</span>
                  </div>
                  <h3 className="mb-3 text-xl font-semibold tracking-tight transition-colors group-hover:text-white">
                    {post.title}
                  </h3>
                  <p className="mb-6 text-sm leading-relaxed text-white/50">
                    {post.excerpt}
                  </p>
                  <div className="flex items-center gap-2 font-mono text-sm text-white/40 transition-colors group-hover:text-white">
                    Read comparison <ArrowRight className="h-4 w-4" />
                  </div>
                </Link>
              </motion.div>
            ))}
          </div>
        </section>
      </div>
    </>
  );
}
