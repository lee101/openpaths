// Indexes video sample generations into the gallery search engine (art_images).
// Posts each curated VIDEO_DEMOS output (rendered test generations we host) to the
// admin-only POST /v1/art/index endpoint, then triggers a semantic reindex.
//
// Usage:
//   OP_ADMIN_KEY=op-xxxx OP_BASE=https://openpaths.io bun scripts/index-video-samples.ts
//   (defaults: OP_BASE=https://openpaths.io)
import { VIDEO_DEMOS, type VideoDemo } from '../src/data/videoDemos';

const BASE = process.env.OP_BASE || 'https://openpaths.io';
const KEY = process.env.OP_ADMIN_KEY || '';

if (!KEY) {
  console.error('OP_ADMIN_KEY required (admin api key)');
  process.exit(1);
}

function slugify(s: string): string {
  return s.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '');
}

function aspectBucket(ar: VideoDemo['aspectRatio']): string {
  if (ar === '1:1') return 'square';
  if (ar === '9:16' || ar === '3:4') return 'portrait';
  if (ar === 'auto') return 'wide';
  return 'wide';
}

function titleFor(modelId: string): string {
  return modelId
    .split(/[/-]/)
    .filter(Boolean)
    .map(p => p.charAt(0).toUpperCase() + p.slice(1))
    .join(' ');
}

async function post(path: string, body: unknown) {
  const res = await fetch(`${BASE}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${KEY}` },
    body: JSON.stringify(body),
  });
  const text = await res.text();
  if (!res.ok) throw new Error(`${path} -> ${res.status}: ${text.slice(0, 200)}`);
  return text;
}

async function main() {
  const entries = Object.entries(VIDEO_DEMOS);
  let ok = 0;
  for (const [modelId, demo] of entries) {
    const duration = Number(demo.duration);
    const item = {
      slug: `video-${slugify(modelId)}`,
      title: titleFor(modelId),
      prompt: demo.prompt,
      mediaType: 'video',
      videoUrl: demo.outputUrl,
      posterUrl: demo.imageUrl || '',
      durationSeconds: Number.isFinite(duration) ? duration : 0,
      aspect: aspectBucket(demo.aspectRatio),
      model: modelId,
      source: 'openpaths-gen',
      tags: ['video', 'video generation'],
    };
    try {
      await post('/v1/art/index', item);
      ok++;
      console.log(`indexed ${item.slug}`);
    } catch (e) {
      console.error(`FAILED ${item.slug}: ${(e as Error).message}`);
    }
  }
  console.log(`\n${ok}/${entries.length} indexed; triggering reindex...`);
  await post('/v1/art/reindex', {});
  console.log('reindex triggered');
}

main().catch(e => {
  console.error(e);
  process.exit(1);
});
