import { Link } from 'react-router-dom';
import { ArrowRight, ExternalLink, KeyRound, Lock, ShieldCheck } from 'lucide-react';
import { Seo } from '../components/Seo';

const PROVIDERS: Array<{ name: string; id: string; keyUrl?: string; note?: string }> = [
  { name: 'OpenAI', id: 'openai', keyUrl: 'https://platform.openai.com/api-keys' },
  { name: 'Anthropic', id: 'anthropic', keyUrl: 'https://console.anthropic.com/settings/keys' },
  { name: 'Google AI Studio', id: 'google', keyUrl: 'https://aistudio.google.com/app/apikey' },
  { name: 'Mistral', id: 'mistral', keyUrl: 'https://console.mistral.ai' },
  { name: 'Groq', id: 'groq', keyUrl: 'https://console.groq.com/keys' },
  { name: 'xAI', id: 'xai', keyUrl: 'https://console.x.ai' },
  { name: 'DeepSeek', id: 'deepseek', keyUrl: 'https://platform.deepseek.com' },
  { name: 'Thinking Machines (Tinker)', id: 'thinkingmachines', keyUrl: 'https://tinker.thinkingmachines.dev' },
  { name: 'OpenRouter', id: 'openrouter', keyUrl: 'https://openrouter.ai/keys' },
  { name: 'Inference.net', id: 'inference_net', keyUrl: 'https://inference.net' },
  { name: 'Together AI', id: 'together', keyUrl: 'https://api.together.xyz' },
  { name: 'MiniMax', id: 'minimax', keyUrl: 'https://minimax.io' },
  { name: 'Netwrck', id: 'netwrck', keyUrl: 'https://netwrck.com' },
  { name: 'Z.ai (incl. GLM Coding Plan)', id: 'zai', keyUrl: 'https://z.ai' },
  { name: 'Sakana AI', id: 'sakana', keyUrl: 'https://sakana.ai' },
  { name: 'fal.ai', id: 'fal', keyUrl: 'https://fal.ai/dashboard/keys' },
  { name: 'Black Forest Labs', id: 'bfl', keyUrl: 'https://bfl.ai' },
  { name: 'OpenAI Codex (Max plan)', id: 'openai_codex', note: 'OAuth sign-in via Account' },
];

const FAQ = [
  {
    q: 'What is BYOK?',
    a: 'BYOK stands for bring-your-own-key. You add a provider API key to your OpenPaths account, and requests routed to that provider run on your key. You pay the provider directly.',
  },
  {
    q: 'Does BYOK cost anything on OpenPaths?',
    a: 'No. Requests served through your own key bypass the OpenPaths balance entirely and are recorded as $0. You only pay the provider for what you use on their side.',
  },
  {
    q: 'Which providers are supported?',
    a: '18 providers: OpenAI, Anthropic, Google, Mistral, Groq, xAI, DeepSeek, Thinking Machines (Tinker), OpenRouter, Inference.net, Together, MiniMax, Netwrck, Z.ai, Sakana, fal.ai, Black Forest Labs, and OpenAI Codex via Max plan OAuth.',
  },
  {
    q: 'Can I mix BYOK keys and OpenPaths credits?',
    a: 'Yes. Requests on your own key bypass the balance; everything else uses OpenPaths credits. If your key fails or is missing for a model, fallback chains still apply under one API.',
  },
  {
    q: 'Is my API key safe?',
    a: 'Keys are stored server-side. The API returns masked previews only, never full keys, and you can delete any key from Account at any time.',
  },
  {
    q: 'How do fallbacks work with BYOK?',
    a: 'If your key fails or does not cover a model, automatic fallback chains route the request to another configured provider. One API still covers fallbacks when your key fails.',
  },
];

const faqJsonLd = {
  '@context': 'https://schema.org',
  '@type': 'FAQPage',
  mainEntity: FAQ.map((item) => ({
    '@type': 'Question',
    name: item.q,
    acceptedAnswer: { '@type': 'Answer', text: item.a },
  })),
};

