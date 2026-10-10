import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { spawnSync } from 'node:child_process';

const key = process.env.GEMINI_API_KEY;
if (!key) throw new Error('GEMINI_API_KEY is required');
const SRC = process.env.SHOWCASE_SOURCE || '/vfast/data/code/memes/fortgameidea.png';
const OUT = 'public/static/blog/gemini-image';
const STYLE = 'Keep the exact camera angle, castle layout, river, bridges, walls, and the bottom build menu and HUD panels so it looks like the same game in a different state.';

type Job = { slug: string; model: string; prompt: string; edit: boolean; aspect?: string };
const jobs: Job[] = [
  { slug: 'fort-night-siege-lite', model: 'gemini-3.1-flash-lite-image', edit: true, prompt: `Turn this strategy game screenshot into a night siege: deep blue moonlit sky, torches and braziers glowing along the walls, small enemy campfires in the forest. ${STYLE} Change the objective banner to "Hold the Walls".` },
  { slug: 'fort-night-siege-nb21', model: 'gemini-nano-banana-2.1', edit: true, prompt: `Turn this strategy game screenshot into a night siege: deep blue moonlit sky, torches and braziers glowing along the walls, small enemy campfires in the forest. ${STYLE} Change the objective banner to "Hold the Walls".` },
  { slug: 'fort-winter', model: 'gemini-3.1-flash-lite-image', edit: true, prompt: `Re-render this game screenshot in deep winter: snow on every roof and wall, frozen river with ice, bare and snow-dusted pines, low pale sun. ${STYLE} Change the year to 128 and the objective to "Survive the Winter".` },
  { slug: 'fort-pixel-art', model: 'gemini-nano-banana-2.1', edit: true, prompt: `Completely redraw this game screenshot as a retro 16-bit SNES-era pixel art game: every object built from large visible square pixels, limited 32-color palette, dithered shading, blocky pixel trees and water, chunky pixel-font UI. It must look unmistakably like pixel art, not a photo filter. ${STYLE}` },
  { slug: 'fort-dragon-attack', model: 'gemini-nano-banana-2.1', edit: true, prompt: `Show the aftermath of a dragon attack on this castle: a large dragon circling above, several towers burning with smoke columns, damaged wall sections, villagers fleeing. ${STYLE} Change the objective to "Repair the Walls".` },
  { slug: 'fort-building-icons', model: 'gemini-3.1-flash-lite-image', edit: true, aspect: '1:1', prompt: 'Using the art style of this game (soft stylized 3D, warm medieval palette), draw a clean 4x3 sprite sheet of twelve build-menu icons on a parchment background: stone wall, wooden palisade, gatehouse, archer tower, moat, keep, barracks, house, market, bridge, farm plot, windmill. Each icon is a separate tidy square cell with no text.' },
  { slug: 'fort-battle-screen', model: 'gemini-nano-banana-2.1', edit: true, prompt: 'Create the battle-mode screen for this same game: same art style and HUD frame, but the camera is closer on the gatehouse, archers on the walls, a knight squad selection panel with unit portraits and health bars at the bottom, an enemy army approaching over the bridge. Title "Aethelgard Chronicles" top left.' },
  { slug: 'game-key-art', model: 'gemini-3.1-flash-lite-image', edit: false, aspect: '16:9', prompt: 'Key art for a cozy medieval castle-builder game called "Aethelgard Chronicles": a hilltop castle at golden hour, rivers and forests, soft stylized 3D look, warm light, title lettering at the top.' },
];

async function run(job: Job) {
  const input: unknown = job.edit
    ? [{ type: 'text', text: job.prompt }, { type: 'image', data: (await readFile(SRC)).toString('base64'), mime_type: 'image/png' }]
    : job.prompt;
  const image_config: Record<string, string> = { image_size: '1K' };
  if (job.aspect) image_config.aspect_ratio = job.aspect;
  const body = { model: `models/${job.model}`, input, generation_config: { temperature: 1, max_output_tokens: 65536, top_p: 0.95, thinking_level: 'minimal', image_config }, response_modalities: ['image', 'text'] };
  const t0 = Date.now();
  const resp = await fetch('https://generativelanguage.googleapis.com/v1beta/interactions', { method: 'POST', headers: { 'x-goog-api-key': key!, 'Content-Type': 'application/json' }, body: JSON.stringify(body), signal: AbortSignal.timeout(240000) });
  const data: any = await resp.json();
  if (!resp.ok) throw new Error(`${job.slug}: ${resp.status} ${JSON.stringify(data).slice(0, 300)}`);
  const part = data.steps?.flatMap((s: any) => s.type === 'model_output' ? s.content || [] : []).find((c: any) => c.type === 'image');
  if (!part) throw new Error(`${job.slug}: no image`);
  const raw = `${OUT}/${job.slug}.jpg`;
  await writeFile(raw, Buffer.from(part.data, 'base64'));
  const conv = spawnSync('python3', ['-c', 'from PIL import Image; import sys; Image.open(sys.argv[1]).convert("RGB").save(sys.argv[2],"WEBP",quality=88)', raw, `${OUT}/${job.slug}.webp`], { stdio: 'inherit' });
  if (conv.status !== 0) throw new Error('webp convert failed');
  spawnSync('rm', ['-f', raw]);
  await writeFile(`${OUT}/${job.slug}.json`, JSON.stringify({ model: job.model, prompt: job.prompt, edit: job.edit, aspect: job.aspect, ms: Date.now() - t0, usage: data.usage?.output_tokens_by_modality }, null, 2));
  console.log(`ok ${job.slug} ${Date.now() - t0}ms`);
}

await mkdir(OUT, { recursive: true });
spawnSync('python3', ['-c', 'from PIL import Image; import sys; Image.open(sys.argv[1]).convert("RGB").save(sys.argv[2],"WEBP",quality=90)', SRC, `${OUT}/fort-source.webp`], { stdio: 'inherit' });
const only = process.argv.slice(2);
const results = await Promise.allSettled(jobs.filter(j => !only.length || only.includes(j.slug)).map(run));
for (const r of results) if (r.status === 'rejected') console.log('FAIL', String(r.reason).slice(0, 300));
