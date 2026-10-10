import React from 'react';
import { Link } from 'react-router-dom';
import { ImageIcon } from 'lucide-react';
import { Seo } from '../components/Seo';
import { CodeBlock } from '../components/CodeBlock';
import { GEMINI_ASPECT_RATIOS, geminiImageModels, imageModels, imagePriceLabel } from '../data/imageCatalog';
import { faqLd, softwareApplicationLd, SITE_URL } from '../lib/seoSchema';
import { modelPath, providerPath } from '../lib/paths';

const PRIMARY = geminiImageModels.find(m => m.id === 'gemini-nano-banana-2.1') || geminiImageModels[0];
const PRIMARY_ID = PRIMARY?.id || 'gemini-nano-banana-2.1';
const BASE = `${SITE_URL}/v1`;

const PARAMS: Array<[string, string, string]> = [
  ['model', 'string', 'Required. A model ID from the table below, an alias such as nano-banana-2.1, or openpaths/auto-image.'],
  ['prompt', 'string', 'Required. What to generate or how to edit the reference images.'],
  ['n', 'integer', 'Number of images. Gemini image models run up to 4 in parallel and return every image that succeeds.'],
  ['size', 'string', 'WIDTHxHEIGHT such as 1024x1024. Gemini models map it to the closest supported aspect ratio.'],
  ['aspect_ratio', 'string', `Explicit aspect ratio for Gemini models, overriding size: ${GEMINI_ASPECT_RATIOS.join(', ')}.`],
  ['image_url / image_urls / reference_image_urls / images', 'string | string[]', 'Reference images for edits and multi-image composition. HTTPS URLs or data: URLs.'],
  ['response_format', 'string', 'url or b64_json. Gemini image models return b64_json.'],
];

const FAQ = [
  {
    q: 'How do I generate images with Gemini Nano Banana through OpenPaths?',
    a: `POST ${BASE}/images/generations with model set to ${PRIMARY_ID}, a prompt, and your OpenPaths API key as a bearer token. The endpoint is OpenAI-compatible, so client.images.generate works with the OpenAI SDK.`,
  },
  {
    q: 'Can I pass reference images to Gemini image models?',
    a: 'Yes. Send image_url, image_urls, reference_image_urls or images with HTTPS or data: URLs and describe the edit in the prompt. The reference images are sent to Gemini together with the prompt.',
  },
  {
    q: 'How does the size parameter work for Gemini image models?',
    a: `The requested WIDTHxHEIGHT is matched to the closest supported aspect ratio (${GEMINI_ASPECT_RATIOS.join(', ')}). Pass aspect_ratio to choose one explicitly. Output is 1K resolution.`,
  },
  {
    q: 'How much does image generation cost?',
    a: `${PRIMARY ? `${PRIMARY.name} costs ${imagePriceLabel(PRIMARY)}. ` : ''}Pricing is per image and listed for every model in the table on this page and on each model page.`,
  },
  {
    q: 'What happens if one of several images fails?',
    a: 'With n greater than 1, successful images are returned and the request only fails when every image fails. If the provider fails, OpenPaths falls back to the next configured image model.',
  },
];

const PY = `from openai import OpenAI
import base64

client = OpenAI(base_url="${BASE}", api_key="op-...")

result = client.images.generate(
    model="${PRIMARY_ID}",
    prompt="A watercolor fox reading a map in a misty forest",
    size="1024x1024",
    n=2,
)

for i, image in enumerate(result.data):
    with open(f"fox-{i}.png", "wb") as f:
        f.write(base64.b64decode(image.b64_json))`;

const CURL = `curl ${BASE}/images/generations \\
  -H "Authorization: Bearer $OPENPATHS_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "${PRIMARY_ID}",
    "prompt": "A watercolor fox reading a map in a misty forest",
    "aspect_ratio": "16:9",
    "n": 1
  }'`;

const EDIT = `curl ${BASE}/images/generations \\
  -H "Authorization: Bearer $OPENPATHS_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "${PRIMARY_ID}",
    "prompt": "Put the product from the first image on the table in the second image",
    "image_urls": [
      "https://example.com/product.png",
      "https://example.com/table.jpg"
    ]
  }'`;

