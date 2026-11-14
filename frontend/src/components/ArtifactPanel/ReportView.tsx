import { useMemo, useState, type CSSProperties } from 'react';
import { REPORT_PREVIEW } from '../../config';
import { useFit } from '../../hooks/useFit';
import { paginateReport, REPORT_PX, REPORT_SANDBOX, reportDocument } from '../../lib/report';

/** A report as the document it will print as: A4 sheets on a neutral backdrop, scaled to the panel's
    width. Each sheet is a window onto one page of the same sandboxed, script-free frame document. */
export function ReportView({ html, title }: { html: string; title: string }) {
  const doc = useMemo(() => reportDocument(html), [html]);
  const [pages, setPages] = useState(1);
  const ref = useFit<HTMLDivElement>(REPORT_PX.width, REPORT_PREVIEW.gutterPx, REPORT_PREVIEW.maxFit);
  const size = {
    '--page-w': REPORT_PX.width + 'px',
    '--page-h': REPORT_PX.height + 'px',
    '--page-m': REPORT_PX.margin + 'px',
    '--content-w': REPORT_PX.contentWidth + 'px',
    '--content-h': REPORT_PX.contentHeight + 'px',
  } as CSSProperties;
  return (
    <div className="art-scroll report" ref={ref} style={size} tabIndex={0}>
      {Array.from({ length: pages }, (_, i) => (
        <Sheet key={i} doc={doc} index={i} pages={pages} title={title} onPaged={setPages} />
      ))}
    </div>
  );
}

/** One page. Its frame holds the whole report, shifted up so that this page shows through. */
function Sheet({ doc, index, pages, title, onPaged }: { doc: string; index: number; pages: number; title: string; onPaged: (pages: number) => void }) {
  const [ready, setReady] = useState(false);
  const label = `Page ${index + 1} of ${pages}`;
  return (
    <figure className={'sheet' + (ready ? ' ready' : '')} aria-label={label}>
      <div className="sheet-page">
        <div className="sheet-clip">
          <iframe
            sandbox={REPORT_SANDBOX}
            referrerPolicy="no-referrer"
            tabIndex={-1}
            title={`${title}, ${label.toLowerCase()}`}
            srcDoc={doc}
            style={{ height: pages * REPORT_PX.contentHeight, transform: `scale(var(--fit)) translateY(${-index * REPORT_PX.contentHeight}px)` }}
            onLoad={(e) => {
              const loaded = e.currentTarget.contentDocument;
              if (loaded) onPaged(paginateReport(loaded));
              setReady(true);
            }}
          />
        </div>
      </div>
      <figcaption>{label}</figcaption>
    </figure>
  );
}
