import React from 'react';
import { Link } from 'react-router-dom';
import { ArrowRight, Coins, Database, Image as ImageIcon, Search, Sparkles, Video, Wand2 } from 'lucide-react';
import { Seo } from '../components/Seo';
import { models, type Model } from '../data/models';

type VideoPrice = {
  id: string;
  name: string;
  provider: string;
  pricePerVideo: number;
  notes: string;
};

const TEXT_MODEL_IDS = [
  'auto-easy-task',
  'gpt-4o-mini',
  'deepseek-chat',
  'minimax-m2.7',
  'gpt-5.4-mini',
  'gpt-5.5',
  'gemini-3.1-pro-preview',
  'claude-opus-4-7',
];

const EMBEDDING_MODEL_IDS = ['text-embedding', 'gemini-embedding-001', 'gemini-embedding-2-preview'];
const LOCAL_EMBEDDING_MODEL_IDS = ['openpaths-embed'];

const IMAGE_MODEL_IDS = [
  'stable-diffusion-3',
  'flux-schnell',
  'zimage',
  'glm-image',
  'flux-pro',
  'ra1',
];

const VIDEO_MODELS: VideoPrice[] = [
  {
    id: 'auto-video',
    name: 'Auto Video',
    provider: 'OpenPaths',
    pricePerVideo: 0.10,
    notes: 'Auto-routed starting point across supported video providers.',
  },
  {
    id: 'ltx-video',
    name: 'LTX Video',
    provider: 'Netwrck',
    pricePerVideo: 0.05,
    notes: 'Fast low-cost generation for lightweight clips.',
  },
  {
    id: 'hailuo-2.3',
    name: 'Hailuo 2.3',
    provider: 'MiniMax',
    pricePerVideo: 0.10,
    notes: 'Natural motion and strong quality-per-dollar.',
  },
  {
    id: 'wan',
    name: 'Wan Video',
    provider: 'Netwrck',
    pricePerVideo: 0.30,
    notes: 'High-quality text-to-video generation.',
  },
  {
    id: 'sora-2',
    name: 'Sora 2',
    provider: 'OpenAI',
    pricePerVideo: 0.80,
    notes: 'Premium video generation on OpenAI infrastructure.',
  },
  {
    id: 'ra2v',
    name: 'RA2V',
    provider: 'Netwrck',
    pricePerVideo: 1.00,
    notes: 'Scene-aware video generation with first-party economics.',
  },
];

const SEARCH_ROWS = [
  {
    id: 'exa-search-base',
    name: 'Exa Search',
    provider: 'Exa',
    unitPrice: 0.0077,
    unitLabel: '/ request',
    notes: 'Instant, Fast, Auto, or Deep search with 1-10 results. Includes the OpenPaths 10% markup.',
  },
  {
    id: 'exa-search-extra-result',
    name: 'Additional Search Result',
    provider: 'Exa',
    unitPrice: 0.0011,
    unitLabel: '/ result',
    notes: 'Applied to each requested result beyond the first 10.',
  },
  {
    id: 'exa-content-page',
    name: 'Content Extraction',
    provider: 'Exa',
    unitPrice: 0.0011,
    unitLabel: '/ page',
    notes: 'Applied when requesting highlights, full webpage text, or structured content extraction.',
  },
  {
    id: 'papers-search',
    name: 'Papers Search',
    provider: 'Papers',
    unitPrice: 0.001,
    unitLabel: '/ search',
    notes: 'Applied AI NZ papers.app.nz research search. Equivalent to $1 per 1,000 searches.',
  },
];

const PRICING_PAGE_TITLE = 'OpenPaths Pricing | Near-Zero Markup AI Model Routing';
const PRICING_PAGE_DESCRIPTION =
  'OpenPaths keeps AI pricing as close to zero markup as practical, makes money from first-party AI services, and supports transparent pay-as-you-go pricing across text, embeddings, image, and video models.';

