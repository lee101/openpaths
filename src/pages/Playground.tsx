import React, { useState, useRef, useEffect, useCallback, useMemo } from 'react';
import { Send, Plus, X, Settings, ChevronDown, Loader2, Trash2, Square, Copy, Check, Zap, RotateCcw } from 'lucide-react';

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
  tokensUsed: number | null;
}

const CHAT_MODELS = [
  { id: 'auto', label: 'Auto (intelligent routing)', provider: 'OpenPaths' },
  { id: 'auto-easy-task', label: 'Auto Easy (cheapest)', provider: 'OpenPaths' },
  { id: 'auto-medium-task', label: 'Auto Medium (balanced)', provider: 'OpenPaths' },
  { id: 'auto-think', label: 'Auto Think (reasoning)', provider: 'OpenPaths' },
  { id: 'gemini-3.1-pro-preview', label: 'Gemini 3.1 Pro', provider: 'Google' },
  { id: 'gemini-2.5-pro', label: 'Gemini 2.5 Pro', provider: 'Google' },
  { id: 'gemini-2.5-flash', label: 'Gemini 2.5 Flash', provider: 'Google' },
  { id: 'gpt-5.4', label: 'GPT-5.4', provider: 'OpenAI' },
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

const QUICK_PROMPTS = [
  'Explain quantum computing in simple terms',
  'Write a Python function to find prime numbers',
  'Compare REST vs GraphQL APIs',
  'Create a React hook for debouncing',
];

let paneCounter = 0;
function makePane(modelId: string): ModelPane {
  return { id: `pane-${++paneCounter}`, modelId, messages: [], streaming: false, error: null, latencyMs: null, tokensUsed: null };
}

// --- Minimal markdown renderer ---

function renderMarkdown(text: string): React.ReactNode[] {
  const nodes: React.ReactNode[] = [];
  const lines = text.split('\n');
  let i = 0;

  while (i < lines.length) {
    const line = lines[i];

    // Fenced code block
    if (line.startsWith('```')) {
      const lang = line.slice(3).trim();
      const codeLines: string[] = [];
      i++;
      while (i < lines.length && !lines[i].startsWith('```')) {
        codeLines.push(lines[i]);
        i++;
      }
      i++; // skip closing ```
      const code = codeLines.join('\n');
      nodes.push(<CodeBlock key={nodes.length} code={code} lang={lang} />);
      continue;
    }

    // Heading
    const headingMatch = line.match(/^(#{1,3})\s+(.+)/);
    if (headingMatch) {
      const level = headingMatch[1].length;
      const text = headingMatch[2];
      const cls = level === 1 ? 'text-lg font-bold mt-4 mb-2' : level === 2 ? 'text-base font-bold mt-3 mb-1.5' : 'text-sm font-bold mt-2 mb-1';
      nodes.push(<div key={nodes.length} className={cls}>{renderInline(text)}</div>);
      i++;
      continue;
    }

    // Bullet list
    if (line.match(/^[\s]*[-*]\s/)) {
      const items: string[] = [];
      while (i < lines.length && lines[i].match(/^[\s]*[-*]\s/)) {
        items.push(lines[i].replace(/^[\s]*[-*]\s/, ''));
        i++;
      }
      nodes.push(
        <ul key={nodes.length} className="list-disc list-inside space-y-0.5 my-1">
          {items.map((item, j) => <li key={j} className="text-sm leading-relaxed">{renderInline(item)}</li>)}
        </ul>
      );
      continue;
    }

    // Numbered list
    if (line.match(/^[\s]*\d+\.\s/)) {
      const items: string[] = [];
      while (i < lines.length && lines[i].match(/^[\s]*\d+\.\s/)) {
        items.push(lines[i].replace(/^[\s]*\d+\.\s/, ''));
        i++;
      }
      nodes.push(
        <ol key={nodes.length} className="list-decimal list-inside space-y-0.5 my-1">
          {items.map((item, j) => <li key={j} className="text-sm leading-relaxed">{renderInline(item)}</li>)}
        </ol>
      );
      continue;
    }

    // Empty line
    if (line.trim() === '') {
      i++;
      continue;
    }

    // Regular paragraph - collect consecutive non-empty lines
    const paraLines: string[] = [];
    while (i < lines.length && lines[i].trim() !== '' && !lines[i].startsWith('```') && !lines[i].match(/^#{1,3}\s/) && !lines[i].match(/^[\s]*[-*]\s/) && !lines[i].match(/^[\s]*\d+\.\s/)) {
      paraLines.push(lines[i]);
      i++;
    }
    nodes.push(
      <p key={nodes.length} className="text-sm leading-relaxed my-1">
        {renderInline(paraLines.join('\n'))}
      </p>
    );
  }

  return nodes;
}

function renderInline(text: string): React.ReactNode[] {
  const parts: React.ReactNode[] = [];
  // Match: **bold**, `code`, *italic*
  const regex = /(\*\*(.+?)\*\*|`([^`]+)`|\*(.+?)\*)/g;
  let lastIndex = 0;
  let match;

  while ((match = regex.exec(text)) !== null) {
    if (match.index > lastIndex) {
      parts.push(text.slice(lastIndex, match.index));
    }
    if (match[2]) {
      parts.push(<strong key={parts.length} className="font-semibold">{match[2]}</strong>);
    } else if (match[3]) {
      parts.push(<code key={parts.length} className="bg-white/10 px-1.5 py-0.5 rounded text-[13px] font-mono">{match[3]}</code>);
    } else if (match[4]) {
      parts.push(<em key={parts.length}>{match[4]}</em>);
    }
    lastIndex = match.index + match[0].length;
  }

  if (lastIndex < text.length) {
    parts.push(text.slice(lastIndex));
  }

  return parts.length > 0 ? parts : [text];
}

function CodeBlock({ code, lang }: { code: string; lang: string; key?: React.Key }) {
  const [copied, setCopied] = useState(false);

  const copy = () => {
    navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="my-2 rounded-lg border border-white/10 overflow-hidden bg-white/[0.03]">
      <div className="flex items-center justify-between px-3 py-1.5 border-b border-white/10 bg-white/[0.02]">
        <span className="text-[10px] font-mono text-white/30 uppercase">{lang || 'code'}</span>
        <button onClick={copy} className="text-white/30 hover:text-white/60 transition-colors p-0.5">
          {copied ? <Check className="w-3 h-3 text-green-400" /> : <Copy className="w-3 h-3" />}
        </button>
      </div>
      <pre className="p-3 overflow-x-auto text-[13px] font-mono leading-relaxed text-white/80">
        <code>{code}</code>
      </pre>
    </div>
  );
}

// --- Main component ---

export function Playground() {
  const [apiKey, setApiKey] = useState(() => localStorage.getItem('op_api_key') || '');
  const [systemPrompt, setSystemPrompt] = useState('You are a helpful assistant.');
  const [temperature, setTemperature] = useState(0.7);
  const [maxTokens, setMaxTokens] = useState(4096);
  const [showSettings, setShowSettings] = useState(false);
  const [input, setInput] = useState('');
  const [panes, setPanes] = useState<ModelPane[]>([makePane('auto')]);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const abortRefs = useRef<Map<string, AbortController>>(new Map());

  const anyStreaming = panes.some(p => p.streaming);

  useEffect(() => {
    const t = setTimeout(() => inputRef.current?.focus(), 100);
    return () => clearTimeout(t);
  }, []);

  useEffect(() => {
    if (apiKey) {
      localStorage.setItem('op_api_key', apiKey);
    }
  }, [apiKey]);

  const baseUrl = window.location.origin;

  const sendToModel = useCallback(async (paneId: string, modelId: string, messages: Message[]) => {
    const controller = new AbortController();
    abortRefs.current.set(paneId, controller);

    const start = performance.now();

    setPanes(prev => prev.map(p =>
      p.id === paneId ? { ...p, streaming: true, error: null, latencyMs: null, tokensUsed: null } : p
    ));

    try {
      const body = {
        model: modelId,
        messages: [
          ...(systemPrompt ? [{ role: 'system', content: systemPrompt }] : []),
          ...messages,
        ],
        stream: true,
        temperature,
        max_tokens: maxTokens,
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
      let totalTokens: number | null = null;

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
            if (parsed.usage?.total_tokens) {
              totalTokens = parsed.usage.total_tokens;
            }
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
        p.id === paneId ? { ...p, streaming: false, tokensUsed: totalTokens } : p
      ));
    } catch (err: any) {
      if (err.name === 'AbortError') {
        setPanes(prev => prev.map(p =>
          p.id === paneId ? { ...p, streaming: false } : p
        ));
        return;
      }
      setPanes(prev => prev.map(p =>
        p.id === paneId ? { ...p, streaming: false, error: err.message } : p
      ));
    } finally {
      abortRefs.current.delete(paneId);
    }
  }, [apiKey, baseUrl, systemPrompt, temperature, maxTokens]);

  function handleSend(text?: string) {
    const msg = (text || input).trim();
    if (!msg || !apiKey) return;

    const userMsg: Message = { role: 'user', content: msg };
    if (!text) setInput('');
    setTimeout(() => inputRef.current?.focus(), 0);

    const updatedPanes = panes.map(p => ({
      ...p,
      messages: [...p.messages, userMsg],
    }));
    setPanes(updatedPanes);

    for (const pane of updatedPanes) {
      sendToModel(pane.id, pane.modelId, pane.messages);
    }
  }

  function stopAll() {
    for (const ctrl of abortRefs.current.values()) ctrl.abort();
    abortRefs.current.clear();
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
    stopAll();
    setPanes(prev => prev.map(p => ({ ...p, messages: [], streaming: false, error: null, latencyMs: null, tokensUsed: null })));
  }

  function retryLast(paneId: string) {
    const pane = panes.find(p => p.id === paneId);
    if (!pane) return;
    // Remove last assistant message and re-send
    const msgs = pane.messages.filter((_, i) => !(i === pane.messages.length - 1 && pane.messages[i].role === 'assistant'));
    setPanes(prev => prev.map(p => p.id === paneId ? { ...p, messages: msgs, error: null } : p));
    sendToModel(paneId, pane.modelId, msgs);
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  }

  const hasMessages = panes.some(p => p.messages.length > 0);

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="border-b border-white/10 px-4 py-2.5 flex items-center gap-2 bg-white/[0.02]">
        <button
          onClick={() => setShowSettings(!showSettings)}
          className={`flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-mono rounded border transition-colors ${showSettings ? 'border-white/30 bg-white/10 text-white' : 'border-white/10 text-white/60 hover:text-white hover:border-white/20'}`}
        >
          <Settings className="w-3.5 h-3.5" /> Settings
        </button>
        <button
          onClick={addPane}
          disabled={panes.length >= 4}
          className="flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-mono rounded border border-white/10 text-white/60 hover:text-white hover:border-white/20 transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
        >
          <Plus className="w-3.5 h-3.5" /> Compare
        </button>
        {hasMessages && (
          <button
            onClick={clearAll}
            className="flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-mono rounded border border-white/10 text-white/60 hover:text-white hover:border-white/20 transition-colors"
          >
            <Trash2 className="w-3.5 h-3.5" /> Clear
          </button>
        )}
        <div className="ml-auto flex items-center gap-3">
          {panes.length > 1 && (
            <span className="text-[10px] font-mono text-white/25">{panes.length}/4 models</span>
          )}
          <span className="text-[10px] font-mono text-white/25">
            temp {temperature} | max {maxTokens}
          </span>
        </div>
      </div>

      {/* Settings Panel */}
      {showSettings && (
        <div className="border-b border-white/10 px-4 py-4 bg-white/[0.02]">
          <div className="max-w-2xl grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="text-[10px] font-mono text-white/40 uppercase tracking-wider block mb-1.5">API Key</label>
              <input
                type="password"
                value={apiKey}
                onChange={e => setApiKey(e.target.value)}
                placeholder="op-..."
                className="w-full bg-black border border-white/10 rounded px-3 py-2 text-sm font-mono text-white placeholder:text-white/20 focus:outline-none focus:border-white/30"
              />
            </div>
            <div>
              <label className="text-[10px] font-mono text-white/40 uppercase tracking-wider block mb-1.5">System Prompt</label>
              <textarea
                value={systemPrompt}
                onChange={e => setSystemPrompt(e.target.value)}
                rows={2}
                className="w-full bg-black border border-white/10 rounded px-3 py-2 text-sm font-mono text-white placeholder:text-white/20 focus:outline-none focus:border-white/30 resize-none"
                placeholder="You are a helpful assistant."
              />
            </div>
            <div>
              <label className="text-[10px] font-mono text-white/40 uppercase tracking-wider block mb-1.5">
                Temperature <span className="text-white/60">{temperature}</span>
              </label>
              <input
                type="range"
                min="0"
                max="2"
                step="0.1"
                value={temperature}
                onChange={e => setTemperature(parseFloat(e.target.value))}
                className="w-full accent-white h-1"
              />
              <div className="flex justify-between text-[9px] font-mono text-white/20 mt-0.5">
                <span>Precise</span><span>Creative</span>
              </div>
            </div>
            <div>
              <label className="text-[10px] font-mono text-white/40 uppercase tracking-wider block mb-1.5">
                Max Tokens <span className="text-white/60">{maxTokens.toLocaleString()}</span>
              </label>
              <input
                type="range"
                min="256"
                max="16384"
                step="256"
                value={maxTokens}
                onChange={e => setMaxTokens(parseInt(e.target.value))}
                className="w-full accent-white h-1"
              />
              <div className="flex justify-between text-[9px] font-mono text-white/20 mt-0.5">
                <span>256</span><span>16,384</span>
              </div>
            </div>
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
              <div className="flex items-center gap-2 shrink-0 ml-auto">
                {pane.latencyMs !== null && (
                  <span className="text-[10px] font-mono text-green-400/70" title="Time to first token">
                    <Zap className="w-2.5 h-2.5 inline mr-0.5" />{pane.latencyMs}ms
                  </span>
                )}
                {pane.tokensUsed !== null && !pane.streaming && (
                  <span className="text-[10px] font-mono text-white/30" title="Total tokens">
                    {pane.tokensUsed.toLocaleString()} tok
                  </span>
                )}
                {pane.streaming && (
                  <Loader2 className="w-3 h-3 animate-spin text-white/40" />
                )}
                {panes.length > 1 && (
                  <button onClick={() => removePane(pane.id)} className="text-white/20 hover:text-white/60">
                    <X className="w-3.5 h-3.5" />
                  </button>
                )}
              </div>
            </div>

            {/* Messages */}
            <div className="flex-1 overflow-y-auto">
              {pane.messages.length === 0 && !pane.error ? (
                <EmptyState onPrompt={handleSend} hasApiKey={!!apiKey} />
              ) : (
                <div className="p-3 space-y-4">
                  {pane.messages.map((msg, i) => (
                    <MessageBubble key={i} message={msg} />
                  ))}
                  {pane.error && (
                    <div className="flex items-start gap-2">
                      <div className="flex-1 text-xs font-mono text-red-400/80 bg-red-400/5 rounded-lg p-3 border border-red-400/10">
                        {pane.error}
                      </div>
                      <button onClick={() => retryLast(pane.id)} className="shrink-0 p-1.5 text-white/30 hover:text-white/60 transition-colors" title="Retry">
                        <RotateCcw className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  )}
                  <ScrollAnchor messages={pane.messages} streaming={pane.streaming} />
                </div>
              )}
            </div>
          </div>
        ))}
      </div>

      {/* Input */}
      <div className="border-t border-white/10 p-3 bg-white/[0.02]">
        <div className="max-w-4xl mx-auto flex gap-2 items-end">
          <textarea
            ref={inputRef}
            value={input}
            onChange={e => {
              setInput(e.target.value);
              const el = e.target;
              el.style.height = 'auto';
              el.style.height = Math.min(el.scrollHeight, 200) + 'px';
            }}
            onKeyDown={handleKeyDown}
            rows={1}
            placeholder={apiKey ? 'Send a message... (Shift+Enter for newline)' : 'Set your API key in Settings first'}
            disabled={!apiKey}
            autoFocus
            data-testid="chat-input"
            className="flex-1 bg-black border border-white/10 rounded-lg px-4 py-3 text-sm text-white placeholder:text-white/20 focus:outline-none focus:border-white/30 focus:ring-1 focus:ring-white/10 resize-none disabled:opacity-30 transition-colors"
            style={{ minHeight: '44px', maxHeight: '200px' }}
          />
          {anyStreaming ? (
            <button
              onClick={stopAll}
              data-testid="chat-stop"
              className="bg-red-500/20 text-red-400 border border-red-500/30 px-4 py-3 rounded-lg font-mono text-sm font-bold hover:bg-red-500/30 transition-colors shrink-0"
              title="Stop generation"
            >
              <Square className="w-4 h-4" />
            </button>
          ) : (
            <button
              onClick={() => handleSend()}
              disabled={!input.trim() || !apiKey}
              data-testid="chat-send"
              className="bg-white text-black px-4 py-3 rounded-lg font-mono text-sm font-bold hover:bg-white/90 transition-colors disabled:opacity-30 disabled:cursor-not-allowed shrink-0"
            >
              <Send className="w-4 h-4" />
            </button>
          )}
        </div>
        {!apiKey && (
          <p className="text-center text-xs font-mono text-white/30 mt-2">
            <button onClick={() => setShowSettings(true)} className="underline hover:text-white/60 transition-colors">Set your API key</button> or <a href="/account" className="underline hover:text-white/60 transition-colors">create an account</a> to get started
          </p>
        )}
      </div>
    </div>
  );
}

// --- Sub-components ---

function EmptyState({ onPrompt, hasApiKey }: { onPrompt: (text: string) => void; hasApiKey: boolean }) {
  return (
    <div className="h-full flex flex-col items-center justify-center px-6 py-12">
      <div className="text-white/10 mb-6">
        <Zap className="w-10 h-10" />
      </div>
      <p className="text-sm font-mono text-white/20 mb-6">Try a prompt to get started</p>
      {hasApiKey && (
        <div className="flex flex-wrap gap-2 justify-center max-w-md">
          {QUICK_PROMPTS.map((prompt, i) => (
            <button
              key={i}
              onClick={() => onPrompt(prompt)}
              className="text-xs font-mono text-white/40 border border-white/10 rounded-lg px-3 py-2 hover:border-white/25 hover:text-white/60 hover:bg-white/[0.02] transition-colors text-left"
            >
              {prompt}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

function MessageBubble({ message }: { message: Message; key?: React.Key }) {
  const [copied, setCopied] = useState(false);
  const isUser = message.role === 'user';

  const copy = () => {
    navigator.clipboard.writeText(message.content);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className={`group ${isUser ? 'flex justify-end' : ''}`}>
      <div className={`relative ${isUser ? 'max-w-[85%]' : 'w-full'}`}>
        {isUser ? (
          <div className="bg-white/10 rounded-2xl rounded-br-sm px-4 py-2.5 text-sm leading-relaxed">
            {message.content}
          </div>
        ) : (
          <div className="text-sm leading-relaxed text-white/90">
            {renderMarkdown(message.content)}
          </div>
        )}
        <button
          onClick={copy}
          className="absolute -right-1 top-0 p-1 opacity-0 group-hover:opacity-100 transition-opacity text-white/20 hover:text-white/50"
          title="Copy message"
        >
          {copied ? <Check className="w-3 h-3 text-green-400" /> : <Copy className="w-3 h-3" />}
        </button>
      </div>
    </div>
  );
}

function ModelSelect({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState('');
  const ref = useRef<HTMLDivElement>(null);
  const searchRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, []);

  useEffect(() => {
    if (open) {
      setSearch('');
      setTimeout(() => searchRef.current?.focus(), 50);
    }
  }, [open]);

  const current = CHAT_MODELS.find(m => m.id === value);

  const filtered = useMemo(() => {
    if (!search) return CHAT_MODELS;
    const q = search.toLowerCase();
    return CHAT_MODELS.filter(m => m.label.toLowerCase().includes(q) || m.provider.toLowerCase().includes(q) || m.id.toLowerCase().includes(q));
  }, [search]);

  const grouped: Record<string, typeof CHAT_MODELS> = useMemo(() => {
    return filtered.reduce<Record<string, typeof CHAT_MODELS>>((acc, m) => {
      (acc[m.provider] ||= []).push(m);
      return acc;
    }, {});
  }, [filtered]);

  return (
    <div ref={ref} className="relative min-w-0">
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-1 text-xs font-mono text-white/80 hover:text-white truncate"
      >
        <span className="truncate">{current?.label || value}</span>
        <ChevronDown className={`w-3 h-3 shrink-0 text-white/30 transition-transform ${open ? 'rotate-180' : ''}`} />
      </button>
      {open && (
        <div className="absolute top-full left-0 mt-1 w-72 max-h-80 overflow-hidden bg-black border border-white/10 rounded-lg shadow-xl z-50 flex flex-col">
          <div className="p-2 border-b border-white/10">
            <input
              ref={searchRef}
              value={search}
              onChange={e => setSearch(e.target.value)}
              placeholder="Search models..."
              className="w-full bg-white/5 border-none rounded px-2.5 py-1.5 text-xs font-mono text-white placeholder:text-white/20 focus:outline-none"
            />
          </div>
          <div className="overflow-y-auto flex-1">
            {Object.keys(grouped).length === 0 ? (
              <div className="px-3 py-4 text-xs font-mono text-white/30 text-center">No models found</div>
            ) : (
              Object.entries(grouped).map(([provider, models]) => (
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
              ))
            )}
          </div>
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
