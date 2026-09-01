import React, { useCallback, useEffect, useRef, useState } from 'react';
import {
  ArrowRight,
  CheckCircle2,
  Download,
  Film,
  Image as ImageIcon,
  Loader2,
  Sparkles,
  Wallet,
} from 'lucide-react';
import { AuthModal } from './AuthModal';
import { TopUpModal } from './TopUpModal';
import { type ZImageArtItem } from '../data/zimageArt';
import { AUTH_EVENT, getApiKey } from '../lib/api';

export type ArtGenerationMode = 'image' | 'video';

type ImageData = { url?: string; b64_json?: string; revised_prompt?: string };
type ImageGenerationResponse = { data?: ImageData[] };
type VideoGenerationResponse = {
  id?: string;
  job_id?: string;
  status?: string;
  video_url?: string;
  result_url?: string;
  result?: { video_url?: string };
};

type ImageModel = {
  id: string;
  label: string;
  price: number;
  sizes: string[];
};

type VideoModel = {
  id: string;
  label: string;
  hint: string;
  resolutions: Array<{ id: string; label: string; rate: number }>;
  durations: number[];
  inputImagePrice?: number;
  audio: 'always' | 'optional' | 'none';
  sourceControlsAspect?: boolean;
};

const IMAGE_MODELS: ImageModel[] = [
  {
    id: 'zimage',
    label: 'ZImage · fast anime & illustration',
    price: 0.007,
    sizes: ['1024x1024', '1024x768', '768x1024', '1024x576', '576x1024'],
  },
  {
    id: 'ra1',
    label: 'RA1 · versatile art',
    price: 0.04,
    sizes: ['1024x1024', '1152x768', '768x1152', '1360x768', '768x1360'],
  },
  {
    id: 'flux-pro',
    label: 'FLUX Pro · prompt fidelity',
    price: 0.04,
    sizes: ['1024x1024', '1152x768', '768x1152', '1360x768', '768x1360'],
  },
  {
    id: 'gpt-image-2',
    label: 'GPT Image 2 · premium',
    price: 0.211,
    sizes: ['1024x1024', '1536x1024', '1024x1536', '2048x1152', '2160x3840'],
  },
];

const VIDEO_MODELS: VideoModel[] = [
  {
    id: 'minimax-h3-max-image-to-video',
    label: 'MiniMax H3 Max',
    hint: 'Best balance · native audio',
    resolutions: [
      { id: '480p', label: '480p', rate: 0.05 },
      { id: '768p', label: '768p', rate: 0.08 },
    ],
    durations: [5, 10, 15],
    audio: 'always',
    sourceControlsAspect: true,
  },
  {
    id: 'wan-3.0-image-to-video',
    label: 'Wan 3.0',
    hint: 'Long shots · native audio',
    resolutions: [
      { id: '480p', label: '480p', rate: 0.05 },
      { id: '720p', label: '720p', rate: 0.1 },
      { id: '1080p', label: '1080p', rate: 0.2 },
    ],
    durations: [5, 10, 15, 20, 30],
    audio: 'optional',
  },
  {
    id: 'grok-imagine-video-1.5',
    label: 'Grok Imagine Video 1.5',
    hint: 'Fast cinematic motion',
    resolutions: [
      { id: '480p', label: '480p', rate: 0.08 },
      { id: '720p', label: '720p', rate: 0.14 },
      { id: '1080p', label: '1080p', rate: 0.25 },
    ],
    durations: [5, 10],
    inputImagePrice: 0.01,
    audio: 'none',
  },
  {
    id: 'seedance-2.0-image-to-video',
    label: 'Seedance 2.0',
    hint: 'Premium motion & adherence',
    resolutions: [
      { id: '720p', label: '720p', rate: 0.33264 },
      { id: '1080p', label: '1080p', rate: 0.682 },
    ],
    durations: [5, 10, 15],
    audio: 'optional',
  },
];

