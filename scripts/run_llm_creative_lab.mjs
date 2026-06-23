import { mkdirSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';

const endpoint = process.env.OPENPATHS_BASE_URL || 'https://openpaths.io/v1/chat/completions';
const apiKey = process.env.OPENPATHS_API_KEY;

const models = [
  { id: 'claude-opus-4-8', label: 'Claude Opus 4.8' },
  { id: 'gpt-5.5', label: 'GPT-5.5 direct', reasoning_effort: 'none' },
  { id: 'gpt-5.5', label: 'GPT-5.5 xhigh', reasoning_effort: 'xhigh' },
  { id: 'gemini-3.5-flash', label: 'Gemini 3.5 Flash' },
  { id: 'qwen3-coder', label: 'Qwen3 Coder' },
];

const tasks = [
  {
    id: 'animated-svg',
    max_tokens: 4096,
    system: 'Output only final code. Do not use markdown or prose.',
    prompt: 'Draw a small robotic lanternfish exploring a coral reef as an animated SVG. Return one complete standalone SVG document only.',
    checks: [
      ['starts_with_svg', text => text.trimStart().startsWith('<svg')],
      ['has_animation', text => /<animate|<animateTransform|@keyframes/.test(text)],
      ['has_subject', text => /fish|lantern|coral|reef/i.test(text)],
    ],
  },
  {
    id: 'fragment-shader',
    max_tokens: 4096,
    system: 'Output only final code. Do not use markdown or prose.',
    prompt: 'Write a compact GLSL fragment shader in Shadertoy style. It should render a loopable aurora over a black ocean with a visible moon reflection. Include mainImage.',
    checks: [
      ['has_main_image', text => /void\s+mainImage\s*\(/.test(text)],
      ['mentions_time', text => /iTime|time/i.test(text)],
      ['has_palette_math', text => /sin|cos|smoothstep|mix/.test(text)],
    ],
  },
  {
    id: 'procedural-video-plan',
    max_tokens: 3072,
    system: 'Answer as concise implementation notes and code. Do not use marketing language.',
    prompt: 'Design a tiny Node or Python script that procedurally generates a 4 second looping WebM clip using ffmpeg rawvideo input. Include the core frame algorithm and encoding command.',
    checks: [
      ['mentions_ffmpeg', text => /ffmpeg/i.test(text)],
      ['mentions_rawvideo', text => /rawvideo|rgb24|yuv420p/i.test(text)],
      ['has_loop_timing', text => /loop|sin|cos|frame|fps/i.test(text)],
    ],
  },
];

if (!apiKey) {
  console.error('OPENPATHS_API_KEY is required.');
  process.exit(1);
}

const results = [];
for (const model of models) {
  for (const task of tasks) {
    results.push(await runOne(model, task));
  }
}

const outDir = join(process.cwd(), 'local/llm-creative-lab');
mkdirSync(outDir, { recursive: true });
const stamp = new Date().toISOString().replaceAll(':', '-').replace(/\.\d+Z$/, 'Z');
const path = join(outDir, `run-${stamp}.json`);
writeFileSync(path, JSON.stringify({ endpoint, models, tasks: tasks.map(({ checks, ...task }) => task), results }, null, 2));
console.log(`Wrote ${path}`);

async function runOne(model, task) {
  const body = {
    model: model.id,
    messages: [
      { role: 'system', content: task.system },
      { role: 'user', content: task.prompt },
    ],
    max_tokens: task.max_tokens,
  };
  if (model.reasoning_effort) {
    body.reasoning_effort = model.reasoning_effort;
  }

  const start = performance.now();
  const response = await fetch(endpoint, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${apiKey}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(body),
  });
  const elapsed_ms = Math.round(performance.now() - start);
  const json = await response.json().catch(() => ({}));
  const content = json.choices?.[0]?.message?.content || '';
  return {
    model: model.label,
    model_id: model.id,
    reasoning_effort: model.reasoning_effort || null,
    task: task.id,
    ok: response.ok,
    status: response.status,
    elapsed_ms,
    usage: json.usage || null,
    checks: Object.fromEntries(task.checks.map(([name, fn]) => [name, Boolean(fn(content))])),
    chars: content.length,
    preview: content.slice(0, 500),
    content,
    error: response.ok ? null : json,
  };
}
