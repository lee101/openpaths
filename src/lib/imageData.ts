export function b64ImageMime(b64: string) {
  if (b64.startsWith('UklGR')) return 'image/webp';
  if (b64.startsWith('/9j/')) return 'image/jpeg';
  return 'image/png';
}

export function b64ImageSrc(b64: string) {
  return `data:${b64ImageMime(b64)};base64,${b64}`;
}

export function b64ImageExt(b64: string) {
  return b64ImageMime(b64) === 'image/webp' ? 'webp' : b64ImageMime(b64) === 'image/jpeg' ? 'jpg' : 'png';
}
