import { useState } from 'react';
import { KeyRound, Trash2 } from 'lucide-react';
import { api } from '../lib/api';

export const BYOK_PROVIDERS: { id: string; name: string; keyUrl?: string; note?: string }[] = [
  { id: 'openai', name: 'OpenAI', keyUrl: 'https://platform.openai.com/api-keys' },
  { id: 'anthropic', name: 'Anthropic', keyUrl: 'https://console.anthropic.com/settings/keys' },
  { id: 'google', name: 'Google AI', keyUrl: 'https://aistudio.google.com/app/apikey' },
  { id: 'mistral', name: 'Mistral', keyUrl: 'https://console.mistral.ai/api-keys' },
  { id: 'groq', name: 'Groq', keyUrl: 'https://console.groq.com/keys' },
  { id: 'xai', name: 'xAI', keyUrl: 'https://console.x.ai' },
  { id: 'deepseek', name: 'DeepSeek', keyUrl: 'https://platform.deepseek.com/api_keys' },
  { id: 'thinkingmachines', name: 'Thinking Machines (Tinker)', keyUrl: 'https://tinker.thinkingmachines.dev' },
  { id: 'openrouter', name: 'OpenRouter', keyUrl: 'https://openrouter.ai/keys' },
  { id: 'inference_net', name: 'Inference.net', keyUrl: 'https://inference.net' },
  { id: 'together', name: 'Together AI', keyUrl: 'https://api.together.xyz/settings/api-keys' },
  { id: 'minimax', name: 'MiniMax', keyUrl: 'https://www.minimax.io' },
  { id: 'netwrck', name: 'Netwrck', keyUrl: 'https://netwrck.com' },
  { id: 'zai', name: 'Z.AI (GLM)', keyUrl: 'https://z.ai', note: 'Coding Plan keys supported' },
  { id: 'sakana', name: 'Sakana AI', keyUrl: 'https://sakana.ai' },
  { id: 'fal', name: 'fal.ai', keyUrl: 'https://fal.ai/dashboard/keys' },
  { id: 'bfl', name: 'Black Forest Labs', keyUrl: 'https://bfl.ai' },
];

export function ProviderKeysPanel({ keys, onChanged }: { keys: any[]; onChanged: () => void }) {
  const [drafts, setDrafts] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState<string | null>(null);

  const byProvider = new Map<string, any>((keys || []).map(k => [k.provider, k]));

  const save = async (providerId: string) => {
    const apiKey = (drafts[providerId] || '').trim();
    if (!apiKey) return;
    setBusy(providerId);
    setError(null);
    setSaved(null);
    try {
      const res = await api('/account/provider-keys', {
        method: 'POST',
        body: JSON.stringify({ provider: providerId, api_key: apiKey }),
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        setError(data.error?.message || `Failed to save ${providerId} key`);
        return;
      }
      setDrafts(d => ({ ...d, [providerId]: '' }));
      setSaved(providerId);
      onChanged();
    } catch {
      setError('Network error');
    } finally {
      setBusy(null);
    }
  };

  const remove = async (providerId: string) => {
    setBusy(providerId);
    setError(null);
    try {
      const res = await api(`/account/provider-keys?provider=${encodeURIComponent(providerId)}`, { method: 'DELETE' });
      if (!res.ok) {
        setError(`Failed to delete ${providerId} key`);
        return;
      }
      onChanged();
    } catch {
      setError('Network error');
    } finally {
      setBusy(null);
    }
  };

  return (
    <section className="mb-8 border border-white/10 bg-white/[0.02] rounded-3xl p-6" data-testid="byok-panel">
      <div className="mb-5">
        <div className="flex items-center gap-2 text-xs font-mono uppercase tracking-[0.18em] text-white/45 mb-2">
          <KeyRound className="w-4 h-4" /> Provider keys (BYOK)
        </div>
        <h2 className="text-xl font-semibold tracking-tight">Bring your own provider keys</h2>
        <p className="text-sm text-white/55 mt-2 max-w-3xl">
          Requests served through your own key bypass the OpenPaths balance entirely - you pay your
          provider directly and we charge nothing. Fallback to OpenPaths credits still applies when a
          BYOK route fails.
        </p>
      </div>

      {error && <p className="mb-4 text-sm text-red-400" data-testid="byok-error">{error}</p>}

      <div className="overflow-x-auto">
        <table className="w-full text-sm" data-testid="byok-table">
          <thead>
            <tr className="text-left text-xs font-mono uppercase tracking-wider text-white/40 border-b border-white/10">
              <th className="px-4 py-3 font-normal">Provider</th>
              <th className="px-4 py-3 font-normal">Status</th>
              <th className="px-4 py-3 font-normal">API key</th>
              <th className="px-4 py-3 font-normal"></th>
            </tr>
          </thead>
          <tbody className="divide-y divide-white/5">
            {BYOK_PROVIDERS.map(p => {
              const existing = byProvider.get(p.id);
              return (
                <tr key={p.id} data-testid={`byok-row-${p.id}`}>
                  <td className="px-4 py-3 align-middle whitespace-nowrap">
                    <span className="font-medium">{p.name}</span>
                    {p.note && <span className="block text-xs text-white/40">{p.note}</span>}
                  </td>
                  <td className="px-4 py-3 align-middle whitespace-nowrap">
                    {existing?.has_key ? (
                      <span className="inline-flex items-center gap-2 text-emerald-300 text-xs font-mono">
                        <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 inline-block" />
                        {existing.key_preview || 'connected'}
                      </span>
                    ) : (
                      <span className="text-white/35 text-xs font-mono">not connected</span>
                    )}
                  </td>
                  <td className="px-4 py-3 w-full min-w-[220px]">
                    <input
                      type="password"
                      value={drafts[p.id] || ''}
                      onChange={e => setDrafts(d => ({ ...d, [p.id]: e.target.value }))}
                      placeholder={existing?.has_key ? 'Replace key' : 'Paste API key'}
                      className="w-full rounded-lg bg-black/30 border border-white/15 px-3 py-2 font-mono text-xs focus:border-sky-400/60 outline-none placeholder:text-white/25"
                      data-testid={`byok-input-${p.id}`}
                    />
                  </td>
                  <td className="px-4 py-3 align-middle whitespace-nowrap">
                    <div className="flex items-center gap-2 justify-end">
                      <button
                        onClick={() => void save(p.id)}
                        disabled={busy === p.id || !(drafts[p.id] || '').trim()}
                        className="rounded-lg border border-sky-400/30 px-3 py-2 text-xs font-mono text-sky-200 hover:bg-sky-500/10 transition-colors disabled:opacity-40"
                        data-testid={`byok-save-${p.id}`}
                      >
                        {busy === p.id ? '...' : saved === p.id ? 'Saved' : 'Save'}
                      </button>
                      {existing?.has_key && (
                        <button
                          onClick={() => void remove(p.id)}
                          disabled={busy === p.id}
                          aria-label={`Delete ${p.name} key`}
                          className="rounded-lg border border-red-400/30 p-2 text-red-300 hover:bg-red-500/10 transition-colors disabled:opacity-40"
                          data-testid={`byok-delete-${p.id}`}
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      <p className="mt-4 text-xs text-white/35 max-w-3xl">
        Keys are stored server-side and only returned as masked previews. The OpenAI Max plan uses
        sign-in instead of a key - see the panel above. See{' '}
        <a href="/byok" className="text-sky-300 hover:underline">openpaths.io/byok</a> for details.
      </p>
    </section>
  );
}
