import { Link, useParams } from 'react-router-dom';
import {
  ArrowRight,
  ChevronRight,
  CircleAlert,
  Code2,
  HelpCircle,
  Route as RouteIcon,
  ShieldCheck,
} from 'lucide-react';
import { Seo } from '../components/Seo';
import { useCases } from '../data/useCases';
import type { UseCase } from '../data/useCases';

const BASE_URL = 'https://openpaths.io';

function faqJsonLd(useCase: UseCase) {
  return {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: useCase.faq.map(item => ({
      '@type': 'Question',
      name: item.question,
      acceptedAnswer: { '@type': 'Answer', text: item.answer },
    })),
  };
}

function Breadcrumbs({ title }: { title: string }) {
  return (
    <nav className="flex flex-wrap items-center gap-1 font-mono text-xs text-white/50">
      <Link to="/" className="hover:text-white">Home</Link>
      <ChevronRight className="h-3 w-3" />
      <Link to="/use-cases" className="hover:text-white">Use cases</Link>
      <ChevronRight className="h-3 w-3" />
      <span className="text-cyan-300">{title}</span>
    </nav>
  );
}

function RoutesTable({ useCase }: { useCase: UseCase }) {
  return (
    <section className="mt-10 overflow-hidden rounded-lg border border-white/20 bg-white/[0.05]">
      <div className="flex items-center gap-2 border-b border-white/20 px-4 py-3">
        <RouteIcon className="h-4 w-4 text-cyan-300" />
        <h2 className="font-mono text-sm uppercase tracking-[0.16em] text-white/65">Recommended routes</h2>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full min-w-[520px] text-left text-sm">
          <thead className="font-mono text-xs uppercase tracking-[0.14em] text-white/50">
            <tr>
              <th className="px-4 py-3">Need</th>
              <th className="px-4 py-3">Model</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-white/10">
            {useCase.routes.map(route => (
              <tr key={route.model}>
                <td className="px-4 py-3 text-white/70">{route.need}</td>
                <td className="px-4 py-3 font-mono text-white">{route.model}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="border-t border-white/10 px-4 py-3 text-xs text-white/50">
        All models are callable from one OpenPaths key via OpenAI-compatible{' '}
        <code className="font-mono text-white/70">POST /v1/chat/completions</code>.
      </div>
    </section>
  );
}

export function UseCasesIndex() {
  return (
    <>
      <Seo
        title="OpenPaths Use Cases | AI Gateway for Every Workload"
        description="How teams use the OpenPaths AI gateway for coding agents, customer support, content pipelines, data extraction, tutoring, and creative media."
        path="/use-cases"
        jsonLd={{
          '@context': 'https://schema.org',
          '@type': 'CollectionPage',
          name: 'OpenPaths use cases',
          url: `${BASE_URL}/use-cases`,
          hasPart: useCases.map(useCase => ({
            '@type': 'WebPage',
            name: useCase.title,
            url: `${BASE_URL}/use-cases/${useCase.slug}`,
          })),
        }}
      />
      <div className="mx-auto max-w-6xl px-6 py-16">
        <h1 className="text-4xl font-semibold tracking-tight md:text-5xl">AI gateway use cases</h1>
        <p className="mt-5 max-w-3xl text-lg leading-relaxed text-white/60">
          One API key reaches chat, images, video, audio, music, embeddings, and search across providers.
          Each guide below maps a real workload to concrete model routes on the OpenPaths gateway, with
          auto-routing, automatic provider fallbacks, BYOK at $0 recorded cost, and public latency stats at /stats.
        </p>
        <div className="mt-12 grid gap-5 md:grid-cols-2 lg:grid-cols-3">
          {useCases.map(useCase => (
            <Link
              key={useCase.slug}
              to={`/use-cases/${useCase.slug}`}
              className="group flex flex-col rounded-lg border border-white/20 bg-white/[0.05] p-6 transition-colors hover:border-cyan-300/60"
            >
              <h2 className="text-xl font-semibold text-white">{useCase.title}</h2>
              <p className="mt-3 flex-1 text-sm leading-relaxed text-white/60">{useCase.hero}</p>
              <span className="mt-5 inline-flex items-center gap-1 font-mono text-xs text-cyan-200 group-hover:text-white">
                Read the guide <ArrowRight className="h-3.5 w-3.5" />
              </span>
            </Link>
          ))}
        </div>
        <section className="mt-14 grid gap-5 md:grid-cols-3">
          <Link to="/evals" className="rounded-lg border border-white/20 p-5 transition-colors hover:border-cyan-300/60">
            <h3 className="font-semibold text-white">Model evals</h3>
            <p className="mt-2 text-sm text-white/60">Leaderboard from Artificial Analysis data plus our own benchmark runs.</p>
          </Link>
          <Link to="/byok" className="rounded-lg border border-white/20 p-5 transition-colors hover:border-cyan-300/60">
            <h3 className="font-semibold text-white">BYOK</h3>
            <p className="mt-2 text-sm text-white/60">Use your own provider keys. Requests bypass your balance and record $0.</p>
          </Link>
          <Link to="/docs" className="rounded-lg border border-white/20 p-5 transition-colors hover:border-cyan-300/60">
            <h3 className="font-semibold text-white">Docs</h3>
            <p className="mt-2 text-sm text-white/60">OpenAI-compatible endpoints, auto-routing strategies, and fallback setup.</p>
          </Link>
        </section>
      </div>
    </>
  );
}

export function UseCaseDetail() {
  const { slug } = useParams<{ slug: string }>();
  const useCase = useCases.find(entry => entry.slug === slug);

  if (!useCase) {
    return (
      <>
        <Seo
          title="Use case not found | OpenPaths"
          description="This use case page does not exist. Browse available OpenPaths AI gateway use cases."
          path="/use-cases"
        />
        <div className="mx-auto max-w-6xl px-6 py-16">
          <h1 className="text-3xl font-semibold tracking-tight">Use case not found</h1>
          <p className="mt-4 max-w-2xl text-white/60">
            No guide exists at /use-cases/{slug}. Browse all guides on the{' '}
            <Link to="/use-cases" className="font-mono text-cyan-200 hover:text-white">use cases index</Link>.
          </p>
        </div>
      </>
    );
  }

  return (
    <>
      <Seo
        title={useCase.metaTitle}
        description={useCase.metaDescription}
        path={`/use-cases/${useCase.slug}`}
        jsonLd={[
          faqJsonLd(useCase),
          {
            '@context': 'https://schema.org',
            '@type': 'WebPage',
            name: useCase.title,
            url: `${BASE_URL}/use-cases/${useCase.slug}`,
            description: useCase.metaDescription,
          },
        ]}
      />
      <div className="mx-auto max-w-6xl px-6 py-16">
        <Breadcrumbs title={useCase.title} />
        <h1 className="mt-6 max-w-4xl text-4xl font-semibold tracking-tight md:text-5xl">{useCase.title}</h1>
        <p className="mt-5 max-w-3xl text-lg leading-relaxed text-white/60">{useCase.hero}</p>

        <section className="mt-14">
          <h2 className="font-mono text-sm uppercase tracking-[0.16em] text-white/65">The problem</h2>
          <div className="mt-5 grid gap-5 md:grid-cols-3">
            {useCase.pains.map(pain => (
              <div key={pain} className="rounded-lg border border-white/20 bg-white/[0.05] p-5">
                <CircleAlert className="h-5 w-5 text-red-300" />
                <p className="mt-3 text-sm leading-relaxed text-white/70">{pain}</p>
              </div>
            ))}
          </div>
        </section>

        <section className="mt-14">
          <h2 className="font-mono text-sm uppercase tracking-[0.16em] text-white/65">Why OpenPaths</h2>
          <ul className="mt-5 grid gap-4 md:grid-cols-2">
            {useCase.why.map(reason => (
              <li key={reason} className="flex items-start gap-3 rounded-lg border border-white/20 bg-white/[0.05] p-5">
                <ShieldCheck className="mt-0.5 h-5 w-5 shrink-0 text-cyan-300" />
                <p className="text-sm leading-relaxed text-white/70">{reason}</p>
              </li>
            ))}
          </ul>
        </section>

        <RoutesTable useCase={useCase} />

        <section className="mt-10 overflow-hidden rounded-lg border border-white/20 bg-white/[0.05]">
          <div className="flex items-center gap-2 border-b border-white/20 px-4 py-3">
            <Code2 className="h-4 w-4 text-cyan-300" />
            <h2 className="font-mono text-sm uppercase tracking-[0.16em] text-white/65">Get started</h2>
          </div>
          <pre className="overflow-x-auto px-4 py-4 font-mono text-xs leading-relaxed text-white/80">{useCase.code}</pre>
        </section>

        <section className="mt-14">
          <h2 className="font-mono text-sm uppercase tracking-[0.16em] text-white/65">FAQ</h2>
          <div className="mt-5 divide-y divide-white/10 rounded-lg border border-white/20 bg-white/[0.05]">
            {useCase.faq.map(item => (
              <details key={item.question} className="group px-5 py-4">
                <summary className="flex cursor-pointer list-none items-center gap-3 text-white">
                  <HelpCircle className="h-4 w-4 shrink-0 text-cyan-300" />
                  <span className="font-medium">{item.question}</span>
                </summary>
                <p className="mt-3 pl-7 text-sm leading-relaxed text-white/60">{item.answer}</p>
              </details>
            ))}
          </div>
        </section>

        <section className="mt-14 grid gap-5 md:grid-cols-3">
          <Link to="/evals" className="rounded-lg border border-white/20 p-5 transition-colors hover:border-cyan-300/60">
            <h3 className="font-semibold text-white">Compare models on /evals</h3>
            <p className="mt-2 text-sm text-white/60">Benchmark snapshot before you pin routes.</p>
          </Link>
          <Link to="/byok" className="rounded-lg border border-white/20 p-5 transition-colors hover:border-cyan-300/60">
            <h3 className="font-semibold text-white">Run BYOK</h3>
            <p className="mt-2 text-sm text-white/60">Your provider keys, $0 against your OpenPaths balance.</p>
          </Link>
          <Link to="/docs" className="rounded-lg border border-white/20 p-5 transition-colors hover:border-cyan-300/60">
            <h3 className="font-semibold text-white">Read the docs</h3>
            <p className="mt-2 text-sm text-white/60">Endpoints, routing_strategy, and fallback chains.</p>
          </Link>
        </section>
      </div>
    </>
  );
}
