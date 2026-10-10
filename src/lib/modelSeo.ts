import { models, type Model } from '../data/models';

const nameCounts = new Map<string, number>();
for (const m of models) nameCounts.set(m.name, (nameCounts.get(m.name) || 0) + 1);

export function modelSeoTitle(model: Model) {
  const label = (nameCounts.get(model.name) || 0) > 1 ? `${model.name} (${model.provider})` : model.name;
  return `${label} API, Pricing, Context Window | OpenPaths`;
}
