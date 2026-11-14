/* Getting an image file ready to send: read it, and shrink it first when it is large. */
import { IMAGE_DOWNSCALE, MAX_IMAGE_BYTES } from '../config';
import type { ImageInput } from '../types';

const JPEG = 'image/jpeg';

export const dataUrl = (i: ImageInput) => `data:${i.mediaType};base64,${i.data}`;

/** Resolves once the browser has this image ready to paint. */
export const preloadImage = (src: string) => Object.assign(new Image(), { src }).decode();

function base64(blob: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.addEventListener('load', () => resolve(String(reader.result).split(',')[1] ?? ''));
    reader.addEventListener('error', () => reject(reader.error));
    reader.readAsDataURL(blob);
  });
}

/** Draws the image at this size and encodes it. JPEG has no transparency, so it gets a white background. */
function encode(image: ImageBitmap, width: number, height: number, type: string): Promise<Blob | null> {
  const canvas = Object.assign(document.createElement('canvas'), { width, height });
  const ctx = canvas.getContext('2d')!;
  if (type === JPEG) {
    ctx.fillStyle = '#fff';
    ctx.fillRect(0, 0, width, height);
  }
  ctx.drawImage(image, 0, 0, width, height);
  return new Promise((resolve) => canvas.toBlob(resolve, type, IMAGE_DOWNSCALE.quality));
}

/** The file as an upload. Small images and GIFs (which may be animated) go as they are; a large one is
    scaled down, as PNG when it was one and still fits, otherwise as JPEG. Throws a message for the user. */
export async function prepareImage(file: File): Promise<ImageInput> {
  const name = file.name || 'That image';
  const tooBig = new Error(`${name} is too large to attach, even after shrinking it.`);
  const asIs = async () => ({ mediaType: file.type, data: await base64(file) });
  if (file.type === 'image/gif') {
    if (file.size > MAX_IMAGE_BYTES) throw tooBig;
    return asIs();
  }
  const image = await createImageBitmap(file).catch(() => {
    throw new Error(`${name} couldn't be read as an image.`);
  });
  const scale = Math.min(1, IMAGE_DOWNSCALE.maxSide / Math.max(image.width, image.height));
  if (scale === 1 && file.size <= MAX_IMAGE_BYTES) return asIs();
  const width = Math.round(image.width * scale);
  const height = Math.round(image.height * scale);
  let blob = file.type === 'image/png' ? await encode(image, width, height, file.type) : null;
  if (!blob || blob.size > MAX_IMAGE_BYTES) blob = await encode(image, width, height, JPEG);
  if (!blob || blob.size > MAX_IMAGE_BYTES) throw tooBig;
  return { mediaType: blob.type, data: await base64(blob) };
}
