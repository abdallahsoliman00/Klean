import { useCopyFeedback } from '../../hooks/useCopyFeedback';
import { artifactFilename } from '../../lib/artifacts';
import { downloadText } from '../../lib/download';
import { getStore } from '../../store';
import type { Artifact } from '../../types';
import { CheckIcon, CopyIcon, DownloadIcon, TrashIcon } from '../common/icons';

/** Copy, download and delete for the artifact on show. */
export function ArtifactActions({ artifact: a }: { artifact: Artifact }) {
  const [copied, copy] = useCopyFeedback();
  const { toast, confirm, deleteArtifact } = getStore();

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
      <button className={'icon-btn' + (copied ? ' done' : '')} aria-label="Copy contents" title="Copy contents" onClick={() => copy(a.content || '')}>
        {copied ? <CheckIcon size={16} /> : <CopyIcon size={16} />}
      </button>
      <button className="icon-btn" aria-label="Download" title="Download" onClick={download}>
        <DownloadIcon />
      </button>
      <button className="icon-btn danger" aria-label="Delete artifact" title="Delete artifact" onClick={remove}>
        <TrashIcon size={16} />
      </button>
    </>
  );
}
