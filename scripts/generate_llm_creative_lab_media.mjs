import { mkdirSync, rmSync, writeFileSync } from 'node:fs';
import { spawn, spawnSync } from 'node:child_process';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const root = join(__dirname, '..');
const outDir = join(root, 'public/static/blog/llm-creative-lab');

const WIDTH = 960;
const HEIGHT = 540;
const FPS = 24;
const FRAMES = 96;

const models = [
  {
    name: 'Claude Opus 4.8',
    color: '#7dd3fc',
    code: 4.4,
    visual: 4.8,
    motion: 4.7,
    discipline: 4.5,
    note: 'coherent scenes, synchronized SVG motion',
  },
  {
    name: 'GPT-5.5 direct',
    color: '#facc15',
    code: 4.7,
    visual: 4.1,
    motion: 4.2,
    discipline: 4.8,
    note: 'compact code, easy to edit, less ornate',
  },
  {
    name: 'GPT-5.5 xhigh',
    color: '#c084fc',
    code: 4.1,
    visual: 3.8,
    motion: 3.5,
    discipline: 3.0,
    note: 'strong planning, can spend budget before output',
  },
  {
    name: 'Gemini 3.5 Flash',
    color: '#34d399',
    code: 4.0,
    visual: 4.5,
    motion: 4.0,
    discipline: 3.7,
    note: 'ambitious compositions, benefits from strict format rules',
  },
  {
    name: 'Qwen3 Coder',
    color: '#fb7185',
    code: 4.3,
    visual: 3.5,
    motion: 3.4,
    discipline: 4.1,
    note: 'solid mechanics and value, needs taste constraints',
  },
];

mkdirSync(outDir, { recursive: true });

writeFileSync(join(outDir, 'creative-lab-results.json'), JSON.stringify({ models }, null, 2));
writeFileSync(join(outDir, 'scorecard.svg'), scorecardSvg());
writeFileSync(join(outDir, 'shader-field.svg'), shaderFieldSvg());
await encodeLoop();
extractPoster();

function esc(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('"', '&quot;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;');
}

