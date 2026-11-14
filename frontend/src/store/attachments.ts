/* The images attached to the message being written. */
import { IMAGE_TYPES, MAX_IMAGES_PER_MESSAGE } from '../config';
import { prepareImage } from '../lib/images';
import type { ImageInput } from '../types';
import type { Slice } from './types';

export interface Attachment extends ImageInput {
  id: number;
}

export interface AttachmentsSlice {
  attachments: Attachment[];
  /** How many images are still being read or shrunk. */
  attaching: number;
  /** Attaches the images among these files; says why when some can't be. */
  attachFiles(files: Iterable<File>): Promise<void>;
  removeAttachment(id: number): void;
  setAttachments(images: ImageInput[]): void;
}

let nextId = 1;
const withId = (image: ImageInput): Attachment => ({ ...image, id: nextId++ });

export const createAttachmentsSlice: Slice<AttachmentsSlice> = (set, get) => ({
  attachments: [],
  attaching: 0,

  async attachFiles(files) {
    const all = [...files];
    let images = all.filter((f) => IMAGE_TYPES.includes(f.type));
    if (images.length < all.length) get().toast('Only PNG, JPEG, WebP and GIF images can be attached.');
    const room = MAX_IMAGES_PER_MESSAGE - get().attachments.length - get().attaching;
    if (images.length > room) {
      get().toast(`A message can carry up to ${MAX_IMAGES_PER_MESSAGE} images.`);
      images = images.slice(0, Math.max(0, room));
    }
    set((s) => ({ attaching: s.attaching + images.length }));
    for (const file of images) {
      // one at a time, so they land in the order they were picked
      const image = await prepareImage(file).catch((e: Error) => void get().toast(e.message));
      set((s) => ({ attaching: s.attaching - 1, attachments: image ? [...s.attachments, withId(image)] : s.attachments }));
    }
  },

  removeAttachment(id) {
    set((s) => ({ attachments: s.attachments.filter((a) => a.id !== id) }));
  },

  setAttachments(images) {
    set({ attachments: images.map(withId) });
  },
});
