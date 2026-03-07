import React, { useState, useRef, useEffect } from 'react';
import { Send, Plus, X, Settings, ChevronDown, Loader2, Trash2 } from 'lucide-react';

interface Message {
  role: 'system' | 'user' | 'assistant';
  content: string;
}

interface ModelPane {
  id: string;
  modelId: string;
  messages: Message[];
  streaming: boolean;
  error: string | null;
  latencyMs: number | null;
}

const CHAT_MODELS = [
  { id: 'auto', label: 'Auto (intelligent routing)', provider: 'OpenPaths' },
  { id: 'auto-easy-task', label: 'Auto Easy (cheapest)', provider: 'OpenPaths' },
  { id: 'auto-medium-task', label: 'Auto Medium (balanced)', provider: 'OpenPaths' },
  { id: 'gemini-3.1-pro-preview', label: 'Gemini 3.1 Pro', provider: 'Google' },
  { id: 'gemini-2.5-pro', label: 'Gemini 2.5 Pro', provider: 'Google' },
  { id: 'gemini-2.5-flash', label: 'Gemini 2.5 Flash', provider: 'Google' },
  { id: 'gpt-5.2', label: 'GPT-5.2', provider: 'OpenAI' },
  { id: 'o3', label: 'o3', provider: 'OpenAI' },
  { id: 'o4-mini', label: 'o4-mini', provider: 'OpenAI' },
  { id: 'gpt-4o', label: 'GPT-4o', provider: 'OpenAI' },
  { id: 'gpt-4o-mini', label: 'GPT-4o Mini', provider: 'OpenAI' },
  { id: 'claude-sonnet-latest', label: 'Claude Sonnet (latest)', provider: 'Anthropic' },
  { id: 'claude-opus-latest', label: 'Claude Opus (latest)', provider: 'Anthropic' },
  { id: 'claude-haiku-4-5-20251001', label: 'Claude Haiku', provider: 'Anthropic' },
  { id: 'grok-4-0709', label: 'Grok 4', provider: 'xAI' },
  { id: 'grok-4-1-fast-reasoning', label: 'Grok 4.1 Fast', provider: 'xAI' },
  { id: 'grok-3-mini', label: 'Grok 3 Mini', provider: 'xAI' },
  { id: 'deepseek-chat', label: 'DeepSeek Chat', provider: 'DeepSeek' },
  { id: 'deepseek-reasoner', label: 'DeepSeek Reasoner', provider: 'DeepSeek' },
  { id: 'mistral-large-latest', label: 'Mistral Large', provider: 'Mistral' },
  { id: 'llama-3.3-70b-versatile', label: 'Llama 3.3 70B', provider: 'Groq' },
  { id: 'glm-5', label: 'GLM-5', provider: 'Together' },
  { id: 'qwen3.5-397b', label: 'Qwen 3.5 397B', provider: 'Together' },
  { id: 'minimax-m2.5-direct', label: 'MiniMax M2.5', provider: 'MiniMax' },
  { id: 'kimi-k2.5', label: 'Kimi K2.5', provider: 'Together' },
];

let paneCounter = 0;
function makePane(modelId: string): ModelPane {
  return { id: `pane-${++paneCounter}`, modelId, messages: [], streaming: false, error: null, latencyMs: null };
}

