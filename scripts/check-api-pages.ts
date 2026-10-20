import assert from 'node:assert/strict';
import { CAPABILITIES, VS_PAIRS, FREE_MODEL_IDS, capabilityBySlug, capabilityForModel, capabilityModels, buildSnippets, vsBySlug } from '../src/data/capabilities';
import { models } from '../src/data/models';
import { apiPrerenderRoutes, apiSitemapPaths, vsVerdict } from '../src/lib/apiPagesSeo';
import { parseResult, requestBody } from '../src/components/CapabilityTryIt';

const unique = (xs: string[]) => new Set(xs).size === xs.length;

assert.ok(unique(CAPABILITIES.map(c => c.slug)), 'capability slugs unique');
assert.ok(unique(CAPABILITIES.map(c => c.path)), 'capability paths unique');
for (const c of CAPABILITIES) {
  assert.equal(c.path, `/api/${c.slug}`);
  assert.ok(c.faq.length >= 2, `${c.slug} faq`);
  for (const r of c.related) assert.ok(capabilityBySlug.has(r), `${c.slug} related ${r}`);
  const list = capabilityModels(c);
  assert.ok(list.length > 0 || c.staticRows?.length, `${c.slug} has models`);
  assert.ok(list.every(m => m.priceInput > 0 || m.priceOutput > 0 || FREE_MODEL_IDS.includes(m.id)), `${c.slug} no placeholder $0 rows`);
  const s = buildSnippets(c, c.defaultModel, 'hello');
  for (const lang of ['python', 'javascript', 'curl'] as const) assert.ok(s[lang].length > 20, `${c.slug} ${lang}`);
  if (c.kind !== 'studio') assert.ok(models.some(m => m.id === c.defaultModel), `${c.slug} default in catalog`);
}

assert.ok(VS_PAIRS.length >= 30, `pairs ${VS_PAIRS.length}`);
assert.ok(unique(VS_PAIRS.map(p => p.slug)), 'vs slugs unique');
for (const p of VS_PAIRS) {
  assert.equal(vsBySlug.get(p.slug), p);
  assert.ok(p.slug.split('-vs-').length === 2, `ambiguous vs slug ${p.slug}`);
  assert.notEqual(p.a.id, p.b.id);
  assert.ok(vsVerdict(p.a, p.b).length > 20);
}

const free = FREE_MODEL_IDS.map(id => models.find(m => m.id === id));
assert.ok(free.every(m => m && m.priceInput === 0 && m.priceOutput === 0), 'free models are $0 in site catalog');
assert.deepEqual(capabilityModels(capabilityBySlug.get('free-llm')!).map(m => m.id).sort(), [...FREE_MODEL_IDS].sort());

const routes = apiPrerenderRoutes();
assert.equal(routes.length, 1 + CAPABILITIES.length + VS_PAIRS.length);
assert.ok(unique(routes.map(r => r.path)), 'prerender paths unique');
for (const r of routes) {
  assert.ok(r.html.includes('<h1>'), `${r.path} h1`);
  assert.ok(r.html.includes('id="static-shell"'), `${r.path} shell id`);
  assert.ok(r.title.length <= 90, `${r.path} title length ${r.title.length}`);
  assert.ok(r.description.length >= 60 && r.description.length <= 300, `${r.path} description length ${r.description.length}`);
}
const sm = new Set(apiSitemapPaths().map(p => p.path));
for (const r of routes) assert.ok(sm.has(r.path), `sitemap has ${r.path}`);

const chat = capabilityBySlug.get('chat-completions')!;
assert.deepEqual(requestBody(chat, 'm', 'hi'), { model: 'm', messages: [{ role: 'user', content: 'hi' }], max_tokens: 512 });
assert.equal(parseResult(chat, { choices: [{ message: { content: 'ok' } }], usage: { prompt_tokens: 1, completion_tokens: 2 } }).kind, 'text');
assert.equal(parseResult(capabilityBySlug.get('embeddings')!, { data: [{ embedding: [1, 2, 3] }] }).kind, 'embedding');
assert.equal(parseResult(capabilityBySlug.get('image-generation')!, { data: [{ url: 'u' }] }).kind, 'image');
assert.throws(() => parseResult(capabilityBySlug.get('text-to-speech')!, {}));

assert.equal(capabilityForModel(models.find(m => m.id === 'openpaths-embed')!)?.slug, 'embeddings');
assert.equal(capabilityForModel(models.find(m => m.id === 'xai-tts')!)?.slug, 'text-to-speech');
assert.equal(capabilityForModel(models.find(m => m.id === 'zimage')!), undefined);

console.log(`api pages ok: ${CAPABILITIES.length} capabilities, ${VS_PAIRS.length} comparisons, ${routes.length} prerender routes`);
