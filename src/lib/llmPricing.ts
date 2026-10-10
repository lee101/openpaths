import { calculatorModels } from '../data/calculator';
import type { Model } from '../data/models';
import pricingConfig from '../data/pricingConfig.generated.json';
import { artificialAnalysisModels, normalizeModelToken, type ArtificialAnalysisModel } from './artificialAnalysis';

export const PRICING_UPDATED = 'October 2026';
export const PRICING_YEAR = 2026;

type RateSet = { input: number; output: number; cacheRead?: number | null };

type ConfigEntry = {
  id: string;
  provider: string;
  input: number;
  output: number;
  context: number;
  maxOutput: number;
  vision: boolean;
  tools: boolean;
  aliases: string[];
  cacheRead?: number;
  longContext?: { threshold: number; input: number; output: number; cacheRead?: number | null };
  schedule?: {
    effectiveAt: string;
    windowsUtc: string[][];
    peak: RateSet;
    offPeak: RateSet;
    weekendOffPeak: boolean;
  };
};

export interface PricedModel {
  id: string;
  slug: string;
  name: string;
  provider: string;
  providerSlug: string;
  description: string;
  input: number;
  output: number;
  cacheRead?: number;
  context: number;
  maxOutput: number;
  vision: boolean;
  tools: boolean;
  tags: string[];
  released: string;
  popularity: number;
  aliases: string[];
  longContext?: { threshold: number; input: number; output: number; cacheRead?: number };
  schedule?: {
    peak: Required<Pick<RateSet, 'input' | 'output'>> & { cacheRead?: number };
    offPeak: Required<Pick<RateSet, 'input' | 'output'>> & { cacheRead?: number };
    windowsUtc: string[][];
    weekendOffPeak: boolean;
    peakFraction: number;
  };
  tokenizerNote?: string;
}

export interface Workload {
  requests: number;
  inputTokens: number;
  outputTokens: number;
  cacheHit: number;
  peakShare?: number;
}

export interface Rates {
  input: number;
  output: number;
  cacheRead?: number;
  tier: 'standard' | 'long-context' | 'blended';
}

export interface Cost {
  total: number;
  inputCost: number;
  outputCost: number;
  perRequest: number;
  per1kRequests: number;
  rates: Rates;
}

export interface Scenario {
  slug: string;
  name: string;
  headline: string;
  description: string;
  workload: Workload;
}

export const SCENARIOS: Scenario[] = [
  { slug: 'chatbot', name: 'Chatbot', headline: 'consumer chatbot', description: 'Multi-turn chat with a short system prompt. Short answers, half the prompt served from cache.', workload: { requests: 300_000, inputTokens: 1_500, outputTokens: 400, cacheHit: 0.5 } },
  { slug: 'customer-support', name: 'Customer support', headline: 'customer support assistant', description: 'Support replies grounded on a policy prompt and ticket history. Heavy prompt reuse, short replies.', workload: { requests: 100_000, inputTokens: 3_000, outputTokens: 350, cacheHit: 0.6 } },
  { slug: 'rag-qa', name: 'RAG question answering', headline: 'RAG Q&A over documents', description: 'Retrieved passages stuffed into the prompt, concise grounded answer.', workload: { requests: 200_000, inputTokens: 8_000, outputTokens: 500, cacheHit: 0.3 } },
  { slug: 'coding-agent', name: 'Coding agent', headline: 'coding agent', description: 'Agent loops that re-send a large repo context every step. Cache hit rate is high, outputs are medium.', workload: { requests: 50_000, inputTokens: 40_000, outputTokens: 2_000, cacheHit: 0.8 } },
  { slug: 'long-document-analysis', name: 'Long document analysis', headline: 'long document analysis', description: '150K-token documents, no caching. Triggers long-context price tiers on models that have them.', workload: { requests: 5_000, inputTokens: 150_000, outputTokens: 3_000, cacheHit: 0 } },
  { slug: 'classification', name: 'Classification and routing', headline: 'classification and routing', description: 'Very high volume, tiny outputs. Input price dominates.', workload: { requests: 5_000_000, inputTokens: 400, outputTokens: 10, cacheHit: 0 } },
  { slug: 'summarization', name: 'Summarization', headline: 'summarization', description: 'Articles and call transcripts in, short summaries out.', workload: { requests: 100_000, inputTokens: 6_000, outputTokens: 600, cacheHit: 0 } },
  { slug: 'data-extraction', name: 'Structured data extraction', headline: 'structured data extraction', description: 'Invoices, emails and forms converted to JSON. Shared schema prompt is cached.', workload: { requests: 500_000, inputTokens: 2_500, outputTokens: 300, cacheHit: 0.2 } },
  { slug: 'content-generation', name: 'Content generation', headline: 'content generation', description: 'Short briefs in, long drafts out. Output price dominates.', workload: { requests: 50_000, inputTokens: 500, outputTokens: 1_500, cacheHit: 0 } },
  { slug: 'agent-tool-loop', name: 'Agent tool loop', headline: 'agent tool loops', description: 'Many small tool-call turns with growing context and a cached prefix.', workload: { requests: 400_000, inputTokens: 12_000, outputTokens: 250, cacheHit: 0.85 } },
];

