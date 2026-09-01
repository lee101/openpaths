import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Box, Loader2, Upload, X } from 'lucide-react';
import { ModelViewer } from './ModelViewer';
import { normalizeUploadedAssetUrl } from '../lib/uploadUrls';

const MODEL_EXT = /\.(glb|gltf)$/i;

function isModelFile(file: File) {
  return MODEL_EXT.test(file.name) || file.type === 'model/gltf-binary' || file.type === 'model/gltf+json';
}

async function parseUpload(resp: Response) {
  const text = await resp.text();
  let data: any = null;
  try { data = text ? JSON.parse(text) : null; } catch { /* non-JSON */ }
  if (!resp.ok) throw new Error(data?.error?.message || data?.error || text || 'Upload failed');
  return data;
}

type Model3DUploadProps = {
  apiKey: string;
  // Called with the public CDN URL once the dropped model finishes uploading.
  onUploaded: (url: string) => void;
  minHeight?: number;
};

// Model3DUpload lets a user drop/choose a .glb/.gltf, previews it locally
// (object URL - instant, no wait) AND uploads it to /v1/files/upload so the
// resulting public URL can be passed to providers that fetch the mesh over HTTP
// (e.g. Fal meshy rigging). Reused by any 3D-to-3D space.
export function Model3DUpload({ apiKey, onUploaded, minHeight = 320 }: Model3DUploadProps) {
  const [objectUrl, setObjectUrl] = useState('');
  const [fileName, setFileName] = useState('');
  const [publicUrl, setPublicUrl] = useState('');
  const [error, setError] = useState('');
  const [uploading, setUploading] = useState(false);
  const [dragging, setDragging] = useState(false);
  const urlRef = useRef('');

  useEffect(() => () => {
    if (urlRef.current) URL.revokeObjectURL(urlRef.current);
  }, []);

  const handleFile = useCallback(async (file: File) => {
    if (!isModelFile(file)) {
      setError('Drop a .glb or .gltf file.');
      return;
    }
    if (!apiKey) {
      setError('Set an OpenPaths API key before uploading.');
      return;
    }
    setError('');
    if (urlRef.current) URL.revokeObjectURL(urlRef.current);
    const localUrl = URL.createObjectURL(file);
    urlRef.current = localUrl;
    setObjectUrl(localUrl);
    setFileName(file.name);
    setPublicUrl('');
    setUploading(true);
    try {
      const form = new FormData();
      form.append('file', file);
      const resp = await fetch('/v1/files/upload', {
        method: 'POST',
        headers: { Authorization: `Bearer ${apiKey}` },
        body: form,
      });
      const data = await parseUpload(resp);
      const url = normalizeUploadedAssetUrl(data.url);
      setPublicUrl(url);
      onUploaded(url);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Upload failed');
    } finally {
      setUploading(false);
    }
  }, [apiKey, onUploaded]);

  const clear = () => {
    if (urlRef.current) URL.revokeObjectURL(urlRef.current);
    urlRef.current = '';
    setObjectUrl('');
    setFileName('');
    setPublicUrl('');
    setError('');
  };

  const dropHandlers = {
    onDragEnter: (e: React.DragEvent) => { e.preventDefault(); e.stopPropagation(); if (!uploading) setDragging(true); },
    onDragOver: (e: React.DragEvent) => { e.preventDefault(); e.stopPropagation(); e.dataTransfer.dropEffect = uploading ? 'none' : 'copy'; if (!uploading) setDragging(true); },
    onDragLeave: (e: React.DragEvent<HTMLElement>) => {
      e.preventDefault(); e.stopPropagation();
      if (!e.currentTarget.contains(e.relatedTarget as Node | null)) setDragging(false);
    },
    onDrop: (e: React.DragEvent) => {
      e.preventDefault(); e.stopPropagation(); setDragging(false);
      if (uploading) return;
      const file = Array.from(e.dataTransfer.files)[0];
      if (file) void handleFile(file);
    },
  };

  if (objectUrl) {
    return (
      <div className="overflow-hidden rounded-lg border border-white/20 bg-white/[0.05]">
        <div className="flex items-center justify-between border-b border-white/20 px-4 py-3">
          <div className="flex min-w-0 items-center gap-2 text-xs font-mono text-white/55">
            <Box className="h-3.5 w-3.5 flex-none" /> <span className="truncate">{fileName}</span>
          </div>
          <div className="flex items-center gap-2">
            {uploading && <span className="inline-flex items-center gap-1.5 text-[10px] font-mono uppercase tracking-[0.14em] text-white/45"><Loader2 className="h-3 w-3 animate-spin" /> Uploading</span>}
            {publicUrl && !uploading && <span className="text-[10px] font-mono uppercase tracking-[0.14em] text-emerald-300">Uploaded</span>}
            <button type="button" onClick={clear} className="inline-flex items-center gap-1.5 rounded border border-white/20 px-2.5 py-1.5 text-[10px] font-mono uppercase tracking-[0.14em] text-white/45 hover:text-white">
              <X className="h-3 w-3" /> Clear
            </button>
          </div>
        </div>
        <div style={{ minHeight }}>
          <ModelViewer src={objectUrl} minHeight={minHeight} />
        </div>
        {error && <p className="border-t border-white/20 px-4 py-2 text-xs font-mono text-red-300">{error}</p>}
        {publicUrl && (
          <a href={publicUrl} target="_blank" rel="noreferrer" className="block truncate border-t border-white/20 px-4 py-2 text-xs font-mono text-emerald-200/80 hover:text-emerald-100">
            {publicUrl}
          </a>
        )}
      </div>
    );
  }

  return (
    <label
      className={`flex cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border border-dashed px-4 text-center text-xs font-mono transition-colors ${dragging ? 'border-emerald-300/50 bg-emerald-400/10 text-emerald-100' : 'border-white/20 bg-black/40 text-white/45 hover:border-white/45'}`}
      style={{ minHeight }}
      {...dropHandlers}
    >
      <Upload className="h-5 w-5" />
      <span>{dragging ? 'Drop to upload' : 'Drop or choose a .glb / .gltf to rig'}</span>
      {error && <span className="text-red-300">{error}</span>}
      <input
        type="file"
        accept=".glb,.gltf,model/gltf-binary,model/gltf+json"
        onChange={e => e.target.files?.[0] && handleFile(e.target.files[0])}
        className="hidden"
      />
    </label>
  );
}
