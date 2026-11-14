import { useStore } from '../../store';
import { shownArtifact } from '../../store/selectors';
import { CloseButton } from '../common/CloseButton';
import { ArtifactActions } from './ArtifactActions';
import { ArtifactSwitcher } from './ArtifactSwitcher';
import { ArtifactTabs } from './ArtifactTabs';
import { ArtifactView } from './ArtifactView';
import { ArtifactMeta } from './ArtifactMeta';
import { ArtifactResizer } from './ArtifactResizer';
import { FullscreenButton } from './FullscreenButton';

/** The right column: the chat's artifacts, one at a time, as a live preview or as code.
    Its inner edge resizes it, and it can cover the whole app. */
export function ArtifactPanel({ ref }: { ref: React.Ref<HTMLElement> }) {
  const a = useStore(shownArtifact);
  const tab = useStore((s) => s.art.tab);
  const full = useStore((s) => s.art.full);
  const closeArtifact = useStore((s) => s.closeArtifact);
  const hasPreview = a?.kind !== 'code';
  const shownTab = hasPreview ? tab : 'code';

  return (
    <aside className={'art' + (full ? ' full' : '')} ref={ref} aria-label="Artifacts">
      <ArtifactResizer />
      <div className="art-inner">
        <div className="art-head">
          {a && <ArtifactSwitcher current={a} />}
          <span className="spacer" />
          <FullscreenButton />
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