export function Playground() {
  const [apiKey, setApiKey] = useState(() => localStorage.getItem('op_api_key') || '');
  const [systemPrompt, setSystemPrompt] = useState('You are a helpful assistant.');
  const [showSettings, setShowSettings] = useState(false);
  const [input, setInput] = useState('');
  const [panes, setPanes] = useState<ModelPane[]>([
    makePane('auto'),
  ]);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const abortRefs = useRef<Map<string, AbortController>>(new Map());

  useEffect(() => {
    if (apiKey) localStorage.setItem('op_api_key', apiKey);
  }, [apiKey]);

  const baseUrl = window.location.origin;

  async function sendToModel(paneId: string, modelId: string, messages: Message[]) {
    const controller = new AbortController();
    abortRefs.current.set(paneId, controller);

    const start = performance.now();

    setPanes(prev => prev.map(p =>
      p.id === paneId ? { ...p, streaming: true, error: null, latencyMs: null } : p
    ));

    try {
      const body = {
        model: modelId,
        messages: [
          ...(systemPrompt ? [{ role: 'system', content: systemPrompt }] : []),
          ...messages,
        ],
        stream: true,
      };

      const resp = await fetch(`${baseUrl}/v1/chat/completions`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${apiKey}`,
        },
        body: JSON.stringify(body),
        signal: controller.signal,
      });

      if (!resp.ok) {
        const errText = await resp.text();
        throw new Error(`${resp.status}: ${errText.slice(0, 200)}`);
      }

      const reader = resp.body?.getReader();
      if (!reader) throw new Error('No response body');

      const decoder = new TextDecoder();
      let assistantContent = '';
      let firstToken = true;

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        const chunk = decoder.decode(value, { stream: true });
        const lines = chunk.split('\n');

        for (const line of lines) {
          if (!line.startsWith('data: ')) continue;
          const data = line.slice(6);
          if (data === '[DONE]') continue;

          try {
            const parsed = JSON.parse(data);
            const delta = parsed.choices?.[0]?.delta?.content;
            if (delta) {
              if (firstToken) {
                firstToken = false;
                const ttft = Math.round(performance.now() - start);
                setPanes(prev => prev.map(p =>
                  p.id === paneId ? { ...p, latencyMs: ttft } : p
                ));
              }
              assistantContent += delta;
              setPanes(prev => prev.map(p => {
                if (p.id !== paneId) return p;
                const msgs = [...p.messages];
                const last = msgs[msgs.length - 1];
                if (last && last.role === 'assistant') {
                  msgs[msgs.length - 1] = { ...last, content: assistantContent };
                } else {
                  msgs.push({ role: 'assistant', content: assistantContent });
                }
                return { ...p, messages: msgs };
              }));
            }
          } catch {}
        }
      }

      setPanes(prev => prev.map(p =>
        p.id === paneId ? { ...p, streaming: false } : p
      ));
    } catch (err: any) {
      if (err.name === 'AbortError') return;
      setPanes(prev => prev.map(p =>
        p.id === paneId ? { ...p, streaming: false, error: err.message } : p
      ));
    } finally {
      abortRefs.current.delete(paneId);
    }
  }

  function handleSend() {
    const msg = input.trim();
    if (!msg || !apiKey) return;

    const userMsg: Message = { role: 'user', content: msg };
    setInput('');

    setPanes(prev => prev.map(p => ({
      ...p,
      messages: [...p.messages, userMsg],
    })));

    for (const pane of panes) {
      const allMsgs = [...pane.messages, userMsg];
      sendToModel(pane.id, pane.modelId, allMsgs);
    }
  }

  function addPane() {
    if (panes.length >= 4) return;
    const usedModels = new Set(panes.map(p => p.modelId));
    const next = CHAT_MODELS.find(m => !usedModels.has(m.id)) || CHAT_MODELS[0];
    setPanes(prev => [...prev, makePane(next.id)]);
  }

  function removePane(id: string) {
    if (panes.length <= 1) return;
    const ctrl = abortRefs.current.get(id);
    if (ctrl) ctrl.abort();
    setPanes(prev => prev.filter(p => p.id !== id));
  }

  function changeModel(paneId: string, modelId: string) {
    setPanes(prev => prev.map(p =>
      p.id === paneId ? { ...p, modelId } : p
    ));
  }

  function clearAll() {
    for (const ctrl of abortRefs.current.values()) ctrl.abort();
    abortRefs.current.clear();
    setPanes(prev => prev.map(p => ({ ...p, messages: [], streaming: false, error: null, latencyMs: null })));
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  }

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="border-b border-white/10 px-4 py-3 flex items-center gap-3 bg-white/[0.02]">
        <button
          onClick={() => setShowSettings(!showSettings)}
          className={`flex items-center gap-2 px-3 py-1.5 text-xs font-mono rounded border transition-colors ${showSettings ? 'border-white/30 bg-white/10 text-white' : 'border-white/10 text-white/60 hover:text-white hover:border-white/20'}`}
        >
          <Settings className="w-3.5 h-3.5" /> Settings
        </button>
        <button
          onClick={addPane}
          disabled={panes.length >= 4}
          className="flex items-center gap-2 px-3 py-1.5 text-xs font-mono rounded border border-white/10 text-white/60 hover:text-white hover:border-white/20 transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
        >
          <Plus className="w-3.5 h-3.5" /> Add Model
        </button>
        <button
          onClick={clearAll}
          className="flex items-center gap-2 px-3 py-1.5 text-xs font-mono rounded border border-white/10 text-white/60 hover:text-white hover:border-white/20 transition-colors"
        >
          <Trash2 className="w-3.5 h-3.5" /> Clear
        </button>
        <div className="ml-auto text-xs font-mono text-white/30">
          {panes.length}/4 models
        </div>
      </div>

      {/* Settings Panel */}
      {showSettings && (
        <div className="border-b border-white/10 px-4 py-4 bg-white/[0.02] space-y-3">
          <div>
            <label className="text-xs font-mono text-white/50 block mb-1">API Key</label>
            <input
              type="password"
              value={apiKey}
              onChange={e => setApiKey(e.target.value)}
              placeholder="op-..."
              className="w-full max-w-md bg-black border border-white/10 rounded px-3 py-2 text-sm font-mono text-white placeholder:text-white/20 focus:outline-none focus:border-white/30"
            />
          </div>
          <div>
            <label className="text-xs font-mono text-white/50 block mb-1">System Prompt</label>
            <textarea
              value={systemPrompt}
              onChange={e => setSystemPrompt(e.target.value)}
              rows={3}
              className="w-full bg-black border border-white/10 rounded px-3 py-2 text-sm font-mono text-white placeholder:text-white/20 focus:outline-none focus:border-white/30 resize-none"
              placeholder="You are a helpful assistant."
            />
          </div>
        </div>
      )}

      {/* Model Panes */}
      <div className="flex-1 flex overflow-hidden">
        {panes.map(pane => (
          <div key={pane.id} className="flex-1 flex flex-col border-r border-white/10 last:border-r-0 min-w-0">
            {/* Pane Header */}
            <div className="flex items-center gap-2 px-3 py-2 border-b border-white/10 bg-white/[0.02]">
              <ModelSelect
                value={pane.modelId}
                onChange={m => changeModel(pane.id, m)}
              />
              {pane.latencyMs !== null && (
                <span className="text-[10px] font-mono text-green-400/70 shrink-0">
                  {pane.latencyMs}ms
                </span>
              )}
              {pane.streaming && <Loader2 className="w-3 h-3 animate-spin text-white/40 shrink-0" />}
              {panes.length > 1 && (
                <button onClick={() => removePane(pane.id)} className="ml-auto text-white/20 hover:text-white/60 shrink-0">
                  <X className="w-3.5 h-3.5" />
                </button>
              )}
            </div>

            {/* Messages */}
            <div className="flex-1 overflow-y-auto p-3 space-y-3">
              {pane.messages.length === 0 && !pane.error && (
                <div className="h-full flex items-center justify-center text-white/15 text-sm font-mono">
                  Send a message to start
                </div>
              )}
              {pane.messages.map((msg, i) => (
                <div key={i} className={`text-sm ${msg.role === 'user' ? 'text-white/50' : 'text-white'}`}>
                  <span className="text-[10px] font-mono text-white/25 uppercase block mb-0.5">
                    {msg.role}
                  </span>
                  <div className="whitespace-pre-wrap break-words leading-relaxed">
                    {msg.content}
                  </div>
                </div>
              ))}
              {pane.error && (
                <div className="text-xs font-mono text-red-400/80 bg-red-400/5 rounded p-2 border border-red-400/10">
                  {pane.error}
                </div>
              )}
              <ScrollAnchor messages={pane.messages} streaming={pane.streaming} />
            </div>
          </div>
        ))}
      </div>

      {/* Input */}
      <div className="border-t border-white/10 p-4 bg-white/[0.02]">
        <div className="max-w-4xl mx-auto flex gap-3">
          <textarea
            ref={inputRef}
            value={input}
            onChange={e => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            rows={1}
            placeholder={apiKey ? 'Send a message to all models...' : 'Set your API key in Settings first'}
            disabled={!apiKey}
            className="flex-1 bg-black border border-white/10 rounded px-4 py-3 text-sm text-white placeholder:text-white/20 focus:outline-none focus:border-white/30 resize-none disabled:opacity-30"
          />
          <button
            onClick={handleSend}
            disabled={!input.trim() || !apiKey}
            className="bg-white text-black px-4 rounded font-mono text-sm font-bold hover:bg-white/90 transition-colors disabled:opacity-30 disabled:cursor-not-allowed shrink-0"
          >
            <Send className="w-4 h-4" />
          </button>
        </div>
      </div>
    </div>
  );
}

function ModelSelect({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, []);

  const current = CHAT_MODELS.find(m => m.id === value);
  const grouped = CHAT_MODELS.reduce<Record<string, typeof CHAT_MODELS>>((acc, m) => {
    (acc[m.provider] ||= []).push(m);
    return acc;
  }, {});

  return (
    <div ref={ref} className="relative flex-1 min-w-0">
      <button
        onClick={() => setOpen(!open)}
        className="w-full flex items-center gap-1 text-xs font-mono text-white/80 hover:text-white truncate"
      >
        <span className="truncate">{current?.label || value}</span>
        <ChevronDown className="w-3 h-3 shrink-0 text-white/30" />
      </button>
      {open && (
        <div className="absolute top-full left-0 mt-1 w-64 max-h-80 overflow-y-auto bg-black border border-white/10 rounded shadow-xl z-50">
          {Object.entries(grouped).map(([provider, models]) => (
            <div key={provider}>
              <div className="px-3 py-1.5 text-[10px] font-mono text-white/30 uppercase tracking-wider sticky top-0 bg-black">
                {provider}
              </div>
              {models.map(m => (
                <button
                  key={m.id}
                  onClick={() => { onChange(m.id); setOpen(false); }}
                  className={`w-full text-left px-3 py-1.5 text-xs font-mono hover:bg-white/5 transition-colors ${m.id === value ? 'text-white bg-white/5' : 'text-white/60'}`}
                >
                  {m.label}
                </button>
              ))}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function ScrollAnchor({ messages, streaming }: { messages: Message[]; streaming: boolean }) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    ref.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages.length, messages[messages.length - 1]?.content, streaming]);
  return <div ref={ref} />;
}
