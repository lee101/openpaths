#!/usr/bin/env node
import fs from 'node:fs/promises';
import path from 'node:path';
import process from 'node:process';

const BASE_URL = 'https://artificialanalysis.ai';
const ROOT = path.resolve(path.dirname(new URL(import.meta.url).pathname), '..');
const CACHE_DIR = path.join(ROOT, 'artificial-analysis');
const OUTPUT_PATH = path.join(ROOT, 'src/data/artificialAnalysis.ts');

const DEFAULT_MODEL_SLUGS = [
  'gpt-5-5',
  'claude-opus-4-7',
];

const args = new Set(process.argv.slice(2));
const refresh = args.has('--refresh');
const modelSlugs = process.argv
  .slice(2)
  .filter(arg => !arg.startsWith('--'))
  .flatMap(arg => arg.split(','))
  .map(arg => arg.trim())
  .filter(Boolean);

async function readCached(urlPath, fileName) {
  await fs.mkdir(CACHE_DIR, { recursive: true });
  const filePath = path.join(CACHE_DIR, fileName);
  if (!refresh) {
    try {
      return await fs.readFile(filePath, 'utf8');
    } catch {
      // Cache miss; fetch below.
    }
  }

  const url = new URL(urlPath, BASE_URL);
  const response = await fetch(url, {
    headers: {
      'user-agent': 'OpenPaths Artificial Analysis crawler (+https://openpaths.ai)',
      accept: 'text/html,application/xhtml+xml',
    },
  });
  if (!response.ok) {
    throw new Error(`Failed to fetch ${url}: ${response.status} ${response.statusText}`);
  }
  const html = await response.text();
  await fs.writeFile(filePath, html);
  return html;
}

function decodeNextFlight(html) {
  const chunks = [];
  const pattern = /self\.__next_f\.push\(\[(\d+),"([\s\S]*?)"\]\)<\/script>/g;
  for (const match of html.matchAll(pattern)) {
    try {
      chunks.push(JSON.parse(`"${match[2]}"`));
    } catch {
      // Keep going. A partial chunk is less useful than the rest of the page.
    }
  }
  return chunks.join('\n');
}

function extractBalancedJson(text, start) {
  const opener = text[start];
  const closer = opener === '{' ? '}' : ']';
  let depth = 0;
  let inString = false;
  let escaped = false;

  for (let index = start; index < text.length; index += 1) {
    const char = text[index];
    if (inString) {
      if (escaped) {
        escaped = false;
      } else if (char === '\\') {
        escaped = true;
      } else if (char === '"') {
        inString = false;
      }
      continue;
    }

    if (char === '"') {
      inString = true;
    } else if (char === opener) {
      depth += 1;
    } else if (char === closer) {
      depth -= 1;
      if (depth === 0) {
        return text.slice(start, index + 1);
      }
    }
  }
  throw new Error(`Could not find balanced JSON starting at ${start}`);
}

function extractAfterMarker(text, marker, opener) {
  const markerIndex = text.indexOf(marker);
  if (markerIndex < 0) return null;
  const start = text.indexOf(opener, markerIndex + marker.length);
  if (start < 0) return null;
  return JSON.parse(extractBalancedJson(text, start));
}

