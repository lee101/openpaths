import React, { useEffect } from 'react';

// Loads Google's <model-viewer> web component once, on demand. It renders
// GLB/GLTF models interactively (orbit, zoom, auto-rotate) and is shared by
// every 3D surface (Image-to-3D, Text-to-3D, and the upload preview).
export function loadModelViewer() {
  if (typeof document === 'undefined') return;
  if (document.querySelector('script[data-model-viewer]')) return;
  const script = document.createElement('script');
  script.type = 'module';
  script.src = 'https://unpkg.com/@google/model-viewer/dist/model-viewer.min.js';
  script.setAttribute('data-model-viewer', 'true');
  document.head.appendChild(script);
}

export function useModelViewer() {
  useEffect(() => {
    loadModelViewer();
  }, []);
}

type ModelViewerProps = {
  src: string;
  alt?: string;
  poster?: string;
  className?: string;
  minHeight?: number | string;
  autoRotate?: boolean;
};

export function ModelViewer({ src, alt = 'Generated 3D model', poster, className, minHeight = '420px', autoRotate = true }: ModelViewerProps) {
  useModelViewer();
  return React.createElement('model-viewer' as any, {
    src,
    alt,
    poster,
    'camera-controls': true,
    'auto-rotate': autoRotate ? true : undefined,
    'shadow-intensity': '0.8',
    exposure: '1',
    className,
    style: {
      width: '100%',
      height: '100%',
      minHeight,
      background: 'linear-gradient(180deg, rgba(255,255,255,0.08), rgba(255,255,255,0.02))',
    },
  });
}
