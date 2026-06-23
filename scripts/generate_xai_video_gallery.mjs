#!/usr/bin/env node
import { createHash, createHmac } from 'node:crypto';
import { existsSync } from 'node:fs';
import { mkdir, readFile, stat, writeFile } from 'node:fs/promises';
import { basename, join, resolve } from 'node:path';
import { spawn } from 'node:child_process';

const ROOT = resolve(new URL('..', import.meta.url).pathname);
const OUT_DIR = resolve(process.env.XAI_VIDEO_GALLERY_OUT_DIR || join(ROOT, 'tmp', 'xai-video-gallery'));
const PUBLIC_BASE = (process.env.R2_PUBLIC_URL || 'https://openpathsstatic.openpaths.io').replace(/\/$/, '');
const DEFAULT_PREFIX = 'static/uploads/landing/video-gallery';
const args = new Set(process.argv.slice(2));

const JOBS = [
  {
    slug: 'grok-imagine-video-data-observatory',
    title: 'Data Observatory',
    prompt: 'A cinematic macro shot of a glass observatory filled with glowing data streams orbiting like constellations, slow dolly forward, realistic reflections, premium AI infrastructure mood, no readable text.',
  },
  {
    slug: 'grok-imagine-video-routing-garden',
    title: 'Routing Garden',
    prompt: 'A miniature indoor garden where fiber optic vines connect small model cards across black stone, soft rain on glass, gentle camera orbit, elegant product demo lighting, no text or logos.',
  },
  {
    slug: 'grok-imagine-video-agent-workshop',
    title: 'Agent Workshop',
    prompt: 'A quiet futuristic workshop where autonomous software agents appear as small luminous tools assembling a clean interface in midair, warm practical lights, shallow depth of field, slow push-in, no readable text.',
  },
];

function valueFlag(name, fallback = '') {
  const prefix = `${name}=`;
  const hit = process.argv.slice(2).find((arg) => arg.startsWith(prefix));
  return hit ? hit.slice(prefix.length) : fallback;
}

const MODEL = valueFlag('--model', process.env.XAI_VIDEO_MODEL || 'grok-imagine-video');
const DURATION = Number(valueFlag('--duration', process.env.XAI_VIDEO_DURATION || '6'));
const RESOLUTION = valueFlag('--resolution', process.env.XAI_VIDEO_RESOLUTION || '480p');
const ASPECT_RATIO = valueFlag('--aspect-ratio', process.env.XAI_VIDEO_ASPECT_RATIO || '16:9');
const COUNT = Number(valueFlag('--count', String(JOBS.length)));
const PREFIX = valueFlag('--prefix', process.env.XAI_VIDEO_GALLERY_PREFIX || DEFAULT_PREFIX).replace(/^\/|\/$/g, '');
const DRY_RUN = args.has('--dry-run');
const SKIP_UPLOAD = args.has('--no-upload');
const SKIP_UPDATE = args.has('--no-update');

async function main() {
  if (!process.env.XAI_API_KEY && !DRY_RUN) {
    throw new Error('XAI_API_KEY is required unless --dry-run is set');
  }
  if (!SKIP_UPLOAD && !DRY_RUN) {
    if (!r2Endpoint()) throw new Error('R2_ENDPOINT is required for upload');
    if (!r2Bucket()) throw new Error('R2_BUCKET or CLOUDFLARE_R2_BUCKET is required for upload');
    if (!r2AccessKey() || !r2SecretKey()) throw new Error('R2 access key/secret are required for upload');
  }
  await mkdir(OUT_DIR, { recursive: true });
  const selected = JOBS.slice(0, Math.max(1, Math.min(COUNT, JOBS.length)));
  const generated = [];
  for (const job of selected) {
    const mp4Path = join(OUT_DIR, `${job.slug}.mp4`);
    const webmPath = join(OUT_DIR, `${job.slug}.webm`);
    console.log(`\n== ${job.slug}`);
    let originalURL = '';
    if (DRY_RUN && existsSync(mp4Path)) {
      console.log(`using existing ${mp4Path}`);
    } else if (!DRY_RUN) {
      originalURL = await generateXaiVideo(job.prompt);
      await downloadFile(originalURL, mp4Path);
    }
    await transcodeWebM(mp4Path, webmPath);
    const size = await stat(webmPath);
    const originalSize = await stat(mp4Path);
    const key = `${PREFIX}/${job.slug}.webm`;
    const videoUrl = SKIP_UPLOAD || DRY_RUN ? `file://${webmPath}` : await uploadR2(key, await readFile(webmPath), 'video/webm');
    generated.push({
      ...job,
      provider: 'xAI',
      model: MODEL,
      videoUrl,
      originalVideoUrl: originalURL || undefined,
      duration: DURATION,
      resolution: RESOLUTION,
      aspectRatio: ASPECT_RATIO,
      bytes: size.size,
      originalBytes: originalSize.size,
    });
    console.log(`webm ${formatBytes(size.size)} from mp4 ${formatBytes(originalSize.size)} -> ${videoUrl}`);
  }

  await writeFile(join(OUT_DIR, 'manifest.json'), JSON.stringify(generated, null, 2) + '\n');
  if (!SKIP_UPDATE && !DRY_RUN) {
    await writeGalleryData(generated);
  }
  console.log(`\nwrote ${join(OUT_DIR, 'manifest.json')}`);
}