function normalizeModel(raw) {
  const creator = raw.model_creators || raw.creator || {};
  return {
    id: raw.id,
    slug: raw.slug,
    name: raw.name,
    shortName: raw.short_name || raw.shortName || raw.name,
    modelUrl: raw.model_url || raw.url || `/models/${raw.slug}`,
    hostsUrl: raw.hosts_url || raw.performanceDataSource?.providerUrl || null,
    creator: {
      id: creator.id || raw.model_creator_id || '',
      name: creator.name || raw.creatorName || '',
      slug: creator.slug || '',
      color: creator.color || '#888888',
      logoSmallUrl: creator.logo_small_url || creator.logo || null,
    },
    releaseDate: raw.release_date || raw.releaseDate || null,
    reasoning: Boolean(raw.reasoning_model || raw.isReasoning),
    deprecated: Boolean(raw.deprecated),
    frontier: Boolean(raw.frontier_model || raw.frontier || raw.chartHighlighted),
    openWeights: Boolean(raw.is_open_weights || raw.isOpenWeights),
    openSourceCategorization: raw.open_source_categorization || raw.openSourceCategorization || null,
    contextWindowTokens: raw.context_window_tokens || raw.contextWindowTokens || null,
    prices: {
      inputPerMTokens: numberOrNull(raw.price_1m_input_tokens ?? raw.price1mInputTokens),
      outputPerMTokens: numberOrNull(raw.price_1m_output_tokens ?? raw.price1mOutputTokens),
      cacheHitPerMTokens: numberOrNull(raw.cache_hit_price ?? raw.cacheHitPrice),
      blendedPerMTokens: numberOrNull(raw.price_1m_blended_7_2_1 ?? raw.price1mBlended7To2To1),
      blendedNoCachePerMTokens: numberOrNull(raw.price_1m_blended_0_1_1 ?? raw.price1mBlended0To1To1),
      cacheHitDiscountPercent: numberOrNull(raw.cache_hit_discount_percent ?? raw.cacheHitDiscountPercent),
    },
    performance: {
      outputTokensPerSecond: numberOrNull(raw.timescaleData?.median_output_speed ?? raw.timescaleData?.medianOutputSpeed),
      medianTotalTime: numberOrNull(raw.end_to_end_response_time_metrics?.total_time ?? raw.endToEndResponseTime?.total),
      timeToFirstAnswerToken: numberOrNull(raw.time_to_first_answer_token_metrics?.total_time ?? raw.timeToFirstAnswerToken?.total),
    },
    costs: {
      intelligenceIndexTotal: numberOrNull(raw.intelligence_index_cost?.total_cost),
      input: numberOrNull(raw.intelligence_index_cost?.input_cost ?? raw.intelligenceIndexCost?.input),
      output: numberOrNull(raw.intelligence_index_cost?.output_cost ?? raw.intelligenceIndexCost?.output),
      reasoning: numberOrNull(raw.intelligence_index_cost?.reasoning_cost ?? raw.intelligenceIndexCost?.reasoning),
      answer: numberOrNull(raw.intelligence_index_cost?.answer_cost ?? raw.intelligenceIndexCost?.answer),
    },
    tokens: {
      input: numberOrNull(raw.intelligence_index_token_counts?.input_tokens || raw.intelligence_index_token_counts?.input || raw.canonicalIntelligenceIndexTokenCount?.input),
      output: numberOrNull(raw.intelligence_index_token_counts?.output_tokens || raw.canonicalIntelligenceIndexTokenCount?.output),
      reasoning: numberOrNull(raw.intelligence_index_token_counts?.reasoning_tokens || raw.intelligence_index_token_counts?.reasoning || raw.canonicalIntelligenceIndexTokenCount?.reasoning),
      answer: numberOrNull(raw.intelligence_index_token_counts?.answer_tokens || raw.intelligence_index_token_counts?.answer || raw.canonicalIntelligenceIndexTokenCount?.answer),
      total: numberOrNull(raw.indexTokensTotal),
    },
    evaluations: {
      intelligenceIndex: numberOrNull(raw.intelligence_index ?? raw.intelligenceIndex),
      estimatedIntelligenceIndex: numberOrNull(raw.estimated_intelligence_index ?? (raw.intelligenceIndexIsEstimated ? raw.intelligenceIndex : null)),
      codingIndex: numberOrNull(raw.coding_index ?? raw.codingIndex),
      agenticIndex: numberOrNull(raw.agentic_index ?? raw.agenticIndex),
      gdpvalElo: numberOrNull(raw.safeGdpval?.elo || raw.gdpval),
      gdpvalNormalized: numberOrNull(raw.gdpval_normalized ?? raw.gdpvalNormalized),
      terminalBenchHard: numberOrNull(raw.terminalbench_hard ?? raw.terminalbenchHard),
      tau2: numberOrNull(raw.tau2),
      lcr: numberOrNull(raw.lcr),
      hle: numberOrNull(raw.hle),
      gpqa: numberOrNull(raw.gpqa),
      liveCodeBench: numberOrNull(raw.livecodebench ?? raw.liveCodeBench),
      sciCode: numberOrNull(raw.scicode ?? raw.sciCode),
      ifBench: numberOrNull(raw.ifbench ?? raw.ifBench),
      aime25: numberOrNull(raw.aime25),
      critPt: numberOrNull(raw.critpt ?? raw.critPt),
      mmmuPro: numberOrNull(raw.mmmu_pro ?? raw.mmmuPro),
      apexAgents: numberOrNull(raw.apex_agents),
      omniscience: numberOrNull(raw.omniscience),
      omniscienceAccuracy: numberOrNull(raw.omniscience_breakdown?.total?.accuracy ?? raw.omniscienceBreakdown?.accuracy),
      omniscienceHallucinationRate: numberOrNull(raw.omniscience_breakdown?.total?.hallucination_rate ?? raw.omniscienceBreakdown?.hallucinationRate),
    },
  };
}

