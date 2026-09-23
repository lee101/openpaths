import { spawnSync } from 'node:child_process';
import { mkdtempSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { models } from '../src/data/models';
import { getProviderLogo, providersByName } from '../src/data/providers';
import { modelOgSlug } from '../src/lib/modelOg';

const money = (v: number) => (v === 0 ? '$0' : `$${v.toFixed(v < 0.01 || Math.abs(v * 100 - Math.round(v * 100)) > 1e-9 ? 3 : 2)}`);
const unit: Record<string, string> = { request: '/ request', chars: '/ 1M chars', hour: '/ hour', second: '/ second', megapixel: '/ megapixel' };

function price(m: (typeof models)[number]) {
  if (m.pricingType && unit[m.pricingType]) return `${money(m.priceInput)} ${unit[m.pricingType]}`;
  if (m.priceInput === 0 && m.priceOutput === 0) return 'Free';
  return `${money(m.priceInput)} in / ${money(m.priceOutput)} out per 1M tokens`;
}

const rows = models
  .filter(m => !m.ogImage)
  .map(m => ({
    slug: modelOgSlug(m.id),
    id: m.id,
    name: m.name,
    provider: m.provider,
    logo: providersByName[m.provider]?.logo || getProviderLogo(m.provider),
    price: price(m),
    context: m.contextLength,
    tags: m.tags.slice(0, 3),
  }));
const file = join(mkdtempSync(join(tmpdir(), 'op-og-')), 'models.json');
writeFileSync(file, JSON.stringify(rows));
const r = spawnSync('python3', ['scripts/gen_og_models.py', file], { stdio: 'inherit' });
process.exit(r.status ?? 1);
