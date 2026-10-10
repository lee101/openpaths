#!/usr/bin/env node
// Upload a generated Wan 3.0 Video clip (mp4 + webm + poster) to R2 under
// static/uploads/playground/wan3.0-video/. Alibaba output links expire in 24h.
// Usage: node --env-file=.env scripts/upload-wan3-demo.mjs <name> <video_url_or_path>
import { createHash, createHmac } from 'node:crypto';
import { mkdtemp, readFile, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { spawnSync } from 'node:child_process';

const PUBLIC_BASE = (process.env.R2_PUBLIC_URL || 'https://openpathsstatic.openpaths.io').replace(/\/$/, '');
const PREFIX = 'static/uploads/playground/wan3.0-video';
const [name, source] = process.argv.slice(2);
if (!name || !source) throw new Error('usage: upload-wan3-demo.mjs <name> <video_url_or_path>');

const dir = await mkdtemp(join(tmpdir(), 'wan3-demo-'));
const mp4 = join(dir, `${name}.mp4`);
if (/^https?:/.test(source)) {
  const res = await fetch(source, { signal: AbortSignal.timeout(120000) });
  if (!res.ok) throw new Error(`download failed HTTP ${res.status}`);
  await writeFile(mp4, new Uint8Array(await res.arrayBuffer()));
} else {
  await writeFile(mp4, await readFile(source));
}
const webm = join(dir, `${name}.webm`);
const poster = join(dir, `${name}.jpg`);
run('ffmpeg', ['-v', 'error', '-y', '-i', mp4, '-c:v', 'libvpx-vp9', '-crf', '32', '-b:v', '0', '-row-mt', '1', '-c:a', 'libopus', '-b:a', '96k', webm]);
run('ffmpeg', ['-v', 'error', '-y', '-ss', '1', '-i', mp4, '-frames:v', '1', '-q:v', '3', poster]);
const out = {
  mp4: await uploadR2(`${PREFIX}/${name}.mp4`, await readFile(mp4), 'video/mp4'),
  webm: await uploadR2(`${PREFIX}/${name}.webm`, await readFile(webm), 'video/webm'),
  poster: await uploadR2(`${PREFIX}/${name}.jpg`, await readFile(poster), 'image/jpeg'),
};
console.log(JSON.stringify(out, null, 2));

function run(cmd, args) {
  const r = spawnSync(cmd, args, { stdio: 'inherit' });
  if (r.status !== 0) throw new Error(`${cmd} exited ${r.status}`);
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
  return `${PUBLIC_BASE}/${key}`;
}
