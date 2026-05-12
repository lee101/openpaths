import React, { useState, useEffect } from 'react';
import { Link, Outlet, useLocation } from 'react-router-dom';

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

export function Layout() {
  const location = useLocation();
  const isPlayground = location.pathname === '/playground';
  const isLoggedIn = useIsLoggedIn();

  return (
    <div className={`bg-black text-white font-sans selection:bg-white selection:text-black flex flex-col ${isPlayground ? 'h-screen overflow-hidden' : 'min-h-screen'}`}>
      <nav className="border-b border-white/10 px-6 py-4 flex items-center justify-between bg-black/80 backdrop-blur-md z-50 shrink-0">
        <Link to="/" className="flex items-center gap-2">
          <img src="/logo.webp" alt="OpenPaths" className="w-6 h-6" />
          <span className="font-mono font-bold text-xl tracking-tighter">OpenPaths</span>
        </Link>
        <div className="hidden md:flex items-center gap-8 text-sm font-mono text-white/60">
          <Link to="/models" className={`transition-colors ${location.pathname === '/models' ? 'text-white' : 'hover:text-white'}`}>Models</Link>
          <Link to="/pricing" className={`transition-colors ${location.pathname === '/pricing' ? 'text-white' : 'hover:text-white'}`}>Pricing</Link>
          <Link to="/providers" className={`transition-colors ${location.pathname === '/providers' ? 'text-white' : 'hover:text-white'}`}>Providers</Link>
          <Link to="/docs" className={`transition-colors ${location.pathname === '/docs' ? 'text-white' : 'hover:text-white'}`}>Docs</Link>
          <Link to="/integrations" className={`transition-colors ${location.pathname === '/integrations' ? 'text-white' : 'hover:text-white'}`}>Integrations</Link>
          <Link to="/playground" className={`transition-colors ${isPlayground ? 'text-white' : 'hover:text-white'}`}>Playground</Link>
          <Link to="/blog" className={`transition-colors ${location.pathname.startsWith('/blog') ? 'text-white' : 'hover:text-white'}`}>Blog</Link>
          <a href="/#api" className="hover:text-white transition-colors">API</a>
        </div>
        <div className="flex items-center gap-4">
          {isLoggedIn ? (
            <>
              <Link to="/account" className="text-sm font-mono text-white/60 hover:text-white transition-colors hidden sm:block" data-testid="nav-account">Account</Link>
              <Link to="/account" className="bg-white text-black px-4 py-2 text-sm font-mono font-bold hover:bg-white/90 transition-colors rounded" data-testid="nav-dashboard">
                Dashboard
              </Link>
            </>
          ) : (
            <>
              <Link to="/account" className="text-sm font-mono text-white/60 hover:text-white transition-colors hidden sm:block" data-testid="nav-signin">Sign In</Link>
              <Link to="/account" className="bg-white text-black px-4 py-2 text-sm font-mono font-bold hover:bg-white/90 transition-colors rounded" data-testid="nav-get-started">
                Get Started
              </Link>
            </>
          )}
        </div>
      </nav>

      <main className="flex-1 min-h-0">
        <Outlet />
      </main>

      {!isPlayground && (
        <footer className="border-t border-white/10 px-6 py-12 mt-auto">
          <div className="max-w-7xl mx-auto">
            <div className="grid grid-cols-2 md:grid-cols-4 gap-8 mb-10">
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
                  <li><Link to="/pricing" className="hover:text-white transition-colors">Pricing</Link></li>
                  <li><Link to="/integrations" className="hover:text-white transition-colors">Integrations</Link></li>
                  <li><Link to="/playground" className="hover:text-white transition-colors">Playground</Link></li>
                  <li><Link to="/blog" className="hover:text-white transition-colors">Blog</Link></li>
                  <li><a href="/#api" className="hover:text-white transition-colors">API Docs</a></li>
                </ul>
              </div>
              <div>
                <h4 className="text-xs font-mono font-bold text-white/60 uppercase tracking-widest mb-3">Our Sites</h4>
                <ul className="space-y-2 text-sm font-mono text-white/40">
                  <li><a href="https://netwrck.com" target="_blank" rel="noreferrer" className="hover:text-white transition-colors">Netwrck.com</a></li>
                  <li><a href="https://text-generator.io" target="_blank" rel="noreferrer" className="hover:text-white transition-colors">Text-Generator.io</a></li>
                  <li><a href="https://ebank.nz" target="_blank" rel="noreferrer" className="hover:text-white transition-colors">ebank.nz — AI Art</a></li>
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
