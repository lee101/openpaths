import React from 'react';
import { Link, Outlet, useLocation } from 'react-router-dom';
import { Network } from 'lucide-react';

export function Layout() {
  const location = useLocation();
  const isPlayground = location.pathname === '/playground';

  return (
    <div className={`bg-black text-white font-sans selection:bg-white selection:text-black flex flex-col ${isPlayground ? 'h-screen overflow-hidden' : 'min-h-screen'}`}>
      <nav className="border-b border-white/10 px-6 py-4 flex items-center justify-between bg-black/80 backdrop-blur-md z-50 shrink-0">
        <Link to="/" className="flex items-center gap-2">
          <Network className="w-6 h-6" />
          <span className="font-mono font-bold text-xl tracking-tighter">OpenPath</span>
        </Link>
        <div className="hidden md:flex items-center gap-8 text-sm font-mono text-white/60">
          <Link to="/models" className={`transition-colors ${location.pathname === '/models' ? 'text-white' : 'hover:text-white'}`}>Models</Link>
          <Link to="/playground" className={`transition-colors ${isPlayground ? 'text-white' : 'hover:text-white'}`}>Playground</Link>
          <Link to="/blog" className={`transition-colors ${location.pathname.startsWith('/blog') ? 'text-white' : 'hover:text-white'}`}>Blog</Link>
          <a href="/#api" className="hover:text-white transition-colors">API</a>
          <a href="/#pricing" className="hover:text-white transition-colors">Pricing</a>
        </div>
        <div className="flex items-center gap-4">
          <Link to="/account" className="text-sm font-mono text-white/60 hover:text-white transition-colors hidden sm:block">Account</Link>
          <Link to="/account" className="bg-white text-black px-4 py-2 text-sm font-mono font-bold hover:bg-white/90 transition-colors rounded">
            Dashboard
          </Link>
        </div>
      </nav>

      <main className="flex-1 min-h-0">
        <Outlet />
      </main>

      {!isPlayground && (
        <footer className="border-t border-white/10 px-6 py-12 text-center md:text-left mt-auto">
          <div className="max-w-7xl mx-auto flex flex-col md:flex-row justify-between items-center gap-6">
            <div className="flex items-center gap-2">
              <Network className="w-5 h-5" />
              <span className="font-mono font-bold tracking-tighter">OpenPath</span>
            </div>
            <div className="flex gap-6 text-sm font-mono text-white/40">
              <a href="#" className="hover:text-white transition-colors">Twitter</a>
              <a href="#" className="hover:text-white transition-colors">GitHub</a>
              <a href="#" className="hover:text-white transition-colors">Discord</a>
              <a href="#" className="hover:text-white transition-colors">Terms</a>
            </div>
          </div>
        </footer>
      )}
    </div>
  );
}
