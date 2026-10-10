import React from 'react';
import { Link } from 'react-router-dom';
import { ArrowLeft } from 'lucide-react';
import { Seo } from '../components/Seo';
import { CodeBlock } from '../components/CodeBlock';
import { faqLd, softwareApplicationLd, SITE_URL } from '../lib/seoSchema';
import { modelPath } from '../lib/paths';

export interface ServerModel {
  id: string;
  name: string;
  provider: string;
  type: string;
  price: string;
  endpoint: string;
  summary: string;
  contextWindow: number;
  aliases?: string[] | null;
  sizes?: string[] | null;
}

export function readServerModel(id: string): ServerModel | null {
  if (typeof document === 'undefined') return null;
  try {
    const raw = document.getElementById('model-seo')?.textContent;
    if (!raw) return null;
    const data = JSON.parse(raw) as ServerModel;
    return data.id === id ? data : null;
  } catch {
    return null;
  }
}

function snippet(m: ServerModel) {
  const body = m.type === 'image'
    ? `{"model": "${m.id}", "prompt": "A watercolor fox reading a map"}`
    : m.type === 'language'
      ? `{"model": "${m.id}", "messages": [{"role": "user", "content": "Hello"}]}`
      : `{"model": "${m.id}"}`;
  return `curl ${SITE_URL}${m.endpoint} \\
  -H "Authorization: Bearer $OPENPATHS_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '${body}'`;
}

export function ServerModelPage({ model }: { model: ServerModel }) {
  const path = modelPath(model.id);
  const faq = [
    { q: `What is the model ID for ${model.name} on OpenPaths?`, a: `Use ${model.id} as the model field when calling ${SITE_URL}${model.endpoint}.` },
    ...(model.price ? [{ q: `How much does ${model.name} cost?`, a: `${model.price}. OpenPaths bills the provider price with near-zero markup.` }] : []),
  ];
  return (
    <>
      <Seo
        title={`${model.name} API, Pricing and Model ID | OpenPaths`}
        description={model.summary}
        path={path}
        breadcrumbName={model.name}
        jsonLd={[softwareApplicationLd({ name: model.name, description: model.summary, path }), faqLd(faq)]}
      />
      <section className="mx-auto max-w-5xl px-6 py-16">
        <Link to="/models" className="mb-8 inline-flex items-center gap-1.5 font-mono text-xs text-white/50 transition-colors hover:text-white">
          <ArrowLeft className="h-3.5 w-3.5" /> Model directory
        </Link>
        <h1 className="mb-4 text-4xl font-bold tracking-tight md:text-5xl">{model.name} API</h1>
        <p className="mb-8 max-w-3xl text-lg leading-relaxed text-white/60">{model.summary}</p>
        <dl className="mb-10 grid gap-4 md:grid-cols-3">
          {[['Model ID', model.id], ['Provider', model.provider], ['Pricing', model.price || 'See pricing'], ['Endpoint', `POST ${model.endpoint}`], ['Context', model.contextWindow ? `${model.contextWindow.toLocaleString()} tokens` : 'N/A']].map(([k, v]) => (
            <div key={k} className="rounded-xl border border-white/20 bg-white/[0.05] p-4">
              <dt className="mb-1 font-mono text-[10px] uppercase tracking-wider text-white/50">{k}</dt>
              <dd className="break-words font-mono text-sm text-white/85">{v}</dd>
            </div>
          ))}
        </dl>
        <CodeBlock language="bash" label="cURL" code={snippet(model)} containerClassName="mb-10 overflow-hidden rounded-lg border border-white/20 bg-black/40" headerClassName="bg-white/[0.06]" preClassName="text-xs" />
        <h2 className="mb-3 text-xl font-bold tracking-tight">Frequently asked questions</h2>
        {faq.map(item => (
          <div key={item.q} className="mb-4">
            <h3 className="font-semibold">{item.q}</h3>
            <p className="text-white/60">{item.a}</p>
          </div>
        ))}
        <p className="mt-8 font-mono text-xs text-white/50">
          <Link to="/docs" className="hover:text-white">API docs</Link> · <Link to="/pricing" className="hover:text-white">Pricing</Link>{model.type === 'image' && <> · <Link to="/docs/image-generation" className="hover:text-white">Image generation API</Link></>}
        </p>
      </section>
    </>
  );
}
