import { DOWNLOAD_REVOKE_MS } from '../config';

/** Saves a blob as a file through a temporary link. */
export function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.append(link);
  link.click();
  link.remove();
  setTimeout(() => URL.revokeObjectURL(url), DOWNLOAD_REVOKE_MS);
}

export const downloadText = (content: string, filename: string) => downloadBlob(new Blob([content], { type: 'application/octet-stream' }), filename);
