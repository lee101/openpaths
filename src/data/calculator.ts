import { models, type Model } from './models';

export const CHAT_TAGS = ['general', 'programming', 'reasoning', 'vision', 'agentic', 'roleplay', 'fast', 'open-source'];

export const NON_TEXT_TAGS = ['video generation', 'text-to-video', 'image-to-video', 'audio', 'embedding', 'text-to-image'];

export const calculatorModels: Model[] = models
  .filter(model => {
    if (model.pricingType && model.pricingType !== 'token') return false;
    if (model.tags.some(tag => NON_TEXT_TAGS.includes(tag))) return false;
    return model.tags.some(tag => CHAT_TAGS.includes(tag));
  })
  .sort((a, b) => b.popularity - a.popularity);

export const defaultSelectionIds = ['gpt-5.6', 'openpaths/auto', 'minimax-m2.7'].filter(id =>
  calculatorModels.some(model => model.id === id)
);

export function formatUsd(value: number): string {
  if (value === 0) return '$0';
  if (Math.abs(value) < 0.005) return `$${value.toFixed(4)}`;
  if (Math.abs(value) < 1) return `$${value.toFixed(3)}`;
  if (Math.abs(value) < 10_000) return `$${value.toFixed(2)}`;
  return `$${value.toLocaleString('en-US', { maximumFractionDigits: 0 })}`;
}
