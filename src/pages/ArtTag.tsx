import React, { useEffect, useState } from 'react';
import { ArrowRight, Loader2 } from 'lucide-react';
import { Link, useParams } from 'react-router-dom';
import { Seo } from '../components/Seo';
import { type ArtTagFacet, type ZImageArtItem } from '../data/zimageArt';
import { artPromptPlaygroundHref, fetchArtList, fetchArtTags } from '../lib/zimageArt';

const PAGE_SIZE = 48;

function titleCase(value: string): string {
  return value.replace(/-/g, ' ').replace(/\b\w/g, c => c.toUpperCase());
}

export function ArtTag() {
  const { slug = '' } = useParams();
  const tag = decodeURIComponent(slug);
  const [items, setItems] = useState<ZImageArtItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [relatedTags, setRelatedTags] = useState<ArtTagFacet[]>([]);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    fetchArtList({ tag, limit: PAGE_SIZE, offset: 0 }).then(list => {
      if (cancelled) return;
      setItems(list?.results || []);
      setTotal(list?.total || 0);
      setLoading(false);
    });
    fetchArtTags(48).then(t => {
      if (!cancelled) setRelatedTags(t.filter(x => x.tag !== tag).slice(0, 24));
    });
    return () => {
      cancelled = true;
    };
  }, [tag]);

  const loadMore = async () => {
    if (loadingMore) return;
    setLoadingMore(true);
    const list = await fetchArtList({ tag, limit: PAGE_SIZE, offset: items.length });
    if (list) {
      setItems(prev => [...prev, ...list.results]);
      setTotal(list.total);
    }
    setLoadingMore(false);
  };

  const display = titleCase(tag);
  const hasMore = items.length < total;

  return (
    <>
      <Seo
        title={`${display} AI Art Prompts | OpenPaths`}
        description={`Browse AI-generated ${tag} art and prompts. Search by visual intent and try any prompt against OpenPaths image models.`}
        path={`/art/tag/${encodeURIComponent(tag)}`}
      />
      <div className="min-h-screen bg-black">
        <div className="mx-auto max-w-7xl px-6 py-10">
          <nav className="mb-6 flex items-center gap-2 font-mono text-xs uppercase tracking-[0.16em] text-white/55">
            <Link to="/art" className="hover:text-white">Art</Link>
            <span>/</span>
            <span className="text-white/45">tag</span>
            <span>/</span>
            <span className="text-white/60">{tag}</span>
          </nav>

          <header className="mb-8 max-w-3xl">
            <h1 className="text-3xl font-bold tracking-tight md:text-5xl">{display} AI art prompts</h1>
            <p className="mt-3 text-base leading-relaxed text-white/60">
              {total ? `${total.toLocaleString('en-US')} ` : ''}AI-generated {tag} images. Click any image for the full prompt, or send it straight to the playground.
            </p>
            <div className="mt-5 flex flex-wrap gap-3">
              <Link to="/art" className="inline-flex items-center gap-2 rounded border border-white/15 px-4 py-2 font-mono text-sm text-white/75 hover:border-white/55 hover:text-white">
                All art <ArrowRight className="h-4 w-4" />
              </Link>
              <Link to="/playground?model=zimage" className="inline-flex items-center gap-2 rounded bg-white px-4 py-2 font-mono text-sm font-bold text-black hover:bg-white/90">
                Open playground
              </Link>
            </div>
          </header>

          {loading ? (
            <div className="flex min-h-80 items-center justify-center rounded border border-white/20 bg-white/[0.05] text-white/45">
              <Loader2 className="mr-2 h-5 w-5 animate-spin" /> Loading {tag} art
            </div>
          ) : items.length === 0 ? (
            <div className="flex min-h-60 items-center justify-center rounded border border-white/20 bg-white/[0.05] text-white/45">
              No images tagged “{tag}” yet.
            </div>
          ) : (
            <>
              <div className="grid auto-rows-fr gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
                {items.map(item => (
                  <article key={item.id || item.slug} className="group flex flex-col overflow-hidden rounded border border-white/20 bg-white/[0.05]">
                    <Link to={`/art/i/${encodeURIComponent(item.slug)}`} className="block overflow-hidden">
                      <img src={item.thumbUrl || item.imageUrl} alt={item.prompt} loading="lazy" className="aspect-square w-full bg-white/[0.06] object-cover transition-transform duration-300 group-hover:scale-[1.03]" />
                    </Link>
                    <div className="flex flex-1 flex-col p-4">
                      <Link to={`/art/i/${encodeURIComponent(item.slug)}`} className="line-clamp-3 min-h-[3.75rem] text-sm leading-relaxed text-white/64 hover:text-white">
                        {item.prompt}
                      </Link>
                      <Link to={artPromptPlaygroundHref(item)} className="mt-3 inline-flex items-center gap-2 font-mono text-xs font-bold uppercase tracking-[0.14em] text-white/70 hover:text-white">
                        Try prompt <ArrowRight className="h-3.5 w-3.5" />
                      </Link>
                    </div>
                  </article>
                ))}
              </div>
              {hasMore && (
                <div className="mt-8 flex justify-center">
                  <button type="button" onClick={loadMore} disabled={loadingMore} className="inline-flex items-center gap-2 rounded border border-white/15 px-6 py-3 font-mono text-sm text-white/75 hover:border-white/55 hover:text-white disabled:opacity-50">
                    {loadingMore ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
                    Load more
                  </button>
                </div>
              )}
            </>
          )}

          {relatedTags.length > 0 && (
            <section className="mt-12 border-t border-white/20 pt-8">
              <h2 className="mb-4 font-mono text-xs uppercase tracking-[0.16em] text-white/55">More tags</h2>
              <div className="flex flex-wrap gap-2">
                {relatedTags.map(t => (
                  <Link key={t.tag} to={`/art/tag/${encodeURIComponent(t.tag)}`} className="rounded border border-white/20 px-3 py-1.5 font-mono text-xs text-white/55 hover:border-white/50 hover:text-white">
                    {t.tag} <span className="opacity-50">{t.count.toLocaleString('en-US')}</span>
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
