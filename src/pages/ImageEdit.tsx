import React from 'react';
import { ImageIcon, Sparkles } from 'lucide-react';
import { ToolSeo } from '../components/ToolSeo';
import { ImageSpacePanel } from '../components/ImageSpacePanel';

export function ImageEdit() {
  return (
    <>
      <ToolSeo slug="image-edit" />
      <div className="px-6 py-12">
        <div className="mx-auto max-w-6xl">
          <div className="mb-8 max-w-3xl">
            <div className="mb-4 inline-flex items-center gap-2 rounded border border-white/20 bg-white/[0.06] px-3 py-1 text-xs font-mono text-white/45"><Sparkles className="h-3.5 w-3.5" /> Image edit</div>
            <h1 className="text-4xl font-bold tracking-tight md:text-5xl">Keep the source. Change the visual world.</h1>
            <p className="mt-4 text-base leading-relaxed text-white/55">Upload a source image, write the transformation, and let the logical OpenPaths image-edit route choose the available editing provider.</p>
          </div>
          <div className="mb-5 flex items-center gap-2 text-xs font-mono uppercase tracking-[0.16em] text-white/45"><ImageIcon className="h-4 w-4" /> GPT Image 2 route · provider fallbacks · one source image</div>
          <ImageSpacePanel
            modelId="openpaths/image-edit"
            modelName="OpenPaths Image Edit"
            imageToImage
            initialPrompt="Restyle this image as a softly lit editorial photograph with warm paper texture, restrained colors, and a premium art-book finish. Preserve the subject, pose, and composition."
            demo={{
              title: 'Source-preserving style transfer',
              description: 'The input image is uploaded to OpenPaths storage, then sent to /v1/images/edits. OpenPaths owns the GPT Image 2 and fallback provider route.',
              imageUrl: 'https://openpathsstatic.openpaths.io/static/uploads/playground/hidream-edit/perfume.jpg',
              outputUrl: 'https://openpathsstatic.openpaths.io/static/uploads/playground/hidream-edit/lipstick.png',
              prompt: 'Restyle the product photograph as a premium editorial still while preserving the scene and composition.',
              payload: { model: 'openpaths/image-edit', prompt: 'Restyle the product photograph as a premium editorial still while preserving the scene and composition.', image_url: 'https://openpathsstatic.openpaths.io/static/uploads/playground/hidream-edit/perfume.jpg', size: '1024x1024', n: 1 },
            }}
          />
        </div>
      </div>
    </>
  );
}
