// Headless browser JS / console / HTTP error scan.
//
// Usage:
//   node check_js.mjs <base_url>                    # legacy: scan only <base_url>, exit 1 on error
//   node check_js.mjs <base_url> --json             # scan default route list, emit JSON, always exit 0
//   node check_js.mjs <base_url> --json --routes /,/playground,/models
//
// JSON shape:
//   { base, scannedAt, routes: [{ path, status, errors: [string] }], hasErrors: bool }
import { chromium } from 'playwright';

const args = process.argv.slice(2);
const base = (args[0] || 'https://openpaths.io').replace(/\/$/, '');
const jsonMode = args.includes('--json');
const routesIdx = args.indexOf('--routes');
const defaultRoutes = ['/', '/playground', '/models', '/pricing', '/providers'];
const routes = routesIdx !== -1 && args[routesIdx + 1]
  ? args[routesIdx + 1].split(',').map(r => r.trim()).filter(Boolean)
  : (jsonMode ? defaultRoutes : ['/']);

const ignorePatterns = [
  /favicon\.ico/i,
  /sentry/i,
  /Failed to load resource: net::ERR_BLOCKED_BY_CLIENT/i,
];

const shouldIgnore = (text) => ignorePatterns.some(p => p.test(text));

async function scanRoute(browser, path) {
  const errors = [];
  const ctx = await browser.newContext();
  const page = await ctx.newPage();
  page.on('pageerror', (err) => {
    if (!shouldIgnore(err.message)) errors.push(`pageerror: ${err.message}`);
  });
  page.on('console', (msg) => {
    if (msg.type() === 'error' && !shouldIgnore(msg.text())) {
      errors.push(`console.error: ${msg.text()}`);
    }
  });
  page.on('requestfailed', (req) => {
    const f = req.failure();
    if (f && !shouldIgnore(f.errorText)) {
      errors.push(`requestfailed: ${req.url()} ${f.errorText}`);
    }
  });
  let status = 0;
  try {
    const resp = await page.goto(base + path, { waitUntil: 'networkidle', timeout: 25000 });
    status = resp ? resp.status() : 0;
    if (!resp || status >= 400) {
      errors.push(`HTTP ${status || 'no response'}`);
    }
    await page.waitForTimeout(2500);
  } catch (e) {
    errors.push(`navigation: ${e.message}`);
  } finally {
    await ctx.close();
  }
  return { path, status, errors };
}

const browser = await chromium.launch({ headless: true });
const results = [];
for (const r of routes) {
  results.push(await scanRoute(browser, r));
}
await browser.close();

const hasErrors = results.some(r => r.errors.length > 0);

if (jsonMode) {
  console.log(JSON.stringify({
    base,
    scannedAt: new Date().toISOString(),
    routes: results,
    hasErrors,
  }, null, 2));
  process.exit(0);
}

// legacy text mode
const flat = results.flatMap(r => r.errors.map(e => `[${r.path}] ${e}`));
if (flat.length > 0) {
  console.log(flat.join('\n'));
  process.exit(1);
}
process.exit(0);
