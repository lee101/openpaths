import React, { useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Agent, Preset, listAgents, fetchPresets, createAgent } from '../lib/agents';
import { getStoredToken } from '../lib/session';

export function Agents() {
  const nav = useNavigate();
  const [agents, setAgents] = useState<Agent[]>([]);
  const [presets, setPresets] = useState<Preset[]>([]);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState('');
  const loggedIn = !!getStoredToken();

  useEffect(() => {
    fetchPresets().then(r => setPresets(r.presets || [])).catch(() => {});
    if (loggedIn) {
      listAgents().then(setAgents).catch(e => setErr(String(e.message || e))).finally(() => setLoading(false));
    } else {
      setLoading(false);
    }
  }, [loggedIn]);

  async function createFromPreset(p: Preset) {
    try {
      const a = await createAgent({ name: p.name, description: p.description, preset: p.key });
      nav(`/agents/${a.id}`);
    } catch (e: any) {
      setErr(String(e.message || e));
    }
  }

  async function createBlank() {
    try {
      const a = await createAgent({ name: 'New agent', model: 'auto', config: { tools: [], max_steps: 8 } });
      nav(`/agents/${a.id}`);
    } catch (e: any) {
      setErr(String(e.message || e));
    }
  }

  return (
    <div className="mx-auto max-w-7xl px-6 py-12">
      <div className="flex items-end justify-between gap-4 flex-wrap">
        <div>
          <h1 className="text-3xl font-bold">Agents</h1>
          <p className="mt-2 text-white/50 font-mono text-sm max-w-2xl">
            Build agents with tool use & computer use, connect documents and databases for embedding search,
            and orchestrate every model on OpenPaths.
          </p>
        </div>
        {loggedIn && (
          <button onClick={createBlank} className="bg-white text-black px-4 py-2 text-sm font-mono font-bold rounded hover:bg-white/90">
            + New agent
          </button>
        )}
      </div>

      {err && <div className="mt-4 text-sm font-mono text-red-400">{err}</div>}

      {!loggedIn && (
        <div className="mt-6 rounded border border-white/10 bg-white/5 p-4 text-sm font-mono text-white/60">
          <Link to="/account" className="text-white underline">Sign in</Link> to build and run your own agents.
        </div>
      )}

      {loggedIn && (
        <section className="mt-10">
          <h2 className="text-lg font-bold mb-4">Your agents</h2>
          {loading ? (
            <div className="text-white/35 font-mono text-sm">Loading…</div>
          ) : agents.length === 0 ? (
            <div className="text-white/40 font-mono text-sm">No agents yet — start from a preset below.</div>
          ) : (
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {agents.map(a => (
                <Link key={a.id} to={`/agents/${a.id}`} className="block rounded border border-white/10 bg-white/5 p-4 hover:border-white/30 transition-colors">
                  <div className="font-bold">{a.name}</div>
                  <div className="mt-1 text-xs font-mono text-white/45 line-clamp-2">{a.description || '—'}</div>
                  <div className="mt-3 flex flex-wrap gap-1">
                    <span className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-white/10 text-white/60">{a.model}</span>
                    {(a.config?.tools || []).map(t => (
                      <span key={t} className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-white/10 text-white/60">{t}</span>
                    ))}
                  </div>
                </Link>
              ))}
            </div>
          )}
        </section>
      )}

      <section className="mt-12">
        <h2 className="text-lg font-bold mb-4">Preset agents</h2>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          {presets.map(p => (
            <div key={p.key} className="flex flex-col rounded border border-white/10 bg-white/5 p-4">
              <div className="font-bold">{p.name}</div>
              <div className="mt-1 flex-1 text-xs font-mono text-white/50">{p.description}</div>
              <div className="mt-3 flex flex-wrap gap-1">
                {(p.config?.tools || []).map(t => (
                  <span key={t} className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-white/10 text-white/60">{t}</span>
                ))}
              </div>
              {loggedIn && (
                <button onClick={() => createFromPreset(p)} className="mt-4 text-sm font-mono border border-white/20 rounded px-3 py-1.5 hover:bg-white/10">
                  Use this preset
                </button>
              )}
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}
