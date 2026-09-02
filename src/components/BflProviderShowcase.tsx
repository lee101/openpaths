import React, { useMemo, useState } from 'react';
import { ArrowRight, Image as ImageIcon, Scissors, Video, WandSparkles } from 'lucide-react';
import { Link } from 'react-router-dom';
import type { Model } from '../data/models';
import { modelPath } from '../lib/paths';

const FLUX2_IDS = [
  'flux-2-max',
  'flux-2-pro-preview',
  'flux-2-flex',
  'flux-2-klein-4b',
  'flux-2-klein-9b-preview',
];
const TOOL_IDS = ['flux-tools/outpainting-v1', 'flux-tools/erase-v1', 'flux-tools/vto-v1'];
const FLUX1_IDS = ['flux-kontext-max', 'flux-kontext-pro', 'flux-pro-1.1-ultra', 'flux-pro-1.1', 'flux-pro-1.0-fill', 'bfl/flux-dev'];

const VIDEO_FEATURES = [
  'Text to video',
  'Image to video',
  'Video to video',
  'Multiple scenes',
  'Multilingual dialogue',
  'Native audio',
  'Reference images',
  'Ordered keyframes',
  'Video continuation',
  'Draft → FHD enhance',
  'Up to 20 seconds',
];

type VideoMode = 'text-image' | 'video';
type Resolution = 'HD' | 'FHD';

