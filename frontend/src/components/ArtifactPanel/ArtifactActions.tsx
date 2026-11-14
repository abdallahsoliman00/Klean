import { useCopyFeedback } from '../../hooks/useCopyFeedback';
import { artifactFilename, isReport } from '../../lib/artifacts';
import { downloadText } from '../../lib/download';
import { getStore, useStore } from '../../store';
import type { Artifact } from '../../types';
import { CheckIcon, CopyIcon, DownloadIcon, SpinnerIcon, TrashIcon } from '../common/icons';

/** Copy, download and delete for the artifact on show. A report downloads as a PDF, which takes a
    moment to make, and has no Copy: its HTML is not something to read. */
export function ArtifactActions({ artifact: a }: { artifact: Artifact }) {
  const [copied, copy] = useCopyFeedback();
  const { toast, confirm, deleteArtifact, downloadReport } = getStore();
  const makingPdf = useStore((s) => s.pdfFor !== null);
  const report = isReport(a);

  const download = () => {
    const name = artifactFilename(a);
    downloadText(a.content || '', name);
    toast(`Downloaded ${name}`);
  };
  const remove = () =>
    confirm({
      title: `Delete “${a.title || 'Untitled'}”?`,
      text: 'It is removed from this chat for good. The messages that mention it stay.',
      okLabel: 'Delete',
      ok: () => deleteArtifact(a),
    });

  return (
    <>
      {report ? (
        <button className="btn small" aria-busy={makingPdf} disabled={makingPdf} onClick={() => downloadReport(a)}>
          {makingPdf ? <SpinnerIcon /> : <DownloadIcon size={14} />}
          {makingPdf ? 'Making the PDF…' : 'Download PDF'}
        </button>
      ) : (
        <>
          <button className={'icon-btn' + (copied ? ' done' : '')} aria-label="Copy contents" title="Copy contents" onClick={() => copy(a.content || '')}>
            {copied ? <CheckIcon size={16} /> : <CopyIcon size={16} />}
          </button>
          <button className="icon-btn" aria-label="Download" title="Download" onClick={download}>
            <DownloadIcon />
          </button>
        </>
      )}
      <button className="icon-btn danger" aria-label="Delete artifact" title="Delete artifact" onClick={remove}>
        <TrashIcon size={16} />
      </button>
    </>
  );
}
