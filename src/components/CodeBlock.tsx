import hljs from 'highlight.js/lib/core';
import bash from 'highlight.js/lib/languages/bash';
import python from 'highlight.js/lib/languages/python';
import javascript from 'highlight.js/lib/languages/javascript';
import typescript from 'highlight.js/lib/languages/typescript';
import json from 'highlight.js/lib/languages/json';
import go from 'highlight.js/lib/languages/go';
import powershell from 'highlight.js/lib/languages/powershell';
import markdown from 'highlight.js/lib/languages/markdown';
import yaml from 'highlight.js/lib/languages/yaml';
import xml from 'highlight.js/lib/languages/xml';
import sql from 'highlight.js/lib/languages/sql';
import css from 'highlight.js/lib/languages/css';
import diff from 'highlight.js/lib/languages/diff';
import shell from 'highlight.js/lib/languages/shell';
import ini from 'highlight.js/lib/languages/ini';
import rust from 'highlight.js/lib/languages/rust';
import java from 'highlight.js/lib/languages/java';
import csharp from 'highlight.js/lib/languages/csharp';
import plaintext from 'highlight.js/lib/languages/plaintext';

hljs.registerLanguage('bash', bash);
hljs.registerLanguage('python', python);
hljs.registerLanguage('javascript', javascript);
hljs.registerLanguage('typescript', typescript);
hljs.registerLanguage('json', json);
hljs.registerLanguage('go', go);
hljs.registerLanguage('powershell', powershell);
hljs.registerLanguage('markdown', markdown);
hljs.registerLanguage('yaml', yaml);
hljs.registerLanguage('xml', xml);
hljs.registerLanguage('sql', sql);
hljs.registerLanguage('css', css);
hljs.registerLanguage('diff', diff);
hljs.registerLanguage('shell', shell);
hljs.registerLanguage('ini', ini);
hljs.registerLanguage('rust', rust);
hljs.registerLanguage('java', java);
hljs.registerLanguage('csharp', csharp);
hljs.registerLanguage('plaintext', plaintext);

type CodeBlockProps = {
  code: string;
  language?: string;
  label?: string;
  containerClassName?: string;
  headerClassName?: string;
  preClassName?: string;
  codeClassName?: string;
  testId?: string;
  hideLabel?: boolean;
};

const LANGUAGE_ALIASES: Record<string, string> = {
  csharp: 'csharp',
  js: 'javascript',
  jsx: 'javascript',
  py: 'python',
  shell: 'bash',
  sh: 'bash',
  ts: 'typescript',
  tsx: 'typescript',
  yml: 'yaml',
  zsh: 'bash',
};

function joinClasses(...classes: Array<string | undefined>) {
  return classes.filter(Boolean).join(' ');
}

function normalizeLanguage(language?: string) {
  if (!language) {
    return '';
  }

  const normalized = language.trim().toLowerCase();
  return LANGUAGE_ALIASES[normalized] || normalized;
}

function highlightCode(code: string, language?: string) {
  const normalizedLanguage = normalizeLanguage(language);

  if (normalizedLanguage && hljs.getLanguage(normalizedLanguage)) {
    return {
      html: hljs.highlight(code, {
        ignoreIllegals: true,
        language: normalizedLanguage,
      }).value,
      language: normalizedLanguage,
    };
  }

  return {
    html: hljs.highlightAuto(code).value,
    language: normalizedLanguage,
  };
}

export function CodeBlock({
  code,
  language,
  label,
  containerClassName,
  headerClassName,
  preClassName,
  codeClassName,
  testId,
  hideLabel,
}: CodeBlockProps) {
  const highlighted = highlightCode(code, language);
  const resolvedLabel = hideLabel ? '' : (label || '');

  return (
    <div className={containerClassName}>
      {resolvedLabel && (
        <div
          className={joinClasses(
            'px-4 py-2 border-b border-white/20 text-[10px] font-mono uppercase tracking-[0.18em] text-white/45',
            headerClassName,
          )}
        >
          {resolvedLabel}
        </div>
      )}
      <pre
        className={joinClasses(
          'overflow-x-auto p-4 text-sm leading-relaxed',
          preClassName,
        )}
        data-testid={testId}
      >
        <code
          tabIndex={0}
          className={joinClasses(
            'hljs block min-w-full font-mono',
            highlighted.language ? `language-${highlighted.language}` : undefined,
            codeClassName,
          )}
          dangerouslySetInnerHTML={{ __html: highlighted.html }}
        />
      </pre>
    </div>
  );
}
