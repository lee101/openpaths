import { Link } from 'react-router-dom';
import { Copy, Terminal } from 'lucide-react';
import { Seo } from '../components/Seo';

const installCommand = 'curl -fsSL https://raw.githubusercontent.com/lee101/pi-infinity/main/install.sh | sh';

export function OpenPathsHarness() {
  return (
    <>
      <Seo
        title="OpenPaths Harness | OpenPaths"
        description="Install the OpenPaths-compatible pinf coding harness in one command."
        path="/op"
      />
      <main className="max-w-4xl mx-auto px-6 py-20">
        <p className="text-xs font-mono uppercase tracking-[0.25em] text-white/45 mb-5">OpenPaths / harness</p>
        <h1 className="text-4xl md:text-6xl font-bold tracking-tight mb-6">A coding harness with OpenPaths built in.</h1>
        <p className="max-w-2xl text-lg text-white/60 leading-relaxed mb-10">
          <code>pinf</code> is a local, autonomous coding agent for long-running work. Point it at OpenPaths-compatible
          models, let OpenRouter choose the lowest-cost provider, and keep the whole session on your machine.
        </p>
        <div className="rounded-2xl border border-white/15 bg-black/35 p-5 mb-10">
          <div className="flex items-center gap-2 text-xs font-mono text-white/45 mb-3"><Terminal className="w-4 h-4" /> Install</div>
          <code className="block text-sm md:text-base text-white/85 break-all">{installCommand}</code>
          <button
            type="button"
            className="mt-4 inline-flex items-center gap-2 text-xs font-mono text-white/55 hover:text-white"
            onClick={() => void navigator.clipboard?.writeText(installCommand)}
          >
            <Copy className="w-3.5 h-3.5" /> Copy command
          </button>
        </div>
        <div className="grid md:grid-cols-3 gap-4 mb-12">
          <div className="border border-white/10 rounded-xl p-5"><h2 className="font-semibold mb-2">Local first</h2><p className="text-sm text-white/55">Your files, sessions, and tools stay on your machine.</p></div>
          <div className="border border-white/10 rounded-xl p-5"><h2 className="font-semibold mb-2">Cost aware</h2><p className="text-sm text-white/55">OpenRouter requests default to lowest-price provider routing.</p></div>
          <div className="border border-white/10 rounded-xl p-5"><h2 className="font-semibold mb-2">Autonomous</h2><p className="text-sm text-white/55">Use <code>--auto-next-steps</code> for long-running implementation loops.</p></div>
        </div>
        <Link to="/blog/openpaths-harness-pinf" className="text-sm font-mono text-white/60 underline underline-offset-4 hover:text-white">Read the launch post →</Link>
      </main>
    </>
  );
}
