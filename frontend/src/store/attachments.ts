/* The images and text files attached to the message being written. */
import { IMAGE_TYPES, MAX_FILES_PER_MESSAGE, MAX_IMAGES_PER_MESSAGE } from '../config';
import { readTextFile } from '../lib/files';
import { prepareImage } from '../lib/images';
import type { FileInput, ImageInput } from '../types';
import type { Slice } from './types';

export interface Attachment extends ImageInput {
  id: number;
}

export interface FileAttachment extends FileInput {
  id: number;
}

export interface AttachmentsSlice {
  attachments: Attachment[];
  attachedFiles: FileAttachment[];
  /** How many images are still being read or shrunk. */
  attaching: number;
  /** Attaches these files: images as images, anything else as text; says why when some can't be. */
  attachFiles(files: Iterable<File>): Promise<void>;
  removeAttachment(id: number): void;
  removeFile(id: number): void;
  setAttachments(images: ImageInput[], files?: FileInput[]): void;
}

let nextId = 1;
const withId = <T>(item: T): T & { id: number } => ({ ...item, id: nextId++ });

export const createAttachmentsSlice: Slice<AttachmentsSlice> = (set, get) => ({
  attachments: [],
  attachedFiles: [],
  attaching: 0,

  async attachFiles(files) {
    const all = [...files];
    let images = all.filter((f) => IMAGE_TYPES.includes(f.type));
    let texts = all.filter((f) => !IMAGE_TYPES.includes(f.type));
    const room = MAX_IMAGES_PER_MESSAGE - get().attachments.length - get().attaching;
    if (images.length > room) {
      get().toast(`A message can carry up to ${MAX_IMAGES_PER_MESSAGE} images.`);
      images = images.slice(0, Math.max(0, room));
    }
    const fileRoom = MAX_FILES_PER_MESSAGE - get().attachedFiles.length;
    if (texts.length > fileRoom) {
      get().toast(`A message can carry up to ${MAX_FILES_PER_MESSAGE} files.`);
      texts = texts.slice(0, Math.max(0, fileRoom));
    }
    for (const file of texts) {
      const read = await readTextFile(file).catch((e: Error) => void get().toast(e.message));
      if (read) set((s) => ({ attachedFiles: [...s.attachedFiles, withId(read)] }));
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

  removeFile(id) {
    set((s) => ({ attachedFiles: s.attachedFiles.filter((f) => f.id !== id) }));
  },

  setAttachments(images, files = []) {
    set({ attachments: images.map(withId), attachedFiles: files.map(withId) });
  },
});
