import React, { useEffect, useRef, useState } from 'react';
import { motion } from 'motion/react';
import { Zap, Code2, ArrowRight, Github, Search, Layers, Activity, Sparkles, ArrowUpRight, Video } from 'lucide-react';
import { Link } from 'react-router-dom';
import { CodeBlock } from '../components/CodeBlock';
import { artGallery } from '../data/artGallery';
import { videoGallery } from '../data/videoGallery';
import { models } from '../data/models';
import { getProviderLogo, providersByName } from '../data/providers';
import { Seo } from '../components/Seo';
import { ArtificialAnalysisBenchmarkSection } from '../components/ArtificialAnalysisCharts';

export function Landing() {
  const [activeTab, setActiveTab] = useState<'python' | 'curl'>('python');
  const apiKey = localStorage.getItem('op_api_key') || 'op_...';
  const galleryGrid = artGallery;
  const videoGrid = videoGallery;

  return (
    <>
      <Seo
        title="OpenPaths | Open Source AI Model Router"
        description="OpenPaths is an OpenAI-compatible model router for chat, image, video, speech, transcription, and embeddings across leading AI providers."
        path="/"
      />

      {/* Hero */}
      <section className="relative overflow-hidden border-b border-white/20">
        <HeroMeshCanvas />
        <div className="absolute inset-0 pointer-events-none bg-[radial-gradient(circle_at_center,_rgba(255,255,255,0.06),_transparent_32%),linear-gradient(180deg,_rgba(0,0,0,0.04),_rgba(0,0,0,0.22)_58%,_#000_96%)]" />
        <div className="relative px-6 py-28 md:py-40 min-h-[680px] max-w-7xl mx-auto flex flex-col items-center justify-center text-center">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5 }}
          className="relative z-10"
        >
          <h1 className="text-5xl md:text-7xl lg:text-8xl font-bold tracking-tighter mb-6 leading-[0.9]">
            The Open Source <br />
            <span className="text-white/55">Model Router.</span>
          </h1>
          <p className="text-lg md:text-xl text-white/60 max-w-2xl mx-auto mb-10 font-light leading-relaxed">
            <span className="text-white font-medium">You don't have to pick a model.</span> Try the <code className="text-white font-mono text-base">openpaths/auto</code> models, auto thinking, and the <code className="text-white font-mono text-base">-latest</code> series to help you stay on the frontier — often better than pinning one provider.
          </p>
          <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
            <Link to="/models" className="w-full sm:w-auto bg-white text-black px-8 py-4 font-mono font-bold flex items-center justify-center gap-2 hover:bg-white/90 transition-colors rounded">
              Explore Models <ArrowRight className="w-4 h-4" />
            </Link>
            <a href="https://codex-infinity.com/@lee101/openpaths" target="_blank" rel="noreferrer" className="w-full sm:w-auto border border-white/20 px-8 py-4 font-mono flex items-center justify-center gap-2 hover:bg-white/10 transition-colors rounded">
              <Github className="w-4 h-4" /> View Source
            </a>
          </div>
        </motion.div>
        </div>
      </section>

      {/* Stats / Marquee-ish */}
      <section className="border-y border-white/20 bg-white/[0.05] py-8 overflow-hidden">
        <div className="max-w-7xl mx-auto px-6 flex flex-wrap justify-center md:justify-between items-center gap-8 font-mono text-sm text-white/55">
          <div className="flex items-center gap-2"><Activity className="w-4 h-4" /> 99.99% Uptime</div>
          <div className="flex items-center gap-2"><Layers className="w-4 h-4" /> 100+ Models</div>
          <div className="flex items-center gap-2"><Zap className="w-4 h-4" /> &lt;50ms Latency</div>
          <div className="flex items-center gap-2"><Search className="w-4 h-4" /> Intelligent Fallbacks</div>
        </div>
      </section>

      {/* OpenPaths Auto */}
      <section id="auto" className="px-6 py-24 max-w-7xl mx-auto border-t border-white/20">
        <div className="mb-12 text-center max-w-3xl mx-auto">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border border-white/15 bg-white/[0.06] text-xs font-mono mb-6">
            <Sparkles className="w-3.5 h-3.5" />
            OpenPaths Auto
          </div>
          <h2 className="text-3xl md:text-5xl font-bold tracking-tight mb-4">Always on the frontier. Zero manual upgrades.</h2>
          <p className="text-white/60 font-mono text-sm leading-relaxed">
            OpenPaths Auto takes the hassle out of model upgrades — you stay on Claude latest, Gemini latest, and GPT latest automatically, without touching your code. Point everything at <code className="text-white">openpaths/auto</code>, or pick a variant when you want a bias.
          </p>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 font-mono text-sm">
          <AutoVariantCard modelId="openpaths/auto" title="Auto" purpose="Default — chat, agents, mixed workloads" backends="Gemini 3.7 Flash, GPT-5.6, GLM-5.3, Claude, DeepSeek" />
          <AutoVariantCard modelId="openpaths/auto-code" title="Auto Code" purpose="Coding agents, bug fixes, refactors" backends="GPT-5.6 Sol, Gemini 3.7 (3D/UI), GLM-5.3, Luna for easy diffs" />
          <AutoVariantCard modelId="openpaths/auto-fast" title="Auto Fast" purpose="Low-latency chat" backends="DeepSeek V4 Flash, Gemini 3.7 Flash" />
          <AutoVariantCard modelId="openpaths/auto-cheap" title="Auto Cheap" purpose="Lowest acceptable cost" backends="GPT-5.6 Luna, Gemini Flash Lite" />
          <AutoVariantCard modelId="openpaths/auto-reasoning" title="Auto Reasoning" purpose="Planning, math, hard problems" backends="Auto thinking depth + GPT-5.6, Gemini 3.7, GLM-5.3" />
          <AutoVariantCard modelId="openpaths/auto-vision" title="Auto Vision" purpose="Image understanding" backends="Gemini 3.7 Flash; Lite for thumbnails" />
          <AutoVariantCard modelId="openpaths/auto-image" title="Auto Image" purpose="Image generation" backends="GPT Image 2 → RA1 fallback" className="md:col-span-2 lg:col-span-1" />
        </div>
        <p className="mt-8 text-center text-white/55 font-mono text-xs">
          Take the hassle out of model upgrades — every variant tracks the frontier for you. Legacy IDs still work: <code className="text-white/60">auto</code>, <code className="text-white/60">auto-easy-task</code>, <code className="text-white/60">auto-think</code>, <code className="text-white/60">auto-image</code>
        </p>
      </section>

      {/* Frontier evidence */}
      <section id="frontier-evals" className="border-y border-white/20 bg-white/[0.025] px-6 py-20">
        <div className="mx-auto max-w-7xl">
          <div className="mb-8 grid gap-4 lg:grid-cols-[1.25fr_0.75fr] lg:items-end">
            <div>
              <div className="mb-4 inline-flex items-center gap-2 rounded-full border border-cyan-300/20 bg-cyan-300/[0.06] px-3 py-1 font-mono text-xs text-cyan-200">
                <Activity className="h-3.5 w-3.5" /> Frontier evidence · refreshed Aug 31, 2026
              </div>
              <h2 className="text-3xl font-bold tracking-tight md:text-5xl">Route from measurements, not model lore.</h2>
            </div>
            <p className="font-mono text-sm leading-relaxed text-white/55">
              External capability, speed, and price data sits beside our own task-level routing benchmark. The route can start cheap, then escalate only when the work earns it.
            </p>
          </div>
          <div className="grid gap-px overflow-hidden rounded-lg border border-white/15 bg-white/15 sm:grid-cols-3">
            <FrontierMetric value="100%" label="Verify + escalate pass rate" detail="27-task local coding suite" />
            <FrontierMetric value="$0.11" label="Cascade total" detail="vs. $2.55 pinned frontier" />
            <FrontierMetric value="30×" label="Lower cost per solved task" detail="Learning to Route result" />
          </div>
          <ArtificialAnalysisBenchmarkSection compact />
          <div className="mt-6 flex flex-wrap items-center justify-between gap-4 font-mono text-xs text-white/45">
            <span>External scores are independent snapshots; local cascade results include exact tasks, costs, and verifier outcomes.</span>
            <div className="flex gap-4">
              <Link to="/evals" className="text-white/70 hover:text-white">All evals <ArrowRight className="ml-1 inline h-3.5 w-3.5" /></Link>
              <Link to="/blog/learning-to-route-whitepaper" className="text-white/70 hover:text-white">Method <ArrowRight className="ml-1 inline h-3.5 w-3.5" /></Link>
            </div>
          </div>
        </div>
      </section>

      {/* Code Snippet */}
      <section id="api" className="px-6 py-24 max-w-5xl mx-auto">
        <div className="mb-12 text-center">
          <h2 className="text-3xl md:text-4xl font-bold tracking-tight mb-4">100% OpenAI Compatible</h2>
          <p className="text-white/60 font-mono text-sm">Just change the base URL and API key. That's it.</p>
        </div>
        
        <div className="border border-white/20 bg-black rounded-lg overflow-hidden shadow-2xl shadow-white/5">
          <div className="flex items-center justify-between px-4 py-3 border-b border-white/20 bg-white/[0.05]">
            <div className="flex items-center gap-2">
              <div className="w-3 h-3 rounded-full bg-white/20" />
              <div className="w-3 h-3 rounded-full bg-white/20" />
              <div className="w-3 h-3 rounded-full bg-white/20" />
            </div>
            <div className="flex gap-2 font-mono text-xs">
              <button 
                onClick={() => setActiveTab('python')}
                className={`px-3 py-1 rounded transition-colors ${activeTab === 'python' ? 'bg-white/10 text-white' : 'text-white/55 hover:text-white'}`}
              >
                Python
              </button>
              <button 
                onClick={() => setActiveTab('curl')}
                className={`px-3 py-1 rounded transition-colors ${activeTab === 'curl' ? 'bg-white/10 text-white' : 'text-white/55 hover:text-white'}`}
              >
                cURL
              </button>
            </div>
          </div>
          <div className="overflow-x-auto">
            {activeTab === 'python' ? (
              <CodeBlock
                language="python"
                preClassName="p-6"
                code={`import openai\n\nclient = openai.OpenAI(\n  base_url="https://openpaths.io/v1",\n  api_key="${apiKey}"\n)\n\nresponse = client.chat.completions.create(\n  model="openpaths/auto",\n  messages=[\n    {"role": "user", "content": "Ship the feature — you pick the model."}\n  ],\n  reasoning_effort="auto",  # none | low | medium | high | auto\n)`}
              />
            ) : (
              <CodeBlock
                language="bash"
                preClassName="p-6"
                code={`curl https://openpaths.io/v1/chat/completions \\\n  -H "Content-Type: application/json" \\\n  -H "Authorization: Bearer ${apiKey}" \\\n  -d '{\n    "model": "openpaths/auto-code",\n    "messages": [\n      {"role": "user", "content": "Fix this git diff and add a test."}\n    ]\n  }'`}
              />
            )}
          </div>
        </div>
        <div className="mt-6 grid grid-cols-1 md:grid-cols-2 gap-4 font-mono text-xs text-white/50">
          <div className="border border-white/20 rounded-lg p-4 bg-white/[0.05]">
            OpenAI-compatible params: <span className="text-white">model</span>, <span className="text-white">messages</span>, <span className="text-white">temperature</span>, <span className="text-white">top_p</span>, <span className="text-white">max_tokens</span>, <span className="text-white">stream</span>, <span className="text-white">tools</span>, <span className="text-white">response_format</span>, <span className="text-white">reasoning_effort</span>.
          </div>
          <div className="border border-white/20 rounded-lg p-4 bg-white/[0.05]">
            <span className="text-white">reasoning_effort</span> supports <span className="text-white">none</span>, <span className="text-white">low</span>, <span className="text-white">medium</span>, <span className="text-white">high</span>, and <span className="text-white">auto</span>.
          </div>
        </div>
      </section>

      {/* Art Playground */}
      <section id="art-playground" className="relative overflow-hidden border-y border-white/20 bg-[radial-gradient(circle_at_top_left,_rgba(245,158,11,0.14),_transparent_30%),radial-gradient(circle_at_bottom_right,_rgba(45,212,191,0.12),_transparent_34%),linear-gradient(180deg,_rgba(255,255,255,0.04),_rgba(255,255,255,0.02))] px-3 py-16 md:px-5 md:py-20 2xl:px-8">
          <div className="absolute inset-0 pointer-events-none bg-[linear-gradient(120deg,transparent_0%,rgba(255,255,255,0.04)_45%,transparent_65%)] opacity-60" />
          <div className="relative mx-auto max-w-[2200px]">
            <div className="grid gap-8">
              <div>
                <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
                  <div className="max-w-3xl">
                    <div className="mb-4 inline-flex items-center gap-2 rounded-full border border-white/15 bg-black/30 px-3 py-1 font-mono text-xs text-white/55">
                      <Sparkles className="h-3.5 w-3.5" /> Live Art Playground · AI Image Gallery
                    </div>
                    <h2 className="text-3xl md:text-5xl font-bold tracking-tight leading-[0.95] mb-4">
                      High-definition image, video, and 3D generation across frontier providers.
                    </h2>
                  </div>
                  <div className="flex flex-col sm:flex-row gap-3 lg:justify-end">
                    <Link
                      to="/models?q=art%20generation"
                      className="px-5 py-3 rounded border border-white/15 bg-white text-black font-mono text-sm font-bold hover:bg-white/90 transition-colors inline-flex items-center justify-center gap-2"
                    >
                      Browse Art Models <ArrowRight className="w-4 h-4" />
                    </Link>
                    <Link
                      to="/providers"
                      className="px-5 py-3 rounded border border-white/15 bg-black/30 font-mono text-sm hover:bg-white/10 transition-colors inline-flex items-center justify-center gap-2"
                    >
                      Explore Providers <ArrowUpRight className="w-4 h-4" />
                    </Link>
                  </div>
                </div>

                <div className="mt-8 grid grid-flow-dense gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5">
                  {galleryGrid.map((item) => (
                    <article
                      key={item.slug}
                      className={`group overflow-hidden rounded-[24px] border border-white/20 bg-black/45 ${item.layout === 'wide' ? 'sm:col-span-2' : ''}`}
                    >
                      <div className={`relative overflow-hidden ${item.layout === 'wide' ? 'aspect-[2/1]' : 'aspect-square'}`}>
                        <Link to={galleryModelPath(item.providerModelId, item.prompt)} className="absolute inset-0 z-0">
                          <img
                            src={item.imageUrl}
                            alt={item.title}
                            className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-[1.03]"
                            loading="lazy"
                          />
                        </Link>
                        <div className="absolute inset-0 bg-gradient-to-t from-black via-black/25 to-transparent" />
                        <div className="absolute top-3 left-3 right-3 flex items-center justify-between gap-3">
                          <div className="flex items-center gap-2 rounded-full border border-white/15 bg-black/55 px-2.5 py-1.5 backdrop-blur">
                            <img src={getProviderLogo(item.provider)} alt={`${item.provider} logo`} className="h-4 w-4 rounded-sm object-contain" />
                            <span className="font-mono text-[11px] text-white/80">{item.provider}</span>
                          </div>
                          <Link
                            to={providerDocsPath(item.provider)}
                            className="rounded-full border border-white/15 bg-black/55 p-2 text-white/70 hover:text-white hover:bg-black/70 transition-colors"
                            aria-label={`${item.provider} docs`}
                          >
                            <ArrowUpRight className="w-3.5 h-3.5" />
                          </Link>
                        </div>
                        <div className="absolute bottom-0 left-0 right-0 p-4">
                          <div className="text-xs uppercase tracking-[0.22em] text-white/50 mb-2">{item.model}</div>
                          <h3 className="relative z-10 text-xl font-bold tracking-tight">
                            <Link to={galleryModelPath(item.providerModelId, item.prompt)} className="hover:underline underline-offset-4">{item.title}</Link>
                          </h3>
                          <PromptText text={item.prompt} className="mt-2 text-sm text-white/65" />
                          <Link to={`/art?q=${encodeURIComponent(item.prompt)}`} className="relative z-10 mt-3 inline-flex text-[11px] font-mono uppercase tracking-[0.14em] text-white/45 hover:text-white">Find similar art <ArrowRight className="ml-1 h-3.5 w-3.5" /></Link>
                        </div>
                      </div>
                    </article>
                  ))}
                </div>

                <div className="mt-12">
                  <div className="mb-5 flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
                    <div>
                      <div className="mb-3 inline-flex items-center gap-2 rounded-full border border-white/15 bg-black/30 px-3 py-1 text-xs font-mono text-white/55">
                        <Video className="h-3.5 w-3.5" /> AI Video Gallery
                      </div>
                      <h3 className="text-2xl font-bold tracking-tight md:text-3xl">A wider, higher-definition range from K-Fold H3 and the newest video models.</h3>
                    </div>
                    <Link to="/models?q=video%20generation" className="inline-flex items-center gap-2 text-sm font-mono text-white/55 hover:text-white">
                      Browse video models <ArrowRight className="h-4 w-4" />
                    </Link>
                  </div>
                  <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
                    {videoGrid.map((item, index) => (
                      <article
                        key={item.slug}
                        className="overflow-hidden rounded-lg border border-white/20 bg-black/45"
                      >
                        <div className="relative aspect-video bg-black">
                          <Link to={galleryModelPath(item.model, item.prompt)} className="block h-full w-full">
                            <video src={item.videoUrl} poster={item.posterUrl} className="h-full w-full object-cover" muted loop playsInline controls preload={index < 4 ? 'metadata' : 'none'} />
                          </Link>
                          <div className="pointer-events-none absolute left-3 top-3 flex items-center gap-2 rounded-full border border-white/15 bg-black/60 px-2.5 py-1.5 backdrop-blur">
                            <img src={getProviderLogo(item.provider)} alt={`${item.provider} logo`} className={`h-4 w-4 rounded-sm object-contain ${item.provider === 'Black Forest Labs' ? 'bg-white p-px' : ''}`} />
                            <span className="font-mono text-[11px] text-white/80">{item.model}</span>
                          </div>
                        </div>
                        <div className="p-4">
                          <div className="mb-2 text-[11px] uppercase tracking-[0.22em] text-white/50">{item.resolution} · {item.duration}s · {item.format || 'WebM'}</div>
                          <h4 className="text-lg font-bold tracking-tight">
                            <Link to={galleryModelPath(item.model, item.prompt)} className="hover:underline underline-offset-4">{item.title}</Link>
                          </h4>
                          <PromptText text={item.prompt} className="mt-2 text-sm text-white/60" />
                          <Link to={`/art?q=${encodeURIComponent(item.prompt)}`} className="mt-3 inline-flex text-[11px] font-mono uppercase tracking-[0.14em] text-white/45 hover:text-white">Find similar art <ArrowRight className="ml-1 h-3.5 w-3.5" /></Link>
                        </div>
                      </article>
                    ))}
                  </div>
                </div>
              </div>

            </div>
          </div>
      </section>

      {/* Features */}
      <section id="features" className="px-6 py-24 max-w-7xl mx-auto border-t border-white/20">
        <div className="mb-16">
          <h2 className="text-3xl md:text-4xl font-bold tracking-tight mb-4">Built for scale.</h2>
          <p className="text-white/60 font-mono text-sm max-w-2xl">We handle the complexity of routing, fallbacks, and payments so you can focus on building your product.</p>
        </div>
        
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-px bg-white/10">
          <FeatureCard 
            icon={<Zap className="w-6 h-6" />}
            title="Millisecond Routing"
            description="Intelligent pathways find the lowest latency and highest throughput provider for your request instantly."
            to="/models"
          />
          <FeatureCard 
            icon={<Code2 className="w-6 h-6" />}
            title="Universal API"
            description="One API key for OpenAI, Anthropic, Meta, Mistral, and dozens of art generators like RA1 and Stable Diffusion."
            to="/docs"
          />
          <FeatureCard
            icon={<Sparkles className="w-6 h-6" />}
            title="OpenPaths Auto"
            description="Always on Claude latest, Gemini latest, and GPT latest — without re-picking models or editing code. One ID stays on the frontier: openpaths/auto, plus auto-code, auto-fast, auto-cheap, auto-reasoning, auto-vision, auto-image."
            to="/blog/how-auto-models-work"
          />
          <FeatureCard 
            icon={<svg className="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M14.5 3h-10l-2 5h10l2-5Z"/><path d="M11.5 11h-10l-2 5h10l2-5Z"/><path d="M14.5 19h-10l-2 5h10l2-5Z"/></svg>}
            title="Solana Payments"
            description="Prefer decentralization? Fund your API usage instantly with Solana (SOL) or USDC on the world's fastest blockchain."
            to="/pricing"
          />
        </div>
      </section>

      {/* CTA */}
      <section className="px-6 py-32 border-t border-white/20 text-center bg-white/[0.05]">
        <h2 className="text-4xl md:text-5xl font-bold tracking-tight mb-6">Ready to find your path?</h2>
        <p className="text-white/60 mb-10 max-w-xl mx-auto">Join thousands of developers building the future of AI with open, transparent, and fast model routing.</p>
        <Link to="/account" className="bg-white text-black px-8 py-4 font-mono font-bold hover:bg-white/90 transition-colors inline-block rounded">
          Create Free Account
        </Link>
      </section>
    </>
  );
}

