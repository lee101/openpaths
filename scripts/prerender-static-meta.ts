import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import {
  artificialAnalysisModels,
  findArtificialAnalysisModel,
} from '../src/lib/artificialAnalysis';
import { seedApps } from '../src/data/seedApps';
import { models as catalogModels } from '../src/data/models';
import { modelOgPath } from '../src/lib/modelOg';
import { modelSeoTitle } from '../src/lib/modelSeo';
import { appOgImage } from '../src/lib/appStats';
import { TOOLS, TOOLS_INDEX_SEO, TOOLS_INDEX_SLUG, toolOgImage } from '../src/data/tools';
import { posts } from '../src/data/blog';
import { providers } from '../src/data/providers';
import { useCases } from '../src/data/useCases';
import { geminiImageModels, imageModels } from '../src/data/imageCatalog';
import { modelJsonLd, withMeta, type StaticMeta } from './lib/staticMeta';
import { allPricingSeo } from '../src/lib/llmPricingSeo';
import { faqLd } from '../src/lib/seoSchema';

const DIST_DIR = process.env.DIST_DIR || 'dist';

const indexHtml = readFileSync(join(DIST_DIR, 'index.html'), 'utf8');

const imageDocsDescription = `Generate and edit images through one OpenAI-compatible endpoint. Gemini Nano Banana 2.1 and 2 Lite, GPT Image 2, Flux and ${Math.max(imageModels.length - 4, 0)}+ more image models with per-image pricing, reference images and aspect ratios.`;

const STATIC_PAGES: StaticMeta[] = [
  { path: '/pricing', title: 'OpenPaths Pricing | Near-Zero Markup AI Model Routing', description: 'OpenPaths keeps AI pricing as close to zero markup as practical, makes money from first-party AI services, and supports transparent pay-as-you-go pricing across text, embeddings, image, and video models.' },
  { path: '/models', title: 'AI Model Directory | OpenPaths', description: `Compare ${catalogModels.length}+ AI models by provider, price, context window, and capability across chat, image, video, audio, and embedding APIs.` },
  { path: '/providers', title: 'AI Providers | OpenPaths', description: `Browse ${providers.length} AI providers available through OpenPaths, including OpenAI, Anthropic, Google, xAI, Netwrck, Text-Generator.io, Fal, and more.` },
  { path: '/docs', title: 'OpenPaths API Docs | OpenAI-Compatible AI Gateway', description: 'OpenPaths API docs for chat completions, images, videos, speech, transcription, embeddings, provider routing, and OpenAI-compatible SDK setup.' },
  {
    path: '/docs/image-generation',
    title: 'Image Generation API - Gemini Nano Banana, GPT Image, Flux | OpenPaths',
    description: imageDocsDescription,
    links: geminiImageModels.map(m => ({ href: `/models/${encodeURIComponent(m.id)}`, label: m.name })),
  },
  { path: '/integrations', title: 'OpenPaths Integrations | OpenAI Agents, Anthropic, LangChain', description: 'Copy-paste OpenPaths integration examples for Claude Code, OpenAI Agents SDK, Anthropic Agent SDK, LangChain, Vercel AI SDK, PydanticAI, Mastra, Langfuse, LiveKit, Hermes Agent, and OpenClaw.' },
  { path: '/mcp', title: 'MCP Server - all models over Model Context Protocol | OpenPaths', description: 'Connect Claude Desktop, Claude Code, Cursor and any MCP client to every OpenPaths model with your secret key. High-performance Streamable HTTP MCP server.' },
  { path: '/byok', title: 'Bring Your Own Key (BYOK) | OpenPaths', description: 'Use your own provider API keys across 18 providers. Requests on your key cost $0 on OpenPaths, with automatic fallbacks under one OpenAI-compatible API.' },
  { path: '/teams', title: 'Teams and Spend Controls | OpenPaths', description: 'OpenPaths organizations: pooled management for AI spend with member invites, org-wide model rules, billing guards, BYOK support, and per-member usage visibility.' },
  { path: '/compound', title: 'Compound Model Designer | OpenPaths', description: 'Build a shareable OpenAI-compatible endpoint from auto models, fallback chains, circuit breakers, and fusion panels.' },
  { path: '/blog', title: 'OpenPaths Blog | AI Model Routing Guides', description: 'Engineering deep dives, AI model comparisons, integration guides, and production notes for building with OpenPaths.' },
  { path: '/calculator', title: 'LLM Cost Calculator | Estimate Monthly Token Spend | OpenPaths', description: 'Estimate monthly LLM spend from your input and output token volume. Compare published per-token prices side by side, see deltas between frontier and efficient models, and learn how BYOK changes the math.' },
  { path: '/status', title: 'System Status | OpenPaths', description: 'Live chat-completion probe results for models served through the OpenPaths API, plus links to real-traffic latency, TTFT, and throughput observability.' },
  { path: '/use-cases', title: 'OpenPaths Use Cases | AI Gateway for Every Workload', description: 'How teams use the OpenPaths AI gateway for coding agents, customer support, content pipelines, data extraction, tutoring, and creative media.' },
  { path: '/search', title: 'AI Search Playground - Gemini, OpenAI, Grok, Exa & Papers | OpenPaths', description: 'A Google-style search console powered by Gemini Flash with Google Search grounding, plus OpenAI web search, Grok live search, Exa neural search and Papers research search - one /v1/search API.' },
];