const MOTION_PROMPT =
  'Bring this artwork to life while preserving the character, composition, and illustration style. Add subtle eye and hair movement, gentle environmental motion, natural parallax, and a slow cinematic push-in. Keep faces and details stable; no added text.';
const CONTROL_SELECT_CLASS =
  'w-full rounded-lg border border-white/20 bg-black px-3 py-2.5 font-mono text-xs text-white outline-none focus:border-white/50';

class ResponseError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = 'ResponseError';
    this.status = status;
  }
}

function formatPrice(value: number): string {
  return value < 0.01 ? `$${value.toFixed(3)}` : `$${value.toFixed(2)}`;
}

function imageSource(image: ImageData): string {
  if (image.url) return image.url;
  if (image.b64_json) return `data:image/png;base64,${image.b64_json}`;
  return '';
}

function responseVideoURL(data: VideoGenerationResponse): string {
  return data.video_url || data.result_url || data.result?.video_url || '';
}

function errorMessage(data: unknown, fallback: string): string {
  const body = data as { error?: string | { message?: string }; message?: string } | null;
  if (typeof body?.error === 'string') return body.error;
  return body?.error?.message || body?.message || fallback;
}

async function parseResponse<T>(response: Response, fallback: string): Promise<T> {
  const text = await response.text();
  let data: unknown = null;
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      if (!response.ok) throw new ResponseError(text.slice(0, 240), response.status);
      throw new Error('The generation API returned invalid JSON.');
    }
  }
  if (!response.ok) throw new ResponseError(errorMessage(data, fallback), response.status);
  return data as T;
}

function closestSize(item: ZImageArtItem, sizes: string[]): string {
  const ratio = item.width && item.height
    ? item.width / item.height
    : item.aspect === 'wide'
      ? 16 / 9
      : item.aspect === 'portrait'
        ? 9 / 16
        : 1;
  return sizes.reduce((best, size) => {
    const [width, height] = size.split('x').map(Number);
    const [bestWidth, bestHeight] = best.split('x').map(Number);
    return Math.abs(width / height - ratio) < Math.abs(bestWidth / bestHeight - ratio) ? size : best;
  }, sizes[0]);
}