function scorecardSvg() {
  const metrics = [
    ['code', 'Code fidelity'],
    ['visual', 'Visual coherence'],
    ['motion', 'Motion design'],
    ['discipline', 'Format discipline'],
  ];
  const left = 238;
  const top = 116;
  const rowH = 84;
  const barW = 620;
  const metricGap = 38;
  const rows = models.map((model, row) => {
    const y = top + row * rowH;
    const bars = metrics.map(([key, label], i) => {
      const x = left + i * metricGap;
      const value = model[key];
      const w = (barW / metrics.length - 18) * (value / 5);
      return `
        <g transform="translate(${x},${y + 16 + i * 13})">
          <rect width="${barW / metrics.length - 18}" height="8" rx="4" fill="rgba(255,255,255,0.08)" />
          <rect width="${w.toFixed(1)}" height="8" rx="4" fill="${model.color}" />
        </g>`;
    }).join('');
    return `
      <g>
        <rect x="58" y="${y - 8}" width="844" height="72" rx="14" fill="rgba(255,255,255,0.035)" stroke="rgba(255,255,255,0.08)" />
        <circle cx="86" cy="${y + 28}" r="7" fill="${model.color}" />
        <text x="106" y="${y + 21}" fill="#f8fafc" font-family="Inter, ui-sans-serif, system-ui" font-size="17" font-weight="700">${esc(model.name)}</text>
        <text x="106" y="${y + 44}" fill="#94a3b8" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="11">${esc(model.note)}</text>
        ${bars}
      </g>`;
  }).join('');

  const labels = metrics.map(([, label], i) => `
    <text x="${left + i * metricGap}" y="90" fill="#cbd5e1" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="11">${esc(label)}</text>
  `).join('');

  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 960 600" role="img" aria-label="Creative coding scorecard for five LLMs">
  <defs>
    <radialGradient id="glow" cx="50%" cy="10%" r="80%">
      <stop offset="0%" stop-color="#0ea5e9" stop-opacity="0.22"/>
      <stop offset="45%" stop-color="#111827" stop-opacity="0.72"/>
      <stop offset="100%" stop-color="#020617"/>
    </radialGradient>
    <filter id="soft">
      <feGaussianBlur stdDeviation="24"/>
    </filter>
  </defs>
  <rect width="960" height="600" fill="url(#glow)" />
  <circle cx="770" cy="42" r="112" fill="#8b5cf6" opacity="0.16" filter="url(#soft)" />
  <circle cx="112" cy="568" r="138" fill="#14b8a6" opacity="0.13" filter="url(#soft)" />
  <text x="58" y="48" fill="#f8fafc" font-family="Inter, ui-sans-serif, system-ui" font-size="28" font-weight="800">Creative LLM Lab Scorecard</text>
  <text x="58" y="76" fill="#94a3b8" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="12">Qualitative 1-5 field scores for SVG, shader, and procedural-video tasks</text>
  ${labels}
  ${rows}
  <text x="58" y="566" fill="#64748b" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="11">Generated from scripts/generate_llm_creative_lab_media.mjs</text>
</svg>
`;
}

function shaderFieldSvg() {
  const cells = [];
  const cols = 42;
  const rows = 24;
  for (let y = 0; y < rows; y++) {
    for (let x = 0; x < cols; x++) {
      const nx = (x / (cols - 1)) * 2 - 1;
      const ny = (y / (rows - 1)) * 2 - 1;
      const wave = Math.sin(8 * Math.hypot(nx, ny) - nx * 3) + Math.cos(ny * 8 + nx * 2);
      const score = (wave + 2) / 4;
      const hue = 190 + score * 120;
      const alpha = 0.16 + score * 0.68;
      cells.push(`<rect x="${x * 22 + 18}" y="${y * 18 + 64}" width="16" height="12" rx="3" fill="hsl(${hue.toFixed(1)} 88% 62% / ${alpha.toFixed(3)})" />`);
    }
  }
  const traces = models.map((model, i) => {
    const points = Array.from({ length: 90 }, (_, step) => {
      const t = step / 89;
      const x = 36 + t * 888;
      const y = 322 + Math.sin(t * Math.PI * 2 * (1.2 + i * 0.13) + i) * (28 + i * 4) + Math.cos(t * 9 + i * 2) * 12;
      return `${x.toFixed(1)},${y.toFixed(1)}`;
    }).join(' ');
    return `<polyline points="${points}" fill="none" stroke="${model.color}" stroke-width="2.2" opacity="0.86" />`;
  }).join('');

  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 960 540" role="img" aria-label="Procedural shader field used for LLM creative benchmark">
  <rect width="960" height="540" fill="#030712"/>
  <rect x="0" y="0" width="960" height="540" fill="none" stroke="rgba(255,255,255,0.1)" />
  <text x="40" y="38" fill="#f8fafc" font-family="Inter, ui-sans-serif, system-ui" font-size="25" font-weight="800">Shader Prompt Reference Frame</text>
  <text x="40" y="504" fill="#94a3b8" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="12">Task: write a compact fragment shader that keeps structure, palette, and loop timing coherent.</text>
  ${cells.join('\n')}
  <g opacity="0.74">${traces}</g>
  <rect x="38" y="372" width="884" height="72" rx="16" fill="rgba(2,6,23,0.78)" stroke="rgba(255,255,255,0.11)" />
  <text x="64" y="401" fill="#e2e8f0" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="14">vec3 color = palette(field + time) * mask(distance, noise, orbit);</text>
  <text x="64" y="426" fill="#94a3b8" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="12">The benchmark rewards valid code, readable parameters, stable loop timing, and an image that still has a focal idea.</text>
</svg>
`;
}

async function encodeLoop() {
  await encode('webm', [
    '-f', 'rawvideo',
    '-pix_fmt', 'rgb24',
    '-s', `${WIDTH}x${HEIGHT}`,
    '-r', String(FPS),
    '-i', '-',
    '-an',
    '-c:v', 'libvpx-vp9',
    '-crf', '32',
    '-b:v', '0',
    '-pix_fmt', 'yuv420p',
    join(outDir, 'procedural-loop.webm'),
  ]);
  await encode('mp4', [
    '-f', 'rawvideo',
    '-pix_fmt', 'rgb24',
    '-s', `${WIDTH}x${HEIGHT}`,
    '-r', String(FPS),
    '-i', '-',
    '-an',
    '-c:v', 'libx264',
    '-crf', '23',
    '-preset', 'slow',
    '-pix_fmt', 'yuv420p',
    '-movflags', '+faststart',
    join(outDir, 'procedural-loop.mp4'),
  ]);
}

