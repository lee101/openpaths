import React, { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { ArrowLeft, ArrowRight, Bolt, Check, Copy, Loader2, Sparkles } from 'lucide-react';
import { Seo } from '../components/Seo';
import type { LibraryPrompt } from '../data/promptLibrary';
import { fetchPrompt, promptPlaygroundHref } from '../lib/promptLibrary';
import { PromptCard } from './Prompts';

export function PromptDetail() {
  const { slug = '' } = useParams();
  const [prompt, setPrompt] = useState<LibraryPrompt | null>(null);
  const [related, setRelated] = useState<LibraryPrompt[]>([]);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setNotFound(false);
    fetchPrompt(slug).then(data => {
      if (cancelled) return;
      if (!data) {
        setNotFound(true);
      } else {
        setPrompt(data.prompt);
        setRelated(data.related);
      }
      setLoading(false);
    });
    return () => {
      cancelled = true;
    };
  }, [slug]);

  const copy = async () => {
    if (!prompt) return;
    try {
      await navigator.clipboard.writeText(prompt.prompt);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1800);
    } catch {
      /* clipboard unavailable */
    }
  };

  if (loading) {
    return (
      <div className="flex min-h-screen items-center gap-2 bg-black px-6 text-white/40">
        <Loader2 className="h-5 w-5 animate-spin" /> Loading prompt…
      </div>
    );
  }

  if (notFound || !prompt) {
    return (
      <div className="min-h-screen bg-black px-6 py-24 text-center">
        <Seo title="Prompt not found | OpenPaths" description="This prompt could not be found." path={`/prompts/${slug}`} />
        <p className="font-mono text-sm text-white/50">Prompt not found.</p>
        <Link to="/prompts" className="mt-4 inline-flex items-center gap-1 font-mono text-sm text-white hover:text-white/80">
          <ArrowLeft className="h-4 w-4" /> Back to the prompt library
        </Link>
      </div>
    );
  }

  return (
    <>
      <Seo
        title={`${prompt.title} — ${prompt.modelName} prompt | OpenPaths`}
        description={prompt.summary}
        path={prompt.url}
      />
      <div className="min-h-screen bg-black">
        <div className="mx-auto max-w-5xl px-6 py-10">
          <div className="mb-6 flex flex-wrap items-center gap-2 font-mono text-xs text-white/45">
            <Link to="/prompts" className="hover:text-white">Prompts</Link>
            <span>/</span>
            <Link to={prompt.categoryUrl} className="hover:text-white">{prompt.categoryName.replace(' prompts', '')}</Link>
            <span>/</span>
            <span className="text-white/70">{prompt.title}</span>
          </div>

          <div className="mb-3 flex flex-wrap items-center gap-2 font-mono text-[11px] uppercase tracking-wide text-white/45">
            <Link to={prompt.modalityUrl} className="rounded border border-white/10 px-2 py-1 hover:border-white/30 hover:text-white">{prompt.modalityName.replace(' prompts', '')}</Link>
            {prompt.isFree && <span className="inline-flex items-center gap-1 rounded border border-emerald-400/30 px-2 py-1 text-emerald-300/80"><Bolt className="h-3 w-3" /> free</span>}
            {prompt.featured && <span className="inline-flex items-center gap-1 rounded border border-white/10 px-2 py-1"><Sparkles className="h-3 w-3" /> featured</span>}
          </div>

          <h1 className="max-w-3xl text-3xl font-bold tracking-tight md:text-5xl">{prompt.title}</h1>
          <p className="mt-3 max-w-3xl text-base leading-relaxed text-white/60">{prompt.summary}</p>

          <div className="mt-4 flex flex-wrap items-center gap-2 text-sm">
            <span className="font-mono text-xs text-white/40">Recommended model:</span>
            <Link to={prompt.modelUrl} className="rounded border border-white/10 px-2 py-1 font-mono text-xs text-white/70 hover:border-white/30 hover:text-white">{prompt.modelName}</Link>
          </div>

          {/* Prompt body */}
          <div className="mt-6 rounded border border-white/10 bg-white/[0.02]">
            <div className="flex items-center justify-between border-b border-white/10 px-4 py-2.5">
              <span className="font-mono text-xs uppercase tracking-wide text-white/40">Prompt</span>
              <button onClick={copy} className="inline-flex items-center gap-1.5 rounded border border-white/10 px-2.5 py-1 font-mono text-xs text-white/60 hover:border-white/30 hover:text-white">
                {copied ? <><Check className="h-3.5 w-3.5" /> Copied</> : <><Copy className="h-3.5 w-3.5" /> Copy</>}
              </button>
            </div>
            <pre className="overflow-x-auto whitespace-pre-wrap px-4 py-4 font-mono text-sm leading-relaxed text-white/80">{prompt.prompt}</pre>
          </div>

          <div className="mt-5 flex flex-wrap gap-3">
            <Link to={promptPlaygroundHref(prompt)} className="inline-flex h-12 items-center justify-center gap-2 rounded border border-white bg-white px-5 font-mono text-sm font-bold text-black hover:bg-white/90">
              Open in Playground <ArrowRight className="h-4 w-4" />
            </Link>
            <button onClick={copy} className="inline-flex h-12 items-center justify-center gap-2 rounded border border-white/15 px-5 font-mono text-sm text-white/75 hover:border-white/35 hover:text-white">
              {copied ? <><Check className="h-4 w-4" /> Copied</> : <><Copy className="h-4 w-4" /> Copy prompt</>}
            </button>
          </div>

          {prompt.tags.length > 0 && (
            <div className="mt-6 flex flex-wrap gap-1.5">
              {prompt.tags.map(tag => (
                <Link key={tag} to={`/prompts?q=${encodeURIComponent(tag)}`} className="rounded bg-white/[0.04] px-2 py-1 font-mono text-xs text-white/45 hover:text-white/70">#{tag}</Link>
              ))}
            </div>
          )}

          {/* Related (gobed-powered) */}
          {related.length > 0 && (
            <section className="mt-12">
              <h2 className="mb-4 text-xl font-semibold text-white">Related prompts</h2>
              <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                {related.map(r => (
                  <PromptCard key={r.slug} prompt={r} />
                ))}
              </div>
            </section>
          )}
        </div>
      </div>
    </>
  );
}
