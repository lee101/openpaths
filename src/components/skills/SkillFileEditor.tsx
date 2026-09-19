import React, { useEffect, useMemo, useRef, useState } from 'react';
import hljs from 'highlight.js/lib/common';

// SkillFileEditor is a syntax-highlighted text editor: a transparent <textarea>
// layered over a highlighted <pre> mirroring its content. Port of app-site's
// CodeEditor (textarea-over-highlight); both layers share font/padding metrics
// so the caret lands on the painted text. No editor dependency.

const LAYER = 'm-0 min-h-full min-w-full whitespace-pre p-3 font-mono text-xs leading-relaxed';

const ALIASES: Record<string, string> = {
  js: 'javascript', jsx: 'javascript', py: 'python', shell: 'bash', sh: 'bash',
  ts: 'typescript', tsx: 'typescript', yml: 'yaml', zsh: 'bash', md: 'markdown',
};

function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

export function SkillFileEditor({ value, onChange, lang, className = '', ariaLabel }: {
  value: string;
  onChange: (next: string) => void;
  lang?: string;
  className?: string;
  ariaLabel?: string;
}) {
  const normalized = (lang || '').trim().toLowerCase();
  const language = ALIASES[normalized] || normalized;
  const [html, setHtml] = useState(() => escapeHtml(value));
  const [scrollTop, setScrollTop] = useState(0);
  const taRef = useRef<HTMLTextAreaElement>(null);
  const preRef = useRef<HTMLPreElement>(null);
  const lineCount = useMemo(() => Math.max(1, value.split('\n').length), [value]);
  const lineNumbers = useMemo(() => Array.from({ length: lineCount }, (_, i) => i + 1), [lineCount]);
  const gutterWidth = `${Math.max(3, String(lineCount).length + 2)}ch`;

  useEffect(() => {
    const padded = value.endsWith('\n') ? `${value} ` : value;
    setHtml(escapeHtml(padded));
    if (!language) return;
    try {
      const out = hljs.getLanguage(language)
        ? hljs.highlight(padded, { language }).value
        : hljs.highlightAuto(padded).value;
      setHtml(out);
    } catch {
      /* keep escaped fallback */
    }
  }, [value, language]);

  const syncScroll = () => {
    if (taRef.current && preRef.current) {
      preRef.current.scrollTop = taRef.current.scrollTop;
      preRef.current.scrollLeft = taRef.current.scrollLeft;
      setScrollTop(taRef.current.scrollTop);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    const ta = taRef.current;
    if (e.key === 'Tab' && !e.shiftKey && !e.altKey && !e.ctrlKey && !e.metaKey) {
      e.preventDefault();
      if (!ta) return;
      const start = ta.selectionStart;
      const end = ta.selectionEnd;
      onChange(value.slice(0, start) + '  ' + value.slice(end));
      window.requestAnimationFrame(() => {
        ta.selectionStart = ta.selectionEnd = start + 2;
      });
    }
  };

  return (
    <div className={`relative overflow-hidden ${className}`}>
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-y-0 left-0 z-10 overflow-hidden border-r border-white/10 bg-black/60 p-3 pr-2 text-right font-mono text-xs leading-relaxed text-white/30"
        style={{ width: gutterWidth }}
      >
        <div style={{ transform: `translateY(-${scrollTop}px)` }}>
          {lineNumbers.map(n => <div key={n}>{n}</div>)}
        </div>
      </div>
      <pre
        ref={preRef}
        aria-hidden="true"
        className={`hljs pointer-events-none absolute inset-0 overflow-auto bg-transparent ${LAYER}`}
        style={{ paddingLeft: `calc(${gutterWidth} + 0.75rem)` }}
      >
        <code className="bg-transparent p-0" dangerouslySetInnerHTML={{ __html: html }} />
      </pre>
      <textarea
        ref={taRef}
        value={value}
        onChange={e => onChange(e.target.value)}
        onScroll={syncScroll}
        onKeyDown={handleKeyDown}
        spellCheck={false}
        aria-label={ariaLabel}
        wrap="off"
        style={{ paddingLeft: `calc(${gutterWidth} + 0.75rem)` }}
        className={`absolute inset-0 h-full w-full resize-none overflow-auto bg-transparent text-transparent caret-white outline-none ${LAYER}`}
      />
    </div>
  );
}
