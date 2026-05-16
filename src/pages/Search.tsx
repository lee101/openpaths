import React, { useMemo, useState } from 'react';
import { Check, Copy, ExternalLink, Loader2, Search as SearchIcon, SlidersHorizontal } from 'lucide-react';
import { CodeBlock } from '../components/CodeBlock';
import { Seo } from '../components/Seo';
import { getStoredAPIKey } from '../lib/session';

const SEARCH_TYPES = [
  { value: 'instant', label: 'Instant', latency: '200ms' },
  { value: 'fast', label: 'Fast', latency: '450ms' },
  { value: 'auto', label: 'Auto', latency: '1s' },
  { value: 'deep', label: 'Deep', latency: '4s-18s' },
] as const;

const CATEGORIES = ['', 'company', 'research paper', 'news article', 'github', 'personal site', 'people', 'financial report'] as const;

type Result = {
  id?: string;
  title?: string;
  url?: string;
  highlights?: string[];
  highlightScores?: number[];
  image?: string;
  favicon?: string;
  publishedDate?: string;
  author?: string;
  text?: string;
};

function splitList(value: string): string[] | undefined {
  const items = value.split(/[\n,]+/).map(v => v.trim()).filter(Boolean);
  return items.length ? items : undefined;
}

function toIsoOrUndefined(value: string, endOfDay = false): string | undefined {
  if (!value) return undefined;
  const suffix = endOfDay ? 'T23:59:59.999Z' : 'T00:00:00.000Z';
  return `${value}${suffix}`;
}

