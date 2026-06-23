#!/usr/bin/env node
// Generate + upload the sync-3 (fal-ai/sync-lipsync/v3/image-to-video) playground
// demo assets: a brand mascot avatar (flux), a branded TTS voice track (kokoro),
// and the lip-synced output video. Uploads input image, audio, output mp4 + webm
// to R2 under static/uploads/playground/sync-lipsync/.
//
// Env: FAL_KEY (or FAL_API_KEY), R2_ENDPOINT, R2_BUCKET, R2_ACCESS_KEY, R2_SECRET_KEY.
// Flags: --reuse (use existing tmp files, skip fal calls), --no-upload, --dry-run.
import { createHash, createHmac } from 'node:crypto';
import { existsSync } from 'node:fs';
import { mkdir, readFile, stat, writeFile } from 'node:fs/promises';
import { join, resolve } from 'node:path';
import { spawn } from 'node:child_process';

const ROOT = resolve(new URL('..', import.meta.url).pathname);
const OUT_DIR = resolve(process.env.SYNC_DEMO_OUT_DIR || join(ROOT, 'tmp', 'sync-demo'));
const PUBLIC_BASE = (process.env.R2_PUBLIC_URL || 'https://openpathsstatic.openpaths.io').replace(/\/$/, '');
const PREFIX = 'static/uploads/playground/sync-lipsync';
const FAL_KEY = process.env.FAL_KEY || process.env.FAL_API_KEY || '';
const args = new Set(process.argv.slice(2));
const REUSE = args.has('--reuse');
const SKIP_UPLOAD = args.has('--no-upload');
const DRY_RUN = args.has('--dry-run');

const AVATAR_PROMPT = 'close-up head and shoulders portrait of a friendly stylized 3D cartoon mascot character wearing green headphones, big expressive eyes, open smiling mouth showing it is speaking, face fills the frame, emerald green and teal brand colors, dark studio gradient background, soft cinematic rim light, clean Pixar-style brand illustration, facing camera';
const SCRIPT = "Hi! I was built with OpenPaths. This whole avatar is talking thanks to Sync 3 lip sync. Just one image and a voice track, and any character comes to life.";

async function main() {
  if (!DRY_RUN && !REUSE && !FAL_KEY) throw new Error('FAL_KEY required');
  await mkdir(OUT_DIR, { recursive: true });

  const avatarPath = join(OUT_DIR, 'avatar2.jpg');
  const audioPath = join(OUT_DIR, 'tts.wav');
  const mp4Path = join(OUT_DIR, 'output.mp4');
  const webmPath = join(OUT_DIR, 'output.webm');

  // 1. avatar image
  let avatarUrl = '';
  if (REUSE && existsSync(avatarPath)) {
    console.log('reuse avatar');
  } else {
    avatarUrl = await falImage(AVATAR_PROMPT);
    await downloadFile(avatarUrl, avatarPath);
  }

  // 2. TTS voice track
  let audioUrl = '';
  if (REUSE && existsSync(audioPath)) {
    console.log('reuse audio');
  } else {
    audioUrl = await falTTS(SCRIPT);
    await downloadFile(audioUrl, audioPath);
  }

  // 3. lip-synced video — needs publicly fetchable input URLs (fal media URLs)
  if (REUSE && existsSync(mp4Path)) {
    console.log('reuse output');
  } else {
    if (!avatarUrl || !audioUrl) throw new Error('need fresh avatar+audio URLs for sync-3 (drop --reuse)');
    const videoUrl = await falSyncLipsync(avatarUrl, audioUrl);
    await downloadFile(videoUrl, mp4Path);
  }

  await transcodeWebM(mp4Path, webmPath);

  const uploads = {};
  if (!SKIP_UPLOAD && !DRY_RUN) {
    uploads.imageUrl = await uploadR2(`${PREFIX}/avatar.jpg`, await readFile(avatarPath), 'image/jpeg');
    uploads.audioUrl = await uploadR2(`${PREFIX}/voice.wav`, await readFile(audioPath), 'audio/wav');
    uploads.outputMp4 = await uploadR2(`${PREFIX}/sync-lipsync-v3-demo.mp4`, await readFile(mp4Path), 'video/mp4');
    uploads.outputWebm = await uploadR2(`${PREFIX}/sync-lipsync-v3-demo.webm`, await readFile(webmPath), 'video/webm');
  } else {
    uploads.imageUrl = `${PUBLIC_BASE}/${PREFIX}/avatar.jpg`;
    uploads.audioUrl = `${PUBLIC_BASE}/${PREFIX}/voice.wav`;
    uploads.outputMp4 = `${PUBLIC_BASE}/${PREFIX}/sync-lipsync-v3-demo.mp4`;
    uploads.outputWebm = `${PUBLIC_BASE}/${PREFIX}/sync-lipsync-v3-demo.webm`;
  }
  console.log(JSON.stringify(uploads, null, 2));
}

async function falImage(prompt) {
  const r = await falPost('https://fal.run/fal-ai/flux/dev', { prompt, image_size: 'square_hd', num_images: 1, num_inference_steps: 28 });
  const url = r?.images?.[0]?.url;
  if (!url) throw new Error('no image url: ' + JSON.stringify(r).slice(0, 300));
  return url;
}

async function falTTS(text) {
  const r = await falPost('https://fal.run/fal-ai/kokoro/american-english', { prompt: text, voice: 'af_heart' });
  const url = r?.audio?.url;
  if (!url) throw new Error('no audio url: ' + JSON.stringify(r).slice(0, 300));
  return url;
}

