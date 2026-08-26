// Types + helpers for the OpenPaths live evals snapshot served by
// GET /v1/evals/results. Public endpoint — fetched without auth.

export type SuiteKey = 'coding' | 'agentic' | 'creative';

export type SuiteAgg = {
  avg_score: number;
  pass_rate: number;
  cases: number;
  median_ttft_ms: number;
  avg_tps: number;
  cost_per_case_micro_usd: number;
};

export type PerModel = {
  model: string;
  by_suite: Record<string, SuiteAgg>;
  overall: SuiteAgg;
};

export type CaseResult = {
  score: number;
  passed: boolean;
  ttft_ms: number;
  total_ms: number;
  tokens_per_sec: number;
  cost_micro_usd: number;
  answer_preview: string;
  error: string | null;
};

export type CaseEntry = {
  suite: string;
  case_id: string;
  results: Record<string, CaseResult>;
};

export type AutoVsBestEntry = {
  best_model: string | null;
  best_score: number | null;
  auto_score: number | null;
  best_cost_per_case_micro_usd?: number;
  auto_cost_per_case_micro_usd?: number;
};

export type EvalSnapshot = {
  ran_at: string | null;
  models: PerModel[];
  cases: CaseEntry[];
  auto_vs_best: Record<string, AutoVsBestEntry>;
};

// Dracula palette — https://draculatheme.com
export const DRACULA = {
  bg: '#282a36',
  currentLine: '#44475a',
  selection: '#44475a',
  foreground: '#f8f8f2',
  comment: '#6272a4',
  cyan: '#8be9fd',
  green: '#50fa7b',
  orange: '#ffb86c',
  pink: '#ff79c6',
  purple: '#bd93f9',
  red: '#ff5555',
  yellow: '#f1fa8c',
} as const;

export const MODEL_META: Record<string, { label: string; color: string }> = {
  'openpaths/auto': { label: 'Auto', color: DRACULA.foreground },
  'gpt-5.6': { label: 'GPT-5.6', color: DRACULA.pink },
  'gpt-5.5': { label: 'GPT-5.5', color: DRACULA.orange },
  'claude-opus-5': { label: 'Claude Opus 5', color: DRACULA.purple },
  'gemini-3.7-flash': { label: 'Gemini 3.7 Flash', color: DRACULA.cyan },
  'deepseek-v4-pro': { label: 'DeepSeek V4 Pro', color: DRACULA.green },
  'grok-4.6': { label: 'Grok 4.6', color: DRACULA.yellow },
  'glm-5.1': { label: 'GLM-5.1', color: DRACULA.red },
};

export function modelLabel(id: string): string {
  return MODEL_META[id]?.label ?? id;
}

export function modelColor(id: string): string {
  return MODEL_META[id]?.color ?? DRACULA.comment;
}

export function formatMicroUSD(micro: number): string {
  if (!micro) return '$0';
  if (micro < 100) return `$${(micro / 100).toFixed(3)}`;
  return `$${(micro / 100).toFixed(2)}`;
}

export function formatRanAt(iso: string | null): string {
  if (!iso) return 'never';
  try {
    return new Date(iso).toLocaleString('en-US', {
      month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit',
    });
  } catch {
    return iso;
  }
}

export async function fetchEvalSnapshot(): Promise<EvalSnapshot | null> {
  try {
    const res = await fetch('/v1/evals/results', { headers: { Accept: 'application/json' } });
    if (!res.ok) return null;
    return (await res.json()) as EvalSnapshot;
  } catch {
    return null;
  }
}

export const SUITES: { key: SuiteKey; title: string; blurb: string }[] = [
  { key: 'coding', title: 'Coding', blurb: 'Deterministic code reasoning: closures, complexity, bugs, SQL boundaries, modular math, regex construction.' },
  { key: 'agentic', title: 'Agentic', blurb: 'Tool calling accuracy, multi-call orchestration, negative tool discipline, format adherence, and error recovery.' },
  { key: 'creative', title: 'Creative SVG', blurb: 'Constraint-checked SVG generation: shapes, proportional charts, wordmarks, and scenes — parsed and graded programmatically.' },
];

export const CASE_TITLES: Record<string, string> = {
  'closure-output': 'JS closure semantics',
  'big-o-complexity': 'Big-O analysis',
  'debug-line-number': 'Bug localization',
  'sql-boundary-count': 'SQL boundary count',
  'mod-arithmetic': 'Modular arithmetic',
  'regex-build': 'Regex construction',
  'tool-single-call': 'Single tool call',
  'tool-two-calls': 'Two tool calls',
  'no-tool-needed': 'No-tool discipline',
  'format-instruction': 'Format following',
  'tool-error-recovery': 'Error recovery',
  'svg-basic-shapes': 'SVG basic shapes',
  'svg-bar-chart': 'SVG proportional chart',
  'svg-wordmark': 'SVG wordmark',
  'svg-mountain-scene': 'SVG scene',
};
