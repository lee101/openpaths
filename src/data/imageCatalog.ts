import { models, type Model } from './models';

export const AUTO_IMAGE = { id: 'openpaths/auto-image', name: 'Auto Image (routed)', price: 0.211 };

const isImageModel = (m: Model) =>
  m.tags.includes('art generation') && !m.tags.includes('video generation') && !m.tags.includes('image-to-3d') && !m.id.startsWith('openpaths/');

export const imageModels: Model[] = models
  .filter(isImageModel)
  .sort((a, b) => (a.popularity <= 0 ? 1e6 : a.popularity) - (b.popularity <= 0 ? 1e6 : b.popularity));

export const geminiImageModels = imageModels.filter(m => m.provider === 'Google' && m.id.startsWith('gemini'));

export const FEATURED_IMAGE_IDS = ['gpt-image-2', 'gemini-nano-banana-2.1', 'gemini-3.1-flash-lite-image', 'ra1', 'flux-pro'];

export const featuredImageModels = FEATURED_IMAGE_IDS
  .map(id => imageModels.find(m => m.id === id))
  .filter((m): m is Model => !!m);

export function imagePriceLabel(m: Model) {
  const price = m.priceInput;
  if (price <= 0) return 'Free';
  const digits = price < 0.1 ? 4 : 3;
  const s = price.toFixed(digits).replace(/0+$/, '').replace(/\.$/, '.00');
  return `$${s}/image`;
}

export const GEMINI_ASPECT_RATIOS = ['1:1', '2:3', '3:2', '3:4', '4:3', '4:5', '5:4', '9:16', '16:9', '21:9'];
