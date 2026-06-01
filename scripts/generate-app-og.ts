import { mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { seedApps } from '../src/data/seedApps';
import { formatTokens, host } from '../src/lib/appStats';

const OUT_DIR = join('public', 'og', 'apps');

mkdirSync(OUT_DIR, { recursive: true });

const allApps = [
  {
    slug: 'openpaths-apps',
    name: 'Apps And Agents',
    url: 'https://openpaths.io/apps',
    description: 'Opt-in app and agent usage stats across OpenPaths and OpenRouter.',
    favicon_url: 'https://openpaths.io/favicon.ico',
    total_tokens: 0,
  },
  ...seedApps,
];

for (const app of allApps) {
  writeFileSync(join(OUT_DIR, `${app.slug}.svg`), renderAppOg(app));
}

console.log(`Wrote ${allApps.length} app OG images to ${OUT_DIR}`);

function renderAppOg(app: {
  slug: string;
  name: string;
  url: string;
  description: string;
  favicon_url: string;
  total_tokens: number;
}) {
  const title = truncate(app.name, 36);
  const description = truncate(app.description, 105);
  const appHost = truncate(host(app.url), 42);
  const tokens = app.total_tokens > 0 ? `${formatTokens(app.total_tokens)} tokens` : 'Usage stats';
  const path = app.slug === 'openpaths-apps' ? 'apps' : `apps/${app.slug}`;

  return `<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="630" viewBox="0 0 1200 630">
  <rect width="1200" height="630" fill="#030303"/>
  <rect x="56" y="54" width="1088" height="522" rx="28" fill="#0a0a0a" stroke="#202020"/>
  <circle cx="120" cy="122" r="21" fill="#0f172a" stroke="#38bdf8" stroke-width="3"/>
  <path d="M112 122h16M120 114v16" stroke="#e5e7eb" stroke-width="3" stroke-linecap="round"/>
  <text x="156" y="132" fill="#f8fafc" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="30" font-weight="700">OpenPaths</text>
  <text x="56" y="608" fill="#525252" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="18">openpaths.io/${escapeXml(path)}</text>
  <image href="${escapeXml(app.favicon_url)}" x="86" y="205" width="116" height="116" preserveAspectRatio="xMidYMid slice"/>
  <text x="230" y="248" fill="#ffffff" font-family="Inter, ui-sans-serif, system-ui, sans-serif" font-size="64" font-weight="800">${escapeXml(title)}</text>
  <text x="232" y="292" fill="#94a3b8" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="24">${escapeXml(appHost)}</text>
  <text x="86" y="384" fill="#cbd5e1" font-family="Inter, ui-sans-serif, system-ui, sans-serif" font-size="30">${escapeXml(description)}</text>
  <rect x="86" y="448" width="300" height="82" rx="16" fill="#111111" stroke="#262626"/>
  <text x="112" y="482" fill="#64748b" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="18">TOKENS</text>
  <text x="112" y="518" fill="#ffffff" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="34" font-weight="800">${escapeXml(tokens)}</text>
  <rect x="414" y="448" width="574" height="82" rx="16" fill="#111111" stroke="#262626"/>
  <text x="440" y="482" fill="#64748b" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="18">APP PAGE</text>
  <text x="440" y="518" fill="#ffffff" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="26" font-weight="700">OpenPaths app usage leaderboard</text>
  <rect x="1010" y="448" width="88" height="82" rx="16" fill="#083344" stroke="#155e75"/>
  <text x="1030" y="498" fill="#67e8f9" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="28" font-weight="800">APP</text>
</svg>
`;
}

function truncate(value: string, max: number) {
  const chars = Array.from(value.trim());
  if (chars.length <= max) return value.trim();
  return `${chars.slice(0, Math.max(0, max - 1)).join('')}...`;
}

function escapeXml(value: string) {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&apos;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;');
}
