import React, { useEffect, useMemo, useState } from 'react';
import { Check, Copy, Plug, Server, Wrench } from 'lucide-react';
import { Link } from 'react-router-dom';
import { CodeBlock } from '../components/CodeBlock';
import { getAPIBaseURL, getStoredAPIKey } from '../lib/session';
import { Seo } from '../components/Seo';

type Tool = {
  name: string;
  desc: string;
  args: string;
};

const TOOLS: Tool[] = [
  { name: 'chat', desc: 'Chat completion against any model (LLMs from OpenAI, Anthropic, Google, xAI, …).', args: 'model, prompt | messages, system?, max_tokens?, temperature?, reasoning_effort?' },
  { name: 'list_models', desc: 'List available models with owner and context window.', args: 'filter?' },
  { name: 'generate_image', desc: 'Generate an image from a prompt, returns image URL(s).', args: 'model, prompt, size?, n?' },
  { name: 'embed', desc: 'Create embedding vectors for text.', args: 'model, input' },
  { name: 'web_search', desc: 'Search the web and return ranked results.', args: 'query, numResults?' },
];

function mcpUrl(apiBase: string): string {
  // apiBase is the OpenAI-compatible base (…/v1). The MCP endpoint lives at /mcp.
  return apiBase.replace(/\/v1\/?$/, '') + '/mcp';
}

