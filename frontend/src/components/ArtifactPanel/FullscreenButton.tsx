import { useStore } from '../../store';
import { ExpandIcon, ShrinkIcon } from '../common/icons';

/** Makes the artifact panel cover the app, and back. Escape also leaves full screen. */
export function FullscreenButton() {
  const full = useStore((s) => s.art.full);
  const toggle = useStore((s) => s.toggleArtifactFull);
  const label = full ? 'Exit full screen' : 'Full screen';
  return (
    <button className="icon-btn full-btn" aria-label={label} aria-pressed={full} title={full ? 'Exit full screen (Esc)' : label} onClick={toggle}>
      {full ? <ShrinkIcon size={16} /> : <ExpandIcon size={16} />}
    </button>
  );
}