async function generateXaiVideo(prompt) {
  const submit = await fetch('https://api.x.ai/v1/videos/generations', {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${process.env.XAI_API_KEY}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ model: MODEL, prompt, duration: DURATION, resolution: RESOLUTION, aspect_ratio: ASPECT_RATIO }),
  });
  const submitText = await submit.text();
  if (!submit.ok) throw new Error(`xAI submit failed HTTP ${submit.status}: ${submitText.slice(0, 500)}`);
  const submitJSON = JSON.parse(submitText);
  if (!submitJSON.request_id) throw new Error(`xAI submit returned no request_id: ${submitText}`);
  console.log(`request_id ${submitJSON.request_id}`);

  const deadline = Date.now() + Number(process.env.XAI_VIDEO_POLL_TIMEOUT_MS || 900_000);
  while (Date.now() < deadline) {
    await sleep(Number(process.env.XAI_VIDEO_POLL_INTERVAL_MS || 5000));
    const poll = await fetch(`https://api.x.ai/v1/videos/${encodeURIComponent(submitJSON.request_id)}`, {
      headers: { Authorization: `Bearer ${process.env.XAI_API_KEY}` },
    });
    const text = await poll.text();
    if (!poll.ok) {
      console.log(`poll HTTP ${poll.status}: ${text.slice(0, 200)}`);
      continue;
    }
    const data = JSON.parse(text);
    process.stdout.write(`status ${data.status || 'unknown'} ${data.progress ?? ''}\r`);
    if (data.status === 'done' && data.video?.url) {
      process.stdout.write('\n');
      return data.video.url;
    }
    if (data.status === 'failed' || data.status === 'expired') {
      throw new Error(`xAI video ${data.status}: ${text.slice(0, 500)}`);
    }
  }
  throw new Error('xAI video generation timed out');
}

async function downloadFile(url, outPath) {
  const res = await fetch(url);
  if (!res.ok || !res.body) throw new Error(`download failed HTTP ${res.status}`);
  const chunks = [];
  for await (const chunk of res.body) chunks.push(chunk);
  const data = Buffer.concat(chunks);
  await writeFile(outPath, data);
  console.log(`downloaded ${basename(outPath)} ${formatBytes(data.length)}`);
}

async function transcodeWebM(input, output) {
  await run('ffmpeg', [
    '-y',
    '-i', input,
    '-map', '0:v:0',
    '-an',
    '-c:v', 'libvpx-vp9',
    '-b:v', '0',
    '-crf', process.env.XAI_VIDEO_WEBM_CRF || '36',
    '-deadline', 'good',
    '-cpu-used', process.env.XAI_VIDEO_WEBM_CPU_USED || '4',
    '-row-mt', '1',
    '-pix_fmt', 'yuv420p',
    output,
  ]);
}

function run(cmd, cmdArgs) {
  return new Promise((resolveRun, reject) => {
    const child = spawn(cmd, cmdArgs, { stdio: ['ignore', 'ignore', 'pipe'] });
    let stderr = '';
    child.stderr.on('data', (chunk) => { stderr += chunk.toString(); });
    child.on('error', reject);
    child.on('close', (code) => {
      if (code === 0) resolveRun();
      else reject(new Error(`${cmd} exited ${code}: ${stderr.slice(-1000)}`));
    });
  });
}

