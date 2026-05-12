import fs from 'node:fs';

const htmlPath = 'dist/index.html';
if (!fs.existsSync(htmlPath)) {
  throw new Error(`missing build output: ${htmlPath}`);
}

const html = fs.readFileSync(htmlPath, 'utf8');
const expected = ['/assets/index.css', '/assets/index.js'];
for (const asset of expected) {
  if (!html.includes(asset)) {
    throw new Error(`expected ${asset} in ${htmlPath}`);
  }
}

const hashedAssetPattern = /\/assets\/index-[A-Za-z0-9_-]+\.(?:js|css)/g;
const matches = html.match(hashedAssetPattern) || [];
if (matches.length > 0) {
  throw new Error(`hashed assets still referenced in ${htmlPath}: ${matches.join(', ')}`);
}

console.log('stable asset references verified');