export function Search() {
  const [apiKey, setApiKey] = useState(() => getStoredAPIKey());
  const [query, setQuery] = useState('Latest news on Nvidia');
  const [numResults, setNumResults] = useState(10);
  const [type, setType] = useState<typeof SEARCH_TYPES[number]['value']>('auto');
  const [category, setCategory] = useState('');
  const [includeDomains, setIncludeDomains] = useState('');
  const [excludeDomains, setExcludeDomains] = useState('reddit.com');
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');
  const [userLocation, setUserLocation] = useState('');
  const [highlights, setHighlights] = useState(true);
  const [highlightQuery, setHighlightQuery] = useState('');
  const [highlightChars, setHighlightChars] = useState(4000);
  const [fullText, setFullText] = useState(false);
  const [textChars, setTextChars] = useState(20000);
  const [mainContentOnly, setMainContentOnly] = useState(true);
  const [structuredOutputs, setStructuredOutputs] = useState(false);
  const [summaryQuery, setSummaryQuery] = useState('Return the key facts as structured JSON.');
  const [summarySchema, setSummarySchema] = useState(`{
  "type": "object",
  "properties": {
    "key_takeaways": {
      "type": "array",
      "items": { "type": "string" }
    }
  },
  "required": ["key_takeaways"]
}`);
  const [livecrawl, setLivecrawl] = useState('');
  const [livecrawlTimeout, setLivecrawlTimeout] = useState(30000);
  const [maxAge, setMaxAge] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [response, setResponse] = useState<any>(null);
  const [copied, setCopied] = useState(false);

  const body = useMemo(() => {
    const contents: Record<string, unknown> = {};
    if (highlights) {
      const h: Record<string, unknown> = {};
      if (highlightQuery.trim()) h.query = highlightQuery.trim();
      if (highlightChars > 0 && highlightChars !== 4000) h.maxCharacters = highlightChars;
      contents.highlights = Object.keys(h).length ? h : true;
    }
    if (fullText) {
      contents.text = { maxCharacters: textChars, ...(mainContentOnly ? { includeHtmlTags: false } : {}) };
    }
    if (structuredOutputs) {
      try {
        contents.summary = {
          query: summaryQuery,
          schema: JSON.parse(summarySchema),
        };
      } catch {
        contents.summary = { query: summaryQuery };
      }
    }

    return {
      query,
      ...(category ? { category } : {}),
      numResults,
      type,
      ...(userLocation.trim() ? { userLocation: userLocation.trim() } : {}),
      ...(splitList(includeDomains) ? { includeDomains: splitList(includeDomains) } : {}),
      ...(splitList(excludeDomains) ? { excludeDomains: splitList(excludeDomains) } : {}),
      ...(toIsoOrUndefined(startDate) ? { startPublishedDate: toIsoOrUndefined(startDate) } : {}),
      ...(toIsoOrUndefined(endDate, true) ? { endPublishedDate: toIsoOrUndefined(endDate, true) } : {}),
      ...(livecrawl ? { livecrawl } : {}),
      ...(maxAge.trim() ? { maxAgeHours: Number(maxAge) } : {}),
      ...(livecrawlTimeout > 0 && livecrawl ? { livecrawlTimeout } : {}),
      ...(Object.keys(contents).length ? { contents } : {}),
    };
  }, [category, endDate, excludeDomains, fullText, highlightChars, highlightQuery, highlights, includeDomains, livecrawl, livecrawlTimeout, mainContentOnly, maxAge, numResults, query, startDate, structuredOutputs, summaryQuery, summarySchema, textChars, type, userLocation]);

  const curl = useMemo(() => `curl ${window.location.origin}/v1/search \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${apiKey || 'op-...'}" \\
  -d '${JSON.stringify(body, null, 2)}'`, [apiKey, body]);

  const estimatedCost = useMemo(() => {
    let cost = 0.0077;
    if (numResults > 10) cost += (numResults - 10) * 0.0011;
    if (body.contents) cost += numResults * 0.0011;
    return cost;
  }, [body.contents, numResults]);

  const runSearch = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setResponse(null);
    setLoading(true);
    try {
      const resp = await fetch('/v1/search', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${apiKey}`,
        },
        body: JSON.stringify(body),
      });
      const text = await resp.text();
      const data = text ? JSON.parse(text) : null;
      if (!resp.ok) {
        throw new Error(data?.error?.message || text || `HTTP ${resp.status}`);
      }
      setResponse(data);
    } catch (err: any) {
      setError(err.message || 'Search failed');
    } finally {
      setLoading(false);
    }
  };

  const copyCurl = async () => {
    await navigator.clipboard.writeText(curl);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1500);
  };

  const results: Result[] = Array.isArray(response?.results) ? response.results : [];

  return (
    <>
      <Seo
        title="Exa Search Playground | OpenPaths"
        description="Build and test Exa Search API requests through OpenPaths with search type, contents, livecrawl, domain filters, and date controls."
        path="/search"
      />
      <div className="min-h-screen bg-black">
        <section className="border-b border-white/10 bg-white/[0.02]">
          <div className="mx-auto max-w-7xl px-6 py-10">
            <div className="flex flex-col gap-5 md:flex-row md:items-start md:justify-between">
              <div className="max-w-3xl">
                <div className="mb-4 flex items-center gap-3">
                  <img src="/logos/exa.svg" alt="" className="h-9 w-9 rounded-lg border border-white/10 bg-white p-2" />
                  <div className="font-mono text-xs uppercase tracking-[0.22em] text-white/45">Search API</div>
                </div>
                <h1 className="text-4xl font-bold tracking-tight md:text-6xl">Exa search playground</h1>
                <p className="mt-4 max-w-2xl text-base leading-relaxed text-white/62">
                  Test OpenPaths `/v1/search` with Exa-compatible request bodies, including highlights, full text, structured outputs, livecrawl, and filters.
                </p>
              </div>
              <a
                href="/exa/docs"
                className="inline-flex items-center justify-center gap-2 rounded border border-white/15 bg-white/[0.03] px-4 py-3 font-mono text-sm text-white/70 hover:border-white/30 hover:text-white"
              >
                Docs <ExternalLink className="h-4 w-4" />
              </a>
            </div>
          </div>
        </section>

        <form onSubmit={runSearch} className="mx-auto grid max-w-7xl gap-6 px-6 py-8 lg:grid-cols-[420px_1fr]">
          <aside className="space-y-4">
            <Panel title="Request" icon={<SearchIcon className="h-4 w-4" />}>
              <label className="block text-xs font-mono text-white/45">OpenPaths API key</label>
              <input
                value={apiKey}
                onChange={e => setApiKey(e.target.value)}
                placeholder="op-..."
                className="mt-2 w-full rounded border border-white/10 bg-black px-3 py-2.5 font-mono text-sm outline-none focus:border-white/35"
              />
              <label className="mt-4 block text-xs font-mono text-white/45">Query</label>
              <textarea
                value={query}
                onChange={e => setQuery(e.target.value)}
                rows={3}
                className="mt-2 w-full resize-none rounded border border-white/10 bg-black px-3 py-2.5 text-sm outline-none focus:border-white/35"
              />
              <div className="mt-4 grid grid-cols-2 gap-3">
                <Field label="Results">
                  <input type="number" min={1} max={100} value={numResults} onChange={e => setNumResults(Number(e.target.value))} className="input" />
                </Field>
                <Field label="Category">
                  <select value={category} onChange={e => setCategory(e.target.value)} className="input">
                    {CATEGORIES.map(c => <option key={c} value={c}>{c || 'Any'}</option>)}
                  </select>
                </Field>
              </div>
              <div className="mt-4 grid grid-cols-4 gap-2">
                {SEARCH_TYPES.map(item => (
                  <button
                    key={item.value}
                    type="button"
                    onClick={() => setType(item.value)}
                    className={`rounded border px-2 py-2 text-left transition-colors ${type === item.value ? 'border-white bg-white text-black' : 'border-white/10 bg-white/[0.03] text-white/65 hover:border-white/25'}`}
                  >
                    <div className="text-xs font-bold">{item.label}</div>
                    <div className="text-[10px] opacity-65">{item.latency}</div>
                  </button>
                ))}
              </div>
            </Panel>

            <Panel title="Contents" icon={<SlidersHorizontal className="h-4 w-4" />}>
              <Toggle label="Highlights" checked={highlights} onChange={setHighlights} />
              {highlights && (
                <div className="mt-3 grid grid-cols-2 gap-3">
                  <Field label="Guiding query">
                    <input value={highlightQuery} onChange={e => setHighlightQuery(e.target.value)} placeholder="key takeaways" className="input" />
                  </Field>
                  <Field label="Max chars">
                    <input type="number" min={1} value={highlightChars} onChange={e => setHighlightChars(Number(e.target.value))} className="input" />
                  </Field>
                </div>
              )}
              <div className="mt-4">
                <Toggle label="Full webpage text" checked={fullText} onChange={setFullText} />
              </div>
              {fullText && (
                <div className="mt-3 grid grid-cols-2 gap-3">
                  <Field label="Text chars">
                    <input type="number" min={1} max={20000} value={textChars} onChange={e => setTextChars(Number(e.target.value))} className="input" />
                  </Field>
                  <label className="flex items-end gap-2 pb-2 text-xs text-white/60">
                    <input type="checkbox" checked={mainContentOnly} onChange={e => setMainContentOnly(e.target.checked)} />
                    Main content only
                  </label>
                </div>
              )}
              <div className="mt-4">
                <Toggle label="Structured outputs" checked={structuredOutputs} onChange={setStructuredOutputs} />
              </div>
              {structuredOutputs && (
                <div className="mt-3 space-y-3">
                  <Field label="Summary query">
                    <input value={summaryQuery} onChange={e => setSummaryQuery(e.target.value)} className="input" />
                  </Field>
                  <Field label="JSON schema">
                    <textarea value={summarySchema} onChange={e => setSummarySchema(e.target.value)} rows={7} className="textarea font-mono text-xs" />
                  </Field>
                </div>
              )}
            </Panel>

            <Panel title="Filters">
              <Field label="Include domains">
                <textarea value={includeDomains} onChange={e => setIncludeDomains(e.target.value)} rows={2} placeholder="exa.ai, docs.exa.ai/reference" className="textarea" />
              </Field>
              <div className="mt-3">
                <Field label="Exclude domains">
                  <textarea value={excludeDomains} onChange={e => setExcludeDomains(e.target.value)} rows={2} placeholder="reddit.com, twitter.com" className="textarea" />
                </Field>
              </div>
              <div className="mt-3 grid grid-cols-2 gap-3">
                <Field label="Start date">
                  <input type="date" value={startDate} onChange={e => setStartDate(e.target.value)} className="input" />
                </Field>
                <Field label="End date">
                  <input type="date" value={endDate} onChange={e => setEndDate(e.target.value)} className="input" />
                </Field>
              </div>
              <div className="mt-3">
                <Field label="User location">
                  <input value={userLocation} onChange={e => setUserLocation(e.target.value)} placeholder="US" className="input" />
                </Field>
              </div>
            </Panel>

            <Panel title="Livecrawl">
              <div className="grid grid-cols-2 gap-3">
                <Field label="Mode">
                  <select value={livecrawl} onChange={e => setLivecrawl(e.target.value)} className="input">
                    <option value="">Default</option>
                    <option value="never">Never</option>
                    <option value="fallback">Fallback</option>
                    <option value="preferred">Preferred</option>
                    <option value="always">Always</option>
                  </select>
                </Field>
                <Field label="Timeout ms">
                  <input type="number" max={30000} value={livecrawlTimeout} onChange={e => setLivecrawlTimeout(Number(e.target.value))} className="input" />
                </Field>
              </div>
              <div className="mt-3">
                <Field label="Max age hours">
                  <input type="number" min={0} value={maxAge} onChange={e => setMaxAge(e.target.value)} placeholder="24" className="input" />
                </Field>
              </div>
            </Panel>

            <button
              type="submit"
              disabled={!apiKey || !query.trim() || loading}
              className="flex w-full items-center justify-center gap-2 rounded border border-white bg-white px-4 py-3 font-mono text-sm font-bold text-black hover:bg-white/90 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <SearchIcon className="h-4 w-4" />}
              Run Search
            </button>
          </aside>

          <main className="space-y-6">
            <div className="grid gap-3 md:grid-cols-3">
              <Stat label="Estimated cost" value={`$${estimatedCost.toFixed(4)}`} />
              <Stat label="Search type" value={type} />
              <Stat label="Results" value={String(numResults)} />
            </div>

            <div className="rounded-2xl border border-white/10 bg-white/[0.02] p-5">
              <div className="mb-3 flex items-center justify-between gap-3">
                <div>
                  <h2 className="font-semibold tracking-tight">cURL</h2>
                  <p className="mt-1 text-sm text-white/50">OpenPaths endpoint with an Exa-compatible body.</p>
                </div>
                <button type="button" onClick={copyCurl} className="inline-flex items-center gap-2 rounded border border-white/10 px-3 py-2 font-mono text-xs text-white/70 hover:border-white/25 hover:text-white">
                  {copied ? <Check className="h-3.5 w-3.5 text-green-400" /> : <Copy className="h-3.5 w-3.5" />}
                  {copied ? 'Copied' : 'Copy'}
                </button>
              </div>
              <CodeBlock code={curl} language="bash" preClassName="rounded-xl border border-white/10 bg-black/60 p-4 text-xs leading-6" />
            </div>

            {error && (
              <div className="rounded-2xl border border-red-400/20 bg-red-500/10 p-4 text-sm text-red-200">{error}</div>
            )}

            <div className="rounded-2xl border border-white/10 bg-white/[0.02] p-5">
              <div className="mb-4 flex items-end justify-between gap-3">
                <div>
                  <h2 className="text-xl font-bold tracking-tight">Results</h2>
                  <p className="mt-1 text-sm text-white/50">
                    {response?.requestId ? `requestId: ${response.requestId}` : 'Search results will appear here.'}
                  </p>
                </div>
                {response?.resolvedSearchType && <span className="rounded bg-white/10 px-2 py-1 font-mono text-xs text-white/60">{response.resolvedSearchType}</span>}
              </div>
              <div className="space-y-3">
                {results.map((result, idx) => (
                  <article key={result.id || result.url || idx} className="rounded-xl border border-white/10 bg-black/35 p-4">
                    <div className="flex gap-3">
                      {result.favicon && <img src={result.favicon} alt="" className="mt-1 h-5 w-5 rounded-sm" />}
                      <div className="min-w-0 flex-1">
                        <a href={result.url} target="_blank" rel="noreferrer" className="font-semibold text-white hover:underline">
                          {result.title || result.url || 'Untitled result'}
                        </a>
                        {result.url && <div className="mt-1 truncate font-mono text-xs text-white/40">{result.url}</div>}
                        {result.highlights && result.highlights.length > 0 && (
                          <div className="mt-3 space-y-2">
                            {result.highlights.map((h, i) => (
                              <p key={i} className="text-sm leading-relaxed text-white/64">{h}</p>
                            ))}
                          </div>
                        )}
                        {!result.highlights?.length && result.text && (
                          <p className="mt-3 line-clamp-4 text-sm leading-relaxed text-white/64">{result.text}</p>
                        )}
                      </div>
                      {result.image && <img src={result.image} alt="" className="hidden h-20 w-28 rounded-lg object-cover md:block" />}
                    </div>
                  </article>
                ))}
                {!loading && response && results.length === 0 && <div className="text-sm text-white/45">No results returned.</div>}
              </div>
            </div>
          </main>
        </form>
      </div>
    </>
  );
}

function Panel({ title, icon, children }: { title: string; icon?: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="rounded-2xl border border-white/10 bg-white/[0.02] p-4">
      <div className="mb-4 flex items-center gap-2 font-mono text-xs uppercase tracking-[0.18em] text-white/45">
        {icon}
        {title}
      </div>
      {children}
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block">
      <span className="mb-1.5 block text-xs font-mono text-white/45">{label}</span>
      {children}
    </label>
  );
}

function Toggle({ label, checked, onChange }: { label: string; checked: boolean; onChange: (value: boolean) => void }) {
  return (
    <label className="flex items-center justify-between gap-3 text-sm text-white/70">
      <span>{label}</span>
      <input type="checkbox" checked={checked} onChange={e => onChange(e.target.checked)} className="h-4 w-4" />
    </label>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-white/10 bg-white/[0.02] p-4">
      <div className="font-mono text-xs uppercase tracking-[0.18em] text-white/35">{label}</div>
      <div className="mt-2 text-2xl font-bold tracking-tight">{value}</div>
    </div>
  );
}
