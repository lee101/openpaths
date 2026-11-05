import React, { useState } from 'react';
import { Link } from 'react-router-dom';
import { ImageIcon, Zap, Sun } from 'lucide-react';
import { ToolSeo } from '../components/ToolSeo';
import { ImageSpacePanel } from '../components/ImageSpacePanel';
import { IMAGE_DEMOS, IMAGE_EXAMPLES } from '../data/imageDemos';
import { GPT_IMAGE_MODELS, GPT_IMAGE_PROMPTS, GPT_IMAGE_SECONDS, gptImageOutput, gptImageThumb } from '../data/gptImageShowcase';
import { modelPath } from '../lib/paths';

export function GptImage() {
  const [modelId, setModelId] = useState(GPT_IMAGE_MODELS[0].id);
  const [promptSlug, setPromptSlug] = useState('poster');
  const model = GPT_IMAGE_MODELS.find(m => m.id === modelId) || GPT_IMAGE_MODELS[0];
  const prompt = GPT_IMAGE_PROMPTS.find(p => p.slug === promptSlug) || GPT_IMAGE_PROMPTS[0];

  return (
    <>
      <ToolSeo slug="gpt-image" />
      <div className="px-4 py-12 sm:px-6 lg:px-10">
        <div className="mx-auto w-full max-w-7xl">
          <div className="mb-8 max-w-3xl">
            <div className="mb-4 inline-flex items-center gap-2 rounded border border-white/20 bg-white/[0.06] px-3 py-1 text-xs font-mono text-white/45"><ImageIcon className="h-3.5 w-3.5" /> OpenAI · new</div>
            <h1 className="text-4xl font-bold tracking-tight md:text-5xl">GPT Image 2.6</h1>
            <p className="mt-4 text-base leading-relaxed text-white/55">
              OpenAI's newest image models, Sunburst and Flare, in versions 2.6 and 2.5. Accurate in-image text, layouts and reference edits through one OpenAI-compatible endpoint. $0.06 per 1K image, $0.11 at 2K and 4K.
            </p>
          </div>

          <div className="mb-5 grid gap-3 sm:grid-cols-2 lg:grid-cols-4" role="tablist" aria-label="Model">
            {GPT_IMAGE_MODELS.map(m => (
              <button key={m.id} type="button" role="tab" aria-selected={m.id === modelId} onClick={() => setModelId(m.id)} data-testid="gpt-image-model-tab"
                className={`rounded-lg border px-4 py-3 text-left transition-colors ${m.id === modelId ? 'border-white/70 bg-white/[0.10]' : 'border-white/20 bg-white/[0.04] hover:border-white/45'}`}>
                <span className="flex items-center gap-2 font-mono text-sm font-bold">{m.tier === 'flare' ? <Zap className="h-3.5 w-3.5" /> : <Sun className="h-3.5 w-3.5" />}{m.name.replace('GPT Image ', '')}</span>
                <span className="mt-1 block font-mono text-[11px] text-white/45">{m.id}</span>
                <span className="mt-1 block text-xs text-white/55">{m.blurb}</span>
              </button>
            ))}
          </div>

          <section id="model-workspace" className="overflow-hidden rounded-xl border border-white/20 bg-white/[0.05]">
            <React.Fragment key={model.id}><ImageSpacePanel
              modelId={model.id}
              modelName={model.name}
              imageToImage
              demo={IMAGE_DEMOS[model.id]}
              examples={IMAGE_EXAMPLES[model.id]}
              priceLabel="$0.06 / image"
            /></React.Fragment>
          </section>

          <section className="mt-10" aria-labelledby="compare-h">
            <h2 id="compare-h" className="mb-1 text-2xl font-bold tracking-tight">Same prompt, four models</h2>
            <p className="mb-4 max-w-3xl text-sm text-white/55">Real outputs from each model at high quality. Pick a prompt to compare them side by side.</p>
            <div className="mb-4 flex flex-wrap gap-2">
              {GPT_IMAGE_PROMPTS.map(p => (
                <button key={p.slug} type="button" onClick={() => setPromptSlug(p.slug)} data-testid="gpt-image-prompt-tab"
                  className={`rounded px-3 py-1.5 font-mono text-xs ${p.slug === promptSlug ? 'bg-white text-black' : 'border border-white/20 text-white/55 hover:text-white'}`}>{p.title}</button>
              ))}
            </div>
            <p className="mb-4 max-w-4xl rounded border border-white/15 bg-black/40 px-3 py-2 font-mono text-xs leading-relaxed text-white/60">{prompt.prompt}</p>
            <div className={`grid grid-cols-2 gap-3 ${prompt.slug === 'infographic' ? '' : 'lg:grid-cols-4'}`} data-testid="gpt-image-compare">
              {GPT_IMAGE_MODELS.map(m => (
                <a key={m.id} href={gptImageOutput(m.id, prompt.slug)} target="_blank" rel="noreferrer" className="overflow-hidden rounded-lg border border-white/20 bg-black hover:border-white/50">
                  <img src={gptImageThumb(m.id, prompt.slug)} alt={`${m.name}: ${prompt.title}`} loading="lazy" className={`w-full bg-black object-cover ${prompt.slug === 'poster' ? 'aspect-[2/3]' : prompt.slug === 'infographic' ? 'aspect-[3/2]' : 'aspect-square'}`} />
                  <span className="block border-t border-white/15 px-3 py-2 font-mono text-[11px] text-white/70">{m.name.replace('GPT Image ', '')}<span className="block text-white/45">{prompt.size} · ~{GPT_IMAGE_SECONDS[`${m.id}/${prompt.slug}`]}s · $0.06</span></span>
                </a>
              ))}
            </div>
          </section>

          <section className="mt-10 overflow-x-auto rounded-lg border border-white/20">
            <table className="w-full text-left text-sm">
              <thead className="bg-white/[0.06] font-mono text-xs uppercase tracking-wider text-white/55">
                <tr><th className="px-4 py-3">Model ID</th><th className="px-4 py-3">Tier</th><th className="px-4 py-3">1K price</th><th className="px-4 py-3">2K / 4K</th><th className="px-4 py-3">Edit input</th></tr>
              </thead>
              <tbody>
                {GPT_IMAGE_MODELS.map(m => (
                  <tr key={m.id} className="border-t border-white/10">
                    <td className="px-4 py-3"><Link to={modelPath(m.id)} className="font-mono text-white underline decoration-white/25 underline-offset-4 hover:decoration-white">{m.id}</Link></td>
                    <td className="px-4 py-3 text-white/60">{m.tier === 'flare' ? 'Flare · fast' : 'Sunburst · detailed'}</td>
                    <td className="px-4 py-3 font-mono text-white/70">$0.06</td>
                    <td className="px-4 py-3 font-mono text-white/70">$0.11</td>
                    <td className="px-4 py-3 font-mono text-white/70">+$0.01 / image</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </section>

          <p className="mt-6 text-sm text-white/55">
            Parameters, edits and size mapping are in the <Link to="/docs/image-generation#gpt-image" className="text-white underline underline-offset-4">image generation API docs</Link>. Looking for the cheapest route? Use the <Link to="/text-to-image" className="text-white underline underline-offset-4">auto image tool</Link>.
          </p>
        </div>
      </div>
    </>
  );
}