const routes: StaticMeta[] = [
  {
    path: TOOLS_INDEX_SEO.path,
    title: TOOLS_INDEX_SEO.seoTitle,
    description: TOOLS_INDEX_SEO.seoDescription,
    image: toolOgImage(TOOLS_INDEX_SLUG),
  },
  ...TOOLS.map(tool => ({
    path: tool.path,
    title: tool.seoTitle,
    description: tool.seoDescription,
    image: toolOgImage(tool.slug),
  })),
  {
    path: '/artifacts',
    title: 'Artifacts Gallery | OpenPaths',
    description: 'Build, host, and publish interactive AI artifacts. An IDE on the web where any OpenPaths model writes, edits, and ships self-contained web apps to a public gallery.',
  },
  {
    path: '/art',
    title: 'ZImage Prompt Search | OpenPaths',
    description: 'Browse and search a large ZImage generated-art prompt index, then try any prompt against OpenPaths image generation models.',
  },
  {
    path: '/evals',
    title: 'AI Model Evals, Pricing, and Speed | OpenPaths',
    description: 'Compare frontier AI model intelligence, coding, agentic performance, speed, and token pricing using the OpenPaths Artificial Analysis benchmark snapshot.',
  },
  {
    path: '/image-evals',
    title: 'Image Generator Evals - Arena Elo, Price & Example Images | OpenPaths',
    description: 'Artificial Analysis Text-to-Image and Image Editing Arena Elo leaderboards, plus a real example image from every generator OpenPaths hosts - RA1, GPT Image 2, FLUX, and more.',
  },
  {
    path: '/compare',
    title: 'Compare AI Models by Evals, Speed, and Price | OpenPaths',
    description: 'Compare frontier AI models head-to-head using Artificial Analysis evals, pricing, context windows, speed, and benchmark run costs.',
  },
  {
    path: '/alternatives',
    title: 'OpenRouter Alternative | 0% AI Gateway Markup | OpenPaths',
    description: 'Compare OpenPaths with OpenRouter, OpenAI, Anthropic, LiteLLM, and more. One API, pooled credits, smart routing, and 0% platform markup.',
  },
  {
    path: '/stats',
    title: 'OpenPaths Stats | AI Model Usage',
    description: 'Daily model usage and public OpenPaths request breakdowns.',
  },
  {
    path: '/works-with-openpaths',
    title: 'Works With OpenPaths | Compatible Apps & Agents',
    description: 'OpenPaths is OpenAI- and Anthropic-compatible, so it drops into every OpenRouter-style app. Browse coding agents, chat UIs, frameworks, and tools that work with an OpenPaths API key.',
  },
  {
    path: '/apps/',
    title: 'Apps | OpenPaths App Usage Stats',
    description: 'Opt-in app and agent usage stats across OpenPaths and OpenRouter, including model and token breakdowns.',
    image: appOgImage('openpaths-apps'),
  },
  {
    path: '/models/sync-lipsync-v3',
    title: 'sync-3 Avatar Lip Sync API & Pricing | OpenPaths',
    description: 'sync-3 image-to-video and video-to-video lip sync via OpenPaths: turn any portrait, illustration, or animated frame plus a voice track into a talking, perfectly lip-synced character. $0.20/sec.',
    image: 'https://openpaths.io/og/og-sync-lipsync.png',
  },
  ...seedApps.map(app => ({
    path: `/apps/${app.slug}/`,
    title: `${app.name} usage stats | OpenPaths Apps`,
    description: `${app.name} model and token usage on OpenPaths, including top models, total tokens, and opt-in attribution stats.`,
    image: appOgImage(app.slug),
  })),
  ...compareRoutes(),
  ...allPricingSeo().map(page => ({
    path: page.path,
    title: page.title,
    description: page.description,
    jsonLd: page.faq.length ? [faqLd(page.faq)] : undefined,
    links: page.links,
  })),
  ...catalogModels.map(model => ({
    path: `/models/${encodeURIComponent(model.id)}`,
    title: modelSeoTitle(model),
    description: clip(`${model.name} from ${model.provider}: ${model.description} Use model ID ${model.id} through the OpenPaths API.`, 160),
    image: model.ogImage || modelOgPath(model.id),
    jsonLd: modelJsonLd({ id: model.id, name: model.name, description: model.description, price: model.priceInput, pricingLabel: `Input price ${model.priceInput} USD (${model.pricingType || 'per 1M tokens'})` }),
    links: [{ href: '/models', label: 'All models' }],
  })),
  ...STATIC_PAGES,
  ...posts.map(post => ({
    path: `/blog/${encodeURIComponent(post.slug)}`,
    title: `${post.title} | OpenPaths Blog`,
    description: post.excerpt,
    type: 'article' as const,
    links: [{ href: '/blog', label: 'Blog' }],
  })),
  ...providers.map(provider => ({
    path: `/providers/${encodeURIComponent(provider.slug)}`,
    title: provider.kind === 'search' ? `${provider.name} Search API And Docs | OpenPaths` : `${provider.name} Models And API Docs | OpenPaths`,
    description: provider.description,
    links: [{ href: '/providers', label: 'All providers' }],
  })),
  ...useCases.map(useCase => ({
    path: `/use-cases/${useCase.slug}`,
    title: useCase.metaTitle,
    description: useCase.metaDescription,
    links: [{ href: '/use-cases', label: 'Use cases' }],
  })),
];

