import { ARTIFICIAL_ANALYSIS_SNAPSHOT } from '../data/artificialAnalysis';

export type ArtificialAnalysisModel = (typeof ARTIFICIAL_ANALYSIS_SNAPSHOT.models)[number];
export type EvaluationKey = keyof ArtificialAnalysisModel['evaluations'];

export const artificialAnalysisModels = [...ARTIFICIAL_ANALYSIS_SNAPSHOT.models] as ArtificialAnalysisModel[];
export const artificialAnalysisSnapshot = ARTIFICIAL_ANALYSIS_SNAPSHOT;

export const EVALUATION_DEFINITIONS: Array<{
  key: EvaluationKey;
  label: string;
  shortLabel: string;
  kind: 'index' | 'percent' | 'elo';
  higherIsBetter: boolean;
}> = [
  { key: 'intelligenceIndex', label: 'Artificial Analysis Intelligence Index', shortLabel: 'AA Index', kind: 'index', higherIsBetter: true },
  { key: 'codingIndex', label: 'Artificial Analysis Coding Index', shortLabel: 'Coding', kind: 'index', higherIsBetter: true },
  { key: 'agenticIndex', label: 'Artificial Analysis Agentic Index', shortLabel: 'Agentic', kind: 'index', higherIsBetter: true },
  { key: 'gdpvalElo', label: 'GDPval-AA Elo', shortLabel: 'GDPval', kind: 'elo', higherIsBetter: true },
  { key: 'terminalBenchHard', label: 'Terminal-Bench Hard', shortLabel: 'Terminal', kind: 'percent', higherIsBetter: true },
  { key: 'tau2', label: 'Tau2', shortLabel: 'Tau2', kind: 'percent', higherIsBetter: true },
  { key: 'lcr', label: 'AA-LCR', shortLabel: 'AA-LCR', kind: 'percent', higherIsBetter: true },
  { key: 'hle', label: "Humanity's Last Exam", shortLabel: 'HLE', kind: 'percent', higherIsBetter: true },
  { key: 'gpqa', label: 'GPQA Diamond', shortLabel: 'GPQA', kind: 'percent', higherIsBetter: true },
  { key: 'sciCode', label: 'SciCode', shortLabel: 'SciCode', kind: 'percent', higherIsBetter: true },
  { key: 'ifBench', label: 'IFBench', shortLabel: 'IFBench', kind: 'percent', higherIsBetter: true },
  { key: 'critPt', label: 'CritPt', shortLabel: 'CritPt', kind: 'percent', higherIsBetter: true },
  { key: 'mmmuPro', label: 'MMMU-Pro', shortLabel: 'MMMU', kind: 'percent', higherIsBetter: true },
  { key: 'apexAgents', label: 'APEX-Agents-AA', shortLabel: 'APEX', kind: 'percent', higherIsBetter: true },
  { key: 'omniscience', label: 'AA-Omniscience Index', shortLabel: 'Omni', kind: 'index', higherIsBetter: true },
  { key: 'omniscienceAccuracy', label: 'AA-Omniscience Accuracy', shortLabel: 'Accuracy', kind: 'percent', higherIsBetter: true },
  { key: 'omniscienceHallucinationRate', label: 'AA-Omniscience Hallucination Rate', shortLabel: 'Halluc.', kind: 'percent', higherIsBetter: false },
];

export function getModelScore(model: ArtificialAnalysisModel, key: EvaluationKey): number | null {
  const value = model.evaluations[key];
  return typeof value === 'number' ? value : null;
}

export function displayScore(value: number | null | undefined, kind: 'index' | 'percent' | 'elo' = 'index'): string {
  if (typeof value !== 'number' || !Number.isFinite(value)) return 'N/A';
  if (kind === 'percent') return `${(value * 100).toFixed(value >= 0.1 ? 1 : 2)}%`;
  if (kind === 'elo') return Math.round(value).toLocaleString('en-US');
  return value.toFixed(1);
}

export function formatDollars(value: number | null | undefined, digits = 2): string {
  if (typeof value !== 'number' || !Number.isFinite(value)) return 'N/A';
  return `$${value.toLocaleString('en-US', { maximumFractionDigits: digits, minimumFractionDigits: value < 10 ? 2 : 0 })}`;
}

export function formatNumber(value: number | null | undefined): string {
  if (typeof value !== 'number' || !Number.isFinite(value)) return 'N/A';
  return value.toLocaleString('en-US');
}

export function normalizeModelToken(value: string): string {
  return value.toLowerCase().replace(/[^a-z0-9]+/g, '');
}

export function findArtificialAnalysisModel(value: string): ArtificialAnalysisModel | undefined {
  const token = normalizeModelToken(value);
  if (!token) return undefined;

  const candidates = artificialAnalysisModels.map(model => ({
    model,
    aliases: [
      model.slug,
      model.name,
      model.shortName,
      `${model.creator.name} ${model.shortName}`,
      model.shortName.replace(/\([^)]*\)/g, ''),
    ].map(normalizeModelToken),
  }));

  return candidates.find(item => item.aliases.includes(token))?.model
    || candidates.find(item => item.aliases.some(alias => alias.includes(token) || token.includes(alias)))?.model;
}

export function topArtificialAnalysisModels(limit = 12): ArtificialAnalysisModel[] {
  return artificialAnalysisModels
    .filter(model => typeof model.evaluations.intelligenceIndex === 'number')
    .slice(0, limit);
}

export function modelColor(model: ArtificialAnalysisModel): string {
  return model.creator.color || '#22d3ee';
}
