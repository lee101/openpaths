export type Light = { dir: [number, number, number]; color: string; intensity: number };
export type SpherePreset = { id: string; name: string; lights: Light[] };

export const SPHERE_PRESETS: SpherePreset[] = [
  { id: 'warm-key-left', name: 'Warm key left', lights: [{ dir: [-0.8, 0.25, 0.55], color: '#ffb36b', intensity: 1 }] },
  { id: 'cool-rim-right', name: 'Cool rim right', lights: [{ dir: [0.95, 0.1, -0.25], color: '#6aa8ff', intensity: 1 }, { dir: [-0.4, 0.2, 0.9], color: '#a9b8d6', intensity: 0.18 }] },
  { id: 'golden-hour', name: 'Golden hour', lights: [{ dir: [-0.75, -0.05, 0.65], color: '#ffb347', intensity: 1 }, { dir: [0.3, 0.7, 0.5], color: '#5a7bb0', intensity: 0.25 }] },
  { id: 'neon', name: 'Neon magenta / cyan', lights: [{ dir: [-0.9, 0, 0.4], color: '#ff2fd0', intensity: 1 }, { dir: [0.9, 0, 0.4], color: '#19e6ff', intensity: 1 }] },
  { id: 'top-down-studio', name: 'Top-down studio', lights: [{ dir: [0, 1, 0.35], color: '#ffffff', intensity: 1 }, { dir: [0, -0.2, 1], color: '#cfd6e6', intensity: 0.12 }] },
  { id: 'candle', name: 'Candle', lights: [{ dir: [0.15, -0.45, 0.8], color: '#ff8a2a', intensity: 0.8 }] },
  { id: 'moonlight', name: 'Moonlight', lights: [{ dir: [0.5, 0.7, 0.5], color: '#8fb4ff', intensity: 0.7 }] },
  { id: 'sunset-backlight', name: 'Sunset backlight', lights: [{ dir: [0.2, 0.3, -0.9], color: '#ff6a2e', intensity: 1 }, { dir: [0, 0, 1], color: '#6b5a7a', intensity: 0.1 }] },
];

export function directionFromAngles(azimuthDeg: number, elevationDeg: number): [number, number, number] {
  const az = (azimuthDeg * Math.PI) / 180;
  const el = (elevationDeg * Math.PI) / 180;
  return [Math.sin(az) * Math.cos(el), Math.sin(el), Math.cos(az) * Math.cos(el)];
}

export function customPreset(azimuthDeg: number, elevationDeg: number, color: string, intensity: number): SpherePreset {
  return { id: 'custom', name: 'Custom', lights: [{ dir: directionFromAngles(azimuthDeg, elevationDeg), color, intensity }] };
}

function hexToRgb(hex: string): [number, number, number] {
  const m = /^#?([0-9a-f]{6})$/i.exec(hex.trim());
  const n = m ? parseInt(m[1], 16) : 0xffffff;
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255];
}

export function shadeSphere(lights: Light[], size: number): Uint8ClampedArray {
  const out = new Uint8ClampedArray(size * size * 4);
  const norm = lights.map(l => {
    const [x, y, z] = l.dir;
    const len = Math.hypot(x, y, z) || 1;
    return { d: [x / len, y / len, z / len] as const, c: hexToRgb(l.color), k: l.intensity };
  });
  const half = size / 2;
  for (let py = 0; py < size; py++) {
    for (let px = 0; px < size; px++) {
      const x = (px + 0.5 - half) / half;
      const y = -(py + 0.5 - half) / half;
      const r2 = x * x + y * y;
      const i = (py * size + px) * 4;
      out[i + 3] = 255;
      if (r2 > 1) continue;
      const z = Math.sqrt(1 - r2);
      let r = 0;
      let g = 0;
      let b = 0;
      for (const l of norm) {
        const diffuse = Math.max(0, x * l.d[0] + y * l.d[1] + z * l.d[2]);
        const hx = l.d[0];
        const hy = l.d[1];
        const hz = l.d[2] + 1;
        const hl = Math.hypot(hx, hy, hz) || 1;
        const spec = Math.pow(Math.max(0, (x * hx + y * hy + z * hz) / hl), 48) * 0.35;
        const k = l.k * (diffuse * 0.95 + spec);
        r += (l.c[0] / 255) * k;
        g += (l.c[1] / 255) * k;
        b += (l.c[2] / 255) * k;
      }
      const edge = Math.min(1, (1 - Math.sqrt(r2)) * size * 0.5);
      out[i] = Math.min(255, 255 * Math.min(1, r) * edge);
      out[i + 1] = Math.min(255, 255 * Math.min(1, g) * edge);
      out[i + 2] = Math.min(255, 255 * Math.min(1, b) * edge);
    }
  }
  return out;
}

export function renderSphere(canvas: HTMLCanvasElement, preset: SpherePreset, size = 512) {
  canvas.width = size;
  canvas.height = size;
  const ctx = canvas.getContext('2d');
  if (!ctx) return;
  const img = ctx.createImageData(size, size);
  img.data.set(shadeSphere(preset.lights, size));
  ctx.putImageData(img, 0, 0);
}

export function sphereBlob(preset: SpherePreset, size = 512): Promise<Blob> {
  const canvas = document.createElement('canvas');
  renderSphere(canvas, preset, size);
  return new Promise((resolve, reject) => canvas.toBlob(b => (b ? resolve(b) : reject(new Error('Could not render sphere'))), 'image/png'));
}
