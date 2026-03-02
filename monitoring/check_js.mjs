import { chromium } from 'playwright';

const url = process.argv[2] || 'https://openpaths.io';
const errors = [];

const browser = await chromium.launch({ headless: true });
const page = await browser.newPage();

page.on('pageerror', (err) => {
  errors.push(`JS Error: ${err.message}`);
});

page.on('console', (msg) => {
  if (msg.type() === 'error') {
    errors.push(`Console Error: ${msg.text()}`);
  }
});

try {
  const resp = await page.goto(url, { waitUntil: 'networkidle', timeout: 20000 });
  if (!resp || resp.status() >= 400) {
    errors.push(`HTTP ${resp ? resp.status() : 'no response'}`);
  }
  // wait a bit for async JS
  await page.waitForTimeout(3000);
} catch (e) {
  errors.push(`Navigation error: ${e.message}`);
}

await browser.close();

if (errors.length > 0) {
  console.log(errors.join('\n'));
  process.exit(1);
}
process.exit(0);