const JS = `import OpenAI from "openai";

const client = new OpenAI({ baseURL: "${BASE}", apiKey: process.env.OPENPATHS_API_KEY });

const result = await client.images.generate({
  model: "${PRIMARY_ID}",
  prompt: "A watercolor fox reading a map in a misty forest",
  size: "1024x1024",
});

console.log(result.data[0].b64_json?.slice(0, 32));`;

const RESPONSE = `{
  "created": 1790000000,
  "data": [
    { "b64_json": "iVBORw0KGgo...", "revised_prompt": "" }
  ]
}`;

export function ImageGenerationDocs() {
  const title = 'Image Generation API - Gemini Nano Banana, GPT Image, Flux | OpenPaths';
  const description = `Generate and edit images through one OpenAI-compatible endpoint. Gemini Nano Banana 2.1 and 2 Lite, GPT Image 2, Flux and ${Math.max(imageModels.length - 4, 0)}+ more image models with per-image pricing, reference images and aspect ratios.`;
  const jsonLd = [
    {
      '@context': 'https://schema.org',
      '@type': 'TechArticle',
      headline: 'Image Generation API',
      description,
      url: `${SITE_URL}/docs/image-generation`,
      author: { '@type': 'Organization', name: 'OpenPaths' },
      publisher: { '@type': 'Organization', name: 'OpenPaths' },
    },
    softwareApplicationLd({ name: 'OpenPaths Image Generation API', description, path: '/docs/image-generation', price: PRIMARY?.priceInput }),
    faqLd(FAQ),
  ];

  return (
    <>
      <Seo title={title} description={description} path="/docs/image-generation" breadcrumbName="Image Generation" jsonLd={jsonLd} />
      <article className="mx-auto max-w-5xl px-6 py-16">
        <nav aria-label="Breadcrumb" className="mb-6 font-mono text-xs text-white/50">
          <Link to="/docs" className="hover:text-white">Docs</Link> / <span className="text-white/70">Image generation</span>
        </nav>
        <div className="mb-10">
          <div className="mb-5 inline-flex items-center gap-2 rounded-full border border-white/20 bg-white/[0.07] px-3 py-1 font-mono text-xs text-white/60">
            <ImageIcon className="h-3.5 w-3.5" /> POST /v1/images/generations
          </div>
          <h1 className="mb-4 text-4xl font-bold tracking-tight md:text-5xl">Image Generation API</h1>
          <p className="max-w-3xl text-lg leading-relaxed text-white/60">
            One OpenAI-compatible endpoint for text-to-image and image editing across Gemini Nano Banana, GPT Image, Flux and more. Try it in the{' '}
            <Link to="/text-to-image" className="text-white underline underline-offset-4">text to image tool</Link> or browse the{' '}
            <Link to="/models" className="text-white underline underline-offset-4">model directory</Link>.
          </p>
        </div>

        <section className="mb-12">
          <h2 className="mb-4 text-2xl font-bold tracking-tight">Quickstart</h2>
          <div className="space-y-4">
            <CodeBlock language="python" label="Python" code={PY} containerClassName="overflow-hidden rounded-lg border border-white/20 bg-black/40" headerClassName="bg-white/[0.06]" preClassName="text-xs" />
            <CodeBlock language="bash" label="cURL" code={CURL} containerClassName="overflow-hidden rounded-lg border border-white/20 bg-black/40" headerClassName="bg-white/[0.06]" preClassName="text-xs" />
            <CodeBlock language="javascript" label="JavaScript" code={JS} containerClassName="overflow-hidden rounded-lg border border-white/20 bg-black/40" headerClassName="bg-white/[0.06]" preClassName="text-xs" />
          </div>
        </section>

        <section className="mb-12">
          <h2 className="mb-4 text-2xl font-bold tracking-tight">Request parameters</h2>
          <div className="overflow-x-auto rounded-lg border border-white/20">
            <table className="w-full text-left text-sm">
              <thead className="bg-white/[0.06] font-mono text-xs uppercase tracking-wider text-white/55">
                <tr><th className="px-4 py-3">Parameter</th><th className="px-4 py-3">Type</th><th className="px-4 py-3">Description</th></tr>
              </thead>
              <tbody>
                {PARAMS.map(([name, type, desc]) => (
                  <tr key={name} className="border-t border-white/10 align-top">
                    <td className="px-4 py-3 font-mono text-white">{name}</td>
                    <td className="px-4 py-3 font-mono text-white/55">{type}</td>
                    <td className="px-4 py-3 text-white/65">{desc}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        <section className="mb-12">
          <h2 className="mb-4 text-2xl font-bold tracking-tight">Gemini Nano Banana image models</h2>
          <p className="mb-4 max-w-3xl text-white/60">
            Gemini image models ({geminiImageModels.map((m, i) => (
              <React.Fragment key={m.id}>{i > 0 && ', '}<Link to={modelPath(m.id)} className="text-white underline underline-offset-4">{m.name}</Link></React.Fragment>
            ))}) are served from{' '}
            <Link to={providerPath('google')} className="text-white underline underline-offset-4">Google</Link>. Pass up to 4 for n, choose an aspect ratio, and add reference images for edits and multi-image composition.
          </p>
          <h3 className="mb-3 text-lg font-semibold">Reference images</h3>
          <CodeBlock language="bash" label="cURL" code={EDIT} containerClassName="mb-6 overflow-hidden rounded-lg border border-white/20 bg-black/40" headerClassName="bg-white/[0.06]" preClassName="text-xs" />
          <h3 className="mb-3 text-lg font-semibold">Response</h3>
          <CodeBlock language="json" label="JSON" code={RESPONSE} containerClassName="overflow-hidden rounded-lg border border-white/20 bg-black/40" headerClassName="bg-white/[0.06]" preClassName="text-xs" />
        </section>

        <section className="mb-12">
          <h2 className="mb-4 text-2xl font-bold tracking-tight">Image models</h2>
          <div className="overflow-x-auto rounded-lg border border-white/20">
            <table className="w-full text-left text-sm">
              <thead className="bg-white/[0.06] font-mono text-xs uppercase tracking-wider text-white/55">
                <tr><th className="px-4 py-3">Model</th><th className="px-4 py-3">Model ID</th><th className="px-4 py-3">Provider</th><th className="px-4 py-3">Price</th><th className="px-4 py-3">Edits</th></tr>
              </thead>
              <tbody>
                {imageModels.map(m => (
                  <tr key={m.id} className="border-t border-white/10">
                    <td className="px-4 py-3"><Link to={modelPath(m.id)} className="text-white underline decoration-white/25 underline-offset-4 hover:decoration-white">{m.name}</Link></td>
                    <td className="px-4 py-3 font-mono text-xs text-white/60">{m.id}</td>
                    <td className="px-4 py-3 text-white/60">{m.provider}</td>
                    <td className="px-4 py-3 font-mono text-white/70">{m.pricingType === 'request' ? imagePriceLabel(m) : `$${m.priceInput}`}</td>
                    <td className="px-4 py-3 text-white/60">{m.tags.includes('image-to-image') ? 'Yes' : 'No'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        <section className="mb-4">
          <h2 className="mb-4 text-2xl font-bold tracking-tight">Frequently asked questions</h2>
          <div className="space-y-5">
            {FAQ.map(item => (
              <div key={item.q}>
                <h3 className="mb-1 text-lg font-semibold">{item.q}</h3>
                <p className="max-w-3xl text-white/60">{item.a}</p>
              </div>
            ))}
          </div>
        </section>
        <p className="mt-10 font-mono text-xs text-white/50">
          <Link to="/docs" className="hover:text-white">All API docs</Link> · <Link to="/pricing" className="hover:text-white">Pricing</Link> · <Link to="/image-edit" className="hover:text-white">Image edit tool</Link>
        </p>
      </article>
    </>
  );
}
