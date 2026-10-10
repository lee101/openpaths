import { describe, expect, test } from 'bun:test';
import {
  PRICED_MODELS,
  SCENARIOS,
  aaFor,
  compareScenarios,
  costFor,
  getPricedModel,
  pairSlug,
  parsePairQuery,
  parsePairSlug,
  ratesFor,
} from '../src/lib/llmPricing';
import { allPricingSeo } from '../src/lib/llmPricingSeo';

const haiku = getPricedModel('claude-haiku-5-5')!;
const flash = getPricedModel('deepseek-v4-flash')!;

describe('llm pricing', () => {
  test('free-text question resolves both models', () => {
    const pair = parsePairQuery('new anthropic haiku 5.5 vs deepseek flash pricing?');
    expect(pair?.map(m => m.id)).toEqual(['claude-haiku-5-5', 'deepseek-v4-flash']);
  });

  test('pair slug round-trips', () => {
    const parsed = parsePairSlug(pairSlug(haiku, flash));
    expect(parsed?.map(m => m.id)).toEqual(['claude-haiku-5-5', 'deepseek-v4-flash']);
  });

  test('haiku 5.5 base rates and long-context tier', () => {
    const short = ratesFor(haiku, { requests: 1, inputTokens: 100_000, outputTokens: 1, cacheHit: 0 });
    expect([short.input, short.output, short.tier]).toEqual([0.1, 0.5, 'standard']);
    const long = ratesFor(haiku, { requests: 1, inputTokens: 100_001, outputTokens: 1, cacheHit: 0 });
    expect([long.input, long.output, long.tier]).toEqual([0.5, 2.5, 'long-context']);
  });

  test('cost math', () => {
    const cost = costFor(haiku, { requests: 1_000_000, inputTokens: 1000, outputTokens: 100, cacheHit: 0 });
    expect(cost.inputCost).toBeCloseTo(100, 6);
    expect(cost.outputCost).toBeCloseTo(50, 6);
    const cached = costFor(haiku, { requests: 1_000_000, inputTokens: 1000, outputTokens: 0, cacheHit: 1 });
    expect(cached.inputCost).toBeCloseTo(10, 6);
  });

  test('deepseek flash blends peak and off-peak', () => {
    expect(flash.schedule).toBeDefined();
    const off = ratesFor(flash, { requests: 1, inputTokens: 1000, outputTokens: 1, cacheHit: 0, peakShare: 0 });
    expect(off.input).toBeCloseTo(0.15, 6);
    const peak = ratesFor(flash, { requests: 1, inputTokens: 1000, outputTokens: 1, cacheHit: 0, peakShare: 1 });
    expect(peak.input).toBeCloseTo(0.3, 6);
  });

  test('long documents flip the haiku vs flash winner', () => {
    const results = compareScenarios(haiku, flash);
    expect(results.find(r => r.scenario.slug === 'long-document-analysis')!.winner).toBe('b');
    expect(results.find(r => r.scenario.slug === 'chatbot')!.winner).toBe('a');
  });

  test('seo pages are unique and cover scenarios', () => {
    const pages = allPricingSeo();
    expect(new Set(pages.map(p => p.path)).size).toBe(pages.length);
    expect(pages.filter(p => p.path.startsWith('/llm-pricing/cost/')).length).toBe(SCENARIOS.length);
    expect(pages.length).toBeGreaterThan(PRICED_MODELS.length);
    for (const p of pages) expect(p.title.length).toBeLessThan(120);
  });

  test('artificial analysis exact match', () => {
    expect(aaFor(flash)?.creator.name).toBe('DeepSeek');
  });
});
