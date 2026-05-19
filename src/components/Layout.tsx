import React, { useState, useEffect } from 'react';
import { Link, Outlet, useLocation } from 'react-router-dom';
import { Menu, User, X } from 'lucide-react';

function useIsLoggedIn() {
  const [loggedIn, setLoggedIn] = useState(() => {
    if (typeof window !== 'undefined') {
      if (window.userData?.authenticated) return true;
      if (localStorage.getItem('op_api_key')) return true;
    }
    return false;
  });

  useEffect(() => {
    const check = () => {
      const authed = !!(window.userData?.authenticated || localStorage.getItem('op_api_key'));
      setLoggedIn(authed);
    };
    // Re-check on storage changes (cross-tab), custom auth-change event (same-tab), and focus
    window.addEventListener('storage', check);
    window.addEventListener('focus', check);
    window.addEventListener('auth-change', check);
    return () => {
      window.removeEventListener('storage', check);
      window.removeEventListener('focus', check);
      window.removeEventListener('auth-change', check);
    };
  }, []);

  return loggedIn;
}

const networkLinks = [
  { label: 'App.nz', href: 'https://app.nz' },
  { label: 'Netwrck', href: 'https://netwrck.com' },
  { label: 'Text-Generator.io', href: 'https://text-generator.io' },
  { label: 'Codex Infinity', href: 'https://codex-infinity.com' },
  { label: 'OpenPaths', href: 'https://openpaths.io' },
  { label: 'AI Art Generator', href: 'https://ebank.nz' },
  { label: 'AIArt-Generator.art', href: 'https://aiart-generator.art' },
  { label: 'SiteSim', href: 'https://sitesim.net' },
  { label: 'SimplexGen', href: 'https://simplexgen.com' },
  { label: 'DictatorFlow', href: 'https://dictatorflow.com' },
  { label: 'WebFiddle', href: 'https://webfiddle.net' },
  { label: 'Ring.nz', href: 'https://ring.nz' },
  { label: 'ChatGibidy', href: 'https://chatgibidy.com' },
  { label: 'BitBank', href: 'https://bitbank.nz' },
  { label: 'ExperimentFlow', href: 'https://experimentflow.com' },
  { label: 'Evangeler', href: 'https://evangeler.com' },
  { label: 'Hires.nz', href: 'https://hires.nz' },
  { label: 'How.nz', href: 'https://how.nz' },
  { label: 'V5 Games', href: 'https://v5games.com' },
  { label: 'Addicting Word Games', href: 'https://addictingwordgames.com' },
  { label: 'Big Multiplayer Chess', href: 'https://bigmultiplayerchess.com' },
  { label: 'Word Smashing', href: 'https://wordsmashing.com' },
  { label: 'reWord Game', href: 'https://rewordgame.com' },
  { label: 'Multiplication Master', href: 'https://multiplicationmaster.com' },
];

const primaryNavLinks = [
  { label: 'Models', to: '/models', match: (path: string) => path === '/models' },
  { label: 'Pricing', to: '/pricing', match: (path: string) => path === '/pricing' },
  { label: 'Providers', to: '/providers', match: (path: string) => path.startsWith('/providers') },
  { label: 'Stats', to: '/stats', match: (path: string) => path === '/stats' },
  { label: 'Docs', to: '/docs', match: (path: string) => path === '/docs' },
  { label: 'Integrations', to: '/integrations', match: (path: string) => path === '/integrations' },
  { label: 'Playground', to: '/playground', match: (path: string) => path === '/playground' },
  { label: 'Image to 3D', to: '/image-to-3d', match: (path: string) => path === '/image-to-3d' },
  { label: 'Search', to: '/search', match: (path: string) => path === '/search' },
  { label: 'Blog', to: '/blog', match: (path: string) => path.startsWith('/blog') },
  { label: 'Alternatives', to: '/alternatives', match: (path: string) => path.startsWith('/alternatives') },
];

function accountInitials() {
  if (typeof window === 'undefined') return '';
  const user = window.userData;
  const source = user?.name || user?.email || '';
  const parts = source
    .replace(/@.*/, '')
    .split(/[\s._-]+/)
    .filter(Boolean);
  return parts.slice(0, 2).map(part => part[0]?.toUpperCase()).join('');
}

