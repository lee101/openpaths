import hljs from 'highlight.js/lib/common';

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
  const resolvedLabel = hideLabel ? '' : label || (highlighted.language ? highlighted.language.toUpperCase() : '');

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