function clip(value: string, max: number) {
  const flat = value.replace(/\s+/g, ' ').trim();
  if (flat.length <= max) return flat;
  const cut = flat.slice(0, max - 1);
  return `${cut.slice(0, Math.max(cut.lastIndexOf(' '), max - 20))}…`;
}

const seen = new Set<string>();
let written = 0;
const manifest: Array<{ path: string; source: string; key: string }> = [];

for (const route of routes) {
  if (seen.has(route.path)) continue;
  seen.add(route.path);
  const html = withMeta(indexHtml, route);
  manifest.push(writeRouteFile(route.path, html));
  written += 1;
}

writeFileSync(join(DIST_DIR, 'prerender-manifest.json'), JSON.stringify(manifest, null, 2));
console.log(`Prerendered static metadata for ${written} routes`);

function compareRoutes(): StaticMeta[] {
  const models = artificialAnalysisModels.filter(model => typeof model.evaluations.intelligenceIndex === 'number');
  const anchor = models[0];
  if (!anchor) return [];

  return [
    compareMeta('/compare/gpt-5.5-vs-opus4.7'),
    ...models.slice(1, 12).map(model => compareMeta(`/compare/${anchor.slug}-vs-${model.slug}`)),
  ];
}

function compareMeta(path: string): StaticMeta {
  const names = path.replace(/^\/compare\//, '')
    .split('-vs-')
    .flatMap(token => {
      const name = findArtificialAnalysisModel(token)?.shortName;
      return name ? [String(name)] : [];
    });
  const subject = names.length >= 2 ? formatNameList(names) : 'AI models';

  return {
    path,
    title: `${subject} | AI Model Comparison | OpenPaths`,
    description: `Compare ${subject} using Artificial Analysis evals, speed, context, and token pricing.`,
  };
}

function writeRouteFile(path: string, html: string) {
  const key = decodeURIComponent(path).replace(/^\//, '') || 'index.html';
  const sourcePath = join(DIST_DIR, 'prerender-routes', `${key.replace(/[^a-z0-9.-]+/gi, '__')}.html`);
  mkdirSync(dirname(sourcePath), { recursive: true });
  writeFileSync(sourcePath, html);

  const indexPath = join(DIST_DIR, key, 'index.html');
  mkdirSync(dirname(indexPath), { recursive: true });
  writeFileSync(indexPath, html);

  return {
    path,
    source: sourcePath,
    key,
  };
}

function formatNameList(names: string[]) {
  if (names.length <= 2) return names.join(' and ');
  return `${names.slice(0, -1).join(', ')}, and ${names[names.length - 1]}`;
}