function providerDocsPath(providerName: string): string {
  const provider = providersByName[providerName];
  return provider ? `/${provider.slug}/docs` : '/providers';
}

function galleryModelPath(modelId: string, prompt: string): string {
  const model = models.find(item => item.id === modelId);
  return model
    ? `/models/${encodeURIComponent(model.id)}?prompt=${encodeURIComponent(prompt)}`
    : `/models?q=${encodeURIComponent(modelId)}`;
}

function HeroMeshCanvas() {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    let frame = 0;
    let width = 0;
    let height = 0;
    let dpr = 1;

    const resize = () => {
      dpr = Math.min(window.devicePixelRatio || 1, 2);
      width = canvas.offsetWidth;
      height = canvas.offsetHeight;
      canvas.width = Math.floor(width * dpr);
      canvas.height = Math.floor(height * dpr);
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    };

    const draw = (time: number) => {
      const t = time * 0.00012;
      ctx.clearRect(0, 0, width, height);

      const centerX = width * 0.52;
      const horizonY = height * 0.43;
      const spacing = Math.max(42, Math.min(width, height) / 12);
      const halfColumns = Math.ceil(width / spacing) + 9;
      const rows = 24;

      ctx.save();
      ctx.translate(0, height * 0.02);
      ctx.rotate(-0.035);

      const drawPoint = (x: number, z: number) => {
        const depth = 1 / (1 + z * 0.075);
        const wave = Math.sin(x * 0.015 + z * 0.4 + t * 7) * 15;
        const drift = Math.cos(z * 0.24 + t * 5) * 26;
        return {
          x: centerX + (x + drift) * depth * 1.55,
          y: horizonY + z * spacing * depth * 0.62 + wave * depth,
          depth,
        };
      };

      const strokeMesh = (path: Array<{ x: number; y: number; depth: number }>, offset: number) => {
        if (path.length < 2) return;
        const shimmer = (Math.sin(t * 11 + offset) + 1) / 2;
        const alpha = 0.075 + shimmer * 0.12;

        ctx.beginPath();
        path.forEach((point, index) => {
          if (index === 0) ctx.moveTo(point.x, point.y);
          else ctx.lineTo(point.x, point.y);
        });
        ctx.strokeStyle = `rgba(255,255,255,${alpha})`;
        ctx.lineWidth = 0.9;
        ctx.stroke();
      };

      for (let row = 0; row < rows; row++) {
        const z = row + 0.6;
        const path = [];
        for (let col = -halfColumns; col <= halfColumns; col++) {
          path.push(drawPoint(col * spacing, z));
        }
        strokeMesh(path, row * 0.41);
      }

      for (let col = -halfColumns; col <= halfColumns; col++) {
        const x = col * spacing;
        const path = [];
        for (let row = 0; row < rows; row++) {
          path.push(drawPoint(x, row + 0.6));
        }
        strokeMesh(path, col * 0.28);
      }

      ctx.restore();

      const fade = ctx.createLinearGradient(0, 0, 0, height);
      fade.addColorStop(0, 'rgba(0,0,0,0.04)');
      fade.addColorStop(0.5, 'rgba(0,0,0,0)');
      fade.addColorStop(1, 'rgba(0,0,0,0.62)');
      ctx.fillStyle = fade;
      ctx.fillRect(0, 0, width, height);

      frame = requestAnimationFrame(draw);
    };

    resize();
    window.addEventListener('resize', resize);
    frame = requestAnimationFrame(draw);

    return () => {
      window.removeEventListener('resize', resize);
      cancelAnimationFrame(frame);
    };
  }, []);

  return (
    <canvas
      ref={canvasRef}
      className="absolute inset-x-0 top-0 h-full min-h-[760px] w-full opacity-95 [mask-image:linear-gradient(to_bottom,black_0%,black_78%,transparent_100%)]"
      aria-hidden="true"
    />
  );
}