async function falSyncLipsync(imageUrl, audioUrl) {
  const submit = await falPost('https://queue.fal.run/fal-ai/sync-lipsync/v3/image-to-video', { image_url: imageUrl, audio_url: audioUrl });
  const rid = submit?.request_id;
  if (!rid) throw new Error('no request_id: ' + JSON.stringify(submit).slice(0, 300));
  const base = `https://queue.fal.run/fal-ai/sync-lipsync/requests/${rid}`;
  for (let i = 0; i < 100; i++) {
    const s = await falGet(`${base}/status`);
    if (s?.status === 'COMPLETED') break;
    if (s?.status === 'FAILED' || s?.status === 'ERROR') throw new Error('sync-3 failed: ' + JSON.stringify(s).slice(0, 300));
    await sleep(6000);
  }
  const res = await falGet(base);
  const url = res?.video?.url;
  if (!url) throw new Error('no video url: ' + JSON.stringify(res).slice(0, 300));
  return url;
}

async function falPost(url, body) {
  const res = await fetch(url, { method: 'POST', headers: { Authorization: `Key ${FAL_KEY}`, 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
  if (!res.ok) throw new Error(`fal POST ${url} HTTP ${res.status}: ${(await res.text()).slice(0, 300)}`);
  return res.json();
}
async function falGet(url) {
  const res = await fetch(url, { headers: { Authorization: `Key ${FAL_KEY}` } });
  if (!res.ok) throw new Error(`fal GET ${url} HTTP ${res.status}`);
  return res.json();
}

async function downloadFile(url, outPath) {
  const res = await fetch(url);
  if (!res.ok || !res.body) throw new Error(`download failed HTTP ${res.status}`);
  const chunks = [];
  for await (const chunk of res.body) chunks.push(chunk);
  const data = Buffer.concat(chunks);
  await writeFile(outPath, data);
  console.log(`downloaded ${outPath} ${(data.length / 1024 / 1024).toFixed(2)} MB`);
}

async function transcodeWebM(input, output) {
  await run('ffmpeg', ['-y', '-i', input, '-c:v', 'libvpx-vp9', '-b:v', '0', '-crf', '34', '-deadline', 'good', '-cpu-used', '4', '-row-mt', '1', '-pix_fmt', 'yuv420p', '-c:a', 'libopus', '-b:a', '96k', output]);
  const sz = await stat(output);
  console.log(`webm ${(sz.size / 1024 / 1024).toFixed(2)} MB`);
}

function run(cmd, cmdArgs) {
  return new Promise((res, rej) => {
    const child = spawn(cmd, cmdArgs, { stdio: ['ignore', 'ignore', 'pipe'] });
    let stderr = '';
    child.stderr.on('data', (c) => { stderr += c.toString(); });
    child.on('error', rej);
    child.on('close', (code) => code === 0 ? res() : rej(new Error(`${cmd} exited ${code}: ${stderr.slice(-800)}`)));
  });
}

async function uploadR2(key, body, contentType) {
  const endpoint = (process.env.R2_ENDPOINT || '').replace(/\/$/, '');
  const bucket = process.env.R2_BUCKET || 'openpathsstatic';
  if (!endpoint) throw new Error('R2_ENDPOINT required');
  const url = `${endpoint}/${bucket}/${key}`;
  const now = new Date();
  const amzDate = now.toISOString().replace(/[:-]|\.\d{3}/g, '');
  const dateStamp = now.toISOString().slice(0, 10).replace(/-/g, '');
  const payloadHash = createHash('sha256').update(body).digest('hex');
  const host = new URL(url).host;
  const canonicalHeaders = `content-type:${contentType}\nhost:${host}\nx-amz-content-sha256:${payloadHash}\nx-amz-date:${amzDate}\n`;
  const signedHeaders = 'content-type;host;x-amz-content-sha256;x-amz-date';
  const canonicalRequest = `PUT\n/${bucket}/${key}\n\n${canonicalHeaders}\n${signedHeaders}\n${payloadHash}`;
  const credentialScope = `${dateStamp}/auto/s3/aws4_request`;
  const stringToSign = `AWS4-HMAC-SHA256\n${amzDate}\n${credentialScope}\n${createHash('sha256').update(Buffer.from(canonicalRequest)).digest('hex')}`;
  const kDate = createHmac('sha256', Buffer.from(`AWS4${process.env.R2_SECRET_KEY}`)).update(dateStamp).digest();
  const kRegion = createHmac('sha256', kDate).update('auto').digest();
  const kService = createHmac('sha256', kRegion).update('s3').digest();
  const kSigning = createHmac('sha256', kService).update('aws4_request').digest();
  const signature = createHmac('sha256', kSigning).update(stringToSign).digest('hex');
  const authorization = `AWS4-HMAC-SHA256 Credential=${process.env.R2_ACCESS_KEY}/${credentialScope}, SignedHeaders=${signedHeaders}, Signature=${signature}`;
  const res = await fetch(url, { method: 'PUT', headers: { Authorization: authorization, 'Content-Type': contentType, 'x-amz-content-sha256': payloadHash, 'x-amz-date': amzDate, 'Cache-Control': 'public, max-age=31536000, immutable' }, body });
  if (!res.ok) throw new Error(`R2 upload failed HTTP ${res.status}: ${(await res.text()).slice(0, 300)}`);
  console.log(`uploaded ${key}`);
  return `${PUBLIC_BASE}/${key}`;
}

function sleep(ms) { return new Promise((r) => setTimeout(r, ms)); }

main().catch((err) => { console.error(err); process.exit(1); });
