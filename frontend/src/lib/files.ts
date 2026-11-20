/* Reading a file the user attached as text, for the model to read. */
import { MAX_FILE_BYTES } from '../config';
import type { FileInput } from '../types';

/** The file's text. Throws a message for the user when it is too large or isn't UTF-8 text. */
export async function readTextFile(file: File): Promise<FileInput> {
  if (file.size > MAX_FILE_BYTES) throw new Error(`${file.name} is too large: a file may be at most ${MAX_FILE_BYTES >> 10} KB.`);
  let text: string;
  try {
    text = new TextDecoder('utf-8', { fatal: true }).decode(await file.arrayBuffer());
  } catch {
    text = '\0'; // not UTF-8: refused below, like any binary file
  }
  if (text.includes('\0')) throw new Error(`${file.name} isn't a text file. Images and text files can be attached.`);
  return { name: file.name, text };
}
