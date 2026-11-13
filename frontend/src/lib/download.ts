import { DOWNLOAD_REVOKE_MS } from '../config';

/** Saves text as a file through a temporary link. */
export function downloadText(content: string, filename: string) {
  const url = URL.createObjectURL(new Blob([content], { type: 'application/octet-stream' }));
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.append(link);
  link.click();
  link.remove();
  setTimeout(() => URL.revokeObjectURL(url), DOWNLOAD_REVOKE_MS);
}