export function BflProviderShowcase({ models }: { models: Model[] }) {
  const [mode, setMode] = useState<VideoMode>('text-image');
  const [draft, setDraft] = useState(true);
  const [resolution, setResolution] = useState<Resolution>('HD');
  const [duration, setDuration] = useState(5);

  const rate = draft ? 0.06 : mode === 'text-image' ? (resolution === 'HD' ? 0.17 : 0.29) : (resolution === 'HD' ? 0.41 : 0.53);
  const total = rate * duration;
  const byId = useMemo(() => new Map(models.map(model => [model.id, model])), [models]);

  const chooseMode = (next: VideoMode) => {
    setMode(next);
    if (next === 'video') setDraft(false);
  };
  const chooseDraft = (next: boolean) => {
    setDraft(next);
    if (next) {
      setMode('text-image');
      setResolution('HD');
    }
  };

  return (
    <div className="space-y-14">
      <section id="flux-3-video" className="scroll-mt-24 rounded-2xl border border-white/12 bg-white/[0.05] overflow-hidden">
        <div className="border-b border-white/20 p-6 md:p-8">
          <div className="mb-3 flex items-center gap-2 font-mono text-xs uppercase tracking-[0.2em] text-white/55">
            <Video className="h-4 w-4" /> Video
          </div>
          <h2 className="text-3xl font-bold tracking-tight md:text-4xl">FLUX 3 Video</h2>
          <p className="mt-3 max-w-3xl text-sm font-light leading-relaxed text-white/60 md:text-base">
            One multimodal model for exploration and production. Generate from text, images, references, keyframes, or existing video, with synchronized audio included at no extra charge.
          </p>
          <div className="mt-5 flex flex-wrap gap-2">
            {VIDEO_FEATURES.map(feature => (
              <span key={feature} className="rounded-full border border-white/20 bg-black/30 px-3 py-1.5 text-xs text-white/60">{feature}</span>
            ))}
          </div>
        </div>

        <div className="grid gap-0 lg:grid-cols-[1.15fr_0.85fr]">
          <div className="overflow-x-auto p-6 md:p-8">
            <h3 className="mb-4 text-lg font-bold">Per-second pricing</h3>
            <table className="w-full min-w-[560px] text-left text-sm">
              <thead className="font-mono text-[10px] uppercase tracking-wider text-white/50">
                <tr className="border-b border-white/20">
                  <th className="pb-3 pr-4">Model</th>
                  <th className="pb-3 px-3">Text / image → video</th>
                  <th className="pb-3 px-3">Video → video</th>
                  <th className="pb-3 pl-3">Best for</th>
                </tr>
              </thead>
              <tbody>
                <tr className="border-b border-white/8 align-top">
                  <td className="py-4 pr-4 font-semibold">FLUX 3 Video Draft</td>
                  <td className="px-3 py-4"><strong>$0.06</strong> <span className="text-white/55">HD</span></td>
                  <td className="px-3 py-4 text-white/45">-</td>
                  <td className="py-4 pl-3 text-white/55">Rapid exploration</td>
                </tr>
                <tr className="align-top">
                  <td className="py-4 pr-4 font-semibold">FLUX 3 Video</td>
                  <td className="px-3 py-4"><strong>$0.17</strong> <span className="text-white/55">HD</span><br /><strong>$0.29</strong> <span className="text-white/55">FHD</span></td>
                  <td className="px-3 py-4"><strong>$0.41</strong> <span className="text-white/55">HD</span><br /><strong>$0.53</strong> <span className="text-white/55">FHD</span></td>
                  <td className="py-4 pl-3 text-white/55">Quality, cost and latency</td>
                </tr>
              </tbody>
            </table>
            <p className="mt-4 text-xs leading-relaxed text-white/50">
              Partial output seconds round up. HD is above 0.5 through 1.0MP per frame (for example 1280×704 or 960×960); FHD is above 1.0 through 2.0MP (for example 1920×1088 or 1440×1440). Draft is HD-only; enhancing a draft to FHD uses the regular FHD rate for the selected mode. Audio is included, and clips can be up to 20 seconds.
            </p>
          </div>

          <div className="border-t border-white/20 bg-black/20 p-6 md:p-8 lg:border-l lg:border-t-0">
            <div className="mb-5 font-mono text-xs uppercase tracking-[0.18em] text-white/55">Pricing calculator</div>
            <Control label="Video mode">
              <Toggle options={[['text-image', 'Text / image → video'], ['video', 'Video → video']]} value={mode} onChange={value => chooseMode(value as VideoMode)} />
            </Control>
            <Control label="Render">
              <Toggle options={[['draft', 'Draft'], ['standard', 'Full quality']]} value={draft ? 'draft' : 'standard'} onChange={value => chooseDraft(value === 'draft')} />
            </Control>
            <Control label="Resolution">
              <Toggle options={[['HD', 'HD'], ['FHD', 'FHD']]} value={resolution} onChange={value => setResolution(value as Resolution)} disabled={draft ? ['FHD'] : []} />
            </Control>
            <label className="block">
              <span className="mb-2 block font-mono text-[10px] uppercase tracking-wider text-white/55">Duration (seconds)</span>
              <input className="w-full accent-white" type="range" min={1} max={20} value={duration} onChange={event => setDuration(Number(event.target.value))} />
              <div className="mt-1 text-right font-mono text-sm">{duration}s</div>
            </label>
            <div className="mt-6 rounded-xl border border-white/20 bg-white/[0.07] p-5">
              <div className="flex justify-between text-sm text-white/50"><span>Rate</span><span>${rate.toFixed(2)} / second</span></div>
              <div className="mt-3 flex items-end justify-between border-t border-white/20 pt-4"><span className="font-mono text-xs uppercase tracking-wider text-white/55">Total</span><strong className="text-3xl">${total.toFixed(2)}</strong></div>
            </div>
            <div className="mt-4 flex flex-wrap gap-3 text-xs">
              <ModelLink model={byId.get('flux-3-video-draft')} />
              <ModelLink model={byId.get('flux-3-video')} />
            </div>
          </div>
        </div>
      </section>

      <FamilySection id="flux-2" eyebrow="Image generation + editing" title="FLUX.2" description="From sub-second interactive generation to maximum-quality grounded images, with exact color control and multi-reference editing. FLUX.2 Pro starts at $0.03/MP for generation and $0.045/MP for edits; prompt upsampling is enabled by default and can be switched off when the original wording must be preserved exactly." icon={<ImageIcon className="h-4 w-4" />} ids={FLUX2_IDS} byId={byId} />
      <FamilySection id="flux-tools" eyebrow="Purpose-built editing" title="FLUX Tools" description="Specialized one-call endpoints for extending scenes, removing objects, and placing garments while preserving the important details." icon={<Scissors className="h-4 w-4" />} ids={TOOL_IDS} byId={byId} />
      <FamilySection id="flux-1" eyebrow="Previous generation + open weights" title="FLUX.1" description="Kontext editing, high-resolution FLUX1.1 Pro generation, masked Fill, and the open-weight development model." icon={<WandSparkles className="h-4 w-4" />} ids={FLUX1_IDS} byId={byId} />
    </div>
  );
}