async function uploadR2(key, body, contentType) {
  const endpoint = r2Endpoint().replace(/\/$/, '');
  const bucket = r2Bucket();
  const url = `${endpoint}/${bucket}/${key}`;
  const now = new Date();
  const amzDate = amz(now);
  const dateStamp = ymd(now);
  const payloadHash = sha256(body);
  const host = new URL(url).host;
  const canonicalHeaders = `content-type:${contentType}\nhost:${host}\nx-amz-content-sha256:${payloadHash}\nx-amz-date:${amzDate}\n`;
  const signedHeaders = 'content-type;host;x-amz-content-sha256;x-amz-date';
  const canonicalRequest = `PUT\n/${bucket}/${key}\n\n${canonicalHeaders}\n${signedHeaders}\n${payloadHash}`;
  const credentialScope = `${dateStamp}/auto/s3/aws4_request`;
  const stringToSign = `AWS4-HMAC-SHA256\n${amzDate}\n${credentialScope}\n${sha256(Buffer.from(canonicalRequest))}`;
  const signature = hmac(signingKey(r2SecretKey(), dateStamp), stringToSign, 'hex');
  const authorization = `AWS4-HMAC-SHA256 Credential=${r2AccessKey()}/${credentialScope}, SignedHeaders=${signedHeaders}, Signature=${signature}`;
  const res = await fetch(url, {
    method: 'PUT',
    headers: {
      Authorization: authorization,
      'Content-Type': contentType,
      'x-amz-content-sha256': payloadHash,
      'x-amz-date': amzDate,
      'Cache-Control': 'public, max-age=31536000, immutable',
    },
    body,
  });
  if (!res.ok) throw new Error(`R2 upload failed HTTP ${res.status}: ${(await res.text()).slice(0, 500)}`);
  return `${PUBLIC_BASE}/${key}`;
}

async function writeGalleryData(items) {
  const lines = [
    'export type VideoGalleryItem = {',
    '  slug: string;',
    '  title: string;',
    '  provider: string;',
    '  model: string;',
    '  prompt: string;',
    '  videoUrl: string;',
    '  originalVideoUrl?: string;',
    "  duration: number;",
    "  resolution: '480p' | '720p';",
    "  aspectRatio: '16:9' | '9:16' | '1:1' | '4:3' | '3:4';",
    '};',
    '',
    'export const videoGallery: VideoGalleryItem[] = [',
  ];
  for (const item of items) {
    lines.push('  {');
    for (const key of ['slug', 'title', 'provider', 'model', 'prompt', 'videoUrl']) {
      if (item[key]) lines.push(`    ${key}: ${JSON.stringify(item[key])},`);
    }
    lines.push(`    duration: ${item.duration},`);
    lines.push(`    resolution: ${JSON.stringify(item.resolution)},`);
    lines.push(`    aspectRatio: ${JSON.stringify(item.aspectRatio)},`);
    lines.push('  },');
  }
  lines.push('];', '');
  await writeFile(join(ROOT, 'src', 'data', 'videoGallery.ts'), lines.join('\n'));
}

function sha256(data) {
  return createHash('sha256').update(data).digest('hex');
}
function hmac(key, data, enc) {
  return createHmac('sha256', key).update(data).digest(enc);
}
function signingKey(secret, dateStamp) {
  const kDate = hmac(Buffer.from(`AWS4${secret}`), dateStamp);
  const kRegion = hmac(kDate, 'auto');
  const kService = hmac(kRegion, 's3');
  return hmac(kService, 'aws4_request');
}
function amz(d) {
  return d.toISOString().replace(/[:-]|\.\d{3}/g, '');
}
function ymd(d) {
  return d.toISOString().slice(0, 10).replace(/-/g, '');
}
function r2AccessKey() {
  return process.env.R2_ACCESS_KEY || process.env.CLOUDFLARE_R2_ACCESS_KEY_ID || '';
}
function r2SecretKey() {
  return process.env.R2_SECRET_KEY || process.env.CLOUDFLARE_R2_SECRET_ACCESS_KEY || '';
}
function r2Endpoint() {
  if (process.env.R2_ENDPOINT) return process.env.R2_ENDPOINT;
  if (process.env.R2_ACCOUNT_ID) return `https://${process.env.R2_ACCOUNT_ID}.r2.cloudflarestorage.com`;
  return '';
}
function r2Bucket() {
  return process.env.R2_BUCKET || process.env.CLOUDFLARE_R2_BUCKET || 'openpathsstatic';
}
function sleep(ms) {
  return new Promise((resolveSleep) => setTimeout(resolveSleep, ms));
}
function formatBytes(n) {
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(2)} MB`;
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
