#!/usr/bin/env node
// Crawls the Artificial Analysis image-generation leaderboards (Text-to-Image
// Arena Elo + Image Editing Arena Elo) into src/data/artificialAnalysisImages.ts.
import fs from 'node:fs/promises';
import path from 'node:path';
import process from 'node:process';

const BASE_URL = 'https://artificialanalysis.ai';
const ROOT = path.resolve(path.dirname(new URL(import.meta.url).pathname), '..');
const CACHE_DIR = path.join(ROOT, 'artificial-analysis');
const OUTPUT_PATH = path.join(ROOT, 'src/data/artificialAnalysisImages.ts');

const LEADERBOARDS = [
  { key: 'textToImage', urlPath: '/image/leaderboard/text-to-image', file: 'image-text-to-image.html' },
  { key: 'editing', urlPath: '/image/leaderboard/editing', file: 'image-editing.html' },
];

const refresh = process.argv.slice(2).includes('--refresh');

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
      // A partial chunk is less useful than the rest of the page.
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
      if (escaped) escaped = false;
      else if (char === '\\') escaped = true;
      else if (char === '"') inString = false;
      continue;
    }
    if (char === '"') inString = true;
    else if (char === opener) depth += 1;
    else if (char === closer) {
      depth -= 1;
      if (depth === 0) return text.slice(start, index + 1);
    }
  }
  throw new Error(`Could not find balanced JSON starting at ${start}`);
}

function numberOrNull(value) {
  return typeof value === 'number' && Number.isFinite(value) ? value : null;
}

// The leaderboard rows live in a chunk shaped like `1b:[[null,[{formatted, values}...]]]`.
// Find every `<id>:[` chunk that contains `"formatted"` + `"elo"`, parse it, and
// collect the `values` payloads.
function extractRows(flight) {
  const rows = [];
  const seen = new Set();
  const collect = node => {
    if (Array.isArray(node)) {
      for (const item of node) collect(item);
    } else if (node && typeof node === 'object') {
      const v = node.values;
      if (v && typeof v.name === 'string' && typeof v.elo === 'number') {
        if (!seen.has(v.id)) {
          seen.add(v.id);
          rows.push(v);
        }
      }
      for (const key of Object.keys(node)) collect(node[key]);
    }
  };

  const chunkPattern = /(?:^|\n)[0-9a-f]+:(\[)/g;
  for (const match of flight.matchAll(chunkPattern)) {
    const start = match.index + match[0].length - 1;
    let json;
    try {
      json = JSON.parse(extractBalancedJson(flight, start));
    } catch {
      continue;
    }
    const slice = flight.slice(start, start + 200);
    if (!slice.includes('"formatted"') && !slice.includes('"elo"')) continue;
    collect(json);
  }
  return rows;
}

function normalizeRow(raw, rank) {
  const creator = raw.creator || {};
  return {
    id: raw.id,
    name: raw.name,
    rank,
    elo: numberOrNull(raw.elo),
    appearances: numberOrNull(raw.appearances),
    winRate: numberOrNull(raw.winRate),
    pricePer1kImages: numberOrNull(raw.pricePer1kImages),
    released: raw.released || null,
    openWeights: Boolean(raw.openWeightsUrl),
    modelUrl: raw.url || null,
    ciLower: numberOrNull(raw.ciLower),
    ciUpper: numberOrNull(raw.ciUpper),
    creator: {
      name: creator.name || '',
      logoUrl: creator.logoUrl || null,
      color: creator.color || '#888888',
    },
  };
}

function normalizeLeaderboard(rows) {
  return rows
    .filter(row => typeof row.elo === 'number')
    .sort((a, b) => b.elo - a.elo)
    .map((row, index) => normalizeRow(row, index + 1));
}

async function crawlLeaderboard({ urlPath, file }) {
  const html = await readCached(urlPath, file);
  const flight = decodeNextFlight(html);
  await fs.writeFile(path.join(CACHE_DIR, `${file}.rsc.txt`), flight);
  return normalizeLeaderboard(extractRows(flight));
}

function renderTypeScript(snapshot) {
  return `// Generated by crawlers/artificial_analysis_images.mjs. Do not edit by hand.\n` +
    `export const ARTIFICIAL_ANALYSIS_IMAGE_SNAPSHOT = ${JSON.stringify(snapshot, null, 2)} as const;\n`;
}

async function main() {
  const snapshot = {
    source: 'Artificial Analysis',
    sourceUrl: `${BASE_URL}/image/leaderboard/text-to-image`,
    crawledAt: new Date().toISOString(),
    textToImage: [],
    editing: [],
  };

  for (const board of LEADERBOARDS) {
    try {
      snapshot[board.key] = await crawlLeaderboard(board);
      console.log(`  ${board.key}: ${snapshot[board.key].length} models`);
    } catch (error) {
      console.warn(`Skipping ${board.key}: ${error.message}`);
    }
  }

  await fs.writeFile(OUTPUT_PATH, renderTypeScript(snapshot));
  console.log(`Wrote image leaderboards to ${path.relative(ROOT, OUTPUT_PATH)}`);
}

main().catch(error => {
  console.error(error);
  process.exit(1);
});
