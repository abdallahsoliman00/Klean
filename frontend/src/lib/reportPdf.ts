/* Report → PDF. Loaded only when Download is pressed, so html2pdf.js stays out of the main bundle.
   The report is laid out and paginated in a hidden frame that cannot run script (see lib/report),
   drawn to a canvas there, and html2pdf.js cuts the canvas into A4 pages and saves the file. */
import html2canvas from 'html2canvas';
import html2pdf from 'html2pdf.js';
import { REPORT_PAGE, REPORT_PDF } from '../config';
import { paginateReport, REPORT_PX, REPORT_SANDBOX, reportDocument } from './report';

/** A hidden frame holding the report, loaded and split into pages. */
function mountReport(html: string): Promise<HTMLIFrameElement> {
  const frame = document.createElement('iframe');
  frame.setAttribute('sandbox', REPORT_SANDBOX);
  frame.setAttribute('aria-hidden', 'true');
  frame.tabIndex = -1;
  // laid out, but off screen (a display:none frame has no layout to draw from)
  frame.style.cssText = `position:fixed;left:-10000px;top:0;width:${REPORT_PX.contentWidth}px;height:${REPORT_PX.contentHeight}px;border:0;opacity:0;pointer-events:none`;
  return new Promise((resolve, reject) => {
    frame.addEventListener('load', () => resolve(frame));
    frame.addEventListener('error', reject);
    frame.srcdoc = reportDocument(html);
    document.body.append(frame);
  });
}

async function render(html: string, filename: string) {
  const frame = await mountReport(html);
  try {
    const doc = frame.contentDocument!;
    const pages = paginateReport(doc);
    // shortcut: one canvas for the whole report, so beyond ~15 pages the scale drops to fit the browser's
    // canvas limit and the text softens. Render page by page if long reports become common.
    const scale = Math.min(REPORT_PDF.scale, REPORT_PDF.maxCanvasPx / (pages * REPORT_PX.contentHeight));
    const canvas = await html2canvas(doc.documentElement, {
      scale,
      width: REPORT_PX.contentWidth,
      windowWidth: REPORT_PX.contentWidth,
      backgroundColor: '#ffffff',
      logging: false,
    });
    await html2pdf()
      .set({
        margin: REPORT_PAGE.marginMm,
        filename,
        image: REPORT_PDF.image,
        jsPDF: { unit: 'mm', format: [REPORT_PAGE.widthMm, REPORT_PAGE.heightMm], orientation: 'portrait' },
      })
      .from(canvas)
      .save();
  } finally {
    frame.remove();
  }
}

/** Renders the report's HTML to a PDF and downloads it. Rejects if it fails or takes too long. */
export function saveReportPdf(html: string, filename: string): Promise<void> {
  let timer: ReturnType<typeof setTimeout>;
  const timeout = new Promise<never>((_, reject) => {
    timer = setTimeout(() => reject(new Error('Rendering the PDF took too long.')), REPORT_PDF.timeoutMs);
  });
  return Promise.race([render(html, filename), timeout]).finally(() => clearTimeout(timer));
}