const STEPS = [
  {
    title: 'Add your provider key',
    body: 'Create a key at your provider console, then paste it under Account > Provider keys.',
  },
  {
    title: 'Call the same endpoints',
    body: 'OpenAI-compatible POST /v1/chat/completions, plus /v1/messages for Anthropic-native clients. No SDK changes.',
  },
  {
    title: 'Requests bill $0 on OpenPaths',
    body: 'Traffic served through your key skips the OpenPaths balance gate entirely and is recorded as $0. The provider bills you directly.',
  },
];

export function Byok() {
  return (
    <>
      <Seo
        title="Bring Your Own Key (BYOK) | OpenPaths"
        description="Use your own provider API keys across 18 providers. Requests on your key cost $0 on OpenPaths, with automatic fallbacks under one OpenAI-compatible API."
        path="/byok"
        jsonLd={faqJsonLd}
      />
      <div className="mx-auto max-w-6xl px-6 py-16">
        <section className="mb-14">
          <div className="mb-4 inline-flex items-center gap-2 font-mono text-xs uppercase tracking-[0.22em] text-cyan-300">
            <KeyRound className="h-4 w-4" />
            Bring your own key
          </div>
          <h1 className="max-w-3xl text-4xl font-semibold tracking-tight text-white md:text-5xl">
            Bring your own keys
          </h1>
          <p className="mt-5 max-w-2xl text-lg text-white/70">
            Pay your provider directly. OpenPaths charges $0 on requests served through your key.
            One API still covers fallbacks when your key fails.
          </p>
          <div className="mt-8 flex flex-wrap gap-3">
            <Link
              to="/account"
              className="inline-flex items-center gap-2 rounded-md bg-cyan-400 px-4 py-2 text-sm font-medium text-black transition hover:bg-cyan-300"
            >
              Manage keys <ArrowRight className="h-4 w-4" />
            </Link>
            <Link
              to="/docs"
              className="inline-flex items-center gap-2 rounded-md border border-white/20 px-4 py-2 text-sm text-white/80 transition hover:border-white/40 hover:text-white"
            >
              Read the docs
            </Link>
            <Link
              to="/pricing"
              className="inline-flex items-center gap-2 rounded-md border border-white/20 px-4 py-2 text-sm text-white/80 transition hover:border-white/40 hover:text-white"
            >
              Pricing
            </Link>
          </div>
        </section>

        <section className="mb-14 grid gap-4 md:grid-cols-3">
          {STEPS.map((step, index) => (
            <div key={step.title} className="rounded-lg border border-white/10 bg-white/[0.04] p-6">
              <div className="mb-3 font-mono text-xs uppercase tracking-[0.22em] text-white/50">
                Step {index + 1}
              </div>
              <h2 className="mb-2 text-lg font-medium text-white">{step.title}</h2>
              <p className="text-sm leading-relaxed text-white/60">{step.body}</p>
            </div>
          ))}
        </section>

        <section className="mb-14">
          <h2 className="mb-2 text-2xl font-semibold text-white">Supported providers</h2>
          <p className="mb-5 max-w-2xl text-sm text-white/60">
            All 18 providers accept a key you bring yourself. Get a key at each console below,
            then paste it into Account &gt; Provider keys.
          </p>
          <div className="overflow-x-auto rounded-lg border border-white/15 bg-white/[0.05]">
            <table className="w-full min-w-[520px] text-left text-sm">
              <thead>
                <tr className="border-b border-white/15 font-mono text-xs uppercase tracking-[0.18em] text-white/50">
                  <th className="px-4 py-3">Provider</th>
                  <th className="px-4 py-3">ID</th>
                  <th className="px-4 py-3">Where to get a key</th>
                </tr>
              </thead>
              <tbody>
                {PROVIDERS.map((provider) => (
                  <tr key={provider.id} className="border-b border-white/5 last:border-b-0">
                    <td className="px-4 py-3 font-medium text-white">{provider.name}</td>
                    <td className="px-4 py-3 font-mono text-xs text-white/50">{provider.id}</td>
                    <td className="px-4 py-3 text-white/70">
                      {provider.keyUrl ? (
                        <a
                          href={provider.keyUrl}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="inline-flex items-center gap-1.5 transition hover:text-cyan-300"
                        >
                          {provider.keyUrl.replace(/^https?:\/\//, '')}
                          <ExternalLink className="h-3.5 w-3.5" />
                        </a>
                      ) : (
                        <span>{provider.note}</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        <section className="mb-14 rounded-lg border border-white/15 bg-white/[0.05] p-8">
          <h2 className="text-xl font-semibold text-white">Anthropic-native endpoint</h2>
          <p className="mt-3 max-w-3xl text-sm leading-relaxed text-white/60">
            Claude Code and Anthropic SDK users can point{' '}
            <code className="rounded bg-black/40 px-1.5 py-0.5 font-mono text-xs text-cyan-300">
              ANTHROPIC_BASE_URL=https://openpaths.io
            </code>{' '}
            and use <code className="font-mono text-xs">POST /v1/messages</code> as-is. Authenticate with
            your Claude key through BYOK, or fall back to OpenPaths credits — same endpoint either way.
          </p>
        </section>

        <section className="mb-14 grid gap-4 md:grid-cols-3">
          <div className="rounded-lg border border-white/10 bg-white/[0.04] p-6">
            <ShieldCheck className="mb-3 h-5 w-5 text-cyan-300" />
            <h3 className="mb-2 font-medium text-white">Stored server-side</h3>
            <p className="text-sm leading-relaxed text-white/60">
              Keys live on OpenPaths servers so requests can be signed without exposing them to browsers.
            </p>
          </div>
          <div className="rounded-lg border border-white/10 bg-white/[0.04] p-6">
            <Lock className="mb-3 h-5 w-5 text-cyan-300" />
            <h3 className="mb-2 font-medium text-white">Masked previews only</h3>
            <p className="text-sm leading-relaxed text-white/60">
              The API returns masked previews of your keys, never the full value.
            </p>
          </div>
          <div className="rounded-lg border border-white/10 bg-white/[0.04] p-6">
            <KeyRound className="mb-3 h-5 w-5 text-cyan-300" />
            <h3 className="mb-2 font-medium text-white">Delete anytime</h3>
            <p className="text-sm leading-relaxed text-white/60">
              Remove any key from Account &gt; Provider keys whenever you want. No lock-in.
            </p>
          </div>
        </section>

        <section className="mb-16">
          <h2 className="mb-6 text-2xl font-semibold text-white">FAQ</h2>
          <div className="space-y-3">
            {FAQ.map((item) => (
              <details
                key={item.q}
                className="group rounded-lg border border-white/10 bg-white/[0.04] px-5 py-4"
              >
                <summary className="cursor-pointer list-none font-medium text-white marker:hidden">
                  {item.q}
                </summary>
                <p className="mt-3 text-sm leading-relaxed text-white/60">{item.a}</p>
              </details>
            ))}
          </div>
        </section>

        <section className="flex flex-col items-start gap-4 rounded-lg border border-cyan-300/25 bg-cyan-300/[0.06] p-8 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 className="text-xl font-semibold text-white">Ready to plug in your keys?</h2>
            <p className="mt-1 text-sm text-white/60">
              Add provider keys in seconds. Your balance stays untouched for everything else.
            </p>
          </div>
          <Link
            to="/account"
            className="inline-flex shrink-0 items-center gap-2 rounded-md bg-cyan-400 px-4 py-2 text-sm font-medium text-black transition hover:bg-cyan-300"
          >
            Manage provider keys <ArrowRight className="h-4 w-4" />
          </Link>
        </section>
      </div>
    </>
  );
}