export function Pricing() {
  const textModels = pickModels(TEXT_MODEL_IDS);
  const embeddingModels = pickModels(EMBEDDING_MODEL_IDS);
  const localEmbeddingModels = pickModels(LOCAL_EMBEDDING_MODEL_IDS);
  const imageModels = pickModels(IMAGE_MODEL_IDS);

  return (
    <>
      <Seo title={PRICING_PAGE_TITLE} description={PRICING_PAGE_DESCRIPTION} path="/pricing" />

      <div className="relative overflow-hidden">
        <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top,_rgba(255,255,255,0.08),_transparent_30%),radial-gradient(circle_at_20%_30%,_rgba(34,197,94,0.09),_transparent_28%),radial-gradient(circle_at_80%_20%,_rgba(14,165,233,0.08),_transparent_24%)]" />

        <section className="relative mx-auto max-w-7xl px-6 py-20 md:py-28">
          <div className="max-w-4xl">
            <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/[0.04] px-3 py-1.5 font-mono text-xs text-white/70">
              <Coins className="h-3.5 w-3.5" />
              Transparent pay-as-you-go pricing
            </div>
            <h1 className="max-w-4xl text-5xl font-bold tracking-tight md:text-7xl">
              Pricing built to stay as close to <span className="text-white/45">0 markup</span> as possible.
            </h1>
            <p className="mt-6 max-w-3xl text-lg leading-relaxed text-white/65 md:text-xl">
              OpenPaths is designed to be a routing layer, not a heavy-tax marketplace. We keep pass-through pricing as tight as practical and make a meaningful share of revenue when requests land on AI services we operate ourselves, including Netwrck image and video models plus our local embedding model and Text-Generator.io embeddings.
            </p>
            <div className="mt-10 flex flex-col gap-3 sm:flex-row">
              <Link
                to="/account"
                className="inline-flex items-center justify-center gap-2 rounded border border-white bg-white px-6 py-3 font-mono text-sm font-bold text-black transition-colors hover:bg-white/90"
              >
                Start Building <ArrowRight className="h-4 w-4" />
              </Link>
              <Link
                to="/models"
                className="inline-flex items-center justify-center gap-2 rounded border border-white/15 bg-black/40 px-6 py-3 font-mono text-sm text-white/80 transition-colors hover:bg-white/10 hover:text-white"
              >
                Browse Full Model Catalog
              </Link>
            </div>
          </div>

          <div className="mt-12 grid gap-4 md:grid-cols-3">
            <ValueCard
              title="No platform subscription"
              body="No seat fee, no monthly minimum, and no forced enterprise contract just to get the API running."
            />
            <ValueCard
              title="Minimal gateway spread"
              body="For third-party models we aim to stay as close as possible to underlying provider economics instead of stacking on large aggregator margins."
            />
            <ValueCard
              title="First-party revenue where it belongs"
              body="We can keep routing margins tight because we also operate some of the AI services in the catalog ourselves, including our local embedding model."
            />
          </div>
        </section>

        <section className="relative border-y border-white/10 bg-white/[0.02]">
          <div className="mx-auto grid max-w-7xl gap-px px-6 py-0 md:grid-cols-2 xl:grid-cols-4">
            <WorkloadCard
              icon={<Sparkles className="h-5 w-5" />}
              title="Text generation"
              body="OpenAI-compatible chat completions, tool use, structured outputs, and long-context models."
              pricing="Priced per 1M input and output tokens"
            />
            <WorkloadCard
              icon={<Database className="h-5 w-5" />}
              title="Embeddings"
              body="Fast local embeddings plus token-priced provider embeddings for search, retrieval, clustering, and reranking pipelines."
              pricing="Per request for the local model, per 1M tokens for provider models"
            />
            <WorkloadCard
              icon={<ImageIcon className="h-5 w-5" />}
              title="Image generation"
              body="Request-priced image models ranging from low-cost open weights to premium first-party generators."
              pricing="Priced per image request"
            />
            <WorkloadCard
              icon={<Video className="h-5 w-5" />}
              title="Video generation"
              body="Text-to-video and scene-aware video models with routing support across OpenPaths and partner providers."
              pricing="Priced per video request"
            />
            <WorkloadCard
              icon={<Search className="h-5 w-5" />}
              title="Web search"
              body="Exa-powered search for AI applications with highlights, content extraction, and freshness controls."
              pricing="Priced per search request and extracted page"
            />
          </div>
        </section>

        <section className="relative mx-auto max-w-7xl px-6 py-20">
          <div className="max-w-3xl">
            <div className="mb-3 font-mono text-xs uppercase tracking-[0.24em] text-white/45">How OpenPaths makes money</div>
            <h2 className="text-3xl font-bold tracking-tight md:text-4xl">Closer to provider pricing by design.</h2>
            <p className="mt-4 text-base leading-relaxed text-white/65 md:text-lg">
              Many AI gateways add a noticeable markup across every single model. We want the default experience to be cleaner than that. OpenPaths can keep pricing tight because some of the inventory is first-party: when you use services like Netwrck, our local embedding model, or Text-Generator.io through OpenPaths, we are not paying another gateway on top.
            </p>
          </div>

          <div className="mt-10 grid gap-4 lg:grid-cols-3">
            <DetailCard
              title="Third-party routing"
              body="For models from providers like OpenAI, Anthropic, Google, MiniMax, and others, our goal is transparent usage pricing with as little extra spread as practical."
            />
            <DetailCard
              title="First-party services"
              body="OpenPaths also includes models where we are the operator or have direct first-party economics. That is where a larger share of platform margin is expected to come from."
            />
            <DetailCard
              title="Bring your own key"
              body="If you need direct vendor billing for certain providers, OpenPaths also supports BYOK flows so you can keep routing features without forcing all spend through a reseller layer."
            />
          </div>
        </section>

        <section className="relative mx-auto max-w-7xl px-6 pb-8">
          <div className="mb-8 max-w-3xl">
            <div className="mb-3 font-mono text-xs uppercase tracking-[0.24em] text-white/45">Representative pricing</div>
            <h2 className="text-3xl font-bold tracking-tight md:text-4xl">Examples by workload type</h2>
            <p className="mt-4 text-base leading-relaxed text-white/65 md:text-lg">
              These tables show representative catalog pricing for common models on OpenPaths. Exact prices can shift as providers update their own rates, but the structure stays simple: token-priced text models, mixed request-priced and token-priced embeddings, and request-priced image and video models.
            </p>
          </div>

          <div className="space-y-10">
            <div>
              <SectionHeading
                icon={<Wand2 className="h-4 w-4" />}
                title="Text Generation and Reasoning"
                body="General chat, coding, tool use, and high-context reasoning models."
              />
              <TokenPricingTable models={textModels} />
            </div>

            <div>
              <SectionHeading
                icon={<Database className="h-4 w-4" />}
                title="Embeddings"
                body="OpenPaths includes a fast local embedding model plus token-priced Google and Text-Generator.io options."
              />
              <RequestPricingTable
                rows={localEmbeddingModels.map((model) => ({
                  id: model.id,
                  name: model.name,
                  provider: model.provider,
                  unitPrice: model.priceInput,
                  unitLabel: '/ request',
                  notes: model.description,
                }))}
              />
              <div className="mt-4" />
              <TokenPricingTable models={embeddingModels} />
              <div className="mt-4 grid gap-4 md:grid-cols-2">
                <DetailCard
                  title="OpenPaths Embed modes"
                  body="Use `openpaths-embed` for the fastest, lowest-cost local embeddings. Keep `long_text_mode=truncate` for speed, or switch to `average_chunks` when you want the full text represented across chunks."
                />
                <DetailCard
                  title="Gemini 2 preview pricing"
                  body="OpenPaths currently exposes the text path for `gemini-embedding-2-preview` at $0.20 / 1M text tokens. Google’s upstream multimodal rates are higher for image, audio, and video inputs."
                />
              </div>
            </div>

            <div>
              <SectionHeading
                icon={<ImageIcon className="h-4 w-4" />}
                title="Image Generation"
                body="Image APIs are billed per generation request instead of per token."
              />
              <RequestPricingTable
                rows={imageModels.map((model) => ({
                  id: model.id,
                  name: model.name,
                  provider: model.provider,
                  unitPrice: model.priceInput,
                  unitLabel: '/ image',
                  notes: model.description,
                }))}
              />
            </div>

            <div>
              <SectionHeading
                icon={<Video className="h-4 w-4" />}
                title="Video Generation"
                body="Video models are request-priced and typically reflect clip-level generation costs."
              />
              <RequestPricingTable
                rows={VIDEO_MODELS.map((model) => ({
                  id: model.id,
                  name: model.name,
                  provider: model.provider,
                  unitPrice: model.pricePerVideo,
                  unitLabel: '/ video',
                  notes: model.notes,
                }))}
              />
            </div>

            <div>
              <SectionHeading
                icon={<Search className="h-4 w-4" />}
                title="Search"
                body="Exa search is available through `/v1/search` and the `/search` playground. Prices include the requested 10% OpenPaths markup."
              />
              <RequestPricingTable rows={SEARCH_ROWS} />
            </div>
          </div>
        </section>

        <section className="relative mx-auto max-w-7xl px-6 py-20">
          <div className="rounded-[28px] border border-white/10 bg-[linear-gradient(135deg,rgba(255,255,255,0.06),rgba(255,255,255,0.02))] p-8 md:p-10">
            <div className="max-w-3xl">
              <div className="mb-3 font-mono text-xs uppercase tracking-[0.24em] text-white/45">Pricing FAQ</div>
              <h2 className="text-3xl font-bold tracking-tight md:text-4xl">Common questions</h2>
            </div>
            <div className="mt-8 grid gap-4 md:grid-cols-3">
              <FaqCard
                question="Is there a subscription fee?"
                answer="No. OpenPaths is usage-based. You pay for the requests and tokens you actually consume."
              />
              <FaqCard
                question="Do you add a markup?"
                answer="We aim to keep markup as close to zero as possible on routed provider traffic. Some margin comes from first-party services we operate ourselves."
              />
              <FaqCard
                question="Which model types are supported?"
                answer="Text generation, reasoning, coding, embeddings, image generation, video generation, and Exa web search are all supported in the current catalog."
              />
            </div>
          </div>
        </section>
      </div>
    </>
  );
}