export function getScenario(slug: string): Scenario | undefined {
  return SCENARIOS.find(s => s.slug === slug);
}

export const DEFAULT_WORKLOAD: Workload = { requests: 100_000, inputTokens: 2_000, outputTokens: 500, cacheHit: 0.3 };

const configById = new Map<string, ConfigEntry>((pricingConfig as ConfigEntry[]).map(entry => [entry.id, entry]));

export function providerSlug(provider: string): string {
  return provider.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
}

export function modelSlug(id: string): string {
  return id.toLowerCase().replace(/\//g, '--').replace(/[^a-z0-9.\-_]+/g, '-');
}

function peakFractionOf(windows: string[][], weekendOffPeak: boolean): number {
  const hours = windows.reduce((sum, [start, end]) => sum + (hourOf(end) - hourOf(start)), 0);
  const daily = Math.max(0, Math.min(24, hours)) / 24;
  return weekendOffPeak ? daily * (5 / 7) : daily;
}

function hourOf(value: string): number {
  const [h, m] = value.split(':').map(Number);
  return (h || 0) + (m || 0) / 60;
}

function toPriced(model: Model, entry: ConfigEntry): PricedModel {
  const scheduleActive = entry.schedule && Date.parse(entry.schedule.effectiveAt) <= Date.now();
  const priced: PricedModel = {
    id: model.id,
    slug: modelSlug(model.id),
    name: model.name,
    provider: model.provider,
    providerSlug: providerSlug(model.provider),
    description: model.description,
    input: entry.input,
    output: entry.output,
    cacheRead: entry.cacheRead,
    context: entry.context,
    maxOutput: entry.maxOutput,
    vision: entry.vision,
    tools: entry.tools,
    tags: model.tags,
    released: model.released,
    popularity: model.popularity,
    aliases: [...(model.aliases || []), ...(entry.aliases || [])],
  };
  if (entry.longContext) {
    priced.longContext = {
      threshold: entry.longContext.threshold,
      input: entry.longContext.input,
      output: entry.longContext.output,
      cacheRead: entry.longContext.cacheRead ?? undefined,
    };
  }
  if (entry.schedule && scheduleActive) {
    const { peak, offPeak, windowsUtc, weekendOffPeak } = entry.schedule;
    priced.schedule = {
      peak: { input: peak.input, output: peak.output, cacheRead: peak.cacheRead ?? undefined },
      offPeak: { input: offPeak.input, output: offPeak.output, cacheRead: offPeak.cacheRead ?? undefined },
      windowsUtc,
      weekendOffPeak,
      peakFraction: peakFractionOf(windowsUtc, weekendOffPeak),
    };
  }
  if (model.id === 'claude-haiku-5-5') {
    priced.tokenizerNote = 'Haiku 5.5 uses a newer tokenizer that produces roughly 30% more tokens than Haiku 4.5 for the same text, so compare cost per task, not just per token.';
  }
  return priced;
}

function buildPricedModels(): PricedModel[] {
  const out: PricedModel[] = [];
  const slugs = new Set<string>();
  for (const model of calculatorModels) {
    if (model.id.startsWith('openpaths/')) continue;
    const entry = configById.get(model.id);
    if (!entry || (entry.input <= 0 && entry.output <= 0)) continue;
    const priced = toPriced(model, entry);
    if (slugs.has(priced.slug)) continue;
    slugs.add(priced.slug);
    out.push(priced);
  }
  return out;
}

export const PRICED_MODELS: PricedModel[] = buildPricedModels();

const bySlug = new Map(PRICED_MODELS.map(m => [m.slug, m]));

export function getPricedModel(slug: string): PricedModel | undefined {
  return bySlug.get(slug.toLowerCase());
}

function words(value: string): string[] {
  return value.toLowerCase().split(/[^a-z0-9.]+/).filter(Boolean).map(w => w.replace(/\./g, '')).filter(Boolean);
}

const STOP_WORDS = new Set([
  'new', 'pricing', 'price', 'prices', 'cost', 'costs', 'api', 'compare', 'comparison', 'cheaper', 'cheapest', 'per', 'token', 'tokens',
  'what', 'whats', 'is', 'the', 'how', 'much', 'does', 'of', 'model', 'models', 'for', 'a', 'an', 'to', 'and', 'in', 'tier', 'tiers',
]);

function haystack(m: PricedModel): string {
  return normalizeModelToken([m.id, m.name, m.provider, ...m.aliases].join(' '));
}

const haystacks = new Map(PRICED_MODELS.map(m => [m.id, haystack(m)]));

export function resolveModel(query: string): PricedModel | undefined {
  const direct = bySlug.get(query.toLowerCase().trim());
  if (direct) return direct;
  const token = normalizeModelToken(query);
  if (!token) return undefined;
  const exact = PRICED_MODELS.find(m => [m.id, m.name, ...m.aliases].some(v => normalizeModelToken(v) === token));
  if (exact) return exact;
  const terms = words(query).filter(w => !STOP_WORDS.has(w));
  if (terms.length === 0) return undefined;
  const matches = PRICED_MODELS.filter(m => {
    const h = haystacks.get(m.id) || '';
    return terms.every(t => h.includes(t));
  });
  return matches.sort((a, b) => a.popularity - b.popularity || a.id.length - b.id.length)[0];
}

export function parsePairQuery(query: string): [PricedModel, PricedModel] | undefined {
  const parts = query.split(/\s+(?:vs\.?|versus|v|or|compared\s+to|against)\s+|\s*,\s*|\s+-vs-\s+/i).map(p => p.trim()).filter(Boolean);
  if (parts.length < 2) return undefined;
  const a = resolveModel(parts[0]);
  const b = resolveModel(parts[1]);
  if (!a || !b || a.id === b.id) return undefined;
  return [a, b];
}

export function parsePairSlug(pair: string): [PricedModel, PricedModel] | undefined {
  const parts = pair.split('-vs-');
  for (let i = 1; i < parts.length; i++) {
    const a = resolveModel(parts.slice(0, i).join('-vs-'));
    const b = resolveModel(parts.slice(i).join('-vs-'));
    if (a && b && a.id !== b.id) return [a, b];
  }
  return undefined;
}

export function pairSlug(a: PricedModel, b: PricedModel): string {
  return `${a.slug}-vs-${b.slug}`;
}

export function ratesFor(m: PricedModel, w: Workload): Rates {
  if (m.longContext && w.inputTokens > m.longContext.threshold) {
    return { input: m.longContext.input, output: m.longContext.output, cacheRead: m.longContext.cacheRead ?? m.cacheRead, tier: 'long-context' };
  }
  if (m.schedule) {
    const p = Math.max(0, Math.min(1, w.peakShare ?? m.schedule.peakFraction));
    const mix = (peak: number, off: number) => peak * p + off * (1 - p);
    const cachePeak = m.schedule.peak.cacheRead;
    const cacheOff = m.schedule.offPeak.cacheRead;
    return {
      input: mix(m.schedule.peak.input, m.schedule.offPeak.input),
      output: mix(m.schedule.peak.output, m.schedule.offPeak.output),
      cacheRead: cachePeak !== undefined && cacheOff !== undefined ? mix(cachePeak, cacheOff) : m.cacheRead,
      tier: 'blended',
    };
  }
  return { input: m.input, output: m.output, cacheRead: m.cacheRead, tier: 'standard' };
}

export function costFor(m: PricedModel, w: Workload): Cost {
  const rates = ratesFor(m, w);
  const hit = rates.cacheRead !== undefined ? Math.max(0, Math.min(1, w.cacheHit)) : 0;
  const effInput = rates.input * (1 - hit) + (rates.cacheRead ?? rates.input) * hit;
  const inputCost = (w.requests * w.inputTokens * effInput) / 1_000_000;
  const outputCost = (w.requests * w.outputTokens * rates.output) / 1_000_000;
  const total = inputCost + outputCost;
  const perRequest = w.requests > 0 ? total / w.requests : 0;
  return { total, inputCost, outputCost, perRequest, per1kRequests: perRequest * 1000, rates };
}

export function blendedPrice(m: PricedModel): number {
  const r = ratesFor(m, { requests: 1, inputTokens: 1000, outputTokens: 250, cacheHit: 0 });
  return (r.input * 3 + r.output) / 4;
}

export interface ScenarioResult {
  scenario: Scenario;
  a: Cost;
  b: Cost;
  winner: 'a' | 'b' | 'tie';
  ratio: number;
}

export function compareScenarios(a: PricedModel, b: PricedModel): ScenarioResult[] {
  return SCENARIOS.map(scenario => {
    const ca = costFor(a, scenario.workload);
    const cb = costFor(b, scenario.workload);
    const winner = Math.abs(ca.total - cb.total) < 1e-9 ? 'tie' : ca.total < cb.total ? 'a' : 'b';
    const lo = Math.min(ca.total, cb.total);
    const hi = Math.max(ca.total, cb.total);
    return { scenario, a: ca, b: cb, winner, ratio: lo > 0 ? hi / lo : 1 };
  });
}

export function rankForScenario(workload: Workload, limit = 12): Array<{ model: PricedModel; cost: Cost }> {
  return PRICED_MODELS
    .map(model => ({ model, cost: costFor(model, workload) }))
    .filter(row => row.cost.total > 0)
    .sort((x, y) => x.cost.total - y.cost.total)
    .slice(0, limit);
}

export function fmtRate(value: number | undefined | null): string {
  if (value === undefined || value === null || !Number.isFinite(value)) return 'n/a';
  if (value === 0) return '$0';
  if (value < 0.1) return `$${value.toFixed(3).replace(/0+$/, '').replace(/\.$/, '')}`;
  return `$${value.toFixed(2)}`;
}

export function fmtTokens(n: number): string {
  if (n >= 1_000_000) return `${+(n / 1_000_000).toFixed(2)}M`;
  if (n >= 1_000) return `${+(n / 1_000).toFixed(1)}K`;
  return String(n);
}

export function fmtInt(n: number): string {
  return Math.round(n).toLocaleString('en-US');
}

export function priceLine(m: PricedModel): string {
  if (m.schedule) {
    return `${fmtRate(m.schedule.offPeak.input)}-${fmtRate(m.schedule.peak.input)} in / ${fmtRate(m.schedule.offPeak.output)}-${fmtRate(m.schedule.peak.output)} out per 1M tokens`;
  }
  return `${fmtRate(m.input)} in / ${fmtRate(m.output)} out per 1M tokens`;
}

export function tierNotes(m: PricedModel): string[] {
  const notes: string[] = [];
  if (m.longContext) {
    notes.push(`Prompts over ${fmtTokens(m.longContext.threshold)} tokens bill the whole request at ${fmtRate(m.longContext.input)} in / ${fmtRate(m.longContext.output)} out per 1M tokens.`);
  }
  if (m.schedule) {
    const windows = m.schedule.windowsUtc.map(([s, e]) => `${s}-${e}`).join(' and ');
    notes.push(`Peak pricing (${fmtRate(m.schedule.peak.input)} in / ${fmtRate(m.schedule.peak.output)} out) applies ${windows ? `${windows} UTC` : 'at peak hours'}${m.schedule.weekendOffPeak ? ' on weekdays' : ''}; off-peak is ${fmtRate(m.schedule.offPeak.input)} in / ${fmtRate(m.schedule.offPeak.output)} out${m.schedule.weekendOffPeak ? ', and all weekend is off-peak' : ''}.`);
  }
  if (m.cacheRead !== undefined) {
    notes.push(`Cached input tokens bill at ${fmtRate(m.cacheRead)} per 1M (${Math.round((1 - m.cacheRead / m.input) * 100)}% off the input rate).`);
  } else {
    notes.push('No cached-input discount is configured for this model, so cache hit rate does not change its cost here.');
  }
  if (m.provider === 'Anthropic') {
    notes.push('Anthropic prompt caching also charges a write premium (1.25x input for 5-minute, 2x for 1-hour caches) on the first write; the calculator excludes it.');
  }
  if (m.tokenizerNote) notes.push(m.tokenizerNote);
  return notes;
}

export function aaFor(m: PricedModel): ArtificialAnalysisModel | undefined {
  const tokens = new Set([m.id, m.name, ...m.aliases].map(normalizeModelToken).filter(Boolean));
  return artificialAnalysisModels.find(model =>
    [model.slug, model.name, model.shortName].map(normalizeModelToken).some(t => tokens.has(t)));
}

export function aaUrl(model: ArtificialAnalysisModel): string {
  return `https://artificialanalysis.ai${model.modelUrl}`;
}

export function aaIndexPerDollar(m: PricedModel): number | null {
  const aa = aaFor(m);
  const idx = aa?.evaluations.intelligenceIndex;
  const price = blendedPrice(m);
  if (typeof idx !== 'number' || price <= 0) return null;
  return idx / price;
}

export function neighbours(m: PricedModel, count = 3): { cheaper: PricedModel[]; pricier: PricedModel[] } {
  const sorted = [...PRICED_MODELS].filter(x => x.id !== m.id).sort((x, y) => blendedPrice(x) - blendedPrice(y));
  const price = blendedPrice(m);
  const cheaper = sorted.filter(x => blendedPrice(x) < price).slice(-count).reverse();
  const pricier = sorted.filter(x => blendedPrice(x) > price).slice(0, count);
  return { cheaper, pricier };
}

export const POPULAR_PAIRS: Array<[string, string]> = [
  ['claude-haiku-5-5', 'deepseek-v4-flash'],
  ['claude-haiku-5-5', 'claude-haiku-4-5-20251001'],
  ['claude-haiku-5-5', 'gemini-3.7-flash'],
  ['claude-haiku-5-5', 'gpt-5.6-luna'],
  ['claude-haiku-5-5', 'claude-sonnet-5-5'],
  ['deepseek-v4-flash', 'deepseek-v4-pro'],
  ['deepseek-v4-flash', 'gemini-3.7-flash'],
  ['deepseek-v4-flash', 'gpt-5.6-luna'],
  ['claude-sonnet-5-5', 'gpt-5.6-terra'],
  ['claude-opus-5-5', 'gpt-5.6-sol'],
  ['claude-sonnet-5-5', 'deepseek-v4-pro'],
  ['gemini-3.7-flash', 'gpt-5.6-luna'],
];

export function popularPairModels(): Array<[PricedModel, PricedModel]> {
  const out: Array<[PricedModel, PricedModel]> = [];
  for (const [x, y] of POPULAR_PAIRS) {
    const a = getPricedModel(modelSlug(x));
    const b = getPricedModel(modelSlug(y));
    if (a && b) out.push([a, b]);
  }
  return out;
}

export function pairVerdict(a: PricedModel, b: PricedModel): { headline: string; bullets: string[]; aWins: number; bWins: number } {
  const results = compareScenarios(a, b);
  const aWins = results.filter(r => r.winner === 'a').length;
  const bWins = results.filter(r => r.winner === 'b').length;
  const cheaper = aWins === bWins ? null : aWins > bWins ? a : b;
  const headline = cheaper
    ? `${cheaper.name} is cheaper in ${Math.max(aWins, bWins)} of ${results.length} workloads`
    : `${a.name} and ${b.name} split the workloads evenly`;
  const bullets: string[] = [];
  const best = (m: 'a' | 'b') => results.filter(r => r.winner === m).sort((x, y) => y.ratio - x.ratio)[0];
  const bestA = best('a');
  const bestB = best('b');
  if (bestA) bullets.push(`${a.name} wins biggest on ${bestA.scenario.headline}: ${formatRatio(bestA.ratio)} cheaper (${fmtMoney(bestA.a.total)} vs ${fmtMoney(bestA.b.total)} per month).`);
  if (bestB) bullets.push(`${b.name} wins biggest on ${bestB.scenario.headline}: ${formatRatio(bestB.ratio)} cheaper (${fmtMoney(bestB.b.total)} vs ${fmtMoney(bestB.a.total)} per month).`);
  const tierModel = [a, b].find(m => m.longContext);
  if (tierModel?.longContext) {
    bullets.push(`${tierModel.name} switches to ${fmtRate(tierModel.longContext.input)} in / ${fmtRate(tierModel.longContext.output)} out above ${fmtTokens(tierModel.longContext.threshold)}-token prompts, which reshuffles the ranking for long documents.`);
  }
  const schedModel = [a, b].find(m => m.schedule);
  if (schedModel?.schedule) {
    bullets.push(`${schedModel.name} is cheaper off-peak (${fmtRate(schedModel.schedule.offPeak.input)} in / ${fmtRate(schedModel.schedule.offPeak.output)} out) than at peak (${fmtRate(schedModel.schedule.peak.input)} in / ${fmtRate(schedModel.schedule.peak.output)} out); batch jobs can run in the cheap window.`);
  }
  return { headline, bullets, aWins, bWins };
}

export function formatRatio(r: number): string {
  if (r >= 10) return `${Math.round(r)}x`;
  return `${r.toFixed(1)}x`;
}

export function fmtMoney(value: number): string {
  if (value === 0) return '$0';
  if (value < 0.01) return `$${value.toFixed(4)}`;
  if (value < 1) return `$${value.toFixed(3)}`;
  if (value < 10_000) return `$${value.toFixed(2)}`;
  return `$${Math.round(value).toLocaleString('en-US')}`;
}

export function modelFaq(m: PricedModel): Array<{ q: string; a: string }> {
  const nb = neighbours(m, 1);
  const chat = costFor(m, SCENARIOS[0].workload);
  const faq = [
    { q: `How much does ${m.name} cost per million tokens?`, a: `${m.name} costs ${priceLine(m)}.${m.longContext ? ` ${tierNotes(m)[0]}` : ''}` },
    { q: `Is there a discount for cached tokens on ${m.name}?`, a: m.cacheRead !== undefined ? `Yes. Cached input is ${fmtRate(m.cacheRead)} per 1M tokens instead of ${fmtRate(m.input)}.` : `No cached-input rate is configured for ${m.name}, so every input token bills at the standard rate.` },
    { q: `What does a typical chatbot cost on ${m.name}?`, a: `A chatbot sending ${fmtInt(SCENARIOS[0].workload.requests)} requests per month (${fmtInt(SCENARIOS[0].workload.inputTokens)} input and ${fmtInt(SCENARIOS[0].workload.outputTokens)} output tokens each, ${Math.round(SCENARIOS[0].workload.cacheHit * 100)}% cached) costs about ${fmtMoney(chat.total)} per month on ${m.name}.` },
    { q: `What is the context window of ${m.name}?`, a: `${m.name} supports ${fmtTokens(m.context)} tokens of context${m.maxOutput ? ` and up to ${fmtTokens(m.maxOutput)} output tokens` : ''}.` },
  ];
  if (nb.cheaper[0]) faq.push({ q: `What is a cheaper alternative to ${m.name}?`, a: `${nb.cheaper[0].name} is cheaper at ${priceLine(nb.cheaper[0])}.` });
  return faq;
}

export function pairFaq(a: PricedModel, b: PricedModel): Array<{ q: string; a: string }> {
  const v = pairVerdict(a, b);
  const results = compareScenarios(a, b);
  const long = results.find(r => r.scenario.slug === 'long-document-analysis');
  return [
    { q: `${a.name} vs ${b.name}: which is cheaper?`, a: `${v.headline}. ${a.name}: ${priceLine(a)}. ${b.name}: ${priceLine(b)}.` },
    { q: `How do the price tiers differ between ${a.name} and ${b.name}?`, a: [...tierNotes(a).slice(0, 2).map(n => `${a.name}: ${n}`), ...tierNotes(b).slice(0, 2).map(n => `${b.name}: ${n}`)].join(' ') },
    ...(long ? [{ q: `Which is cheaper for long documents, ${a.name} or ${b.name}?`, a: `For 150K-token documents, ${a.name} costs ${fmtMoney(long.a.total)} and ${b.name} costs ${fmtMoney(long.b.total)} per ${fmtInt(long.scenario.workload.requests)} requests.` }] : []),
    { q: `Can I call both ${a.name} and ${b.name} with one API key?`, a: `Yes. OpenPaths exposes both through one OpenAI-compatible API with pay-as-you-go billing at the published per-token rates; set the model to ${a.id} or ${b.id}.` },
  ];
}