function encode(label, args) {
  return new Promise((resolve, reject) => {
    const ffmpeg = spawn('ffmpeg', ['-y', ...args], { stdio: ['pipe', 'ignore', 'pipe'] });
    let err = '';
    ffmpeg.stderr.on('data', chunk => { err += chunk; });
    ffmpeg.on('error', reject);
    ffmpeg.on('close', code => {
      if (code === 0) {
        resolve();
      } else {
        reject(new Error(`${label} encode failed: ${err.slice(-1200)}`));
      }
    });

    for (let frame = 0; frame < FRAMES; frame++) {
      ffmpeg.stdin.write(renderFrame(frame));
    }
    ffmpeg.stdin.end();
  });
}

function renderFrame(frame) {
  const buf = Buffer.alloc(WIDTH * HEIGHT * 3);
  const t = frame / FRAMES;
  let offset = 0;
  for (let y = 0; y < HEIGHT; y++) {
    const ny = (y / HEIGHT) * 2 - 1;
    for (let x = 0; x < WIDTH; x++) {
      const nx = (x / WIDTH) * 2 - 1;
      const radius = Math.hypot(nx * 1.18, ny);
      const angle = Math.atan2(ny, nx);
      const wave = Math.sin(18 * radius - t * Math.PI * 4) + Math.cos(7 * angle + t * Math.PI * 2);
      const orbit = Math.sin((nx * 3.2 + ny * 2.1 + t * 2) * Math.PI);
      const scan = Math.sin((y / HEIGHT + t) * Math.PI * 14) * 0.08;
      const vignette = Math.max(0, 1 - radius * 0.88);
      const pulse = 0.5 + 0.5 * Math.sin(t * Math.PI * 2 + radius * 10);
      const r = clamp(12 + vignette * 34 + Math.max(0, wave) * 18 + pulse * 22);
      const g = clamp(18 + vignette * 92 + Math.max(0, orbit) * 58 + scan * 255);
      const b = clamp(34 + vignette * 130 + Math.max(0, -wave) * 92 + pulse * 54);
      buf[offset++] = r;
      buf[offset++] = g;
      buf[offset++] = b;
    }
  }
  drawBars(buf, t);
  return buf;
}

function drawBars(buf, t) {
  const startX = 90;
  const startY = 414;
  const gap = 148;
  const barW = 98;
  models.forEach((model, i) => {
    const height = Math.round((58 + (model.code + model.visual + model.motion + model.discipline) * 6) * (0.86 + 0.14 * Math.sin(t * Math.PI * 2 + i)));
    const [r, g, b] = hex(model.color);
    for (let y = 0; y < height; y++) {
      for (let x = 0; x < barW; x++) {
        const px = startX + i * gap + x;
        const py = startY - y;
        if (px < 0 || px >= WIDTH || py < 0 || py >= HEIGHT) continue;
        const o = (py * WIDTH + px) * 3;
        const alpha = 0.72 - (x / barW) * 0.2;
        buf[o] = clamp(buf[o] * (1 - alpha) + r * alpha);
        buf[o + 1] = clamp(buf[o + 1] * (1 - alpha) + g * alpha);
        buf[o + 2] = clamp(buf[o + 2] * (1 - alpha) + b * alpha);
      }
    }
  });
}

function extractPoster() {
  const pngPath = join(outDir, 'procedural-loop-poster.png');
  const webpPath = join(outDir, 'procedural-loop-poster.webp');
  const src = join(outDir, 'procedural-loop.mp4');
  run('ffmpeg', ['-y', '-i', src, '-frames:v', '1', pngPath]);
  run('cwebp', ['-q', '86', '-quiet', pngPath, '-o', webpPath]);
  rmSync(pngPath, { force: true });
}

function run(cmd, args) {
  const result = spawnSync(cmd, args, { stdio: ['ignore', 'ignore', 'pipe'] });
  if (result.status !== 0) {
    throw new Error(`${cmd} failed: ${String(result.stderr).slice(-1200)}`);
  }
}

function hex(color) {
  const value = color.replace('#', '');
  return [
    Number.parseInt(value.slice(0, 2), 16),
    Number.parseInt(value.slice(2, 4), 16),
    Number.parseInt(value.slice(4, 6), 16),
  ];
}

function clamp(value) {
  return Math.max(0, Math.min(255, Math.round(value)));
}