export function ArtGenerationStudio({
  item,
  mode,
  onModeChange,
}: {
  item: ZImageArtItem;
  mode: ArtGenerationMode;
  onModeChange: (mode: ArtGenerationMode) => void;
}) {
  const [apiKey, setApiKey] = useState(getApiKey);
  const [balanceUSD, setBalanceUSD] = useState<number | null>(null);
  const [imagePrompt, setImagePrompt] = useState(item.prompt);
  const [imageModel, setImageModel] = useState('zimage');
  const [imageSize, setImageSize] = useState(() => closestSize(item, IMAGE_MODELS[0].sizes));
  const [imageCount, setImageCount] = useState(1);
  const [videoPrompt, setVideoPrompt] = useState(MOTION_PROMPT);
  const [sourceImageURL, setSourceImageURL] = useState(item.imageUrl);
  const [videoModel, setVideoModel] = useState(VIDEO_MODELS[0].id);
  const [resolution, setResolution] = useState(VIDEO_MODELS[0].resolutions.at(-1)?.id || '768p');
  const [duration, setDuration] = useState(5);
  const [aspectRatio, setAspectRatio] = useState('auto');
  const [generateAudio, setGenerateAudio] = useState(true);
  const [images, setImages] = useState<ImageData[]>([]);
  const [videoURL, setVideoURL] = useState('');
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState('');
  const [error, setError] = useState('');
  const [authOpen, setAuthOpen] = useState(false);
  const [topUpOpen, setTopUpOpen] = useState(false);
  const pendingAction = useRef<ArtGenerationMode | null>(null);
  const pollController = useRef<AbortController | null>(null);

  const selectedImageModel = IMAGE_MODELS.find(model => model.id === imageModel) || IMAGE_MODELS[0];
  const selectedVideoModel = VIDEO_MODELS.find(model => model.id === videoModel) || VIDEO_MODELS[0];
  const selectedResolution = selectedVideoModel.resolutions.find(option => option.id === resolution) || selectedVideoModel.resolutions[0];
  const imagePrice = selectedImageModel.price * imageCount;
  const videoPrice = selectedResolution.rate * duration + (selectedVideoModel.inputImagePrice || 0);

  const refreshBalance = useCallback(async (key = getApiKey()): Promise<number | null> => {
    if (!key) {
      setBalanceUSD(null);
      return null;
    }
    try {
      const response = await fetch('/account/balance', {
        headers: { Authorization: `Bearer ${key}` },
      });
      if (!response.ok) return null;
      const data = await response.json() as { balance_usd?: number; balance_cents?: number };
      const balance = typeof data.balance_usd === 'number'
        ? data.balance_usd
        : typeof data.balance_cents === 'number'
          ? data.balance_cents / 10000
          : null;
      setBalanceUSD(balance);
      return balance;
    } catch {
      return null;
    }
  }, []);

  useEffect(() => {
    setImagePrompt(item.prompt);
    setSourceImageURL(item.imageUrl);
    setImages([]);
    setVideoURL('');
    setError('');
    setImageSize(closestSize(item, selectedImageModel.sizes));
  }, [item.id, item.imageUrl, item.prompt]);

  useEffect(() => {
    const syncAuth = () => {
      const key = getApiKey();
      setApiKey(key);
      void refreshBalance(key);
    };
    syncAuth();
    window.addEventListener(AUTH_EVENT, syncAuth);
    window.addEventListener('auth-change', syncAuth);
    return () => {
      window.removeEventListener(AUTH_EVENT, syncAuth);
      window.removeEventListener('auth-change', syncAuth);
    };
  }, [refreshBalance]);

  useEffect(() => () => pollController.current?.abort(), []);

  useEffect(() => {
    const nextModel = IMAGE_MODELS.find(model => model.id === imageModel) || IMAGE_MODELS[0];
    if (!nextModel.sizes.includes(imageSize)) setImageSize(closestSize(item, nextModel.sizes));
  }, [imageModel]);

  useEffect(() => {
    const nextModel = VIDEO_MODELS.find(model => model.id === videoModel) || VIDEO_MODELS[0];
    if (!nextModel.resolutions.some(option => option.id === resolution)) {
      setResolution(nextModel.resolutions.at(-1)?.id || nextModel.resolutions[0].id);
    }
    if (!nextModel.durations.includes(duration)) setDuration(nextModel.durations[0]);
    if (nextModel.audio === 'always') setGenerateAudio(true);
    if (nextModel.audio === 'none') setGenerateAudio(false);
  }, [videoModel]);

  const onGenerationError = (reason: unknown) => {
    const message = reason instanceof Error ? reason.message : 'Generation failed';
    setError(message);
    setStatus('');
    if (reason instanceof ResponseError && reason.status === 401) {
      pendingAction.current = mode;
      setAuthOpen(true);
    }
    if (reason instanceof ResponseError && reason.status === 402) setTopUpOpen(true);
  };

  const generateImages = useCallback(async (key = getApiKey()) => {
    if (!imagePrompt.trim()) {
      setError('Add a prompt before generating.');
      return;
    }
    setLoading(true);
    setError('');
    setStatus(`Creating ${imageCount} image${imageCount === 1 ? '' : 's'}…`);
    try {
      const response = await fetch('/v1/images/generations', {
        method: 'POST',
        headers: { Authorization: `Bearer ${key}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({
          model: imageModel,
          prompt: imagePrompt.trim(),
          size: imageSize,
          n: imageCount,
          num_images: imageCount,
        }),
      });
      const data = await parseResponse<ImageGenerationResponse>(response, 'Image generation failed');
      const generated = (data.data || []).filter(image => imageSource(image));
      if (!generated.length) throw new Error('Image generation completed without an image.');
      setImages(generated);
      setStatus(`${generated.length} image${generated.length === 1 ? '' : 's'} ready`);
      await refreshBalance(key);
    } catch (reason) {
      onGenerationError(reason);
    } finally {
      setLoading(false);
    }
  }, [imageCount, imageModel, imagePrompt, imageSize, refreshBalance]);

  const pollVideo = useCallback(async (jobID: string, key: string, signal: AbortSignal): Promise<string> => {
    for (let attempt = 0; attempt < 240; attempt += 1) {
      await new Promise<void>((resolve, reject) => {
        const onAbort = () => {
          window.clearTimeout(timer);
          reject(new DOMException('Aborted', 'AbortError'));
        };
        const timer = window.setTimeout(() => {
          signal.removeEventListener('abort', onAbort);
          resolve();
        }, 2000);
        signal.addEventListener('abort', onAbort, { once: true });
      });
      const response = await fetch(`/v1/videos/generations/${encodeURIComponent(jobID)}`, {
        headers: { Authorization: `Bearer ${key}` },
        signal,
      });
      const data = await parseResponse<VideoGenerationResponse>(response, 'Could not check video progress');
      const result = responseVideoURL(data);
      if (data.status === 'completed' || result) return result;
      if (data.status === 'failed') throw new Error(errorMessage(data, 'Video generation failed'));
      setStatus(data.status === 'queued' || data.status === 'pending' ? 'Video queued…' : 'Animating artwork…');
    }
    throw new Error('Video generation timed out.');
  }, []);

  const generateVideo = useCallback(async (key = getApiKey()) => {
    if (!sourceImageURL.trim()) {
      setError('This artwork does not have an image URL to animate.');
      return;
    }
    if (!videoPrompt.trim()) {
      setError('Describe how the artwork should move.');
      return;
    }
    pollController.current?.abort();
    const controller = new AbortController();
    pollController.current = controller;
    setLoading(true);
    setVideoURL('');
    setError('');
    setStatus('Starting video…');
    try {
      const response = await fetch('/v1/videos/generations', {
        method: 'POST',
        headers: { Authorization: `Bearer ${key}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({
          model: videoModel,
          prompt: videoPrompt.trim(),
          image_url: sourceImageURL.trim(),
          resolution,
          duration,
          aspect_ratio: selectedVideoModel.sourceControlsAspect ? 'auto' : aspectRatio,
          generate_audio: selectedVideoModel.audio === 'always' ? true : selectedVideoModel.audio === 'optional' ? generateAudio : false,
          async: true,
        }),
        signal: controller.signal,
      });
      const data = await parseResponse<VideoGenerationResponse>(response, 'Video generation failed');
      let result = responseVideoURL(data);
      const jobID = data.id || data.job_id;
      if (!result && jobID) result = await pollVideo(jobID, key, controller.signal);
      if (!result) throw new Error('Video generation completed without a video.');
      setVideoURL(result);
      setStatus('Video ready');
      await refreshBalance(key);
    } catch (reason) {
      if (reason instanceof DOMException && reason.name === 'AbortError') return;
      onGenerationError(reason);
    } finally {
      if (pollController.current === controller) pollController.current = null;
      setLoading(false);
    }
  }, [aspectRatio, duration, generateAudio, pollVideo, refreshBalance, resolution, selectedVideoModel.audio, selectedVideoModel.sourceControlsAspect, sourceImageURL, videoModel, videoPrompt]);

  const beginGeneration = useCallback(async (requestedMode = mode) => {
    const key = getApiKey();
    if (!key) {
      pendingAction.current = requestedMode;
      setAuthOpen(true);
      return;
    }
    if (balanceUSD !== null && balanceUSD <= 0) {
      setTopUpOpen(true);
      return;
    }
    if (requestedMode === 'image') await generateImages(key);
    else await generateVideo(key);
  }, [balanceUSD, generateImages, generateVideo, mode]);

  const handleAuthSuccess = async () => {
    const key = getApiKey();
    const action = pendingAction.current;
    pendingAction.current = null;
    setApiKey(key);
    const balance = await refreshBalance(key);
    if (balance !== null && balance <= 0) {
      setTopUpOpen(true);
      return;
    }
    if (action === 'image') await generateImages(key);
    if (action === 'video') await generateVideo(key);
  };

  const useImageForVideo = (image: ImageData) => {
    const source = imageSource(image);
    if (!source) return;
    setSourceImageURL(source);
    onModeChange('video');
    setVideoURL('');
    setStatus('Generated image selected as the first frame');
  };

  const balanceLabel = !apiKey
    ? 'Sign in to generate'
    : balanceUSD === null
      ? 'Checking balance…'
      : balanceUSD <= 0
        ? 'No credits remaining'
        : `${formatPrice(balanceUSD)} balance`;

  return (
    <section id="create-from-art" className="mt-12 overflow-hidden rounded-xl border border-white/20 bg-white/[0.05]" data-testid="art-generation-studio">
      <div className="border-b border-white/15 bg-[radial-gradient(circle_at_top_left,rgba(99,102,241,0.22),transparent_42%),rgba(255,255,255,0.02)] px-5 py-6 sm:px-7">
        <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <div className="mb-2 flex items-center gap-2 font-mono text-[11px] uppercase tracking-[0.18em] text-indigo-200/70">
              <Sparkles className="h-4 w-4" /> Create from this artwork
            </div>
            <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">Make a variation or bring it to life</h2>
            <p className="mt-2 max-w-2xl text-sm leading-relaxed text-white/55">
              The prompt and first frame are already loaded. Pick a model, see the exact estimate, and generate without leaving this page.
            </p>
          </div>
          <button
            type="button"
            onClick={() => apiKey && balanceUSD !== null && balanceUSD <= 0 ? setTopUpOpen(true) : undefined}
            className={`inline-flex items-center gap-2 self-start rounded-full border px-3 py-2 font-mono text-xs ${apiKey && balanceUSD !== null && balanceUSD <= 0 ? 'border-amber-300/30 bg-amber-300/10 text-amber-100 hover:bg-amber-300/15' : 'border-white/15 bg-black/30 text-white/55'}`}
            data-testid="art-generation-balance"
          >
            <Wallet className="h-3.5 w-3.5" /> {balanceLabel}
          </button>
        </div>
      </div>

      <div className="grid lg:grid-cols-[minmax(0,1fr)_minmax(320px,0.8fr)]">
        <div className="border-b border-white/15 p-5 sm:p-7 lg:border-b-0 lg:border-r">
          <div className="mb-5 grid grid-cols-2 rounded-lg border border-white/15 bg-black/40 p-1" role="tablist" aria-label="Generation mode">
            <button
              type="button"
              role="tab"
              aria-selected={mode === 'image'}
              disabled={loading}
              onClick={() => onModeChange('image')}
              className={`inline-flex items-center justify-center gap-2 rounded-md px-3 py-2.5 font-mono text-xs font-bold transition-colors disabled:cursor-wait ${mode === 'image' ? 'bg-white text-black' : 'text-white/55 hover:text-white'}`}
              data-testid="art-generate-image-tab"
            >
              <ImageIcon className="h-4 w-4" /> Generate image
            </button>
            <button
              type="button"
              role="tab"
              aria-selected={mode === 'video'}
              disabled={loading}
              onClick={() => onModeChange('video')}
              className={`inline-flex items-center justify-center gap-2 rounded-md px-3 py-2.5 font-mono text-xs font-bold transition-colors disabled:cursor-wait ${mode === 'video' ? 'bg-white text-black' : 'text-white/55 hover:text-white'}`}
              data-testid="art-generate-video-tab"
            >
              <Film className="h-4 w-4" /> Animate image
            </button>
          </div>

          {mode === 'image' ? (
            <div data-testid="art-image-controls">
              <label className="block">
                <span className="mb-1.5 block font-mono text-[10px] uppercase tracking-[0.16em] text-white/55">Prompt</span>
                <textarea value={imagePrompt} onChange={event => setImagePrompt(event.target.value)} rows={6} className="w-full resize-y rounded-lg border border-white/20 bg-black px-3 py-3 text-sm leading-relaxed text-white outline-none focus:border-white/50" />
              </label>
              <div className="mt-3 grid gap-3 sm:grid-cols-3">
                <Control label="Model">
                  <select value={imageModel} onChange={event => setImageModel(event.target.value)} className={CONTROL_SELECT_CLASS}>
                    {IMAGE_MODELS.map(model => <option key={model.id} value={model.id}>{model.label}</option>)}
                  </select>
                </Control>
                <Control label="Canvas">
                  <select value={imageSize} onChange={event => setImageSize(event.target.value)} className={CONTROL_SELECT_CLASS}>
                    {selectedImageModel.sizes.map(size => <option key={size}>{size}</option>)}
                  </select>
                </Control>
                <Control label="Images">
                  <select value={imageCount} onChange={event => setImageCount(Number(event.target.value))} className={CONTROL_SELECT_CLASS}>
                    {[1, 2, 4].map(count => <option key={count} value={count}>{count}</option>)}
                  </select>
                </Control>
              </div>
              <PriceLine label={`${imageCount} image${imageCount === 1 ? '' : 's'} · ${formatPrice(selectedImageModel.price)} each`} price={imagePrice} />
            </div>
          ) : (
            <div data-testid="art-video-controls">
              <div className="mb-3 flex items-center gap-3 rounded-lg border border-white/15 bg-black/40 p-2.5">
                <img src={sourceImageURL} alt="First frame" className="h-16 w-16 shrink-0 rounded-md bg-white/[0.06] object-cover" />
                <div className="min-w-0">
                  <div className="font-mono text-[10px] uppercase tracking-[0.16em] text-white/45">First frame locked</div>
                  <p className="mt-1 truncate text-xs text-white/65">{sourceImageURL === item.imageUrl ? 'This artwork' : 'Your generated variation'}</p>
                </div>
                {sourceImageURL !== item.imageUrl && (
                  <button type="button" onClick={() => setSourceImageURL(item.imageUrl)} className="ml-auto shrink-0 font-mono text-[10px] text-white/45 hover:text-white">Reset</button>
                )}
              </div>
              <label className="block">
                <span className="mb-1.5 block font-mono text-[10px] uppercase tracking-[0.16em] text-white/55">Motion prompt</span>
                <textarea value={videoPrompt} onChange={event => setVideoPrompt(event.target.value)} rows={5} className="w-full resize-y rounded-lg border border-white/20 bg-black px-3 py-3 text-sm leading-relaxed text-white outline-none focus:border-white/50" />
              </label>
              <div className="mt-3 grid gap-3 sm:grid-cols-2">
                <Control label="Model">
                  <select value={videoModel} onChange={event => setVideoModel(event.target.value)} className={CONTROL_SELECT_CLASS}>
                    {VIDEO_MODELS.map(model => <option key={model.id} value={model.id}>{model.label} · {model.hint}</option>)}
                  </select>
                </Control>
                <Control label="Resolution">
                  <select value={resolution} onChange={event => setResolution(event.target.value)} className={CONTROL_SELECT_CLASS}>
                    {selectedVideoModel.resolutions.map(option => <option key={option.id} value={option.id}>{option.label} · {formatPrice(option.rate)}/s</option>)}
                  </select>
                </Control>
                <Control label="Duration">
                  <select value={duration} onChange={event => setDuration(Number(event.target.value))} className={CONTROL_SELECT_CLASS}>
                    {selectedVideoModel.durations.map(seconds => <option key={seconds} value={seconds}>{seconds} seconds</option>)}
                  </select>
                </Control>
                <Control label="Frame shape">
                  <select value={selectedVideoModel.sourceControlsAspect ? 'auto' : aspectRatio} disabled={selectedVideoModel.sourceControlsAspect} onChange={event => setAspectRatio(event.target.value)} className={`${CONTROL_SELECT_CLASS} disabled:text-white/40`}>
                    <option value="auto">Match source</option>
                    {['16:9', '9:16', '1:1', '4:3', '3:4'].map(aspect => <option key={aspect}>{aspect}</option>)}
                  </select>
                </Control>
              </div>
              <label className={`mt-3 flex items-center justify-between gap-3 rounded-lg border border-white/15 bg-black/40 px-3 py-2.5 text-xs ${selectedVideoModel.audio !== 'none' ? 'text-white/65' : 'text-white/30'}`}>
                <span>{selectedVideoModel.audio === 'always' ? 'Native synchronized audio included' : 'Generate synchronized audio'}</span>
                <input type="checkbox" checked={selectedVideoModel.audio === 'always' || (selectedVideoModel.audio === 'optional' && generateAudio)} disabled={selectedVideoModel.audio !== 'optional'} onChange={event => setGenerateAudio(event.target.checked)} className="h-4 w-4 accent-white" />
              </label>
              <PriceLine label={`${duration}s · ${selectedResolution.label}${selectedVideoModel.inputImagePrice ? ` · ${formatPrice(selectedVideoModel.inputImagePrice)} first frame` : ''}`} price={videoPrice} />
            </div>
          )}

          <button
            type="button"
            onClick={() => void beginGeneration()}
            disabled={loading}
            className="mt-4 inline-flex w-full items-center justify-center gap-2 rounded-lg bg-white px-4 py-3.5 font-mono text-sm font-bold text-black transition-colors hover:bg-white/90 disabled:cursor-wait disabled:opacity-60"
            data-testid="art-generate-submit"
          >
            {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : balanceUSD !== null && balanceUSD <= 0 ? <Wallet className="h-4 w-4" /> : <Sparkles className="h-4 w-4" />}
            {loading
              ? mode === 'image' ? 'Generating…' : 'Animating…'
              : !apiKey
                ? 'Sign in to generate'
                : balanceUSD !== null && balanceUSD <= 0
                  ? 'Add credits to generate'
                  : `${mode === 'image' ? 'Generate' : 'Animate'} · ${formatPrice(mode === 'image' ? imagePrice : videoPrice)}`}
          </button>
          <p className="mt-2 text-center font-mono text-[10px] text-white/35">Pay per generation · no subscription required</p>
          {error && <p className="mt-3 rounded-lg border border-red-400/20 bg-red-400/10 px-3 py-2.5 text-xs leading-relaxed text-red-200" role="alert" data-testid="art-generation-error">{error}</p>}
        </div>

        <div className="flex min-h-[420px] flex-col bg-black/20 p-5 sm:p-7" data-testid="art-generation-result">
          <div className="mb-4 flex items-center justify-between gap-3 font-mono text-[10px] uppercase tracking-[0.16em] text-white/45">
            <span>Result</span>
            {status && <span className="inline-flex items-center gap-1.5 text-white/60">{loading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <CheckCircle2 className="h-3.5 w-3.5 text-emerald-300" />}{status}</span>}
          </div>
          {mode === 'image' && images.length > 0 ? (
            <div className={`grid gap-3 ${images.length > 1 ? 'sm:grid-cols-2' : ''}`}>
              {images.map((image, index) => {
                const source = imageSource(image);
                return (
                  <div key={`${source.slice(0, 80)}-${index}`} className="overflow-hidden rounded-lg border border-white/15 bg-black">
                    <img src={source} alt={`Generated variation ${index + 1}`} className="w-full object-contain" />
                    <div className="grid grid-cols-2 border-t border-white/15 font-mono text-[10px]">
                      <a href={source} download={`openpaths-variation-${index + 1}.png`} className="inline-flex items-center justify-center gap-1.5 border-r border-white/15 px-2 py-2.5 text-white/50 hover:text-white"><Download className="h-3.5 w-3.5" /> Download</a>
                      <button type="button" onClick={() => useImageForVideo(image)} className="inline-flex items-center justify-center gap-1.5 px-2 py-2.5 text-white/65 hover:text-white"><Film className="h-3.5 w-3.5" /> Animate</button>
                    </div>
                  </div>
                );
              })}
            </div>
          ) : mode === 'video' && videoURL ? (
            <div className="overflow-hidden rounded-lg border border-white/15 bg-black">
              <video src={videoURL} poster={sourceImageURL} controls autoPlay playsInline className="w-full" data-testid="art-generated-video" />
              <a href={videoURL} download="openpaths-art-video.mp4" className="flex items-center justify-center gap-2 border-t border-white/15 px-3 py-3 font-mono text-xs text-white/60 hover:text-white"><Download className="h-4 w-4" /> Download video</a>
            </div>
          ) : loading ? (
            <div className="flex flex-1 flex-col items-center justify-center gap-4 rounded-lg border border-dashed border-white/15 bg-white/[0.02] text-center">
              <Loader2 className="h-7 w-7 animate-spin text-white/55" />
              <div><p className="text-sm text-white/70">{status}</p><p className="mt-1 font-mono text-[10px] text-white/35">You can keep this page open while it renders.</p></div>
            </div>
          ) : (
            <div className="flex flex-1 flex-col items-center justify-center rounded-lg border border-dashed border-white/15 bg-white/[0.02] p-8 text-center">
              {mode === 'image' ? <ImageIcon className="h-8 w-8 text-white/25" /> : <Film className="h-8 w-8 text-white/25" />}
              <p className="mt-4 text-sm text-white/60">{mode === 'image' ? 'Your new variations will appear here.' : 'Your animated artwork will play here.'}</p>
              <p className="mt-2 max-w-xs text-xs leading-relaxed text-white/35">All results stay on this page so you can download them or continue from a generated frame.</p>
            </div>
          )}
          {(images.length > 0 || videoURL) && !loading && (
            <button type="button" onClick={() => void beginGeneration()} className="mt-4 inline-flex items-center justify-center gap-2 rounded-lg border border-white/15 px-3 py-2.5 font-mono text-xs text-white/60 hover:border-white/40 hover:text-white">
              Generate again <ArrowRight className="h-3.5 w-3.5" />
            </button>
          )}
        </div>
      </div>

      <AuthModal open={authOpen} onClose={() => { setAuthOpen(false); pendingAction.current = null; }} onSuccess={() => void handleAuthSuccess()} />
      <TopUpModal open={topUpOpen} initialAmount={25} onClose={() => { setTopUpOpen(false); void refreshBalance(); }} />
    </section>
  );
}

function Control({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label>
      <span className="mb-1.5 block font-mono text-[10px] uppercase tracking-[0.16em] text-white/55">{label}</span>
      {children}
    </label>
  );
}

function PriceLine({ label, price }: { label: string; price: number }) {
  return (
    <div className="mt-4 flex items-center justify-between gap-4 rounded-lg border border-indigo-300/20 bg-indigo-300/[0.07] px-3 py-2.5 font-mono text-xs">
      <span className="text-white/50">{label}</span>
      <strong className="text-indigo-100">Estimated {formatPrice(price)}</strong>
    </div>
  );
}