function pickModels(ids: string[]) {
  const lookup = new Map(models.map((model) => [model.id, model]));
  return ids
    .map((id) => lookup.get(id))
    .filter((model): model is Model => Boolean(model));
}

function formatDollars(amount: number) {
  if (amount === 0) return '$0';
  if (amount < 0.01) return `$${amount.toFixed(3)}`;
  return `$${amount.toFixed(2)}`;
}

function ValueCard({ title, body }: { title: string; body: string }) {
  return (
    <div className="rounded-3xl border border-white/10 bg-white/[0.03] p-6">
      <h2 className="text-lg font-semibold tracking-tight">{title}</h2>
      <p className="mt-3 text-sm leading-relaxed text-white/62">{body}</p>
    </div>
  );
}

function WorkloadCard({
  icon,
  title,
  body,
  pricing,
}: {
  icon: React.ReactNode;
  title: string;
  body: string;
  pricing: string;
}) {
  return (
    <div className="border border-white/10 bg-black/40 p-6">
      <div className="mb-4 text-white/75">{icon}</div>
      <h3 className="text-lg font-semibold tracking-tight">{title}</h3>
      <p className="mt-3 text-sm leading-relaxed text-white/60">{body}</p>
      <div className="mt-4 font-mono text-[11px] uppercase tracking-[0.18em] text-white/40">{pricing}</div>
    </div>
  );
}

