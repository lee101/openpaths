import 'dotenv/config';
import { mkdir, writeFile } from 'node:fs/promises';
import { spawnSync } from 'node:child_process';
import { IMAGE_DEMOS } from '../src/data/imageDemos';

const key = process.env.BFL_API_KEY;
if (!key) throw new Error('BFL_API_KEY is required to generate the FLUX 3 showcase');
const { model, ...input } = IMAGE_DEMOS['flux-3-image'].payload;
const headers = { 'x-key': key, 'Content-Type': 'application/json' };
const submitted = await fetch('https://api.bfl.ai/v1/flux-3-image', { method: 'POST', headers, body: JSON.stringify(input), signal: AbortSignal.timeout(60000) });
if (!submitted.ok) throw new Error(`BFL submission failed: ${submitted.status} ${await submitted.text()}`);
const task = await submitted.json();
if (!task.polling_url) throw new Error('BFL returned no polling_url');
console.log(`Submitted FLUX 3 Image; cost: ${task.cost}`);
const deadline = Date.now() + 10 * 60 * 1000;
let result;
while (Date.now() < deadline) {
  const resp = await fetch(task.polling_url, { headers, signal: AbortSignal.timeout(60000) });
  if (!resp.ok) throw new Error(`BFL polling failed: ${resp.status}`);
  const data = await resp.json();
  if (data.status === 'Ready') { result = data.result; break; }
  if (/error|failed|moderated/i.test(data.status)) throw new Error(`BFL task failed: ${data.status}`);
  await new Promise(resolve => setTimeout(resolve, 2000));
}
if (!result?.sample) throw new Error('BFL did not return an image before the deadline');
const downloaded = await fetch(result.sample, { signal: AbortSignal.timeout(60000) });
if (!downloaded.ok) throw new Error(`BFL image download failed: ${downloaded.status}`);
const dir = 'public/static/image-gallery/bfl';
await mkdir(dir, {recursive:true});
const source = `${dir}/flux-3-forest-observatory-source.jpg`;
const output = `${dir}/flux-3-forest-observatory.webp`;
await writeFile(source, new Uint8Array(await downloaded.arrayBuffer()));
const converted = spawnSync('python3', ['-c', 'from PIL import Image; import sys; Image.open(sys.argv[1]).convert("RGB").save(sys.argv[2],"WEBP",quality=92)', source, output], {stdio:'inherit'});
if (converted.status !== 0) throw new Error('Could not create showcase WebP');
await writeFile(`${dir}/flux-3-forest-observatory.json`, JSON.stringify({ model, input, id:task.id, cost:task.cost, generatedAt:new Date().toISOString(), prompt:result.prompt }, null, 2));
console.log(`Saved ${output}`);