function numberOrNull(value) {
  return typeof value === 'number' && Number.isFinite(value) ? value : null;
}

function mergeModels(models) {
  const bySlug = new Map();
  for (const model of models) {
    if (!model.slug) continue;
    bySlug.set(model.slug, { ...bySlug.get(model.slug), ...model });
  }
  return Array.from(bySlug.values()).sort((a, b) => {
    const aScore = a.evaluations.intelligenceIndex ?? -Infinity;
    const bScore = b.evaluations.intelligenceIndex ?? -Infinity;
    return bScore - aScore;
  });
}

function renderTypeScript(snapshot) {
  return `// Generated by crawlers/artificial_analysis.mjs. Do not edit by hand.\n` +
    `export const ARTIFICIAL_ANALYSIS_SNAPSHOT = ${JSON.stringify(snapshot, null, 2)} as const;\n`;
}

async function extractHomeModels() {
  const html = await readCached('/', 'home.html');
  const flight = decodeNextFlight(html);
  await fs.writeFile(path.join(CACHE_DIR, 'home.rsc.txt'), flight);

  const initial = extractAfterMarker(flight, '"initialData":', '{');
  if (!initial?.initialData?.length) {
    // Artificial Analysis occasionally changes the homepage leaderboard shape.
    // Model pages remain the stable source, so allow explicit slugs to refresh
    // even while the broad homepage extractor catches up with a schema change.
    console.warn('Home leaderboard schema changed; continuing with explicit model pages');
    return [];
  }
  return initial.initialData.map(normalizeModel);
}

async function extractModelPageModels(slug) {
  const html = await readCached(`/models/${slug}`, `model-${slug}.html`);
  const flight = decodeNextFlight(html);
  await fs.writeFile(path.join(CACHE_DIR, `model-${slug}.rsc.txt`), flight);
  const models = extractAfterMarker(flight, '"selectModelsByDefault":', '[');
  if (Array.isArray(models) && models.length) return models.map(normalizeModel);
  const pageDataStart = flight.indexOf('"currentModel":');
  if (pageDataStart < 0) return [];
  const pageData = flight.slice(pageDataStart);
  const currentModel = extractAfterMarker(pageData, '"currentModel":', '{');
  const comparisonModels = extractAfterMarker(pageData, '"models":', '[');
  return [currentModel, ...(Array.isArray(comparisonModels) ? comparisonModels : [])]
    .filter(Boolean)
    .map(normalizeModel);
}

async function main() {
  const slugs = modelSlugs.length > 0 ? modelSlugs : DEFAULT_MODEL_SLUGS;
  const gathered = [...await extractHomeModels()];

  for (const slug of slugs) {
    try {
      gathered.push(...await extractModelPageModels(slug));
    } catch (error) {
      console.warn(`Skipping ${slug}: ${error.message}`);
    }
  }

  // Keep the client snapshot focused and bundle-friendly. The comparison
  // payload now contains hundreds of historical variants; the top 80 by the
  // current Intelligence Index covers the active frontier without shipping a
  // multi-megabyte archive in every frontend bundle.
  const merged = mergeModels(gathered).slice(0, 80);
  const snapshot = {
    source: 'Artificial Analysis',
    sourceUrl: BASE_URL,
    crawledAt: new Date().toISOString(),
    modelCount: merged.length,
    models: merged,
  };

  await fs.writeFile(OUTPUT_PATH, renderTypeScript(snapshot));
  console.log(`Wrote ${snapshot.modelCount} Artificial Analysis models to ${path.relative(ROOT, OUTPUT_PATH)}`);
}

main().catch(error => {
  console.error(error);
  process.exit(1);
});
