import React, { useEffect, useState } from 'react';
import { ArrowLeft, ArrowRight, Check, Copy, Loader2 } from 'lucide-react';
import { Link, useParams } from 'react-router-dom';
import { Seo } from '../components/Seo';
import { type ZImageArtItem } from '../data/zimageArt';
import { artPromptPlaygroundHref, fetchArtItem } from '../lib/zimageArt';

export function ArtDetail() {
  const { slug = '' } = useParams();
  const [item, setItem] = useState<ZImageArtItem | null>(null);
  const [related, setRelated] = useState<ZImageArtItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setNotFound(false);
    fetchArtItem(slug).then(data => {
      if (cancelled) return;
      if (!data) {
        setNotFound(true);
      } else {
        setItem(data.item);
        setRelated(data.related);
      }
      setLoading(false);
    });
    return () => {
      cancelled = true;
    };
  }, [slug]);

  const copyPrompt = async () => {
    if (!item) return;
    try {
      await navigator.clipboard.writeText(item.prompt);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1600);
    } catch {
      /* clipboard unavailable */
    }
  };

  if (loading) {
    return (
      <div className="flex min-h-[60vh] items-center justify-center bg-black text-white/45">
        <Loader2 className="mr-2 h-5 w-5 animate-spin" /> Loading
      </div>
    );
  }

  if (notFound || !item) {
    return (
      <>
        <Seo title="Art not found | OpenPaths" description="This AI art prompt could not be found." path={`/art/i/${slug}`} />
        <div className="mx-auto flex min-h-[60vh] max-w-3xl flex-col items-center justify-center gap-4 px-6 text-center">
          <h1 className="text-2xl font-semibold">Art not found</h1>
          <Link to="/art" className="inline-flex items-center gap-2 font-mono text-sm text-white/70 hover:text-white">
            <ArrowLeft className="h-4 w-4" /> Back to art search
          </Link>
        </div>
      </>
    );
  }

  const title = item.prompt.length > 70 ? `${item.prompt.slice(0, 70).trim()}…` : item.prompt;
  const dims = item.width && item.height ? `${item.width}×${item.height}` : item.aspect || '';
  const jsonLd = {
    '@context': 'https://schema.org',
    '@type': 'ImageObject',
    contentUrl: item.imageUrl,
    thumbnailUrl: item.thumbUrl || item.imageUrl,
    caption: item.prompt,
    description: item.prompt,
    width: item.width,
    height: item.height,
    creditText: 'OpenPaths',
    isAccessibleForFree: true,
  };

  return (
    <>
      <Seo
        title={`${title} | AI Art Prompt | OpenPaths`}
        description={`AI-generated art for the prompt: ${item.prompt.slice(0, 150)}. Try it in the OpenPaths playground.`}
        path={`/art/i/${encodeURIComponent(item.slug)}`}
        image={item.imageUrl}
      />
      <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }} />
      <div className="min-h-screen bg-black">
        <div className="mx-auto max-w-6xl px-6 py-8">
          <nav className="mb-6 flex items-center gap-2 font-mono text-xs uppercase tracking-[0.16em] text-white/55">
            <Link to="/art" className="hover:text-white">Art</Link>
            <span>/</span>
            {item.tags && item.tags[0] ? (
              <>
                <Link to={`/art/tag/${encodeURIComponent(item.tags[0])}`} className="hover:text-white">{item.tags[0]}</Link>
                <span>/</span>
              </>
            ) : null}
            <span className="truncate text-white/45">{item.slug}</span>
          </nav>

          <div className="grid gap-8 lg:grid-cols-[minmax(0,1fr)_380px]">
            <div className="overflow-hidden rounded-lg border border-white/20 bg-white/[0.05]">
              {item.mediaType === 'video' && item.videoUrl ? (
                <video src={item.videoUrl} poster={item.posterUrl || item.thumbUrl || undefined} controls autoPlay muted loop playsInline className="w-full bg-black object-contain" data-testid="art-detail-video" />
              ) : (
                <img src={item.imageUrl} alt={item.prompt} className="w-full bg-white/[0.06] object-contain" />
              )}
            </div>
            <div className="flex flex-col gap-6">
              <div>
                <div className="mb-3 flex flex-wrap items-center gap-2 font-mono text-[11px] uppercase tracking-[0.16em] text-white/55">
                  <span className="rounded border border-white/20 px-2 py-1">{item.model || 'zimage'}</span>
                  {dims ? <span className="rounded border border-white/20 px-2 py-1">{dims}</span> : null}
                  {item.aspect ? <span className="rounded border border-white/20 px-2 py-1">{item.aspect}</span> : null}
                </div>
                <h1 className="text-2xl font-semibold leading-snug tracking-tight">{item.title || 'AI generated art'}</h1>
              </div>

              <div className="rounded border border-white/20 bg-black/40 p-4">
                <div className="mb-2 flex items-center justify-between font-mono text-[11px] uppercase tracking-[0.16em] text-white/55">
                  <span>Prompt</span>
                  <button type="button" onClick={copyPrompt} className="inline-flex items-center gap-1 text-white/55 hover:text-white">
                    {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
                    {copied ? 'Copied' : 'Copy'}
                  </button>
                </div>
                <p className="text-sm leading-relaxed text-white/72">{item.prompt}</p>
              </div>

              <Link to={artPromptPlaygroundHref(item)} className="inline-flex items-center justify-center gap-2 rounded bg-white px-4 py-3 font-mono text-sm font-bold text-black hover:bg-white/90">
                Try this prompt in the playground <ArrowRight className="h-4 w-4" />
              </Link>

              {item.tags && item.tags.length > 0 && (
                <div className="flex flex-wrap gap-2">
                  {item.tags.map(tag => (
                    <Link key={tag} to={`/art/tag/${encodeURIComponent(tag)}`} className="rounded border border-white/20 px-2.5 py-1 font-mono text-xs text-white/55 hover:border-white/50 hover:text-white">
                      {tag}
                    </Link>
                  ))}
                </div>
              )}
            </div>
          </div>

          {related.length > 0 && (
            <section className="mt-12">
              <h2 className="mb-4 text-lg font-semibold tracking-tight">Related art</h2>
              <div className="grid auto-rows-fr gap-4 sm:grid-cols-2 lg:grid-cols-4">
                {related.map(r => (
                  <Link key={r.id || r.slug} to={`/art/i/${encodeURIComponent(r.slug)}`} className="group overflow-hidden rounded border border-white/20 bg-white/[0.05]">
                    <img src={r.thumbUrl || r.imageUrl} alt={r.prompt} loading="lazy" className="aspect-square w-full bg-white/[0.06] object-cover transition-transform duration-300 group-hover:scale-[1.03]" />
                    <p className="line-clamp-2 p-3 text-xs leading-relaxed text-white/60">{r.prompt}</p>
                  </Link>
                ))}
              </div>
            </section>
          )}
        </div>
      </div>
    </>
  );
}
