import { PanelIcon } from '../common/icons';
import { useStore } from '../../store';

/** How many artifacts this chat has; opens or closes the panel. Hidden when there are none. */
export function ArtifactsButton() {
  const n = useStore((s) => s.chat?.artifacts.length ?? 0);
  const toggleArtifacts = useStore((s) => s.toggleArtifacts);
  return (
    <button
      className="ctx art-btn"
      title="Artifacts in this chat"
      aria-label="Artifacts in this chat"
      hidden={!n}
      onClick={(e) => toggleArtifacts(e.currentTarget)}
    >
      <PanelIcon />
      <span>{n}</span>
      <span className="n-label">{n === 1 ? 'artifact' : 'artifacts'}</span>
    </button>
  );
}
