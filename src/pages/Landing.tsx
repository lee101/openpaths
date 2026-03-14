import React, { useState } from 'react';
import { motion } from 'motion/react';
import { Terminal, Zap, Code2, ArrowRight, Github, Search, Layers, Activity, Sparkles, KeyRound, Cpu } from 'lucide-react';
import { Link } from 'react-router-dom';
import { CodeBlock } from '../components/CodeBlock';

export function Landing() {
  const [activeTab, setActiveTab] = useState<'openai' | 'anthropic' | 'curl'>('openai');
  const openAIPythonExample = `import openai

client = openai.OpenAI(
    base_url="https://openpaths.io/v1",
    api_key="op-...",
)

response = client.chat.completions.create(
    model="anthropic-opus-latest",
    messages=[{"role": "user", "content": "Write a tiny SSE server in Go."}],
)`;
  const anthropicPythonExample = `from anthropic import Anthropic

client = Anthropic(
    base_url="https://openpaths.io",
    api_key="op-...",
)

message = client.messages.create(
    model="anthropic-opus-latest",
    max_tokens=1024,
    messages=[{"role": "user", "content": "Write a tiny SSE server in Go."}],
)`;
  const curlExample = `curl https://openpaths.io/v1/chat/completions \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer op-..." \\
  -d '{
    "model": "anthropic-opus-latest",
    "messages": [
      {"role": "user", "content": "Write a tiny SSE server in Go."}
    ]
  }'`;

  return (
    <>
      {/* Hero */}
      <section className="px-6 py-24 md:py-32 max-w-7xl mx-auto flex flex-col items-center text-center relative overflow-hidden">
        {/* Decorative background elements */}
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[800px] h-[800px] border border-white/5 rounded-full pointer-events-none" />
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[600px] border border-white/5 rounded-full pointer-events-none" />
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[400px] h-[400px] border border-white/5 rounded-full pointer-events-none" />
        
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5 }}
          className="relative z-10"
        >
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border border-white/10 bg-white/5 text-xs font-mono mb-8">
            <span className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
            v1.1.0 is now live
          </div>
          <h1 className="text-5xl md:text-7xl lg:text-8xl font-bold tracking-tighter mb-6 leading-[0.9]">
            The Open Source <br />
            <span className="text-white/40">Model Router.</span>
          </h1>
          <p className="text-lg md:text-xl text-white/60 max-w-2xl mx-auto mb-10 font-light leading-relaxed">
            <span className="text-white font-medium">Search and we shall find open pathways.</span> Newly learned pathways for millisecond routing between large model providers or art generators.
          </p>
          <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
            <Link to="/models" className="w-full sm:w-auto bg-white text-black px-8 py-4 font-mono font-bold flex items-center justify-center gap-2 hover:bg-white/90 transition-colors rounded">
              Explore Models <ArrowRight className="w-4 h-4" />
            </Link>
            <a href="https://github.com" target="_blank" rel="noreferrer" className="w-full sm:w-auto border border-white/20 px-8 py-4 font-mono flex items-center justify-center gap-2 hover:bg-white/10 transition-colors rounded">
              <Github className="w-4 h-4" /> View Source
            </a>
          </div>
        </motion.div>
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
          <h2 className="text-3xl md:text-4xl font-bold tracking-tight mb-4">OpenAI And Anthropic SDK Compatible</h2>
          <p className="text-white/60 font-mono text-sm mb-4">Keep your existing Python client. Just change the base URL and API key.</p>
          <div className="flex flex-wrap items-center justify-center gap-2 text-xs font-mono text-white/50">
            <span className="px-3 py-1 rounded-full border border-white/10 bg-white/[0.03]">OpenAI SDK {'->'} <code>https://openpaths.io/v1</code></span>
            <span className="px-3 py-1 rounded-full border border-white/10 bg-white/[0.03]">Anthropic SDK {'->'} <code>https://openpaths.io</code></span>
          </div>
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
                onClick={() => setActiveTab('openai')}
                className={`px-3 py-1 rounded transition-colors ${activeTab === 'openai' ? 'bg-white/10 text-white' : 'text-white/40 hover:text-white'}`}
              >
                OpenAI Python
              </button>
              <button 
                onClick={() => setActiveTab('anthropic')}
                className={`px-3 py-1 rounded transition-colors ${activeTab === 'anthropic' ? 'bg-white/10 text-white' : 'text-white/40 hover:text-white'}`}
              >
                Anthropic Python
              </button>
              <button 
                onClick={() => setActiveTab('curl')}
                className={`px-3 py-1 rounded transition-colors ${activeTab === 'curl' ? 'bg-white/10 text-white' : 'text-white/40 hover:text-white'}`}
              >
                cURL
              </button>
            </div>
          </div>
          <div className="p-6 overflow-x-auto">
            {activeTab === 'openai' ? (
              <CodeBlock code={openAIPythonExample} language="python" />
            ) : activeTab === 'anthropic' ? (
              <CodeBlock code={anthropicPythonExample} language="python" />
            ) : (
              <CodeBlock code={curlExample} language="bash" />
            )}
          </div>
        </div>
      </section>

      {/* Features */}
      <section id="features" className="px-6 py-24 max-w-7xl mx-auto border-t border-white/10">
        <div className="mb-16">
          <h2 className="text-3xl md:text-4xl font-bold tracking-tight mb-4">Built for scale.</h2>
          <p className="text-white/60 font-mono text-sm max-w-2xl">We handle the complexity of routing, fallbacks, and payments so you can focus on building your product.</p>
        </div>
        
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-px bg-white/10">
          <FeatureCard
            icon={<Zap className="w-6 h-6" />}
            title="Millisecond Routing"
            description="Intelligent pathways find the lowest latency and highest throughput provider for your request instantly."
          />
          <FeatureCard
            icon={<Code2 className="w-6 h-6" />}
            title="Universal API"
            description="One API key for OpenAI, Anthropic, Meta, Mistral, and dozens of art generators like RA1, FLUX, and Stable Diffusion."
          />
          <FeatureCard
            icon={<Sparkles className="w-6 h-6" />}
            title="Auto Models"
            description="Always on the price frontier. AI neural routing picks the best frontier model for every task -- just use auto-coding-latest and stay ahead automatically."
          />
          <FeatureCard
            icon={<KeyRound className="w-6 h-6" />}
            title="Bring Your Own Keys"
            description="Use your own API keys from OpenAI, Anthropic, Google, and more. Free routing and fallbacks with zero markup on your keys."
          />
          <FeatureCard
            icon={<Cpu className="w-6 h-6" />}
            title="Codex Spark Models"
            description="Use GPT-5.3 Codex and Codex Spark with your OpenAI Max plan. Low reasoning mode for fast, cheap coding tasks with automatic token refresh."
          />
          <FeatureCard
            icon={<svg className="w-6 h-6" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M14.5 3h-10l-2 5h10l2-5Z"/><path d="M11.5 11h-10l-2 5h10l2-5Z"/><path d="M14.5 19h-10l-2 5h10l2-5Z"/></svg>}
            title="Solana Payments"
            description="Prefer decentralization? Fund your API usage instantly with Solana (SOL) or USDC on the world's fastest blockchain."
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

function FeatureCard({ icon, title, description }: { icon: React.ReactNode, title: string, description: string }) {
  return (
    <div className="bg-black p-8 group hover:bg-white/[0.02] transition-colors h-full">
      <div className="mb-6 text-white/40 group-hover:text-white transition-colors">
        {icon}
      </div>
      <h3 className="text-xl font-bold mb-3 tracking-tight">{title}</h3>
      <p className="text-sm text-white/60 leading-relaxed font-light">{description}</p>
    </div>
  );
}
