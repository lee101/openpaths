import { Link } from 'react-router-dom';
import { motion } from 'motion/react';
import { ArrowRight, CheckCircle2, CreditCard, Route, Sparkles } from 'lucide-react';
import { posts } from '../data/blog';
import { Seo } from '../components/Seo';

const firstPartyGuides = [
  {
    slug: 'provider-manifoldgen',
    name: 'ManifoldGen',
    body: 'First-party video, music, speech, image, and creative tools from the ManifoldGen GPU studio.',
  },
  {
    slug: 'provider-netwrck',
    name: 'Netwrck',
    body: 'RA1 images, ZImage art, and RA2V video for the creative workloads that need a dependable lane.',
  },
  {
    slug: 'provider-cutedsl',
    name: 'CuteDSL',
    body: 'Triton-accelerated Z-Image generation and Chronos2 time-series forecasting.',
  },
  {
    slug: 'provider-text-generator',
    name: 'Text-Generator.io',
    body: 'ModernBERT embeddings for semantic search, RAG, clustering, and recommendations.',
  },
  {
    slug: 'provider-papers',
    name: 'Papers',
    body: 'Research search for agents: papers, methods, datasets, and GitHub code in compact markdown.',
  },
  {
    slug: 'provider-openpaths',
    name: 'OpenPaths',
    body: 'Auto routes that pick the right model for the task, with fallbacks when a provider has a bad day.',
  },
] as const;

const comparisonRows = [
  ['Platform markup', '0%', '5.5% on pay-as-you-go'],
  ['Billing', 'One balance across the stack', 'OpenRouter credits'],
  ['Model choice', 'Pin a model or use task-aware auto routes', 'Browse and select models'],
  ['Beyond chat', 'Images, video, audio, embeddings, search, and 3D', 'Catalog-dependent'],
  ['Fallbacks', 'Direct lanes plus provider fallbacks', 'OpenRouter routing'],
] as const;

const faqs = [
  {
    question: 'Is OpenPaths really 0% markup?',
    answer: 'OpenPaths does not add a platform markup to the listed model rates. We make money from first-party services we operate, and publish current model prices on the models page.',
  },
  {
    question: 'What does OpenRouter charge?',
    answer: 'OpenRouter currently lists a 5.5% platform fee on its pay-as-you-go plan. Free-model and enterprise terms are separate.',
  },
  {
    question: 'What is the practical difference?',
    answer: 'OpenRouter is a broad model marketplace. OpenPaths is a routing layer with one pooled balance, task-aware auto routes, direct provider lanes, first-party media, and fallbacks under one API.',
  },
] as const;

const alternativesJsonLd = {
  '@context': 'https://schema.org',
  '@graph': [
    {
      '@type': 'WebPage',
      name: 'OpenRouter Alternative | 0% AI Gateway Markup | OpenPaths',
      url: 'https://openpaths.io/alternatives',
      description: 'Compare OpenPaths with OpenRouter and other AI gateways by markup, routing, billing, and provider coverage.',
    },
    {
      '@type': 'FAQPage',
      mainEntity: faqs.map(faq => ({
        '@type': 'Question',
        name: faq.question,
        acceptedAnswer: {
          '@type': 'Answer',
          text: faq.answer,
        },
      })),
    },
  ],
};

function competitorName(postTitle: string) {
  return postTitle.split(' Alternative:')[0] || postTitle;
}