export function Layout() {
  const location = useLocation();
  const isPlayground = location.pathname === '/playground';
  const isLoggedIn = useIsLoggedIn();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const initials = accountInitials();

  useEffect(() => {
    setMobileMenuOpen(false);
  }, [location.pathname]);

  return (
    <div className={`bg-black text-white font-sans selection:bg-white selection:text-black flex flex-col ${isPlayground ? 'h-screen overflow-hidden' : 'min-h-screen'}`}>
      <nav className="relative border-b border-white/10 px-4 py-3 sm:px-6 sm:py-4 flex items-center justify-between bg-black/90 backdrop-blur-md z-50 shrink-0">
        <Link to="/" className="flex items-center gap-2">
          <img src="/logo.webp" alt="OpenPaths" className="w-6 h-6" />
          <span className="font-mono font-bold text-xl tracking-tighter">OpenPaths</span>
        </Link>
        <div className="hidden xl:flex items-center gap-6 text-sm font-mono text-white/60">
          {primaryNavLinks.map(link => (
            <Link key={link.to} to={link.to} className={`transition-colors ${link.match(location.pathname) ? 'text-white' : 'hover:text-white'}`}>
              {link.label}
            </Link>
          ))}
          <a href="/#api" className="hover:text-white transition-colors">API</a>
        </div>
        <div className="flex items-center gap-2 sm:gap-4">
          {isLoggedIn ? (
            <>
              <Link to="/account" className="text-sm font-mono text-white/60 hover:text-white transition-colors hidden sm:block" data-testid="nav-account">Account</Link>
              <Link to="/account" className="hidden sm:block bg-white text-black px-4 py-2 text-sm font-mono font-bold hover:bg-white/90 transition-colors rounded" data-testid="nav-dashboard">
                Dashboard
              </Link>
              <Link
                to="/account"
                className="flex h-9 w-9 items-center justify-center rounded-full border border-white/15 bg-white text-sm font-mono font-bold text-black transition-colors hover:bg-white/90 sm:hidden"
                aria-label="Account dashboard"
                data-testid="nav-dashboard-mobile"
              >
                {initials || <User className="h-4 w-4" />}
              </Link>
            </>
          ) : (
            <>
              <Link to="/account" className="text-sm font-mono text-white/60 hover:text-white transition-colors hidden sm:block" data-testid="nav-signin">Sign In</Link>
              <Link to="/account" className="hidden sm:block bg-white text-black px-3 py-2 sm:px-4 text-sm font-mono font-bold hover:bg-white/90 transition-colors rounded" data-testid="nav-get-started">
                Get Started
              </Link>
            </>
          )}
          <button
            type="button"
            className="inline-flex h-9 w-9 items-center justify-center rounded border border-white/10 bg-white/[0.03] text-white/70 transition-colors hover:border-white/25 hover:text-white xl:hidden"
            aria-label={mobileMenuOpen ? 'Close menu' : 'Open menu'}
            aria-expanded={mobileMenuOpen}
            onClick={() => setMobileMenuOpen(open => !open)}
          >
            {mobileMenuOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
          </button>
        </div>
        {mobileMenuOpen && (
          <div className="absolute left-0 right-0 top-full border-b border-white/10 bg-black/95 px-4 py-4 shadow-2xl backdrop-blur-md xl:hidden">
            <div className="grid grid-cols-2 gap-2 font-mono text-sm text-white/60">
              {primaryNavLinks.map(link => (
                <Link
                  key={link.to}
                  to={link.to}
                  className={`rounded border px-3 py-2 transition-colors ${link.match(location.pathname) ? 'border-white/25 bg-white/[0.08] text-white' : 'border-white/8 bg-white/[0.02] hover:border-white/20 hover:text-white'}`}
                >
                  {link.label}
                </Link>
              ))}
              <a href="/#api" className="rounded border border-white/8 bg-white/[0.02] px-3 py-2 transition-colors hover:border-white/20 hover:text-white">
                API
              </a>
              {!isLoggedIn && (
                <Link to="/account" className="rounded border border-white/25 bg-white px-3 py-2 font-bold text-black transition-colors hover:bg-white/90">
                  Get Started
                </Link>
              )}
            </div>
          </div>
        )}
      </nav>

      <main className="flex-1 min-h-0">
        <Outlet />
      </main>

      {!isPlayground && (
        <footer className="border-t border-white/10 px-6 py-12 mt-auto">
          <div className="max-w-7xl mx-auto">
            <div className="grid grid-cols-2 lg:grid-cols-6 gap-8 mb-10">
              <div>
                <div className="flex items-center gap-2 mb-4">
                  <img src="/logo.webp" alt="OpenPaths" className="w-5 h-5" />
                  <span className="font-mono font-bold tracking-tighter">OpenPaths</span>
                </div>
                <p className="text-xs font-mono text-white/30 leading-relaxed">Open source model router. Millisecond routing across 400+ AI models.</p>
              </div>
              <div>
                <h4 className="text-xs font-mono font-bold text-white/60 uppercase tracking-widest mb-3">Product</h4>
                <ul className="space-y-2 text-sm font-mono text-white/40">
                  <li><Link to="/models" className="hover:text-white transition-colors">Models</Link></li>
                  <li><Link to="/providers" className="hover:text-white transition-colors">Providers</Link></li>
                  <li><Link to="/stats" className="hover:text-white transition-colors">Stats</Link></li>
                  <li><Link to="/pricing" className="hover:text-white transition-colors">Pricing</Link></li>
                  <li><Link to="/integrations" className="hover:text-white transition-colors">Integrations</Link></li>
                  <li><Link to="/playground" className="hover:text-white transition-colors">Playground</Link></li>
                  <li><Link to="/image-to-3d" className="hover:text-white transition-colors">Image to 3D</Link></li>
                  <li><Link to="/search" className="hover:text-white transition-colors">Search</Link></li>
                  <li><Link to="/blog" className="hover:text-white transition-colors">Blog</Link></li>
                  <li><Link to="/alternatives" className="hover:text-white transition-colors">Alternatives</Link></li>
                  <li><a href="/#api" className="hover:text-white transition-colors">API Docs</a></li>
                </ul>
              </div>
              <div className="col-span-2 lg:col-span-3">
                <h4 className="text-xs font-mono font-bold text-white/60 uppercase tracking-widest mb-3">Network</h4>
                <ul className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-x-6 gap-y-2 text-sm font-mono text-white/40">
                  {networkLinks.map((link) => (
                    <li key={link.href}>
                      <a href={link.href} target="_blank" rel="noopener noreferrer" className="hover:text-white transition-colors">
                        {link.label}
                      </a>
                    </li>
                  ))}
                </ul>
              </div>
              <div>
                <h4 className="text-xs font-mono font-bold text-white/60 uppercase tracking-widest mb-3">Social</h4>
                <ul className="space-y-2 text-sm font-mono text-white/40">
                  <li><a href="https://twitter.com/Netwrck" target="_blank" rel="noreferrer" className="hover:text-white transition-colors">Twitter</a></li>
                  <li><a href="https://github.com/Netwrck" target="_blank" rel="noreferrer" className="hover:text-white transition-colors">GitHub</a></li>
                  <li><a href="https://codex-infinity.com/@lee101/openpaths" target="_blank" rel="noreferrer" className="hover:text-white transition-colors">Codex</a></li>
                </ul>
              </div>
            </div>
            <div className="border-t border-white/10 pt-6 flex flex-col sm:flex-row justify-between items-center gap-3 text-xs font-mono text-white/20">
              <span>© {new Date().getFullYear()} OpenPaths. Open source model routing.</span>
              <div className="flex gap-4">
                <Link to="/docs" className="hover:text-white transition-colors">Docs</Link>
                <Link to="/stats" className="hover:text-white transition-colors">Stats</Link>
                <Link to="/alternatives" className="hover:text-white transition-colors">Alternatives</Link>
                <Link to="/integrations" className="hover:text-white transition-colors">Integrations</Link>
                <Link to="/account" className="hover:text-white transition-colors">Account</Link>
              </div>
            </div>
          </div>
        </footer>
      )}
    </div>
  );
}