function DetailCard({ title, body }: { title: string; body: string }) {
  return (
    <div className="rounded-3xl border border-white/10 bg-black/45 p-6">
      <h3 className="text-lg font-semibold tracking-tight">{title}</h3>
      <p className="mt-3 text-sm leading-relaxed text-white/62">{body}</p>
    </div>
  );
}

function SectionHeading({
  icon,
  title,
  body,
}: {
  icon: React.ReactNode;
  title: string;
  body: string;
}) {
  return (
    <div className="mb-4 flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
      <div>
        <div className="mb-2 inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/[0.04] px-3 py-1 font-mono text-xs text-white/65">
          {icon}
          {title}
        </div>
        <p className="text-sm leading-relaxed text-white/60">{body}</p>
      </div>
    </div>
  );
}

function TokenPricingTable({ models }: { models: Model[] }) {
  return (
    <div className="overflow-hidden rounded-[28px] border border-white/10 bg-black/45">
      <div className="overflow-x-auto">
        <table className="min-w-full text-left">
          <thead className="border-b border-white/10 bg-white/[0.03] font-mono text-[11px] uppercase tracking-[0.18em] text-white/45">
            <tr>
              <th className="px-4 py-3">Model</th>
              <th className="px-4 py-3">Provider</th>
              <th className="px-4 py-3">Context</th>
              <th className="px-4 py-3">Input</th>
              <th className="px-4 py-3">Output</th>
            </tr>
          </thead>
          <tbody>
            {models.map((model) => (
              <tr key={model.id} className="border-b border-white/6 last:border-b-0">
                <td className="px-4 py-4">
                  <div className="font-semibold">{model.name}</div>
                  <div className="mt-1 text-sm text-white/50">{model.description}</div>
                </td>
                <td className="px-4 py-4 text-sm text-white/72">{model.provider}</td>
                <td className="px-4 py-4 font-mono text-sm text-white/62">{model.contextLength}</td>
                <td className="px-4 py-4 font-mono text-sm text-white">{formatDollars(model.priceInput)} / 1M</td>
                <td className="px-4 py-4 font-mono text-sm text-white">{formatDollars(model.priceOutput)} / 1M</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function RequestPricingTable({
  rows,
}: {
  rows: Array<{
    id: string;
    name: string;
    provider: string;
    unitPrice: number;
    unitLabel: string;
    notes: string;
  }>;
}) {
  return (
    <div className="overflow-hidden rounded-[28px] border border-white/10 bg-black/45">
      <div className="overflow-x-auto">
        <table className="min-w-full text-left">
          <thead className="border-b border-white/10 bg-white/[0.03] font-mono text-[11px] uppercase tracking-[0.18em] text-white/45">
            <tr>
              <th className="px-4 py-3">Model</th>
              <th className="px-4 py-3">Provider</th>
              <th className="px-4 py-3">Price</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr key={row.id} className="border-b border-white/6 last:border-b-0">
                <td className="px-4 py-4">
                  <div className="font-semibold">{row.name}</div>
                  <div className="mt-1 text-sm text-white/50">{row.notes}</div>
                </td>
                <td className="px-4 py-4 text-sm text-white/72">{row.provider}</td>
                <td className="px-4 py-4 font-mono text-sm text-white">
                  {formatDollars(row.unitPrice)} {row.unitLabel}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function FaqCard({ question, answer }: { question: string; answer: string }) {
  return (
    <div className="rounded-3xl border border-white/10 bg-black/45 p-6">
      <h3 className="text-lg font-semibold tracking-tight">{question}</h3>
      <p className="mt-3 text-sm leading-relaxed text-white/62">{answer}</p>
    </div>
  );
}