function Control({ label, children }: { label: string; children: React.ReactNode }) {
  return <div className="mb-5"><div className="mb-2 font-mono text-[10px] uppercase tracking-wider text-white/55">{label}</div>{children}</div>;
}

function Toggle({ options, value, onChange, disabled = [] }: { options: string[][]; value: string; onChange: (value: string) => void; disabled?: string[] }) {
  return (
    <div className="grid grid-cols-2 gap-2">
      {options.map(([key, label]) => (
        <button key={key} type="button" disabled={disabled.includes(key)} onClick={() => onChange(key)} className={`rounded border px-3 py-2 text-xs transition-colors ${value === key ? 'border-white bg-white text-black' : 'border-white/20 bg-white/[0.06] text-white/55 hover:border-white/45 hover:text-white'} disabled:cursor-not-allowed disabled:opacity-25`}>
          {label}
        </button>
      ))}
    </div>
  );
}

function FamilySection({ id, eyebrow, title, description, icon, ids, byId }: { id: string; eyebrow: string; title: string; description: string; icon: React.ReactNode; ids: string[]; byId: Map<string, Model> }) {
  const items = ids.map(modelId => byId.get(modelId)).filter((model): model is Model => Boolean(model));
  return (
    <section id={id} className="scroll-mt-24">
      <div className="mb-5">
        <div className="mb-2 flex items-center gap-2 font-mono text-xs uppercase tracking-[0.18em] text-white/55">{icon}{eyebrow}</div>
        <h2 className="text-3xl font-bold tracking-tight">{title}</h2>
        <p className="mt-2 max-w-3xl text-sm font-light leading-relaxed text-white/58">{description}</p>
      </div>
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {items.map(model => (
          <Link key={model.id} to={modelPath(model.id)} className="group rounded-xl border border-white/20 bg-white/[0.05] p-5 transition-colors hover:border-white/45 hover:bg-white/[0.07]">
            <div className="mb-3 flex items-start justify-between gap-3">
              <div><h3 className="font-bold tracking-tight">{model.name}</h3><code className="mt-1 block text-[10px] text-white/50">{model.id}</code></div>
              <span className="shrink-0 rounded bg-white/8 px-2 py-1 font-mono text-[10px] text-white/50">{catalogPrice(model)}</span>
            </div>
            <p className="line-clamp-3 text-sm font-light leading-relaxed text-white/55">{model.description}</p>
            <div className="mt-4 flex items-center gap-1 font-mono text-[10px] text-white/50 transition-colors group-hover:text-white/70">Model details <ArrowRight className="h-3 w-3" /></div>
          </Link>
        ))}
      </div>
    </section>
  );
}

function catalogPrice(model: Model) {
  if (model.priceInput === 0) return 'Local';
  const price = `$${model.priceInput < 0.1 ? model.priceInput.toFixed(3).replace(/0+$/, '').replace(/\.$/, '') : model.priceInput.toFixed(2)}`;
  if (model.pricingType === 'megapixel') return `${price}/MP`;
  return `${price}/image`;
}

function ModelLink({ model }: { model?: Model }) {
  if (!model) return null;
  return <Link to={modelPath(model.id)} className="inline-flex items-center gap-1 text-white/50 transition-colors hover:text-white">{model.name} <ArrowRight className="h-3 w-3" /></Link>;
}
