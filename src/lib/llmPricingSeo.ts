import {
  PRICED_MODELS,
  PRICING_YEAR,
  SCENARIOS,
  blendedPrice,
  costFor,
  fmtInt,
  fmtMoney,
  fmtRate,
  fmtTokens,
  modelFaq,
  pairFaq,
  pairSlug,
  pairVerdict,
  popularPairModels,
  priceLine,
  rankForScenario,
  type PricedModel,
  type Scenario,
} from './llmPricing';

export interface PricingSeo {
  path: string;
  title: string;
  description: string;
  faq: Array<{ q: string; a: string }>;
  links: Array<{ href: string; label: string }>;
}

export function hubSeo(): PricingSeo {
  return {
    path: '/llm-pricing',
    title: `LLM API Pricing Calculator & Comparison (${PRICING_YEAR}) | OpenPaths`,
    description: `Compare ${PRICED_MODELS.length}+ LLM API prices side by side. Tier-aware cost calculator for Claude Haiku 5.5, DeepSeek, GPT, Gemini and more, with Artificial Analysis benchmarks.`,
    faq: [],
    links: [
      ...popularPairModels().map(([a, b]) => ({ href: `/llm-pricing/compare/${pairSlug(a, b)}`, label: `${a.name} vs ${b.name} pricing` })),
      ...SCENARIOS.map(s => ({ href: `/llm-pricing/cost/${s.slug}`, label: `Cheapest LLM for ${s.headline}` })),
    ],
  };
}

export function modelSeo(m: PricedModel): PricingSeo {
  return {
    path: `/llm-pricing/${m.slug}`,
    title: `${m.name} API Pricing (${PRICING_YEAR}): ${fmtRate(m.input)} in / ${fmtRate(m.output)} out per 1M tokens | OpenPaths`,
    description: `${m.name} pricing: ${priceLine(m)}.${m.longContext ? ` Prompts over ${fmtTokens(m.longContext.threshold)} tokens cost more.` : ''} Tier-aware calculator, cheaper alternatives and Artificial Analysis benchmarks.`,
    faq: modelFaq(m),
    links: [{ href: '/llm-pricing', label: 'LLM pricing calculator' }, { href: `/llm-pricing/provider/${m.providerSlug}`, label: `${m.provider} pricing` }],
  };
}

export function pairSeo(a: PricedModel, b: PricedModel): PricingSeo {
  const v = pairVerdict(a, b);
  return {
    path: `/llm-pricing/compare/${pairSlug(a, b)}`,
    title: `${a.name} vs ${b.name} Pricing (${PRICING_YEAR}): Tiers, Cost per 1M Tokens | OpenPaths`,
    description: `${v.headline}. ${a.name}: ${priceLine(a)}. ${b.name}: ${priceLine(b)}. Tier-aware cost for 10 workloads plus Artificial Analysis benchmarks.`,
    faq: pairFaq(a, b),
    links: [
      { href: '/llm-pricing', label: 'LLM pricing calculator' },
      { href: `/llm-pricing/${a.slug}`, label: `${a.name} pricing` },
      { href: `/llm-pricing/${b.slug}`, label: `${b.name} pricing` },
    ],
  };
}

export function providerSeo(slug: string, models: PricedModel[]): PricingSeo {
  const sorted = [...models].sort((a, b) => blendedPrice(a) - blendedPrice(b));
  const name = sorted[0].provider;
  return {
    path: `/llm-pricing/provider/${slug}`,
    title: `${name} API Pricing (${PRICING_YEAR}): All ${sorted.length} Models Compared | OpenPaths`,
    description: `${name} model prices per 1M tokens: ${sorted.slice(0, 3).map(m => `${m.name} ${fmtRate(m.input)}/${fmtRate(m.output)}`).join(', ')}. Compare tiers, cache discounts and benchmarks.`,
    faq: [],
    links: sorted.slice(0, 12).map(m => ({ href: `/llm-pricing/${m.slug}`, label: `${m.name} pricing` })),
  };
}

export function scenarioSeo(sc: Scenario): PricingSeo {
  const ranking = rankForScenario(sc.workload, 15);
  return {
    path: `/llm-pricing/cost/${sc.slug}`,
    title: `Cheapest LLM API for ${sc.name} (${PRICING_YEAR}): Cost Ranking | OpenPaths`,
    description: `Cheapest LLM for ${sc.headline}: ${ranking.slice(0, 3).map(r => `${r.model.name} ${fmtMoney(r.cost.total)}/mo`).join(', ')} for ${fmtInt(sc.workload.requests)} requests. Full ranking of ${PRICED_MODELS.length} models.`,
    faq: [],
    links: ranking.slice(0, 10).map(r => ({ href: `/llm-pricing/${r.model.slug}`, label: `${r.model.name} pricing` })),
  };
}

export function allPricingSeo(): PricingSeo[] {
  const out: PricingSeo[] = [hubSeo()];
  out.push(...PRICED_MODELS.map(modelSeo));
  const seen = new Set<string>();
  const addPair = (a: PricedModel, b: PricedModel) => {
    const key = pairSlug(a, b);
    if (seen.has(key)) return;
    seen.add(key);
    out.push(pairSeo(a, b));
  };
  popularPairModels().forEach(([a, b]) => addPair(a, b));
  const flagship = PRICED_MODELS.filter(m => ['claude-haiku-5-5', 'deepseek-v4-flash'].includes(m.id));
  for (const anchor of flagship) {
    const near = [...PRICED_MODELS]
      .filter(m => m.id !== anchor.id)
      .sort((x, y) => Math.abs(Math.log(blendedPrice(x) / blendedPrice(anchor))) - Math.abs(Math.log(blendedPrice(y) / blendedPrice(anchor))))
      .slice(0, 12);
    near.forEach(m => addPair(anchor, m));
  }
  const providers = new Map<string, PricedModel[]>();
  PRICED_MODELS.forEach(m => providers.set(m.providerSlug, [...(providers.get(m.providerSlug) || []), m]));
  providers.forEach((models, slug) => out.push(providerSeo(slug, models)));
  out.push(...SCENARIOS.map(scenarioSeo));
  return out;
}

export { costFor };
