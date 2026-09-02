import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import CodeMirror from '@uiw/react-codemirror';
import { oneDark } from '@codemirror/theme-one-dark';
import { html } from '@codemirror/lang-html';
import { css } from '@codemirror/lang-css';
import { javascript } from '@codemirror/lang-javascript';
import { json } from '@codemirror/lang-json';
import { ArrowLeft, FilePlus, Trash2, Play, Save, Send, Loader2, Sparkles, X } from 'lucide-react';
import { Seo } from '../components/Seo';
import { getStoredAPIKey, getAPIBaseURL } from '../lib/session';
import {
  ArtifactFile,
  ArtifactInput,
  ARTIFACT_SYSTEM_PROMPT,
  buildPreviewDoc,
  createArtifact,
  getMine,
  isLoggedIn,
  langFromPath,
  mergeFiles,
  parseAgentFiles,
  updateArtifact,
} from '../lib/artifacts';

const STARTER: ArtifactFile[] = [
  {
    path: 'index.html',
    content: `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>My Artifact</title>
    <style>
      body { font-family: system-ui, sans-serif; display: grid; place-items: center; height: 100vh; margin: 0; background: #0b0b0f; color: #eaeaf0; }
      h1 { font-weight: 600; }
    </style>
  </head>
  <body>
    <h1>Hello from your artifact 👋</h1>
    <script>
      console.log('ready');
    </script>
  </body>
</html>`,
  },
];

const DEFAULT_MODELS = ['auto', 'claude-opus-4-8', 'claude-sonnet-4-6', 'gpt-5.5', 'gemini-3-pro', 'deepseek-v4-pro', 'kimi-k2'];

interface ChatMsg { role: 'user' | 'assistant'; content: string }

function langExt(path: string) {
  switch (langFromPath(path)) {
    case 'html': return [html()];
    case 'css': return [css()];
    case 'json': return [json()];
    case 'javascript': return [javascript({ jsx: true, typescript: true })];
    default: return [];
  }
}