export function Alternatives() {
  const alternativePosts = posts.filter(post => post.alternativePath);

  return (
    <>
      <Seo
        title="OpenRouter Alternative | 0% AI Gateway Markup | OpenPaths"
        description="Compare OpenPaths with OpenRouter, OpenAI, Anthropic, LiteLLM, and more. One API, pooled credits, smart routing, and 0% platform markup."
        path="/alternatives"
        jsonLd={alternativesJsonLd}
      />

      <div className="mx-auto max-w-7xl px-6 py-14">
        <section className="mb-12 max-w-4xl">
          <div className="mb-4 inline-flex items-center gap-2 rounded-full border border-white/20 bg-white/[0.06] px-3 py-1 font-mono text-xs uppercase tracking-[0.18em] text-white/45">
            AI gateway alternatives
          </div>
          <h1 className="text-4xl font-semibold tracking-tight md:text-6xl">
            Compare OpenPaths with other AI gateways.
          </h1>
          <p className="mt-5 max-w-3xl text-xl leading-relaxed text-white/65">
            Stop paying the gateway tax. OpenPaths gives you one OpenAI-compatible API, one balance, and 0% platform markup on listed model rates.
          </p>
          <p className="mt-3 max-w-3xl text-base leading-relaxed text-white/45">
            Use the model you want, or let task-aware routing choose it. Chat, code, images, video, audio, embeddings, search, and 3D all sit behind the same key.
          </p>
          <div className="mt-8 flex flex-wrap gap-3">
            <Link to="/account" className="inline-flex items-center gap-2 rounded bg-white px-4 py-2 font-mono text-sm font-bold text-black transition-colors hover:bg-white/90">
              Start routing <ArrowRight className="h-4 w-4" />
            </Link>
            <Link to="/pricing" className="inline-flex items-center gap-2 rounded border border-white/12 px-4 py-2 font-mono text-sm text-white/70 transition-colors hover:border-white/50 hover:text-white">
              See pricing
            </Link>
          </div>
        </section>

        <section aria-label="Platform markup comparison" className="mb-14 grid gap-4 md:grid-cols-2">
          <div className="rounded-xl border border-emerald-200/30 bg-emerald-200/[0.08] p-6">
            <div className="flex items-start justify-between gap-4">
              <div>
                <div className="font-mono text-xs uppercase tracking-[0.18em] text-emerald-100/65">OpenPaths</div>
                <div className="mt-3 text-6xl font-semibold tracking-tight text-emerald-100">0%</div>
                <div className="mt-1 text-lg text-white/75">platform markup</div>
              </div>
              <CreditCard className="h-6 w-6 text-emerald-200/70" />
            </div>
            <p className="mt-5 max-w-md text-sm leading-relaxed text-white/60">
              We keep the spread at zero on listed model rates and earn from services we run ourselves. BYOK requests are also $0 through OpenPaths.
            </p>
          </div>

          <div className="rounded-xl border border-white/20 bg-white/[0.05] p-6">
            <div className="flex items-start justify-between gap-4">
              <div>
                <div className="font-mono text-xs uppercase tracking-[0.18em] text-white/50">OpenRouter</div>
                <div className="mt-3 text-6xl font-semibold tracking-tight text-white/85">5.5%</div>
                <div className="mt-1 text-lg text-white/65">pay-as-you-go fee</div>
              </div>
              <CreditCard className="h-6 w-6 text-white/40" />
            </div>
            <p className="mt-5 max-w-md text-sm leading-relaxed text-white/50">
              OpenRouter’s current pricing page lists a 5.5% platform fee on pay-as-you-go usage. <a href="https://openrouter.ai/pricing" target="_blank" rel="noopener noreferrer" className="text-white/75 underline decoration-white/30 underline-offset-4 hover:text-white">See their pricing.</a>
            </p>
          </div>
        </section>

        <section className="mb-16 overflow-hidden rounded-xl border border-white/20 bg-white/[0.04]">
          <div className="border-b border-white/20 p-6">
            <div className="mb-2 font-mono text-xs uppercase tracking-[0.18em] text-white/45">The short version</div>
            <h2 className="text-3xl font-semibold tracking-tight">A cheaper route to a bigger stack.</h2>
            <p className="mt-3 max-w-2xl text-sm leading-relaxed text-white/50">
              $100 of usage means $0 in OpenPaths platform markup versus $5.50 in OpenRouter’s listed pay-as-you-go fee. The bigger win is operational: one key, one budget, fewer provider dashboards.
            </p>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full min-w-[680px] text-left text-sm">
              <thead>
                <tr className="border-b border-white/20 font-mono text-xs uppercase tracking-[0.14em] text-white/45">
                  <th className="px-6 py-4 font-normal">Need</th>
                  <th className="px-6 py-4 font-normal text-emerald-100/75">OpenPaths</th>
                  <th className="px-6 py-4 font-normal text-white/55">OpenRouter</th>
                </tr>
              </thead>
              <tbody>
                {comparisonRows.map(([need, openPaths, openRouter]) => (
                  <tr key={need} className="border-b border-white/10 last:border-0">
                    <th className="px-6 py-4 font-medium text-white/70">{need}</th>
                    <td className="px-6 py-4 text-white/60"><span className="inline-flex items-start gap-2"><CheckCircle2 className="mt-0.5 h-4 w-4 flex-none text-emerald-300" />{openPaths}</span></td>
                    <td className="px-6 py-4 text-white/45">{openRouter}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        <section className="mb-16">
          <div className="mb-6 flex items-end justify-between gap-4">
            <div>
              <div className="mb-2 font-mono text-xs uppercase tracking-[0.18em] text-white/45">Why teams switch</div>
              <h2 className="text-3xl font-semibold tracking-tight">Keep your stack moving.</h2>
            </div>
            <Route className="hidden h-7 w-7 text-white/35 sm:block" />
          </div>
          <div className="grid gap-4 md:grid-cols-3">
            <div className="rounded-lg border border-white/15 bg-white/[0.04] p-5">
              <CreditCard className="mb-4 h-5 w-5 text-white/60" />
              <h3 className="text-lg font-semibold">One balance</h3>
              <p className="mt-2 text-sm leading-relaxed text-white/50">Stop splitting budget across OpenAI, Anthropic, Google, DeepSeek, media APIs, and search.</p>
            </div>
            <div className="rounded-lg border border-white/15 bg-white/[0.04] p-5">
              <Route className="mb-4 h-5 w-5 text-white/60" />
              <h3 className="text-lg font-semibold">One route</h3>
              <p className="mt-2 text-sm leading-relaxed text-white/50">Use a stable model alias. Auto routes handle task difficulty, fallbacks, and provider health.</p>
            </div>
            <div className="rounded-lg border border-white/15 bg-white/[0.04] p-5">
              <Sparkles className="mb-4 h-5 w-5 text-white/60" />
              <h3 className="text-lg font-semibold">More than chat</h3>
              <p className="mt-2 text-sm leading-relaxed text-white/50">Ship image, video, audio, embeddings, forecasts, research search, and 3D without another billing system.</p>
            </div>
          </div>
        </section>

        <section className="mb-16">
          <div className="mb-6 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <div className="mb-2 font-mono text-xs uppercase tracking-[0.18em] text-white/45">First-party providers</div>
              <h2 className="text-3xl font-semibold tracking-tight">The services we actually run.</h2>
              <p className="mt-3 max-w-2xl text-sm leading-relaxed text-white/50">First-party lanes give us tighter economics, direct integration, and more control over capacity. Read the useful bits behind each one.</p>
            </div>
            <Link to="/providers" className="font-mono text-sm text-white/45 transition-colors hover:text-white">All providers <ArrowRight className="ml-1 inline h-4 w-4" /></Link>
          </div>

          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {firstPartyGuides.map((guide, index) => {
              const post = posts.find(candidate => candidate.slug === guide.slug);
              if (!post) return null;
              return (
                <motion.div
                  key={guide.slug}
                  initial={{ opacity: 0, y: 8 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ duration: 0.25, delay: index * 0.04 }}
                >
                  <Link to={`/blog/${post.slug}`} className="group block h-full rounded-lg border border-white/15 bg-white/[0.04] p-5 transition-all hover:border-white/40 hover:bg-white/[0.07]">
                    <div className="mb-3 flex items-center justify-between gap-3">
                      <h3 className="text-lg font-semibold tracking-tight group-hover:text-white">{guide.name}</h3>
                      <ArrowRight className="h-4 w-4 text-white/35 transition-transform group-hover:translate-x-1 group-hover:text-white" />
                    </div>
                    <p className="text-sm leading-relaxed text-white/50">{guide.body}</p>
                    <div className="mt-5 font-mono text-xs text-white/40">Read provider guide</div>
                  </Link>
                </motion.div>
              );
            })}
          </div>
        </section>

        <section className="mb-16">
          <div className="mb-6 flex items-end justify-between gap-4">
            <div>
              <div className="mb-2 font-mono text-xs uppercase tracking-[0.18em] text-white/45">Alternative guides</div>
              <h2 className="text-3xl font-semibold tracking-tight">Pick the comparison you need.</h2>
            </div>
            <Link to="/blog" className="font-mono text-sm text-white/45 transition-colors hover:text-white">All posts <ArrowRight className="ml-1 inline h-4 w-4" /></Link>
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            {alternativePosts.map((post, index) => (
              <motion.div
                key={post.slug}
                initial={{ opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.25, delay: index * 0.04 }}
              >
                <Link to={post.alternativePath || `/blog/${post.slug}`} className="group block h-full rounded-lg border border-white/15 bg-white/[0.04] p-6 transition-all hover:border-white/40 hover:bg-white/[0.07]">
                  <div className="mb-4 flex flex-wrap items-center gap-2">
                    <span className="rounded border border-white/20 bg-white/[0.07] px-2 py-1 font-mono text-[10px] uppercase tracking-[0.14em] text-white/45">{competitorName(post.title)}</span>
                    <span className="font-mono text-xs text-white/40">{post.readTime}</span>
                  </div>
                  <h3 className="mb-3 text-xl font-semibold tracking-tight transition-colors group-hover:text-white">{post.title}</h3>
                  <p className="mb-6 text-sm leading-relaxed text-white/50">{post.excerpt}</p>
                  <div className="flex items-center gap-2 font-mono text-sm text-white/55 transition-colors group-hover:text-white">Read comparison <ArrowRight className="h-4 w-4" /></div>
                </Link>
              </motion.div>
            ))}
          </div>
        </section>

        <section aria-labelledby="alternatives-faq">
          <div className="mb-6">
            <div className="mb-2 font-mono text-xs uppercase tracking-[0.18em] text-white/45">FAQ</div>
            <h2 id="alternatives-faq" className="text-3xl font-semibold tracking-tight">Pricing and routing questions.</h2>
          </div>
          <div className="grid gap-4 md:grid-cols-3">
            {faqs.map(faq => (
              <div key={faq.question} className="rounded-lg border border-white/15 bg-white/[0.04] p-5">
                <h3 className="font-semibold leading-snug">{faq.question}</h3>
                <p className="mt-3 text-sm leading-relaxed text-white/50">{faq.answer}</p>
              </div>
            ))}
          </div>
        </section>
      </div>
    </>
  );
}
