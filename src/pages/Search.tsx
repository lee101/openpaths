import React, { useMemo, useState } from 'react';
import { Check, Copy, ExternalLink, Globe, Loader2, Search as SearchIcon, SlidersHorizontal } from 'lucide-react';
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
const PAPERS_TYPES = ['papers', 'methods', 'datasets', 'github_code'] as const;

// Provider registry. Add a new search provider by appending one entry here —
// the UI, tabs, curl example and cost estimate all derive from it.
type ProviderKind = 'answer' | 'results';
type ProviderDef = {
  id: string; // value sent as `provider`
  label: string;
  kind: ProviderKind;
  logo: string;
  blurb: string;
  docs?: string;
  defaultModel?: string;
  costUSD: number;
};

const PROVIDERS: ProviderDef[] = [
  { id: 'gemini', label: 'Gemini Flash', kind: 'answer', logo: '/logos/google.svg', blurb: 'Google Flash with live Google Search grounding', defaultModel: 'gemini-3.7-flash', costUSD: 0.004 },
  { id: 'openai', label: 'OpenAI', kind: 'answer', logo: '/logos/openai.svg', blurb: 'GPT with the web_search tool', defaultModel: 'gpt-4o', costUSD: 0.01 },
  { id: 'grok', label: 'Grok', kind: 'answer', logo: '/logos/xai-64.webp', blurb: 'xAI Grok live search', defaultModel: 'grok-latest', costUSD: 0.01 },
  { id: 'exa', label: 'Exa', kind: 'results', logo: '/logos/exa.svg', blurb: 'Neural web search with full contents', docs: '/exa/docs', costUSD: 0.0077 },
  { id: 'papers', label: 'Papers', kind: 'results', logo: '/logos/papers.webp', blurb: 'Research papers, methods, datasets & code', docs: '/papers/docs', costUSD: 0.001 },
];

type Result = {
  id?: string;
  title?: string;
  url?: string;
  highlights?: string[];
  image?: string;
  favicon?: string;
  publishedDate?: string;
  author?: string;
  text?: string;
};

