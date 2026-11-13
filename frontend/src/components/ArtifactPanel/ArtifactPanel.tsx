import { useStore } from '../../store';
import { shownArtifact } from '../../store/selectors';
import { CloseButton } from '../common/CloseButton';
import { ArtifactActions } from './ArtifactActions';
import { ArtifactSwitcher } from './ArtifactSwitcher';
import { ArtifactTabs } from './ArtifactTabs';
import { ArtifactView } from './ArtifactView';
import { ArtifactMeta } from './ArtifactMeta';

/** The right column: the chat's artifacts, one at a time, as a live preview or as code. */
export function ArtifactPanel({ ref }: { ref: React.Ref<HTMLElement> }) {
  const a = useStore(shownArtifact);
  const tab = useStore((s) => s.art.tab);
  const closeArtifact = useStore((s) => s.closeArtifact);
  const hasPreview = a?.kind !== 'code';
  const shownTab = hasPreview ? tab : 'code';

  return (
    <aside className="art" ref={ref} aria-label="Artifacts">
      <div className="art-inner">
        <div className="art-head">
          {a && <ArtifactSwitcher current={a} />}
          <span className="spacer" />
          <CloseButton label="Close artifact panel" title="Close (Esc)" onClick={() => closeArtifact(true)} />
        </div>
        <div className="art-bar">
          <ArtifactTabs tab={shownTab} hidden={!hasPreview} />
          {a && <ArtifactMeta artifact={a} />}
          <span className="spacer" />
          {a && <ArtifactActions artifact={a} />}
        </div>
        <ArtifactView artifact={a} tab={shownTab} />
      </div>
    </aside>
  );
}
