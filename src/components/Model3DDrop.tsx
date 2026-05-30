import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Box, Upload, X } from 'lucide-react';
import { ModelViewer } from './ModelViewer';

const MODEL_EXT = /\.(glb|gltf)$/i;

function isModelFile(file: File) {
  return MODEL_EXT.test(file.name) || file.type === 'model/gltf-binary' || file.type === 'model/gltf+json';
}

// Model3DDrop previews any 3D asset the user drops in, entirely client-side
// (object URL, no upload). Used to preview uploaded GLB/GLTF files alongside
// the generated 3D pages.
export function Model3DDrop({ minHeight = 320 }: { minHeight?: number }) {
  const [objectUrl, setObjectUrl] = useState('');
  const [fileName, setFileName] = useState('');
  const [error, setError] = useState('');
  const [dragging, setDragging] = useState(false);
  const urlRef = useRef('');

  const setModel = useCallback((file: File) => {
    if (!isModelFile(file)) {
      setError('Drop a .glb or .gltf file to preview it.');
      return;
    }
    setError('');
    if (urlRef.current) URL.revokeObjectURL(urlRef.current);
    const url = URL.createObjectURL(file);
    urlRef.current = url;
    setObjectUrl(url);
    setFileName(file.name);
  }, []);

  useEffect(() => () => {
    if (urlRef.current) URL.revokeObjectURL(urlRef.current);
  }, []);

  const clear = () => {
    if (urlRef.current) URL.revokeObjectURL(urlRef.current);
    urlRef.current = '';
    setObjectUrl('');
    setFileName('');
  };

  const dropHandlers = {
    onDragEnter: (e: React.DragEvent) => { e.preventDefault(); e.stopPropagation(); setDragging(true); },
    onDragOver: (e: React.DragEvent) => { e.preventDefault(); e.stopPropagation(); e.dataTransfer.dropEffect = 'copy'; setDragging(true); },
    onDragLeave: (e: React.DragEvent<HTMLLabelElement>) => {
      e.preventDefault(); e.stopPropagation();
      if (!e.currentTarget.contains(e.relatedTarget as Node | null)) setDragging(false);
    },
    onDrop: (e: React.DragEvent) => {
      e.preventDefault(); e.stopPropagation(); setDragging(false);
      const file = Array.from(e.dataTransfer.files)[0];
      if (file) setModel(file);
    },
  };

  if (objectUrl) {
    return (
      <div className="overflow-hidden rounded-lg border border-white/10 bg-white/[0.02]">
        <div className="flex items-center justify-between border-b border-white/10 px-4 py-3">
          <div className="flex items-center gap-2 text-xs font-mono text-white/55">
            <Box className="h-3.5 w-3.5" /> {fileName}
          </div>
          <button type="button" onClick={clear} className="inline-flex items-center gap-1.5 rounded border border-white/10 px-2.5 py-1.5 text-[10px] font-mono uppercase tracking-[0.14em] text-white/45 hover:text-white">
            <X className="h-3 w-3" /> Clear
          </button>
        </div>
        <div style={{ minHeight }}>
          <ModelViewer src={objectUrl} minHeight={minHeight} />
        </div>
      </div>
    );
  }

  return (
    <label
      className={`flex cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border border-dashed px-4 text-center text-xs font-mono transition-colors ${dragging ? 'border-emerald-300/50 bg-emerald-400/10 text-emerald-100' : 'border-white/10 bg-black/40 text-white/45 hover:border-white/25'}`}
      style={{ minHeight }}
      {...dropHandlers}
    >
      <Upload className="h-5 w-5" />
      <span>{dragging ? 'Drop to preview' : 'Drop or choose a .glb / .gltf to preview'}</span>
      {error && <span className="text-red-300">{error}</span>}
      <input
        type="file"
        accept=".glb,.gltf,model/gltf-binary,model/gltf+json"
        onChange={e => e.target.files?.[0] && setModel(e.target.files[0])}
        className="hidden"
      />
    </label>
  );
}
