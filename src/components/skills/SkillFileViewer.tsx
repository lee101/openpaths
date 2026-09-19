import React, { useMemo } from 'react';

// SkillFileViewer renders a skill attachment by kind: image/audio/video/pdf via
// native tags, markdown as formatted prose, text as a scrollable pre, office
// docs via a graceful fallback (no office deps in this bundle), and an
// unsupported state for everything else. Port of app-site's FilePreview,
// trimmed to the kinds the skills marketplace serves.

export type SkillPreviewKind = 'image' | 'audio' | 'video' | 'pdf' | 'markdown' | 'text' | 'office' | 'none';

const IMAGE_RE = /\.(png|jpe?g|gif|webp|svg|avif|ico|bmp)$/i;
const AUDIO_RE = /\.(mp3|wav|flac|ogg|m4a|aac)$/i;
const VIDEO_RE = /\.(mp4|mov|webm|mkv|avi)$/i;
const PDF_RE = /\.pdf$/i;
const MARKDOWN_RE = /\.(md|markdown|mdx)$/i;
const OFFICE_RE = /\.(docx?|xlsx?|csv|pptx?)$/i;

export function previewKindForSkillFile(path: string, mime: string): SkillPreviewKind {
  const m = (mime || '').toLowerCase();
  if (m.startsWith('image/')) return 'image';
  if (m.startsWith('audio/')) return 'audio';
  if (m.startsWith('video/')) return 'video';
  if (m === 'application/pdf') return 'pdf';
  if (m === 'text/markdown' || m === 'text/x-markdown') return 'markdown';
  if (IMAGE_RE.test(path)) return 'image';
  if (AUDIO_RE.test(path)) return 'audio';
  if (VIDEO_RE.test(path)) return 'video';
  if (PDF_RE.test(path)) return 'pdf';
  if (MARKDOWN_RE.test(path)) return 'markdown';
  if (OFFICE_RE.test(path)) return 'office';
  if (m.startsWith('text/') || m === 'application/json' || m === '') return 'text';
  return 'none';
}

function renderMarkdownInline(src: string): React.ReactNode[] {
  // Minimal inline markdown: `code`, **bold**, links, *italic*. Block structure
  // is handled below; this covers spans only.
  const parts: React.ReactNode[] = [];
  const re = /(`[^`]+`|\*\*[^*]+\*\*|\[[^\]]+\]\([^)]+\)|\*[^*]+\*)/g;
  let last = 0;
  let k = 0;
  for (const m of src.matchAll(re)) {
    const idx = m.index ?? 0;
    if (idx > last) parts.push(src.slice(last, idx));
    const tok = m[0];
    if (tok.startsWith('`')) parts.push(<code key={k++} className="rounded bg-white/10 px-1 font-mono text-[0.9em]">{tok.slice(1, -1)}</code>);
    else if (tok.startsWith('**')) parts.push(<strong key={k++}>{tok.slice(2, -2)}</strong>);
    else if (tok.startsWith('[')) {
      const label = tok.slice(1, tok.indexOf(']'));
      const href = tok.slice(tok.indexOf('(') + 1, -1);
      parts.push(<a key={k++} href={href} target="_blank" rel="noreferrer" className="text-sky-300 underline">{label}</a>);
    } else parts.push(<em key={k++}>{tok.slice(1, -1)}</em>);
    last = idx + tok.length;
  }
  if (last < src.length) parts.push(src.slice(last));
  return parts;
}

