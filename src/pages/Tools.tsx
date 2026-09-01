import React from 'react';
import { Link } from 'react-router-dom';
import { ArrowRight, AudioLines, Box, Boxes, Clapperboard, Drama, GitMerge, ImageIcon, Layers, Music, Palette, PersonStanding, Scissors, Sparkles, WandSparkles } from 'lucide-react';
import { Seo } from '../components/Seo';

const TOOLS = [
  {
    to: '/tools/google-tts',
    icon: AudioLines,
    name: 'Gemini Flash TTS',
    tagline: '30 voices · multi-speaker',
    description: 'Direct expressive narration and two-person scenes with control over voice, style, pace, accent, context, and emotion.',
    price: 'Gemini token pricing',
  },
  {
    to: '/tools/lyria',
    icon: Music,
    name: 'Lyria 3 Music Studio',
    tagline: 'Pro songs · 30s clips · Opus',
    description: 'Compose full songs, scores, and loops with precise control over genre, mood, arrangement, vocals, lyrics, and structure.',
    price: '$0.04 clip · $0.08 pro',
  },
  {
    to: '/text-to-image',
    icon: ImageIcon,
    name: 'Text to Image',
    tagline: 'Auto image endpoint',
    description: 'Generate an image from a prompt. The auto image route picks the best model (GPT Image 2, RA1, Flux), or pin your own.',
    price: 'from ~$0.04 / image',
  },
  {
    to: '/image-edit',
    icon: WandSparkles,
    name: 'Image Style Transfer',
    tagline: 'GPT Image 2 · routed edits',
    description: 'Upload a source image and describe the new visual world. The shared image-edit route handles GPT Image 2 and fallbacks.',
    price: '~$0.30 / edit',
  },
  {
    to: '/text-to-video',
    icon: Clapperboard,
    name: 'Text to Video',
    tagline: 'Wan 3.0 · native audio',
    description: 'Describe a shot and get up to 30 seconds of cinematic video with native audio. Smart duration or pin exact seconds, per-resolution pricing.',
    price: '$0.05 – $0.20 / sec',
  },
  {
    to: '/image-to-video',
    icon: Clapperboard,
    name: 'Image to Video',
    tagline: 'Wan 3.0 · first + last frame',
    description: 'Animate any still with native audio, optionally driving it to an end frame. Smart duration or pin exact seconds.',
    price: '$0.05 – $0.20 / sec',
  },
  {
    to: '/video-extension',
    icon: Clapperboard,
    name: 'Video Extension',
    tagline: 'Grok Imagine Video',
    description: 'Upload or paste an MP4, describe what happens next, and extend it through the Grok Imagine video API.',
    price: '~$0.08 / sec',
  },
  {
    to: '/character-animator',
    icon: Drama,
    name: 'Character Animator',
    tagline: 'Wan-Animate · first-party',
    description: 'Animate a reference character with a driving performance video. Standard, fast, and x-fast latency lanes.',
    price: '$0.15 – $0.60 / sec',
  },
  {
    to: '/music-generator',
    icon: Music,
    name: 'Music Generator',
    tagline: 'MiniMax-Music3 · first-party',
    description: 'Full songs with vocals from a prompt and optional lyrics. Pin 30–300 seconds.',
    price: 'from $0.35 / track',
  },
  {
    to: '/remove-video-background',
    icon: Scissors,
    name: 'Background Remover',
    tagline: 'Transparent video · first-party',
    description: 'Key any clip to alpha-transparent WebM, optionally compositing a solid backdrop color.',
    price: '~$0.005 / sec',
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
  {
    to: '/fusion',
    icon: GitMerge,
    name: 'Model Fusion',
    tagline: 'OpenRouter fusion beta',
    description: 'Run multiple models side by side, analyze consensus and contradictions, then fuse the best result into one answer.',
    price: 'panel + judge cost',
  },
];

export function Tools() {
  return (
    <>
      <Seo
        title="Tools | OpenPaths"
        description="First-party OpenPaths tools: text-to-image, image-to-3D, text-to-3D, model fusion, and the multi-model playground. Each tool has its own API."
        path="/tools"
      />

      <div className="px-6 py-12">
        <div className="mx-auto max-w-6xl">
          <div className="mb-10">
            <div className="mb-4 inline-flex items-center gap-2 rounded border border-white/20 bg-white/[0.06] px-3 py-1 text-xs font-mono text-white/45">
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
                  className="group flex flex-col gap-3 rounded-lg border border-white/20 bg-white/[0.05] p-6 transition-colors hover:border-white/50 hover:bg-white/[0.07]"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <span className="inline-flex h-10 w-10 items-center justify-center rounded border border-white/20 bg-white/[0.07] text-white/70">
                        <Icon className="h-5 w-5" />
                      </span>
                      <div>
                        <div className="text-lg font-bold tracking-tight">{tool.name}</div>
                        <div className="text-[11px] font-mono uppercase tracking-[0.16em] text-white/50">{tool.tagline}</div>
                      </div>
                    </div>
                    <ArrowRight className="h-4 w-4 text-white/45 transition-colors group-hover:text-white/70" />
                  </div>
                  <p className="text-sm leading-relaxed text-white/55">{tool.description}</p>
                  <div className="mt-auto text-xs font-mono text-white/55">{tool.price}</div>
                </Link>
              );
            })}
          </div>

          <p className="mt-8 text-sm font-mono text-white/55">
            Looking for raw endpoints? See the <Link to="/docs" className="text-white underline underline-offset-4">API docs</Link>.
          </p>
        </div>
      </div>
    </>
  );
}
