import type { Model } from '../data/models';

export function formatCurrency(value: number) {
  if (value === 0) return '$0';
  const precision = value < 0.01 || Math.abs(value * 100 - Math.round(value * 100)) > 1e-9 ? 3 : 2;
  return `$${value.toFixed(precision)}`;
}

const UNIT: Record<string, string> = {
  request: '/ request',
  chars: '/ 1M chars',
  hour: '/ hour',
  second: '/ second',
  megapixel: '/ megapixel',
};

export function priceUnit(model: Model) {
  return model.pricingType ? UNIT[model.pricingType] : '/ 1M tokens';
}

export function isTokenPriced(model: Model) {
  return !model.pricingType || model.pricingType === 'token';
}

export function isFreeModel(model: Model) {
  return model.priceInput === 0 && model.priceOutput === 0;
}

/** Compact one-line price: "$0.50 in / $4.50 out per 1M tokens" or "$0.04 / request". */
export function priceSummary(model: Model) {
  if (isFreeModel(model)) return 'Free';
  if (isTokenPriced(model) && model.priceOutput > 0 && model.priceOutput !== model.priceInput) {
    return `${formatCurrency(model.priceInput)} in / ${formatCurrency(model.priceOutput)} out per 1M tokens`;
  }
  return `${formatCurrency(model.priceInput)} ${priceUnit(model)}`;
}

export function slugify(value: string) {
  return value.toLowerCase().replace(/[^a-z0-9.]+/g, '-').replace(/^-+|-+$/g, '');
}
