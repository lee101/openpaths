import { mkdir, writeFile } from 'node:fs/promises';
import { spawnSync } from 'node:child_process';

const key = process.env.OPENAI_API_KEY;
if (!key) throw new Error('OPENAI_API_KEY is required');
const OUT = 'public/static/blog/gpt-image-2-6';
const REF = 'https://openpathsstatic.openpaths.io/static/uploads/playground/hidream-edit/perfume.jpg';
const MODELS = ['gpt-image-2.6-sunburst', 'gpt-image-2.6-flare', 'gpt-image-2.5-sunburst', 'gpt-image-2.5-flare'];

type Prompt = { slug: string; prompt: string; size: string; edit?: boolean };
const PROMPTS: Prompt[] = [
  { slug: 'mug', size: '1024x1024', prompt: "A beige ceramic coffee mug on a wooden table, soft natural window light, editorial product photo, with the handwritten text 'OPEN PATHS' on a chalkboard behind it" },
  { slug: 'poster', size: '1024x1536', prompt: 'Minimalist poster for a jazz night titled MIDNIGHT ROUTES, bold typography, deep blue and gold, saxophone silhouette, small text at the bottom: Fri 9pm, Blue Door Club' },
  { slug: 'infographic', size: '1536x1024', prompt: 'Clean infographic titled "How an API gateway routes a request" with four numbered steps in a left-to-right flow: 1 Client request, 2 Auth and rate limit, 3 Pick best model, 4 Stream response. Flat vector style, white background, teal and orange accents, crisp legible labels.' },
  { slug: 'perfume-edit', size: '1024x1024', edit: true, prompt: 'Change the label text to say OPENPATHS and make the liquid emerald green' },
];

async function run(model: string, p: Prompt) {
  const t0 = Date.now();
  let resp: Response;
  if (p.edit) {
    const img = await (await fetch(REF)).blob();
    const form = new FormData();
    form.append('model', model); form.append('prompt', p.prompt); form.append('size', p.size); form.append('quality', 'high');
    form.append('image', img, 'perfume.jpg');
    resp = await fetch('https://api.openai.com/v1/images/edits', { method: 'POST', headers: { Authorization: `Bearer ${key}` }, body: form, signal: AbortSignal.timeout(400000) });
  } else {
    resp = await fetch('https://api.openai.com/v1/images/generations', { method: 'POST', headers: { Authorization: `Bearer ${key}`, 'Content-Type': 'application/json' }, body: JSON.stringify({ model, prompt: p.prompt, size: p.size, quality: 'high' }), signal: AbortSignal.timeout(400000) });
  }
  const data: any = await resp.json();
  if (!resp.ok) throw new Error(`${model}/${p.slug}: ${resp.status} ${JSON.stringify(data).slice(0, 300)}`);
  const base = `${OUT}/${model}-${p.slug}`;
  await writeFile(`${base}.png`, Buffer.from(data.data[0].b64_json, 'base64'));
  const conv = spawnSync('python3', ['-c', 'import sys;from PIL import Image;im=Image.open(sys.argv[1]).convert("RGB");im.save(sys.argv[2],"WEBP",quality=86);im.thumbnail((800,800));im.save(sys.argv[3],"JPEG",quality=82)', `${base}.png`, `${base}.webp`, `${base}-email.jpg`], { stdio: 'inherit' });
  if (conv.status !== 0) throw new Error('convert failed');
  spawnSync('rm', ['-f', `${base}.png`]);
  const ms = Date.now() - t0;
  await writeFile(`${base}.json`, JSON.stringify({ model, prompt: p.prompt, size: p.size, edit: !!p.edit, quality: 'high', ms, usage: data.usage }, null, 2));
  console.log(`ok ${model} ${p.slug} ${ms}ms out_tokens=${data.usage?.output_tokens}`);
}

await mkdir(OUT, { recursive: true });
const only = process.argv.slice(2);
const jobs = MODELS.flatMap(m => PROMPTS.map(p => ({ m, p }))).filter(j => !only.length || only.includes(`${j.m}/${j.p.slug}`));
const results = await Promise.allSettled(jobs.map(j => run(j.m, j.p)));
for (const r of results) if (r.status === 'rejected') console.log('FAIL', String(r.reason).slice(0, 300));
