import React, { useMemo, useState } from 'react';
import { Check, Copy, ExternalLink, Github, PlugZap } from 'lucide-react';
import { Link } from 'react-router-dom';
import { Seo } from '../components/Seo';
import { CodeBlock } from '../components/CodeBlock';
import { getAPIBaseURL, getStoredAPIKey } from '../lib/session';
import {
  WORKS_WITH_CATEGORIES,
  WorksWithApp,
  WorksWithStatus,
  worksWithApps,
  worksWithFavicon,
} from '../data/worksWith';

const STATUS_LABEL: Record<WorksWithStatus, string> = {
  'native-merged': 'Native support',
  'native-pr': 'PR open',
  compatible: 'Works today',
  listed: 'Compatible (BYO key)',
};

const STATUS_CLASS: Record<WorksWithStatus, string> = {
  'native-merged': 'border-green-400/30 text-green-300 bg-green-400/10',
  'native-pr': 'border-cyan-400/30 text-cyan-300 bg-cyan-400/10',
  compatible: 'border-white/15 text-white/70 bg-white/[0.04]',
  listed: 'border-white/10 text-white/45 bg-white/[0.02]',
};

const appIcon = (app: WorksWithApp) => worksWithFavicon(app.url);

export function WorksWith() {
  const apiBase = getAPIBaseURL();
  const exampleKey = getStoredAPIKey() || 'sk-openpaths-...';
  const [copied, setCopied] = useState(false);

  const grouped = useMemo(() => {
    return WORKS_WITH_CATEGORIES.map(category => ({
      category,
      apps: worksWithApps.filter(a => a.category === category),
    })).filter(group => group.apps.length > 0);
  }, []);

  const counts = useMemo(() => {
    const oss = worksWithApps.filter(a => a.oss).length;
    return { total: worksWithApps.length, oss };
  }, []);

  const snippet = `curl ${apiBase}/chat/completions \\
  -H "Authorization: Bearer ${exampleKey}" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"openpaths/auto","messages":[{"role":"user","content":"hi"}]}'`;

  const copySnippet = async () => {
    try {
      await navigator.clipboard.writeText(snippet);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      /* ignore */
    }
  };

  return (
    <>
      <Seo
        title="Works With OpenPaths | Compatible Apps & Agents"
        description="OpenPaths is OpenAI- and Anthropic-compatible, so it drops into every OpenRouter-style app. Browse coding agents, chat UIs, frameworks, and tools that work with an OpenPaths API key."
        path="/works-with-openpaths"
      />
      <section className="max-w-6xl mx-auto px-6 py-16">
        <div className="mb-10">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border border-white/10 bg-white/[0.04] text-xs font-mono text-white/60 mb-6">
            <PlugZap className="w-3.5 h-3.5" />
            {counts.total} apps · {counts.oss} open source
          </div>
          <h1 className="text-4xl md:text-5xl font-bold tracking-tight mb-4">Works With OpenPaths</h1>
          <p className="text-white/60 max-w-3xl font-light leading-relaxed">
            OpenPaths is a drop-in OpenAI- and Anthropic-compatible model gateway. Any app that
            speaks OpenRouter speaks OpenPaths — point it at the base URL below with your
            OpenPaths key and use <code className="text-white/80">openpaths/auto</code>. We are
            also upstreaming first-class OpenPaths providers into the open-source ones. See the{' '}
            <Link to="/integrations" className="text-white underline underline-offset-4">integration examples</Link>{' '}
            for SDK snippets.
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
          <div className="border border-white/10 bg-white/[0.02] rounded-2xl p-5">
            <div className="text-xs font-mono text-white/40 mb-2">Base URL</div>
            <code className="text-sm text-white/80 break-all" data-testid="works-base-url">{apiBase}</code>
          </div>
          <div className="border border-white/10 bg-white/[0.02] rounded-2xl p-5">
            <div className="text-xs font-mono text-white/40 mb-2">API Key</div>
            <code className="text-sm text-white/80 break-all">{exampleKey}</code>
          </div>
          <div className="border border-white/10 bg-white/[0.02] rounded-2xl p-5">
            <div className="text-xs font-mono text-white/40 mb-2">Model IDs</div>
            <div className="text-sm text-white/70 font-mono">openpaths/auto · auto-code · auto-reasoning</div>
          </div>
        </div>

        <div className="border border-white/10 bg-white/[0.02] rounded-2xl overflow-hidden mb-12">
          <div className="flex items-center justify-between px-5 py-3 border-b border-white/10">
            <div className="text-xs font-mono text-white/40">Verify your key works anywhere</div>
            <button
              onClick={copySnippet}
              className="inline-flex items-center gap-2 border border-white/10 px-3 py-1.5 rounded-lg text-xs font-mono text-white/70 hover:text-white hover:border-white/20 transition-colors"
            >
              {copied ? <Check className="w-3.5 h-3.5 text-green-400" /> : <Copy className="w-3.5 h-3.5" />}
              {copied ? 'Copied' : 'Copy'}
            </button>
          </div>
          <CodeBlock code={snippet} language="bash" />
        </div>

        {grouped.map(group => (
          <div key={group.category} className="mb-12">
            <h2 className="text-sm font-mono uppercase tracking-[0.24em] text-white/40 mb-4">{group.category}</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {group.apps.map(app => (
                <article
                  key={app.slug}
                  className="border border-white/10 bg-white/[0.02] rounded-2xl p-5 flex flex-col gap-3"
                  data-testid={`works-app-${app.slug}`}
                >
                  <div className="flex items-start gap-3">
                    <img
                      src={appIcon(app)}
                      alt=""
                      loading="lazy"
                      className="w-9 h-9 rounded-lg bg-white/5 border border-white/10 shrink-0"
                    />
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2 flex-wrap">
                        <h3 className="text-lg font-bold tracking-tight">{app.name}</h3>
                        <span className={`text-[10px] font-mono px-2 py-0.5 rounded-full border ${STATUS_CLASS[app.status]}`}>
                          {app.prUrl ? (
                            <a href={app.prUrl} target="_blank" rel="noreferrer" className="hover:underline">
                              {STATUS_LABEL[app.status]}
                            </a>
                          ) : STATUS_LABEL[app.status]}
                        </span>
                      </div>
                      <p className="text-sm text-white/55 leading-relaxed mt-1">{app.description}</p>
                    </div>
                  </div>

                  {app.setup && (
                    <p className="text-xs text-white/45 font-mono leading-relaxed border-l border-white/10 pl-3">
                      {app.setup}
                    </p>
                  )}

                  <div className="flex items-center gap-4 text-xs font-mono mt-auto pt-1">
                    <a href={app.url} target="_blank" rel="noreferrer" className="inline-flex items-center gap-1 text-white/60 hover:text-white transition-colors">
                      <ExternalLink className="w-3.5 h-3.5" /> Site
                    </a>
                    {app.repo && (
                      <a href={app.repo} target="_blank" rel="noreferrer" className="inline-flex items-center gap-1 text-white/60 hover:text-white transition-colors">
                        <Github className="w-3.5 h-3.5" /> Repo
                      </a>
                    )}
                    {app.oss && <span className="text-white/30">open source</span>}
                  </div>
                </article>
              ))}
            </div>
          </div>
        ))}

        <div className="border border-white/10 bg-white/[0.02] rounded-2xl p-6 text-sm text-white/60">
          Building one of these — or your own app? Point it at{' '}
          <code className="text-white/80">{apiBase}</code>, grab a key on the{' '}
          <Link to="/account" className="text-white underline underline-offset-4">account page</Link>, and
          it just works. Want a first-class OpenPaths provider in your project? Open an issue or PR — we
          help with the integration.
        </div>
      </section>
    </>
  );
}
