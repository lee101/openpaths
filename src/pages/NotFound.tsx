import { ArrowRight } from 'lucide-react';
import { Link, useLocation } from 'react-router-dom';
import { Seo } from '../components/Seo';

export function NotFound() {
  const location = useLocation();

  return (
    <>
      <Seo
        title="Page not found | OpenPaths"
        description="That page does not exist. Browse AI art, try the text-to-image tool, or explore model providers on OpenPaths."
        path={location.pathname}
      />
      <div className="min-h-screen bg-black">
        <div className="mx-auto flex max-w-3xl flex-col items-start px-6 py-20">
          <span className="font-mono text-xs uppercase tracking-[0.18em] text-white/40">Error 404</span>
          <h1 className="mt-4 text-4xl font-bold tracking-tight md:text-6xl">This path leads nowhere</h1>
          <p className="mt-4 max-w-xl text-base leading-relaxed text-white/60">
            We couldn&rsquo;t find{' '}
            <code className="rounded bg-white/[0.06] px-1.5 py-0.5 font-mono text-sm text-white/75">
              {location.pathname}
            </code>
            . It may have moved or never existed. Try browsing AI art or generating your own with the
            text-to-image tool.
          </p>

          <div className="mt-8 flex flex-wrap gap-3">
            <Link
              to="/art"
              className="inline-flex items-center gap-2 rounded bg-white px-4 py-2 font-mono text-sm font-bold text-black hover:bg-white/90"
            >
              Browse AI art <ArrowRight className="h-4 w-4" />
            </Link>
            <Link
              to="/text-to-image"
              className="inline-flex items-center gap-2 rounded border border-white/15 px-4 py-2 font-mono text-sm text-white/75 hover:border-white/35 hover:text-white"
            >
              Text-to-image tool <ArrowRight className="h-4 w-4" />
            </Link>
            <Link
              to="/"
              className="inline-flex items-center gap-2 rounded border border-white/15 px-4 py-2 font-mono text-sm text-white/75 hover:border-white/35 hover:text-white"
            >
              Home
            </Link>
          </div>

          <nav className="mt-12 w-full border-t border-white/10 pt-8">
            <h2 className="mb-4 font-mono text-xs uppercase tracking-[0.16em] text-white/40">Popular destinations</h2>
            <div className="flex flex-wrap gap-2">
              {[
                { to: '/art/tag/portrait', label: 'Portrait art' },
                { to: '/art/tag/anime', label: 'Anime art' },
                { to: '/models', label: 'Models' },
                { to: '/providers', label: 'Providers' },
                { to: '/playground', label: 'Playground' },
                { to: '/pricing', label: 'Pricing' },
              ].map(link => (
                <Link
                  key={link.to}
                  to={link.to}
                  className="rounded border border-white/10 px-3 py-1.5 font-mono text-xs text-white/55 hover:border-white/30 hover:text-white"
                >
                  {link.label}
                </Link>
              ))}
            </div>
          </nav>
        </div>
      </div>
    </>
  );
}
