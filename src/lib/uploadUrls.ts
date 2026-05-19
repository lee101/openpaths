const OPENPATHS_R2_UPLOAD_RE = /^https:\/\/[^/]+\.r2\.cloudflarestorage\.com\/openpathsstatic\/(uploads\/.+)$/i;

export function normalizeUploadedAssetUrl(url: string): string {
  const trimmed = url.trim();
  const match = trimmed.match(OPENPATHS_R2_UPLOAD_RE);
  if (match) {
    return `https://openpathsstatic.openpaths.io/${match[1]}`;
  }
  return trimmed;
}
