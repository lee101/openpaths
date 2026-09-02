import React, { useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { ArrowRight, AudioLines, Box, Boxes, Clapperboard, Drama, GitMerge, ImageIcon, Layers, Music, Palette, PersonStanding, Scissors, Search, Sparkles, WandSparkles } from 'lucide-react';
import { Seo } from '../components/Seo';
import { BASE_URL, TOOLS, TOOLS_INDEX_SEO, TOOLS_INDEX_SLUG, toolArtImage, toolOgImage } from '../data/tools';

const ICONS: Record<string, React.ComponentType<{ className?: string }>> = {
  'google-tts': AudioLines,
  lyria: Music,
  'text-to-image': ImageIcon,
  'image-edit': WandSparkles,
  'text-to-video': Clapperboard,
  'image-to-video': Clapperboard,
  'video-extension': Clapperboard,
  'character-animator': Drama,
  'music-generator': Music,
  'remove-video-background': Scissors,
  'image-to-3d': Box,
  'text-to-3d': Boxes,
  'rig-3d': PersonStanding,
  'retexture-3d': Palette,
  playground: Layers,
  fusion: GitMerge,
};

const JSON_LD = {
  '@context': 'https://schema.org',
  '@type': 'ItemList',
  name: 'OpenPaths Tools',
  description: TOOLS_INDEX_SEO.seoDescription,
  numberOfItems: TOOLS.length,
  itemListElement: TOOLS.map((tool, index) => ({
    '@type': 'ListItem',
    position: index + 1,
    url: `${BASE_URL}${tool.path}`,
    name: tool.name,
    description: tool.description,
    image: toolOgImage(tool.slug),
  })),
};

export function Tools() {
  const [query, setQuery] = useState('');

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return TOOLS;
    const terms = q.split(/\s+/);
    return TOOLS.filter(tool => {
      const haystack = `${tool.name} ${tool.tagline} ${tool.description} ${tool.keywords} ${tool.path}`.toLowerCase();
      return terms.every(term => haystack.includes(term));
    });
  }, [query]);

  return (
    <>
      <Seo
        title={TOOLS_INDEX_SEO.seoTitle}
        description={TOOLS_INDEX_SEO.seoDescription}
        path={TOOLS_INDEX_SEO.path}
        image={toolOgImage(TOOLS_INDEX_SLUG)}
        jsonLd={JSON_LD}
      />

      <div className="px-4 py-12 sm:px-6 lg:px-10">
        <div className="mx-auto w-full max-w-[2400px]">
          <div className="mb-8 flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
            <div>
              <div className="mb-4 inline-flex items-center gap-2 rounded border border-white/20 bg-white/[0.06] px-3 py-1 text-xs font-mono text-white/45">
                <Sparkles className="h-3.5 w-3.5" /> Tools
              </div>
              <h1 className="text-4xl font-bold tracking-tight md:text-5xl">OpenPaths Tools</h1>
              <p className="mt-4 max-w-2xl text-base leading-relaxed text-white/55">
                Every tool is a thin wrapper over the API. Copy the snippet to ship it.
              </p>
            </div>
            <div className="w-full lg:max-w-md">
              <label className="relative block">
                <Search className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-white/45" />
                <input
                  type="search"
                  value={query}
                  onChange={event => setQuery(event.target.value)}
                  placeholder="Search tools: video, 3D, music, voice..."
                  aria-label="Search tools"
                  data-testid="tools-search"
                  className="h-12 w-full rounded border border-white/20 bg-white/[0.06] pl-11 pr-4 font-mono text-sm text-white outline-none transition-colors placeholder:text-white/35 focus:border-white/60"
                />
              </label>
              <div className="mt-2 font-mono text-[11px] text-white/45" data-testid="tools-count">
                {filtered.length} of {TOOLS.length} tools
              </div>
            </div>
          </div>

          {filtered.length === 0 ? (
            <div className="rounded-lg border border-white/20 bg-white/[0.05] p-12 text-center">
              <p className="text-sm text-white/55">No tools match “{query}”.</p>
              <button type="button" onClick={() => setQuery('')} className="mt-3 font-mono text-xs text-white underline underline-offset-4">Clear search</button>
            </div>
          ) : (
            <div data-testid="tools-grid" className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5">
              {filtered.map(tool => {
                const Icon = ICONS[tool.slug] ?? Sparkles;
                return (
                  <Link
                    key={tool.path}
                    to={tool.path}
                    className="group flex flex-col overflow-hidden rounded-lg border border-white/20 bg-white/[0.05] transition-colors hover:border-white/50 hover:bg-white/[0.07]"
                  >
                    <div className="relative aspect-[16/9] overflow-hidden bg-black">
                      <img
                        src={toolArtImage(tool.slug)}
                        alt={`${tool.name} - ${tool.tagline}`}
                        loading="lazy"
                        width={1152}
                        height={768}
                        className="h-full w-full object-cover opacity-80 transition-all duration-500 group-hover:scale-[1.03] group-hover:opacity-100"
                      />
                      <div className="absolute inset-0 bg-gradient-to-t from-black via-black/25 to-transparent" />
                    </div>
                    <div className="flex flex-1 flex-col gap-3 p-5">
                      <div className="flex items-center justify-between gap-3">
                        <div className="flex min-w-0 items-center gap-3">
                          <span className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded border border-white/20 bg-white/[0.07] text-white/70">
                            <Icon className="h-5 w-5" />
                          </span>
                          <div className="min-w-0">
                            <div className="truncate text-lg font-bold tracking-tight">{tool.name}</div>
                            <div className="truncate text-[11px] font-mono uppercase tracking-[0.16em] text-white/50">{tool.tagline}</div>
                          </div>
                        </div>
                        <ArrowRight className="h-4 w-4 shrink-0 text-white/45 transition-colors group-hover:text-white/70" />
                      </div>
                      <p className="text-sm leading-relaxed text-white/55">{tool.description}</p>
                      <div className="mt-auto pt-1 text-xs font-mono text-white/55">{tool.price}</div>
                    </div>
                  </Link>
                );
              })}
            </div>
          )}

          <p className="mt-8 text-sm font-mono text-white/55">
            Looking for raw endpoints? See the <Link to="/docs" className="text-white underline underline-offset-4">API docs</Link>.
          </p>
        </div>
      </div>
    </>
  );
}