export function ArtifactEditor({ isEdit }: { isEdit?: boolean }) {
  const { id } = useParams();
  const navigate = useNavigate();
  const loggedIn = isLoggedIn();

  const [files, setFiles] = useState<ArtifactFile[]>(STARTER);
  const [active, setActive] = useState('index.html');
  const [entry, setEntry] = useState('index.html');
  const [title, setTitle] = useState('Untitled');
  const [description, setDescription] = useState('');
  const [imageUrl, setImageUrl] = useState('');
  const [tags, setTags] = useState('');
  const [visibility, setVisibility] = useState<'private' | 'public' | 'unlisted'>('private');
  const [savedId, setSavedId] = useState<string | null>(isEdit && id ? id : null);
  const [loading, setLoading] = useState(isEdit);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const [model, setModel] = useState('auto');
  const [chat, setChat] = useState<ChatMsg[]>([]);
  const [prompt, setPrompt] = useState('');
  const [streaming, setStreaming] = useState(false);
  const abortRef = useRef<AbortController | null>(null);

  const [preview, setPreview] = useState(0); // bump to refresh iframe
  const previewDoc = useMemo(() => buildPreviewDoc(files, entry), [files, entry, preview]);
  const activeFile = files.find(f => f.path === active) || files[0];

  useEffect(() => {
    if (!isEdit || !id) return;
    let cancelled = false;
    (async () => {
      try {
        const a = await getMine(id);
        if (cancelled) return;
        setFiles(a.files && a.files.length ? a.files : STARTER);
        setActive((a.files && a.files[0]?.path) || 'index.html');
        setEntry(a.entry || 'index.html');
        setTitle(a.title);
        setDescription(a.description);
        setImageUrl(a.image_url);
        setTags((a.tags || []).join(', '));
        setVisibility(a.visibility);
        setSavedId(a.id);
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Failed to load');
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [isEdit, id]);

  const setContent = useCallback((value: string) => {
    setFiles(prev => prev.map(f => (f.path === active ? { ...f, content: value } : f)));
  }, [active]);

  const addFile = () => {
    const name = window.prompt('File name (e.g. style.css, app.js)');
    if (!name) return;
    const path = name.replace(/^\.?\//, '').trim();
    if (!path || files.some(f => f.path === path)) return;
    setFiles(prev => [...prev, { path, content: '' }]);
    setActive(path);
  };

  const removeFile = (path: string) => {
    if (files.length <= 1) return;
    setFiles(prev => prev.filter(f => f.path !== path));
    if (active === path) setActive(files.find(f => f.path !== path)!.path);
  };

  const buildInput = (): ArtifactInput => ({
    title: title.trim() || 'Untitled',
    description,
    image_url: imageUrl,
    files,
    entry,
    visibility,
    tags: tags.split(',').map(t => t.trim()).filter(Boolean),
  });

  const save = async (publish?: boolean) => {
    if (!loggedIn) { setError('Sign in to save artifacts.'); return; }
    setSaving(true);
    setError('');
    try {
      const input = buildInput();
      if (publish) input.visibility = visibility === 'private' ? 'public' : visibility;
      const result = savedId ? await updateArtifact(savedId, input) : await createArtifact(input);
      setSavedId(result.id);
      setVisibility(result.visibility);
      if (publish && result.visibility === 'public') {
        navigate(`/artifacts/${result.slug || result.id}`);
        return;
      }
      if (!savedId) navigate(`/artifacts/${result.id}/edit`, { replace: true });
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save');
    } finally {
      setSaving(false);
    }
  };

  const runAgent = async () => {
    const text = prompt.trim();
    if (!text || streaming) return;
    if (!loggedIn) { setError('Sign in to use the agent (it uses your credits).'); return; }
    setPrompt('');
    setError('');
    const history = [...chat, { role: 'user' as const, content: text }];
    setChat([...history, { role: 'assistant', content: '' }]);
    setStreaming(true);
    const controller = new AbortController();
    abortRef.current = controller;

    const filesContext = files.map(f => `=== ${f.path} ===\n${f.content}`).join('\n\n');
    const messages = [
      { role: 'system', content: ARTIFACT_SYSTEM_PROMPT },
      { role: 'user', content: `Current files:\n\n${filesContext}` },
      ...history.map(m => ({ role: m.role, content: m.content })),
    ];

    try {
      const resp = await fetch(`${getAPIBaseURL()}/chat/completions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${getStoredAPIKey()}` },
        body: JSON.stringify({ model, messages, stream: true, max_tokens: 8192 }),
        signal: controller.signal,
      });
      if (!resp.ok || !resp.body) {
        throw new Error(`${resp.status}: ${(await resp.text()).slice(0, 200)}`);
      }
      const reader = resp.body.getReader();
      const decoder = new TextDecoder();
      let acc = '';
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        for (const line of decoder.decode(value, { stream: true }).split('\n')) {
          if (!line.startsWith('data: ')) continue;
          const data = line.slice(6);
          if (data === '[DONE]') continue;
          try {
            const delta = JSON.parse(data).choices?.[0]?.delta?.content;
            if (delta) {
              acc += delta;
              setChat(prev => {
                const next = [...prev];
                next[next.length - 1] = { role: 'assistant', content: acc };
                return next;
              });
            }
          } catch { /* ignore partial */ }
        }
      }
      const { files: produced, prose } = parseAgentFiles(acc);
      if (produced.length) {
        setFiles(prev => {
          const merged = mergeFiles(prev, produced);
          return merged;
        });
        setActive(produced[0].path);
        setPreview(p => p + 1);
      }
      setChat(prev => {
        const next = [...prev];
        next[next.length - 1] = { role: 'assistant', content: prose || (produced.length ? `Updated ${produced.map(f => f.path).join(', ')}.` : acc) };
        return next;
      });
    } catch (e) {
      if ((e as Error).name === 'AbortError') return;
      setError(e instanceof Error ? e.message : 'Agent failed');
      setChat(prev => prev.slice(0, -1));
    } finally {
      setStreaming(false);
      abortRef.current = null;
    }
  };

  if (loading) {
    return <div className="flex h-[60vh] items-center justify-center text-white/55"><Loader2 className="h-6 w-6 animate-spin" /></div>;
  }

  return (
    <>
      <Seo title={`${title} - Artifact Editor | OpenPaths`} description="Build and edit a web artifact with an AI agent." path="/artifacts/new" />
      <div className="mx-auto max-w-[1600px] px-4 py-6">
        <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
          <button onClick={() => navigate('/artifacts')} className="inline-flex items-center gap-1.5 font-mono text-xs text-white/50 hover:text-white">
            <ArrowLeft className="h-3.5 w-3.5" /> Artifacts
          </button>
          <div className="flex items-center gap-2">
            <select value={visibility} onChange={e => setVisibility(e.target.value as any)} className="h-9 rounded-lg border border-white/20 bg-black px-2 font-mono text-xs text-white">
              <option value="private">Private</option>
              <option value="unlisted">Unlisted</option>
              <option value="public">Public</option>
            </select>
            <button onClick={() => save(false)} disabled={saving} className="inline-flex h-9 items-center gap-1.5 rounded-lg border border-white/15 bg-white/[0.07] px-3 font-mono text-xs text-white hover:bg-white/[0.11] disabled:opacity-50">
              {saving ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Save className="h-3.5 w-3.5" />} Save
            </button>
            <button onClick={() => save(true)} disabled={saving} className="inline-flex h-9 items-center gap-1.5 rounded-lg bg-cyan-300 px-3 font-mono text-xs font-semibold text-black hover:bg-cyan-200 disabled:opacity-50">
              Publish
            </button>
          </div>
        </div>

        {error && <div className="mb-3 rounded-lg border border-red-400/20 bg-red-500/10 p-3 font-mono text-xs text-red-200">{error}</div>}

        <div className="grid gap-4 lg:grid-cols-[200px_1fr_1fr] xl:grid-cols-[220px_1fr_1fr_360px]">
          {/* File tree */}
          <div className="rounded-lg border border-white/20 bg-white/[0.05] p-2">
            <div className="mb-2 flex items-center justify-between px-1">
              <span className="font-mono text-[10px] uppercase tracking-wider text-white/50">Files</span>
              <button onClick={addFile} title="Add file" className="text-white/55 hover:text-white"><FilePlus className="h-3.5 w-3.5" /></button>
            </div>
            <ul className="space-y-0.5">
              {files.map(f => (
                <li key={f.path} className={`group flex items-center justify-between rounded px-2 py-1 font-mono text-xs ${active === f.path ? 'bg-white/10 text-white' : 'text-white/55 hover:bg-white/[0.07]'}`}>
                  <button onClick={() => setActive(f.path)} className="flex-1 truncate text-left">{f.path}</button>
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100">
                    <button onClick={() => setEntry(f.path)} title="Set as preview entry" className={entry === f.path ? 'text-cyan-300' : 'text-white/45 hover:text-white'}><Play className="h-3 w-3" /></button>
                    {files.length > 1 && <button onClick={() => removeFile(f.path)} title="Delete" className="text-white/45 hover:text-red-300"><Trash2 className="h-3 w-3" /></button>}
                  </div>
                </li>
              ))}
            </ul>
          </div>

          {/* Editor */}
          <div className="overflow-hidden rounded-lg border border-white/20">
            <div className="border-b border-white/20 bg-white/[0.05] px-3 py-1.5 font-mono text-[11px] text-white/45">{active}</div>
            <CodeMirror
              value={activeFile?.content || ''}
              height="640px"
              theme={oneDark}
              extensions={langExt(active)}
              onChange={setContent}
              basicSetup={{ lineNumbers: true, foldGutter: true, highlightActiveLine: true }}
            />
          </div>

          {/* Preview */}
          <div className="overflow-hidden rounded-lg border border-white/20 bg-white">
            <div className="flex items-center justify-between border-b border-white/20 bg-white/[0.05] px-3 py-1.5">
              <span className="font-mono text-[11px] text-white/45">Preview · {entry}</span>
              <button onClick={() => setPreview(p => p + 1)} className="font-mono text-[11px] text-cyan-300 hover:text-cyan-200">Refresh</button>
            </div>
            <iframe key={preview} title="preview" srcDoc={previewDoc} sandbox="allow-scripts allow-modals allow-forms allow-popups" className="h-[640px] w-full bg-white" />
          </div>

          {/* Agent + meta */}
          <div className="flex flex-col gap-4 xl:row-span-1">
            <div className="flex flex-col rounded-lg border border-white/20 bg-white/[0.05]">
              <div className="flex items-center gap-2 border-b border-white/20 px-3 py-2 font-mono text-[11px] uppercase tracking-wider text-white/45">
                <Sparkles className="h-3.5 w-3.5 text-cyan-300" /> Agent
              </div>
              <div className="flex items-center gap-2 border-b border-white/20 px-3 py-2">
                <input list="artifact-models" value={model} onChange={e => setModel(e.target.value)} className="h-8 flex-1 rounded border border-white/20 bg-black px-2 font-mono text-xs text-white outline-none focus:border-cyan-300" />
                <datalist id="artifact-models">
                  {DEFAULT_MODELS.map(m => <option key={m} value={m} />)}
                </datalist>
              </div>
              <div className="max-h-[300px] min-h-[120px] flex-1 space-y-3 overflow-y-auto p-3">
                {chat.length === 0 && <p className="font-mono text-xs text-white/45">Describe what to build or change. The agent edits your files and uses your credits.</p>}
                {chat.map((m, i) => (
                  <div key={i} className={`rounded-lg px-3 py-2 text-xs leading-relaxed ${m.role === 'user' ? 'bg-cyan-300/10 text-white' : 'bg-white/[0.07] text-white/75'}`}>
                    {m.content || (streaming && i === chat.length - 1 ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : '')}
                  </div>
                ))}
              </div>
              <form onSubmit={e => { e.preventDefault(); void runAgent(); }} className="border-t border-white/20 p-2">
                <div className="flex items-end gap-2">
                  <textarea
                    value={prompt}
                    onChange={e => setPrompt(e.target.value)}
                    onKeyDown={e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); void runAgent(); } }}
                    rows={2}
                    placeholder="Build a snake game…"
                    className="flex-1 resize-none rounded-lg border border-white/20 bg-black px-2 py-1.5 font-mono text-xs text-white outline-none focus:border-cyan-300"
                  />
                  {streaming ? (
                    <button type="button" onClick={() => abortRef.current?.abort()} className="grid h-9 w-9 place-items-center rounded-lg border border-white/15 text-white/60 hover:text-white"><X className="h-4 w-4" /></button>
                  ) : (
                    <button type="submit" className="grid h-9 w-9 place-items-center rounded-lg bg-cyan-300 text-black hover:bg-cyan-200"><Send className="h-4 w-4" /></button>
                  )}
                </div>
              </form>
            </div>

            <div className="space-y-2 rounded-lg border border-white/20 bg-white/[0.05] p-3">
              <input value={title} onChange={e => setTitle(e.target.value)} placeholder="Title" className="w-full rounded border border-white/20 bg-black px-2 py-1.5 text-sm text-white outline-none focus:border-cyan-300" />
              <textarea value={description} onChange={e => setDescription(e.target.value)} placeholder="Description" rows={2} className="w-full resize-none rounded border border-white/20 bg-black px-2 py-1.5 text-xs text-white/80 outline-none focus:border-cyan-300" />
              <input value={imageUrl} onChange={e => setImageUrl(e.target.value)} placeholder="Cover image URL" className="w-full rounded border border-white/20 bg-black px-2 py-1.5 font-mono text-xs text-white/70 outline-none focus:border-cyan-300" />
              <input value={tags} onChange={e => setTags(e.target.value)} placeholder="tags, comma, separated" className="w-full rounded border border-white/20 bg-black px-2 py-1.5 font-mono text-xs text-white/70 outline-none focus:border-cyan-300" />
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
