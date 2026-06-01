import React from 'react';
import { Link } from 'react-router-dom';
import { ArrowRight, Box, Boxes, ImageIcon, Layers, Palette, PersonStanding, Sparkles } from 'lucide-react';
import { Seo } from '../components/Seo';

const TOOLS = [
  {
    to: '/text-to-image',
    icon: ImageIcon,
    name: 'Text to Image',
    tagline: 'Auto image endpoint',
    description: 'Generate an image from a prompt. The auto image route picks the best model (GPT Image 2, RA1, Flux), or pin your own.',
    price: 'from ~$0.04 / image',
  },
  {
    to: '/image-to-3d',
    icon: Box,
    name: 'Image to 3D',
    tagline: 'Pixal3D · Meshy v6 · Tripo p1',
    description: 'Upload or paste an object image and get back a textured GLB you can preview, rotate, and download. Choose Pixal3D (fast), Meshy v6 (quality), or Tripo p1 (sharp geometry).',
    price: '$0.30 – $0.80',
  },
  {
    to: '/text-to-3d',
    icon: Boxes,
    name: 'Text to 3D',
    tagline: 'Auto Image + Pixal3D',
    description: 'Type a prompt: OpenPaths generates an image then converts it to a textured GLB in one call. Pay image + 3D price.',
    price: 'image + 3D price',
  },
  {
    to: '/rig-3d',
    icon: PersonStanding,
    name: '3D Auto-Rigging',
    tagline: 'Fal Meshy Rigging',
    description: 'Upload a humanoid GLB and get back a rigged character (GLB + FBX) with walk/run animations — preview it right in the browser.',
    price: '$0.20 – $0.32',
  },
  {
    to: '/retexture-3d',
    icon: Palette,
    name: '3D Retexture',
    tagline: 'Fal Trellis-2',
    description: 'Upload an existing mesh plus a reference image and re-skin it with a fresh texture — preview and download the new GLB.',
    price: '$0.20 – $0.24',
  },
  {
    to: '/playground',
    icon: Layers,
    name: 'Playground',
    tagline: 'Chat · image · video · audio',
    description: 'Multi-pane studio to test any model across modalities side by side, with live cost and latency.',
    price: 'pay per request',
  },
];

export function Tools() {
  return (
    <>
      <Seo
        title="Tools | OpenPaths"
        description="First-party OpenPaths tools: text-to-image, image-to-3D, text-to-3D, and the multi-model playground. Each tool has its own API."
        path="/tools"
      />

      <div className="px-6 py-12">
        <div className="mx-auto max-w-6xl">
          <div className="mb-10">
            <div className="mb-4 inline-flex items-center gap-2 rounded border border-white/10 bg-white/[0.03] px-3 py-1 text-xs font-mono text-white/45">
              <Sparkles className="h-3.5 w-3.5" /> Tools
            </div>
            <h1 className="text-4xl font-bold tracking-tight md:text-5xl">OpenPaths Tools</h1>
            <p className="mt-4 max-w-2xl text-base leading-relaxed text-white/55">
              Hands-on surfaces for our first-party generation endpoints. Every tool is a thin wrapper over the API — copy the snippet to ship it.
            </p>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            {TOOLS.map(tool => {
              const Icon = tool.icon;
              return (
                <Link
                  key={tool.to}
                  to={tool.to}
                  className="group flex flex-col gap-3 rounded-lg border border-white/10 bg-white/[0.02] p-6 transition-colors hover:border-white/30 hover:bg-white/[0.04]"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <span className="inline-flex h-10 w-10 items-center justify-center rounded border border-white/10 bg-white/[0.04] text-white/70">
                        <Icon className="h-5 w-5" />
                      </span>
                      <div>
                        <div className="text-lg font-bold tracking-tight">{tool.name}</div>
                        <div className="text-[11px] font-mono uppercase tracking-[0.16em] text-white/35">{tool.tagline}</div>
                      </div>
                    </div>
                    <ArrowRight className="h-4 w-4 text-white/30 transition-colors group-hover:text-white/70" />
                  </div>
                  <p className="text-sm leading-relaxed text-white/55">{tool.description}</p>
                  <div className="mt-auto text-xs font-mono text-white/40">{tool.price}</div>
                </Link>
              );
            })}
          </div>

          <p className="mt-8 text-sm font-mono text-white/40">
            Looking for raw endpoints? See the <Link to="/docs" className="text-white underline underline-offset-4">API docs</Link>.
          </p>
        </div>
      </div>
    </>
  );
}
