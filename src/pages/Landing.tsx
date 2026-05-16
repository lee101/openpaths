import React, { useEffect, useRef, useState } from 'react';
import { motion } from 'motion/react';
import { Zap, Code2, ArrowRight, Github, Search, Layers, Activity, Sparkles, ArrowUpRight, Image as ImageIcon } from 'lucide-react';
import { Link } from 'react-router-dom';
import { CodeBlock } from '../components/CodeBlock';
import { artGallery } from '../data/artGallery';
import { getProviderLogo, providersByName } from '../data/providers';
import { Seo } from '../components/Seo';

export function Landing() {
  const [activeTab, setActiveTab] = useState<'python' | 'curl'>('python');
  const apiKey = localStorage.getItem('op_api_key') || 'op_...';
  const featuredArt = artGallery[0];
  const galleryGrid = artGallery.slice(1);

  return (
    <>
      <Seo
        title="OpenPaths | Open Source AI Model Router"
        description="OpenPaths is an OpenAI-compatible model router for chat, image, video, speech, transcription, and embeddings across leading AI providers."
        path="/"
      />

      {/* Hero */}
      <section className="relative overflow-hidden border-b border-white/10">
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
            <span className="text-white/40">Model Router.</span>
          </h1>
          <p className="text-lg md:text-xl text-white/60 max-w-2xl mx-auto mb-10 font-light leading-relaxed">
            <span className="text-white font-medium">Search and we shall find.</span> Neural learned paths for 1ms routing to find the best model. Try Open Pathways.
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
      <section className="border-y border-white/10 bg-white/[0.02] py-8 overflow-hidden">
        <div className="max-w-7xl mx-auto px-6 flex flex-wrap justify-center md:justify-between items-center gap-8 font-mono text-sm text-white/40">
          <div className="flex items-center gap-2"><Activity className="w-4 h-4" /> 99.99% Uptime</div>
          <div className="flex items-center gap-2"><Layers className="w-4 h-4" /> 100+ Models</div>
          <div className="flex items-center gap-2"><Zap className="w-4 h-4" /> &lt;50ms Latency</div>
          <div className="flex items-center gap-2"><Search className="w-4 h-4" /> Intelligent Fallbacks</div>
        </div>
      </section>

      {/* Code Snippet */}
      <section id="api" className="px-6 py-24 max-w-5xl mx-auto">
        <div className="mb-12 text-center">
          <h2 className="text-3xl md:text-4xl font-bold tracking-tight mb-4">100% OpenAI Compatible</h2>
          <p className="text-white/60 font-mono text-sm">Just change the base URL and API key. That's it.</p>
        </div>
        
        <div className="border border-white/10 bg-black rounded-lg overflow-hidden shadow-2xl shadow-white/5">
          <div className="flex items-center justify-between px-4 py-3 border-b border-white/10 bg-white/[0.02]">
            <div className="flex items-center gap-2">
              <div className="w-3 h-3 rounded-full bg-white/20" />
              <div className="w-3 h-3 rounded-full bg-white/20" />
              <div className="w-3 h-3 rounded-full bg-white/20" />
            </div>
            <div className="flex gap-2 font-mono text-xs">
              <button 
                onClick={() => setActiveTab('python')}
                className={`px-3 py-1 rounded transition-colors ${activeTab === 'python' ? 'bg-white/10 text-white' : 'text-white/40 hover:text-white'}`}
              >
                Python
              </button>
              <button 
                onClick={() => setActiveTab('curl')}
                className={`px-3 py-1 rounded transition-colors ${activeTab === 'curl' ? 'bg-white/10 text-white' : 'text-white/40 hover:text-white'}`}
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
                code={`import openai\n\nclient = openai.OpenAI(\n  base_url="https://openpaths.io/v1",\n  api_key="${apiKey}"\n)\n\nresponse = client.chat.completions.create(\n  model="auto-hard-task",\n  messages=[\n    {"role": "user", "content": "Make a 3D simulation of cogs in a clock."}\n  ],\n  reasoning_effort="auto",\n)`}
              />
            ) : (
              <CodeBlock
                language="bash"
                preClassName="p-6"
                code={`curl https://openpaths.io/v1/chat/completions \\\n  -H "Content-Type: application/json" \\\n  -H "Authorization: Bearer ${apiKey}" \\\n  -d '{\n    "model": "auto-hard-task",\n    "messages": [\n      {\n        "role": "user",\n        "content": "Make a 3D simulation of cogs in a clock."\n      }\n    ],\n    "reasoning_effort": "auto"\n  }'`}
              />
            )}
          </div>
        </div>
        <div className="mt-6 grid grid-cols-1 md:grid-cols-2 gap-4 font-mono text-xs text-white/50">
          <div className="border border-white/10 rounded-lg p-4 bg-white/[0.02]">
            OpenAI-compatible params: <span className="text-white">model</span>, <span className="text-white">messages</span>, <span className="text-white">temperature</span>, <span className="text-white">top_p</span>, <span className="text-white">max_tokens</span>, <span className="text-white">stream</span>, <span className="text-white">tools</span>, <span className="text-white">response_format</span>, <span className="text-white">reasoning_effort</span>.
          </div>
          <div className="border border-white/10 rounded-lg p-4 bg-white/[0.02]">
            <span className="text-white">reasoning_effort</span> supports <span className="text-white">none</span>, <span className="text-white">low</span>, <span className="text-white">medium</span>, <span className="text-white">high</span>, and <span className="text-white">auto</span>.
          </div>
        </div>
      </section>

      {/* Art Playground */}
      <section id="art-playground" className="px-6 py-24 max-w-7xl mx-auto border-t border-white/10">
        <div className="relative overflow-hidden rounded-[32px] border border-white/10 bg-[radial-gradient(circle_at_top_left,_rgba(245,158,11,0.14),_transparent_30%),radial-gradient(circle_at_bottom_right,_rgba(45,212,191,0.12),_transparent_34%),linear-gradient(180deg,_rgba(255,255,255,0.04),_rgba(255,255,255,0.02))] p-6 md:p-8 lg:p-10">
          <div className="absolute inset-0 pointer-events-none bg-[linear-gradient(120deg,transparent_0%,rgba(255,255,255,0.04)_45%,transparent_65%)] opacity-60" />
          <div className="relative">
            <div className="grid gap-8">
              <div>
                <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border border-white/15 bg-black/30 text-xs font-mono mb-6">
                  <ImageIcon className="w-3.5 h-3.5" />
                  Live Art Playground
                </div>
                <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
                  <div className="max-w-3xl">
                    <h2 className="text-3xl md:text-5xl font-bold tracking-tight leading-[0.95] mb-4">
                      Image generation with real providers, not mock thumbnails.
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

                <motion.article
                  initial={{ opacity: 0, y: 24 }}
                  whileInView={{ opacity: 1, y: 0 }}
                  viewport={{ once: true, amount: 0.2 }}
                  transition={{ duration: 0.45 }}
                  className="mt-8 overflow-hidden rounded-[28px] border border-white/10 bg-black/50"
                >
                  <div className="relative aspect-[16/10] overflow-hidden">
                    <img
                      src={featuredArt.imageUrl}
                      alt={featuredArt.title}
                      className="h-full w-full object-cover"
                      loading="lazy"
                    />
                    <div className="absolute inset-0 bg-gradient-to-t from-black via-black/30 to-transparent" />
                    <div className="absolute top-4 left-4 flex items-center gap-3 rounded-full border border-white/15 bg-black/55 px-3 py-2 backdrop-blur">
                      <img src={getProviderLogo(featuredArt.provider)} alt={`${featuredArt.provider} logo`} className="h-5 w-5 rounded-sm object-contain" />
                      <div className="font-mono text-xs text-white/75">
                        <div>{featuredArt.provider}</div>
                        <div className="text-white">{featuredArt.model}</div>
                      </div>
                    </div>
                    <div className="absolute bottom-0 left-0 right-0 p-5 md:p-6">
                      <div className="max-w-4xl">
                        <div className="text-xs uppercase tracking-[0.22em] text-white/40 mb-2">Featured Study</div>
                        <h3 className="text-2xl md:text-4xl font-bold tracking-tight">{featuredArt.title}</h3>
                        <PromptText text={featuredArt.prompt} className="mt-3 text-sm md:text-base text-white/70 max-w-3xl" />
                      </div>
                    </div>
                  </div>
                </motion.article>

                <div className="mt-5 grid gap-4 md:grid-cols-2 xl:grid-cols-4">
                  {galleryGrid.map((item, index) => (
                    <motion.article
                      key={item.slug}
                      initial={{ opacity: 0, y: 18 }}
                      whileInView={{ opacity: 1, y: 0 }}
                      viewport={{ once: true, amount: 0.15 }}
                      transition={{ duration: 0.35, delay: index * 0.04 }}
                      className="overflow-hidden rounded-[24px] border border-white/10 bg-black/45 group"
                    >
                      <div className="relative aspect-square overflow-hidden">
                        <img
                          src={item.imageUrl}
                          alt={item.title}
                          className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-[1.03]"
                          loading="lazy"
                        />
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
                          <div className="text-xs uppercase tracking-[0.22em] text-white/35 mb-2">{item.model}</div>
                          <h3 className="text-xl font-bold tracking-tight">{item.title}</h3>
                          <PromptText text={item.prompt} className="mt-2 text-sm text-white/65" />
                        </div>
                      </div>
                    </motion.article>
                  ))}
                </div>
              </div>

            </div>
          </div>
        </div>
      </section>

      {/* Features */}
      <section id="features" className="px-6 py-24 max-w-7xl mx-auto border-t border-white/10">
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
            title="Auto Models"
            description="Always on the price frontier. Static embedding model-based routing picks the best frontier model for every task — just use auto-coding-latest and stay ahead automatically."
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
      <section className="px-6 py-32 border-t border-white/10 text-center bg-white/[0.02]">
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

      for (let pulse = 0; pulse < 3; pulse++) {
        const progress = ((t * 0.18 + pulse / 3) % 1);
        const z = progress * rows;
        const y = horizonY + z * spacing * (1 / (1 + z * 0.075)) * 0.62;
        const radiusX = width * (0.18 + progress * 0.48);
        const radiusY = 18 + progress * 72;
        const gradient = ctx.createRadialGradient(centerX, y, 0, centerX, y, radiusX);
        gradient.addColorStop(0, 'rgba(255,255,255,0.14)');
        gradient.addColorStop(0.32, 'rgba(255,255,255,0.055)');
        gradient.addColorStop(0.68, 'rgba(0,0,0,0.22)');
        gradient.addColorStop(1, 'rgba(0,0,0,0)');
        ctx.fillStyle = gradient;
        ctx.beginPath();
        ctx.ellipse(centerX, y, radiusX, radiusY, 0, 0, Math.PI * 2);
        ctx.fill();
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

function FeatureCard({ icon, title, description, to }: { icon: React.ReactNode, title: string, description: string, to: string }) {
  return (
    <Link to={to} className="bg-black p-8 group hover:bg-white/[0.02] transition-colors h-full block focus:outline-none focus-visible:ring-2 focus-visible:ring-white/40">
      <div className="mb-6 text-white/40 group-hover:text-white transition-colors flex items-start justify-between gap-4">
        {icon}
        <ArrowUpRight className="w-4 h-4 text-white/20 group-hover:text-white/60 transition-colors shrink-0" />
      </div>
      <h3 className="text-xl font-bold mb-3 tracking-tight">{title}</h3>
      <p className="text-sm text-white/60 leading-relaxed font-light">{description}</p>
    </Link>
  );
}
