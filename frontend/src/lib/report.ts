/* Reports: the model writes a whole HTML document, and the user gets pages.
   That HTML is untrusted. It is only ever placed in a frame that (1) is sandboxed without allow-scripts,
   so nothing in it can run, (2) carries a Content-Security-Policy that loads nothing from the network,
   and (3) was sanitised first: no script, no event handlers, no forms, no redirects, no links out.
   The preview and the PDF are both made from that one frame document, split into pages the same way. */
import DOMPurify from 'dompurify';
import { CSS_PX_PER_MM, REPORT_CSP, REPORT_PAGE, REPORT_PDF } from '../config';

const px = (mm: number) => mm * CSS_PX_PER_MM;
const innerMm = { width: REPORT_PAGE.widthMm - 2 * REPORT_PAGE.marginMm, height: REPORT_PAGE.heightMm - 2 * REPORT_PAGE.marginMm };
const contentWidth = Math.round(px(innerMm.width));

/** The page in CSS pixels. The content area (the page inside its margins) is what the report is laid out in;
    its height is rounded the way html2pdf.js slices the rendered canvas, so both cut at the same line. */
export const REPORT_PX = {
  width: px(REPORT_PAGE.widthMm),
  height: px(REPORT_PAGE.heightMm),
  margin: px(REPORT_PAGE.marginMm),
  contentWidth,
  contentHeight: Math.floor((contentWidth * REPORT_PDF.scale * innerMm.height) / innerMm.width) / REPORT_PDF.scale,
} as const;

/** The only way a frame document may be loaded: without scripts, forms, popups or navigating the app. */
export const REPORT_SANDBOX = 'allow-same-origin';

const purify = DOMPurify(window); // its own instance: these hooks must not touch the chat's markdown
const URL_ATTRS = new Set(['href', 'xlink:href', 'src', 'srcset', 'poster', 'background', 'action', 'formaction', 'data', 'ping']);
const LOCAL_URL = /^\s*(#|data:image\/)/i;
purify.addHook('uponSanitizeAttribute', (_node, attr) => {
  if (URL_ATTRS.has(attr.attrName) && !LOCAL_URL.test(attr.attrValue)) attr.keepAttr = false;
});

/** The report as a frame document that cannot run script or reach the network. */
export function reportDocument(html: string): string {
  const safe = purify.sanitize(html, {
    WHOLE_DOCUMENT: true,
    ADD_TAGS: ['style'],
    FORBID_TAGS: ['meta', 'base', 'link', 'form', 'input', 'button', 'select', 'textarea', 'audio', 'video', 'source', 'track'],
  });
  // the policy comes first, so it governs everything after it
  return `<!doctype html><meta http-equiv="Content-Security-Policy" content="${REPORT_CSP}">${safe}`;
}

const HTML_NS = 'http://www.w3.org/1999/xhtml';
const BREAKS = ['always', 'page', 'left', 'right'];
const AVOIDS = ['avoid', 'avoid-page'];

/** Empty space that pushes what follows down by `height`. Inside a table it has to be a row. */
function spacer(doc: Document, before: Element, height: number): HTMLElement {
  const style = `height:${height}px;padding:0;border:0;margin:0`;
  if (before.localName !== 'tr') return Object.assign(doc.createElement('div'), { style: `display:block;${style}` });
  const row = doc.createElement('tr');
  row.innerHTML = `<td colspan="1000" style="${style}"></td>`;
  return row;
}

/** Splits a loaded report into pages, in place, and returns how many there are. The rules are those of
    html2pdf.js's page-break modes in REPORT_PDF.pagebreakModes ("css": break-before, break-after and
    break-inside; "legacy": a break after .html2pdf__page-break), plus REPORT_PDF.avoid: where a page
    would end in the middle of such an element, space is added above it so it starts the next page. */
export function paginateReport(doc: Document): number {
  const view = doc.defaultView!;
  const page = REPORT_PX.contentHeight;
  const modes: readonly string[] = REPORT_PDF.pagebreakModes;
  const css = modes.includes('css');
  const legacy = modes.includes('legacy');
  const toNextPage = (y: number) => (y % page < 1 ? 0 : page - (y % page)); // nothing to add at the top of a page

  for (const el of doc.body.querySelectorAll('*')) {
    const parent = el.parentElement;
    if (!parent || parent.namespaceURI !== HTML_NS) continue; // not inside an SVG drawing
    const style = view.getComputedStyle(el);
    let before = css && BREAKS.includes(style.breakBefore);
    const after = (css && BREAKS.includes(style.breakAfter)) || (legacy && el.classList.contains('html2pdf__page-break'));
    const avoid = (css && AVOIDS.includes(style.breakInside)) || el.matches(REPORT_PDF.avoid);

    const rect = el.getBoundingClientRect();
    const top = rect.top + view.scrollY;
    const bottom = rect.bottom + view.scrollY;
    // split across two pages, yet short enough to fit on one
    if (avoid && !before && rect.height <= page && Math.floor(top / page) !== Math.floor((bottom - 1) / page)) before = true;
    if (before && toNextPage(top)) parent.insertBefore(spacer(doc, el, toNextPage(top)), el);
    else if (after && toNextPage(bottom)) parent.insertBefore(spacer(doc, el, toNextPage(bottom)), el.nextSibling);
  }
  return Math.max(1, Math.ceil((doc.documentElement.scrollHeight - 1) / page));
}