function PromptText({ text, className = '' }: { text: string; className?: string }) {
  return (
    <p
      className={className}
      style={{
        display: '-webkit-box',
        WebkitBoxOrient: 'vertical',
        WebkitLineClamp: 3,
        overflow: 'hidden',
      }}
    >
      {text}
    </p>
  );
}

function AutoVariantCard({
  modelId,
  title,
  purpose,
  backends,
  className = '',
}: {
  modelId: string;
  title: string;
  purpose: string;
  backends: string;
  className?: string;
}) {
  return (
    <div className={`border border-white/20 rounded-lg p-5 bg-black/40 hover:bg-white/[0.06] transition-colors ${className}`}>
      <div className="text-white font-bold mb-1">{title}</div>
      <code className="text-xs text-emerald-400/90 block mb-3">{modelId}</code>
      <p className="text-white/70 text-xs leading-relaxed mb-2">{purpose}</p>
      <p className="text-white/55 text-[11px] leading-relaxed">{backends}</p>
    </div>
  );
}

function FrontierMetric({ value, label, detail }: { value: string; label: string; detail: string }) {
  return (
    <div className="bg-black/75 p-5">
      <div className="font-mono text-3xl font-bold tracking-tight text-white">{value}</div>
      <div className="mt-2 text-sm font-semibold text-white/80">{label}</div>
      <div className="mt-1 font-mono text-[11px] text-white/45">{detail}</div>
    </div>
  );
}

function FeatureCard({ icon, title, description, to }: { icon: React.ReactNode, title: string, description: string, to: string }) {
  return (
    <Link to={to} className="bg-black p-8 group hover:bg-white/[0.05] transition-colors h-full block focus:outline-none focus-visible:ring-2 focus-visible:ring-white/40">
      <div className="mb-6 text-white/55 group-hover:text-white transition-colors flex items-start justify-between gap-4">
        {icon}
        <ArrowUpRight className="w-4 h-4 text-white/35 group-hover:text-white/60 transition-colors shrink-0" />
      </div>
      <h3 className="text-xl font-bold mb-3 tracking-tight">{title}</h3>
      <p className="text-sm text-white/60 leading-relaxed font-light">{description}</p>
    </Link>
  );
}
