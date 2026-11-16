/* Download PDF: the backend's PDF when it can make one (real, selectable text), else one made here. */
import { ApiError } from '../api/errors';
import { STATUS_NOT_IMPLEMENTED } from '../config';
import type { Artifact } from '../types';
import { downloadBlob } from './download';

/** The backend said it has no browser to render with. That won't change while the page is open, so it is asked once. */
let serverCantRender = false;

/** Saves a report as a PDF. `fromServer` (when given) is asked first; if it fails for any reason, the PDF is made
    in the browser instead (an image of each page). Rejects only when that fails too. */
export async function downloadReportPdf(a: Artifact, filename: string, fromServer?: (id: string) => Promise<Blob>): Promise<void> {
  if (fromServer && !serverCantRender) {
    try {
      return downloadBlob(await fromServer(a.id), filename);
    } catch (e) {
      if (e instanceof ApiError && e.status === STATUS_NOT_IMPLEMENTED) serverCantRender = true;
    }
  }
  const { saveReportPdf } = await import('./reportPdf'); // html2pdf.js is fetched on first use
  await saveReportPdf(a.content || '', filename);
}
