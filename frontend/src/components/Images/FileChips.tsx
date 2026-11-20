import type { FileInput } from '../../types';
import { CloseIcon, PageIcon } from '../common/icons';

/** Text files as a row of name chips; in the composer each can also be removed. */
export function FileChips({ files, onRemove }: { files: FileInput[]; onRemove?: (i: number) => void }) {
  return (
    <ul className="file-chips">
      {files.map((f, i) => (
        <li key={i} className="file-chip" title={`${f.name} · ${f.text.split('\n').length} lines`}>
          <PageIcon />
          <span className="file-name">{f.name}</span>
          {onRemove && (
            <button type="button" className="file-x" aria-label={`Remove ${f.name}`} title="Remove" onClick={() => onRemove(i)}>
              <CloseIcon />
            </button>
          )}
        </li>
      ))}
    </ul>
  );
}
