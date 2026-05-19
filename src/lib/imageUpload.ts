const WEBP_UPLOAD_QUALITY = 0.85;

function imageBitmapToWebPBlob(bitmap: ImageBitmap): Promise<Blob> {
  const canvas = document.createElement('canvas');
  canvas.width = bitmap.width;
  canvas.height = bitmap.height;
  const ctx = canvas.getContext('2d');
  if (!ctx) throw new Error('Could not prepare image canvas');
  ctx.drawImage(bitmap, 0, 0);
  return new Promise((resolve, reject) => {
    canvas.toBlob(blob => {
      if (blob) resolve(blob);
      else reject(new Error('Could not convert image to WebP'));
    }, 'image/webp', WEBP_UPLOAD_QUALITY);
  });
}

function webpFilename(name: string): string {
  return name.replace(/\.[^.]*$/, '') + '.webp';
}

export async function prepareUploadFile(file: File): Promise<File> {
  if (!file.type.startsWith('image/') || file.type === 'image/webp') {
    return file;
  }

  const bitmap = await createImageBitmap(file);
  try {
    const blob = await imageBitmapToWebPBlob(bitmap);
    return new File([blob], webpFilename(file.name || 'upload.png'), {
      type: 'image/webp',
      lastModified: file.lastModified,
    });
  } finally {
    bitmap.close();
  }
}