export function Mcp() {
  const [apiKey, setApiKey] = useState(() => getStoredAPIKey());
  const [copied, setCopied] = useState<string | null>(null);
  const apiBase = getAPIBaseURL();
  const url = mcpUrl(apiBase);
  const exampleKey = apiKey || 'OPENPATHS_API_KEY';
  const tools = useMemo(() => TOOLS, []);

  useEffect(() => {
    const sync = () => setApiKey(getStoredAPIKey());
    window.addEventListener('storage', sync);
    window.addEventListener('auth-change', sync);
    return () => {
      window.removeEventListener('storage', sync);
      window.removeEventListener('auth-change', sync);
    };
  }, []);

  const copy = (id: string, text: string) => {
    navigator.clipboard?.writeText(text);
    setCopied(id);
    setTimeout(() => setCopied((c) => (c === id ? null : c)), 1500);
  };

  const claudeConfig = `{
  "mcpServers": {
    "openpaths": {
      "type": "http",
      "url": "${url}",
      "headers": {
        "Authorization": "Bearer ${exampleKey}"
      }
    }
  }
}`;

  const claudeCodeCmd = `claude mcp add --transport http openpaths ${url} \\
  --header "Authorization: Bearer ${exampleKey}"`;

  const curlExample = `curl -s ${url} \\
  -H "Authorization: Bearer ${exampleKey}" \\
  -H "Content-Type: application/json" \\
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "chat",
      "arguments": { "model": "auto", "prompt": "Say hi in one word." }
    }
  }'`;

  return (
    <>
      <Seo
        title="MCP Server — all models over Model Context Protocol | OpenPaths"
        description="Connect Claude Desktop, Claude Code, Cursor and any MCP client to every OpenPaths model with your secret key. High-performance Streamable HTTP MCP server."
        path="/mcp"
      />

      <section className="max-w-5xl mx-auto px-6 py-16">
        <div className="mb-12">
          <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border border-white/10 bg-white/[0.04] text-xs font-mono text-white/60 mb-6">
            <Server className="w-3.5 h-3.5" />
            Model Context Protocol
          </div>
          <h1 className="text-4xl md:text-5xl font-bold tracking-tight mb-4">MCP Server</h1>
          <p className="text-white/60 max-w-3xl font-light leading-relaxed">
            Every OpenPaths model — chat, image, embeddings and web search — exposed over the{' '}
            <span className="text-white">Model Context Protocol</span>. Point any MCP client at one
            URL, authenticate with your secret key, and your agents get the whole catalog with
            automatic routing, fallback and billing.
          </p>
        </div>

        {/* Connection details */}
        <div className="grid md:grid-cols-2 gap-4 mb-12">
          <div className="border border-white/10 bg-white/[0.02] rounded-xl p-5">
            <div className="text-xs font-mono text-white/40 mb-2">ENDPOINT (Streamable HTTP)</div>
            <button
              onClick={() => copy('url', url)}
              className="flex items-center gap-2 font-mono text-sm text-white hover:text-white/80 break-all text-left"
            >
              {copied === 'url' ? <Check className="w-4 h-4 shrink-0 text-green-400" /> : <Copy className="w-4 h-4 shrink-0" />}
              {url}
            </button>
          </div>
          <div className="border border-white/10 bg-white/[0.02] rounded-xl p-5">
            <div className="text-xs font-mono text-white/40 mb-2">AUTH HEADER</div>
            <button
              onClick={() => copy('key', `Authorization: Bearer ${exampleKey}`)}
              className="flex items-center gap-2 font-mono text-sm text-white hover:text-white/80 break-all text-left"
            >
              {copied === 'key' ? <Check className="w-4 h-4 shrink-0 text-green-400" /> : <Copy className="w-4 h-4 shrink-0" />}
              Bearer {apiKey ? exampleKey.slice(0, 11) + '••••••••' : 'OPENPATHS_API_KEY'}
            </button>
            {!apiKey && (
              <p className="text-xs text-white/40 mt-2">
                <Link to="/account" className="underline hover:text-white/70">Sign in</Link> to inject your real key into these snippets.
              </p>
            )}
          </div>
        </div>

        {/* Tools */}
        <div className="mb-12">
          <h2 className="text-2xl font-bold tracking-tight mb-4 flex items-center gap-2">
            <Wrench className="w-5 h-5 text-white/60" /> Tools
          </h2>
          <div className="border border-white/10 rounded-xl overflow-hidden">
            {tools.map((t, i) => (
              <div
                key={t.name}
                className={`p-4 ${i > 0 ? 'border-t border-white/10' : ''} bg-white/[0.02]`}
              >
                <div className="flex flex-wrap items-baseline gap-3 mb-1">
                  <span className="font-mono text-sm text-white font-bold">{t.name}</span>
                  <span className="font-mono text-xs text-white/40">{t.args}</span>
                </div>
                <p className="text-sm text-white/60 font-light">{t.desc}</p>
              </div>
            ))}
          </div>
          <p className="text-sm text-white/40 mt-3">
            Use <span className="font-mono text-white/70">model: "auto"</span> to let OpenPaths pick
            the best model per request, or any id from{' '}
            <Link to="/models" className="underline hover:text-white/70">the model directory</Link>.
          </p>
        </div>

        {/* Claude Desktop */}
        <div className="mb-10">
          <h2 className="text-2xl font-bold tracking-tight mb-3 flex items-center gap-2">
            <Plug className="w-5 h-5 text-white/60" /> Claude Desktop / Cursor / Cline
          </h2>
          <p className="text-white/60 font-light mb-4">
            Add OpenPaths to your client's MCP config (e.g. <span className="font-mono text-white/70">claude_desktop_config.json</span>):
          </p>
          <CodeBlock code={claudeConfig} language="json" label="mcp config" />
        </div>

        {/* Claude Code */}
        <div className="mb-10">
          <h2 className="text-2xl font-bold tracking-tight mb-3">Claude Code (CLI)</h2>
          <CodeBlock code={claudeCodeCmd} language="bash" label="terminal" />
        </div>

        {/* Raw */}
        <div className="mb-10">
          <h2 className="text-2xl font-bold tracking-tight mb-3">Raw JSON-RPC</h2>
          <p className="text-white/60 font-light mb-4">
            The server speaks JSON-RPC 2.0 over HTTP POST (stateless, no session id needed). Call a
            tool directly:
          </p>
          <CodeBlock code={curlExample} language="bash" label="curl" />
        </div>

        <div className="border border-white/10 bg-white/[0.02] rounded-xl p-5 text-sm text-white/50 font-light">
          Pay-as-you-go billing applies per call to paid tools (chat, image, embed, web_search), using
          the same credits as the API. <Link to="/pricing" className="underline hover:text-white/70">See pricing</Link>.
          BYOK provider keys are honored automatically when configured in your{' '}
          <Link to="/account" className="underline hover:text-white/70">account</Link>.
        </div>
      </section>
    </>
  );
}