type AnswerResponse = {
  provider?: string;
  model?: string;
  answer?: string;
  citations?: Result[];
  searchQueries?: string[];
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

function faviconFor(url?: string): string | undefined {
  if (!url) return undefined;
  try {
    const host = new URL(url).hostname;
    return `https://www.google.com/s2/favicons?domain=${host}&sz=64`;
  } catch {
    return undefined;
  }
}

function hostLabel(url?: string): string {
  if (!url) return '';
  try {
    return new URL(url).hostname.replace(/^www\./, '');
  } catch {
    return url;
  }
}

export function Search() {
  const [apiKey, setApiKey] = useState(() => getStoredAPIKey());
  const [providerId, setProviderId] = useState<string>('gemini');
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
  const [papersType, setPapersType] = useState<typeof PAPERS_TYPES[number]>('papers');
  const [papersFormat, setPapersFormat] = useState<'json' | 'markdown'>('markdown');
  const [papersRecent, setPapersRecent] = useState(false);
  const [papersHasCode, setPapersHasCode] = useState(false);
  const [papersIncludeGithubCode, setPapersIncludeGithubCode] = useState(false);
  const [model, setModel] = useState('');
  const [showOptions, setShowOptions] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [response, setResponse] = useState<any>(null);
  const [copied, setCopied] = useState(false);

  const provider = useMemo(() => PROVIDERS.find(p => p.id === providerId) || PROVIDERS[0], [providerId]);
  const isAnswer = provider.kind === 'answer';

  const body = useMemo(() => {
    if (provider.id === 'papers') {
      return {
        provider: 'papers',
        query,
        numResults,
        type: papersType,
        ...(papersFormat === 'markdown' ? { format: 'markdown' } : {}),
        ...(papersRecent ? { sort: 'recent' } : {}),
        ...(papersHasCode ? { hasCode: true } : {}),
        ...(papersIncludeGithubCode ? { includeGithubCode: true } : {}),
      };
    }

    if (provider.kind === 'answer') {
      return {
        provider: provider.id,
        query,
        ...(model.trim() ? { model: model.trim() } : {}),
      };
    }

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
        contents.summary = { query: summaryQuery, schema: JSON.parse(summarySchema) };
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
  }, [category, endDate, excludeDomains, fullText, highlightChars, highlightQuery, highlights, includeDomains, livecrawl, livecrawlTimeout, mainContentOnly, maxAge, model, numResults, papersFormat, papersHasCode, papersIncludeGithubCode, papersRecent, papersType, provider, query, startDate, structuredOutputs, summaryQuery, summarySchema, textChars, type, userLocation]);

  const curl = useMemo(() => `curl ${window.location.origin}/v1/search \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${apiKey || 'op-...'}" \\
  -d '${JSON.stringify(body, null, 2)}'`, [apiKey, body]);

  const estimatedCost = useMemo(() => {
    if (provider.kind === 'answer') return provider.costUSD;
    if (provider.id === 'papers') return 0.001;
    let cost = 0.0077;
    if (numResults > 10) cost += (numResults - 10) * 0.0011;
    if ((body as any).contents) cost += numResults * 0.0011;
    return cost;
  }, [body, numResults, provider]);

  const runSearch = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!query.trim()) return;
    setError('');
    setResponse(null);
    setLoading(true);
    try {
      const resp = await fetch('/v1/search', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${apiKey}` },
        body: JSON.stringify(body),
      });
      const text = await resp.text();
      const contentType = resp.headers.get('content-type') || '';
      const data = contentType.includes('application/json') && text ? JSON.parse(text) : text;
      if (!resp.ok) {
        throw new Error((typeof data === 'object' && data?.error?.message) || text || `HTTP ${resp.status}`);
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

  const responseText = typeof response === 'string' ? response : '';
  const answer: AnswerResponse | null = response && typeof response === 'object' && !Array.isArray(response) ? response : null;
  const results: Result[] = Array.isArray(response?.results) ? response.results : [];

  return (
    <>
      <Seo
        title="AI Search Playground — Gemini, OpenAI, Grok, Exa & Papers | OpenPaths"
        description="A Google-style search console powered by Gemini Flash with Google Search grounding, plus OpenAI web search, Grok live search, Exa neural search and Papers research search — one /v1/search API."
        path="/search"
      />
      <div className="min-h-screen bg-black">
        {/* Hero / search box */}
        <section className="border-b border-white/20 bg-gradient-to-b from-white/[0.04] to-transparent">
          <div className="mx-auto max-w-4xl px-6 pb-8 pt-14 text-center">
            <div className="mb-4 inline-flex items-center gap-2 rounded-full border border-white/20 bg-white/[0.06] px-3 py-1 font-mono text-[11px] uppercase tracking-[0.22em] text-white/45">
              <Globe className="h-3.5 w-3.5" /> OpenPaths Search
            </div>
            <h1 className="text-4xl font-bold tracking-tight md:text-6xl">Search the web with AI</h1>
            <p className="mx-auto mt-3 max-w-2xl text-base leading-relaxed text-white/60">
              One <span className="font-mono text-white/80">/v1/search</span> endpoint, many engines. {provider.blurb}.
            </p>

            <form onSubmit={runSearch} className="mt-7">
              <div className="flex items-center gap-2 rounded-full border border-white/15 bg-white/[0.07] px-4 py-2 shadow-lg shadow-black/30 focus-within:border-white/40">
                <SearchIcon className="h-5 w-5 shrink-0 text-white/55" />
                <input
                  value={query}
                  onChange={e => setQuery(e.target.value)}
                  placeholder="Ask anything…"
                  aria-label="Search query"
                  className="w-full bg-transparent px-1 py-2 text-base outline-none placeholder:text-white/45"
                />
                <button
                  type="submit"
                  disabled={!apiKey || !query.trim() || loading}
                  className="inline-flex shrink-0 items-center gap-2 rounded-full border border-white bg-white px-5 py-2 font-mono text-sm font-bold text-black hover:bg-white/90 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <SearchIcon className="h-4 w-4" />}
                  Search
                </button>
              </div>

              {/* Provider tabs */}
              <div className="mt-4 flex flex-wrap items-center justify-center gap-2">
                {PROVIDERS.map(p => (
                  <button
                    key={p.id}
                    type="button"
                    onClick={() => setProviderId(p.id)}
                    className={`inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-sm transition-colors ${provider.id === p.id ? 'border-white bg-white text-black' : 'border-white/20 bg-white/[0.06] text-white/65 hover:border-white/50'}`}
                  >
                    <img src={p.logo} alt="" className={`h-4 w-4 rounded-sm ${provider.id === p.id ? '' : 'opacity-80'}`} />
                    {p.label}
                  </button>
                ))}
              </div>
            </form>

            {!apiKey && (
              <p className="mt-4 text-xs text-white/45">
                Enter your OpenPaths API key below to run searches. <span className="font-mono">op-…</span>
              </p>
            )}
          </div>
        </section>

        <div className="mx-auto grid max-w-6xl gap-6 px-6 py-8 lg:grid-cols-[1fr_360px]">
          {/* Results column */}
          <main className="order-2 space-y-6 lg:order-1">
            {error && (
              <div className="rounded-2xl border border-red-400/20 bg-red-500/10 p-4 text-sm text-red-200">{error}</div>
            )}

            {/* Answer block (answer providers) */}
            {isAnswer && answer?.answer && (
              <div className="rounded-2xl border border-white/20 bg-white/[0.06] p-6">
                <div className="mb-3 flex items-center gap-2 font-mono text-xs uppercase tracking-[0.18em] text-white/45">
                  <img src={provider.logo} alt="" className="h-4 w-4 rounded-sm" />
                  {provider.label} answer{answer.model ? ` · ${answer.model}` : ''}
                </div>
                <div className="whitespace-pre-wrap text-[15px] leading-relaxed text-white/85">{answer.answer}</div>
                {answer.searchQueries && answer.searchQueries.length > 0 && (
                  <div className="mt-4 flex flex-wrap gap-2">
                    {answer.searchQueries.map((q, i) => (
                      <span key={i} className="rounded-full border border-white/20 bg-white/[0.07] px-2.5 py-1 font-mono text-[11px] text-white/50">{q}</span>
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* Sources / results */}
            {responseText && (
              <CodeBlock code={responseText} language="markdown" preClassName="rounded-xl border border-white/20 bg-black/60 p-4 text-xs leading-6" />
            )}

            {results.length > 0 && (
              <div className="rounded-2xl border border-white/20 bg-white/[0.05] p-5">
                <div className="mb-4 flex items-end justify-between gap-3">
                  <h2 className="text-lg font-bold tracking-tight">{isAnswer ? 'Sources' : 'Results'}</h2>
                  <span className="font-mono text-xs text-white/55">{results.length} link{results.length === 1 ? '' : 's'}</span>
                </div>
                <div className="space-y-3">
                  {results.map((result, idx) => {
                    const fav = result.favicon || faviconFor(result.url);
                    return (
                      <article key={result.id || result.url || idx} className="rounded-xl border border-white/20 bg-black/35 p-4 hover:border-white/40">
                        <div className="flex gap-3">
                          {fav && <img src={fav} alt="" className="mt-0.5 h-5 w-5 rounded-sm" loading="lazy" />}
                          <div className="min-w-0 flex-1">
                            {result.url && <div className="truncate font-mono text-[11px] text-white/55">{hostLabel(result.url)}</div>}
                            <a href={result.url} target="_blank" rel="noreferrer" className="font-semibold text-white hover:underline">
                              {result.title || result.url || 'Untitled result'}
                            </a>
                            {result.highlights && result.highlights.length > 0 && (
                              <div className="mt-2 space-y-2">
                                {result.highlights.map((h, i) => (
                                  <p key={i} className="text-sm leading-relaxed text-white/64">{h}</p>
                                ))}
                              </div>
                            )}
                            {!result.highlights?.length && result.text && (
                              <p className="mt-2 line-clamp-4 text-sm leading-relaxed text-white/64">{result.text}</p>
                            )}
                          </div>
                          {result.image && <img src={result.image} alt="" className="hidden h-20 w-28 rounded-lg object-cover md:block" />}
                        </div>
                      </article>
                    );
                  })}
                </div>
              </div>
            )}

            {!loading && response && results.length === 0 && !responseText && !answer?.answer && (
              <div className="rounded-2xl border border-white/20 bg-white/[0.05] p-6 text-sm text-white/45">No results returned.</div>
            )}

            {!response && !loading && !error && (
              <div className="rounded-2xl border border-dashed border-white/20 bg-white/[0.03] p-10 text-center text-sm text-white/55">
                Results will appear here. Pick an engine above and hit Search.
              </div>
            )}

            {/* Code example */}
            <div className="rounded-2xl border border-white/20 bg-white/[0.05] p-5">
              <div className="mb-3 flex items-center justify-between gap-3">
                <div>
                  <h2 className="font-semibold tracking-tight">Code example</h2>
                  <p className="mt-1 text-sm text-white/50">OpenPaths <span className="font-mono">/v1/search</span> request for the current settings.</p>
                </div>
                <button type="button" onClick={copyCurl} className="inline-flex items-center gap-2 rounded border border-white/20 px-3 py-2 font-mono text-xs text-white/70 hover:border-white/45 hover:text-white">
                  {copied ? <Check className="h-3.5 w-3.5 text-green-400" /> : <Copy className="h-3.5 w-3.5" />}
                  {copied ? 'Copied' : 'Copy'}
                </button>
              </div>
              <CodeBlock code={curl} language="bash" preClassName="rounded-xl border border-white/20 bg-black/60 p-4 text-xs leading-6" />
            </div>
          </main>

          {/* Settings column */}
          <aside className="order-1 space-y-4 lg:order-2">
            <Panel title="Request" icon={<SearchIcon className="h-4 w-4" />}>
              <div className="flex items-center gap-2">
                <img src={provider.logo} alt="" className="h-7 w-7 rounded-lg border border-white/20 bg-white p-1.5" />
                <div className="min-w-0">
                  <div className="truncate text-sm font-semibold">{provider.label}</div>
                  <div className="truncate text-[11px] text-white/45">{provider.blurb}</div>
                </div>
                {provider.docs && (
                  <a href={provider.docs} className="ml-auto inline-flex items-center gap-1 rounded border border-white/20 px-2 py-1 font-mono text-[11px] text-white/55 hover:border-white/50 hover:text-white">
                    Docs <ExternalLink className="h-3 w-3" />
                  </a>
                )}
              </div>

              <label className="mt-4 block text-xs font-mono text-white/45">OpenPaths API key</label>
              <input
                value={apiKey}
                onChange={e => setApiKey(e.target.value)}
                placeholder="op-..."
                className="mt-2 w-full rounded border border-white/20 bg-black px-3 py-2.5 font-mono text-sm outline-none focus:border-white/60"
              />

              {isAnswer && (
                <>
                  <label className="mt-4 block text-xs font-mono text-white/45">Model (optional)</label>
                  <input
                    value={model}
                    onChange={e => setModel(e.target.value)}
                    placeholder={provider.defaultModel}
                    className="mt-2 w-full rounded border border-white/20 bg-black px-3 py-2.5 font-mono text-sm outline-none focus:border-white/60"
                  />
                </>
              )}

              {provider.id === 'exa' && (
                <>
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
                        className={`rounded border px-2 py-2 text-left transition-colors ${type === item.value ? 'border-white bg-white text-black' : 'border-white/20 bg-white/[0.06] text-white/65 hover:border-white/45'}`}
                      >
                        <div className="text-xs font-bold">{item.label}</div>
                        <div className="text-[10px] opacity-65">{item.latency}</div>
                      </button>
                    ))}
                  </div>
                </>
              )}

              {provider.id === 'papers' && (
                <div className="mt-4 grid grid-cols-2 gap-3">
                  <Field label="Results">
                    <input type="number" min={1} max={100} value={numResults} onChange={e => setNumResults(Number(e.target.value))} className="input" />
                  </Field>
                  <Field label="Type">
                    <select value={papersType} onChange={e => setPapersType(e.target.value as typeof PAPERS_TYPES[number])} className="input">
                      {PAPERS_TYPES.map(item => <option key={item} value={item}>{item}</option>)}
                    </select>
                  </Field>
                  <Field label="Format">
                    <select value={papersFormat} onChange={e => setPapersFormat(e.target.value as 'json' | 'markdown')} className="input">
                      <option value="markdown">Markdown</option>
                      <option value="json">JSON</option>
                    </select>
                  </Field>
                  <label className="flex items-end gap-2 pb-2 text-xs text-white/60">
                    <input type="checkbox" checked={papersRecent} onChange={e => setPapersRecent(e.target.checked)} />
                    Recent first
                  </label>
                  <label className="flex items-end gap-2 pb-2 text-xs text-white/60">
                    <input type="checkbox" checked={papersHasCode} onChange={e => setPapersHasCode(e.target.checked)} />
                    Has code
                  </label>
                  <label className="flex items-end gap-2 pb-2 text-xs text-white/60">
                    <input type="checkbox" checked={papersIncludeGithubCode} onChange={e => setPapersIncludeGithubCode(e.target.checked)} />
                    Include GitHub code
                  </label>
                </div>
              )}
            </Panel>

            <div className="grid grid-cols-2 gap-3">
              <Stat label="Est. cost" value={`$${estimatedCost.toFixed(4)}`} />
              <Stat label="Engine" value={provider.label} />
            </div>

            {provider.id === 'exa' && (
              <>
                <div>
                  <button type="button" onClick={() => setShowOptions(v => !v)} className="flex w-full items-center justify-between rounded border border-white/20 bg-white/[0.05] px-3 py-2 text-xs font-mono uppercase tracking-[0.18em] text-white/45 hover:border-white/45">
                    <span className="inline-flex items-center gap-2"><SlidersHorizontal className="h-3.5 w-3.5" /> Advanced options</span>
                    <span>{showOptions ? '–' : '+'}</span>
                  </button>
                </div>
                {showOptions && (
                  <>
                    <Panel title="Contents">
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
                  </>
                )}
              </>
            )}
          </aside>
        </div>
      </div>
    </>
  );
}

function Panel({ title, icon, children }: { title: string; icon?: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="rounded-2xl border border-white/20 bg-white/[0.05] p-4">
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
    <div className="rounded-2xl border border-white/20 bg-white/[0.05] p-4">
      <div className="font-mono text-xs uppercase tracking-[0.18em] text-white/50">{label}</div>
      <div className="mt-2 truncate text-xl font-bold tracking-tight">{value}</div>
    </div>
  );
}