export function SkillMarkdown({ source }: { source: string }) {
  const blocks = useMemo(() => {
    const lines = source.split('\n');
    const out: React.ReactNode[] = [];
    let i = 0;
    let k = 0;
    while (i < lines.length) {
      const line = lines[i];
      if (/^```/.test(line)) {
        const lang = line.slice(3).trim();
        const buf: string[] = [];
        i++;
        while (i < lines.length && !/^```/.test(lines[i])) buf.push(lines[i++]);
        i++;
        out.push(
          <pre key={k++} className="overflow-x-auto rounded border border-white/15 bg-black/60 p-3 font-mono text-xs leading-relaxed text-white/85">
            {lang && <div className="mb-1 font-mono text-[10px] uppercase tracking-wide text-white/40">{lang}</div>}
            {buf.join('\n')}
          </pre>,
        );
        continue;
      }
      const h = /^(#{1,4})\s+(.*)/.exec(line);
      if (h) {
        const level = h[1].length;
        const cls = level === 1 ? 'text-xl font-bold' : level === 2 ? 'text-lg font-bold' : 'text-base font-semibold';
        out.push(<div key={k++} className={`${cls} mt-4 text-white`}>{renderMarkdownInline(h[2])}</div>);
        i++;
        continue;
      }
      if (/^\s*[-*]\s+/.test(line)) {
        const items: string[] = [];
        while (i < lines.length && /^\s*[-*]\s+/.test(lines[i])) items.push(lines[i].replace(/^\s*[-*]\s+/, ''));
        out.push(<ul key={k++} className="list-disc space-y-1 pl-5">{items.map((t, j) => <li key={j}>{renderMarkdownInline(t)}</li>)}</ul>);
        continue;
      }
      if (/^\s*$/.test(line)) { i++; continue; }
      out.push(<p key={k++} className="leading-relaxed">{renderMarkdownInline(line)}</p>);
      i++;
    }
    return out;
  }, [source]);
  return <div className="space-y-2 text-sm text-white/80">{blocks}</div>;
}

export function SkillFileViewer({ path, mime, src, text, className = '' }: {
  path: string;
  mime: string;
  src?: string;
  text?: string;
  className?: string;
}) {
  const kind = previewKindForSkillFile(path, mime);

  if (kind === 'image' && src) {
    return (
      <div className={`grid place-items-center bg-black p-6 ${className}`}>
        <img src={src} alt={path} className="max-h-[60vh] max-w-full rounded border border-white/15 bg-white/5" />
      </div>
    );
  }
  if (kind === 'audio' && src) {
    return (
      <div className={`grid place-items-center bg-black p-6 ${className}`}>
        <audio src={src} controls className="w-full max-w-2xl" />
      </div>
    );
  }
  if (kind === 'video' && src) {
    return (
      <div className={`grid place-items-center bg-black p-4 ${className}`}>
        <video src={src} controls className="max-h-[60vh] max-w-full rounded border border-white/15 bg-black" />
      </div>
    );
  }
  if (kind === 'pdf' && src) {
    return (
      <object data={src} type="application/pdf" className={`min-h-[60vh] w-full bg-black ${className}`}>
        <iframe title={path} src={src} className="h-[60vh] w-full bg-black" />
      </object>
    );
  }
  if (kind === 'markdown' && text != null) {
    return (
      <div className={`overflow-auto bg-black px-6 py-4 ${className}`}>
        <div className="mx-auto max-w-3xl pb-6"><SkillMarkdown source={text} /></div>
      </div>
    );
  }
  if (kind === 'text' && text != null) {
    return <pre className={`min-h-[240px] overflow-auto whitespace-pre-wrap bg-black p-5 font-mono text-sm leading-6 text-white/80 ${className}`}>{text}</pre>;
  }
  if (kind === 'office') {
    return (
      <div className={`grid min-h-[240px] place-items-center bg-black px-4 py-10 text-center ${className}`}>
        <div>
          <p className="font-mono text-sm text-white/70">{path}</p>
          <p className="mt-2 text-sm text-white/45">Office preview isn't available in this browser view.</p>
          {src && <a href={src} download={path} className="mt-3 inline-block rounded border border-white/20 px-3 py-1.5 font-mono text-xs text-white/70 hover:border-white/50 hover:text-white">Download file</a>}
        </div>
      </div>
    );
  }
  return (
    <div className={`grid min-h-[240px] place-items-center bg-black px-4 py-10 text-center font-mono text-sm text-white/45 ${className}`}>
      No preview for this file type ({path.split('.').pop() || 'unknown'}).
      {src && <a href={src} download={path} className="mt-3 rounded border border-white/20 px-3 py-1.5 text-xs text-white/70 hover:border-white/50 hover:text-white">Download</a>}
    </div>
  );
}
