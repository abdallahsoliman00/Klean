import { useMemo } from 'react';
import { isReport } from '../../lib/artifacts';
import { useStore } from '../../store';
import { shownArtifact } from '../../store/selectors';
import { CloseButton } from '../common/CloseButton';
import { ArtifactActions } from './ArtifactActions';
import { ArtifactSwitcher } from './ArtifactSwitcher';
import { ArtifactTabs } from './ArtifactTabs';
import { ArtifactView } from './ArtifactView';
import { ArtifactVersions, useVersions } from './ArtifactVersions';
import { ArtifactResizer } from './ArtifactResizer';
import { FullscreenButton } from './FullscreenButton';

/** The right column: the chat's artifacts, one at a time, as a live preview or as code (a report: as pages only).
    A revised artifact's older versions can be looked at, copied and downloaded too.
    Its inner edge resizes it, and it can cover the whole app. */
export function ArtifactPanel({ ref }: { ref: React.Ref<HTMLElement> }) {
  const latest = useStore(shownArtifact);
  const picked = useStore((s) => s.art.version);
  const showVersion = useStore((s) => s.showVersion);
  const versions = useVersions(latest);
  // an older version, when one is picked: its title and content in place of the latest
  const old = versions?.find((v) => v.version === picked && v.version !== latest?.version);
  const a = useMemo(() => (latest && old ? { ...latest, ...old } : latest), [latest, old]);
  const tab = useStore((s) => s.art.tab);
  const full = useStore((s) => s.art.full);
  const closeArtifact = useStore((s) => s.closeArtifact);
  // code has no preview, and a report has no code to show: only what has both gets the tabs
  const hasPreview = a?.kind !== 'code';
  const hasCode = !isReport(a);
  const shownTab = !hasPreview ? 'code' : !hasCode ? 'preview' : tab;

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
          <ArtifactTabs tab={shownTab} hidden={!hasPreview || !hasCode} />
          {a && <ArtifactVersions artifact={a} versions={versions} />}
          <span className="spacer" />
          {a && <ArtifactActions artifact={a} />}
        </div>
        {old && (
          <div className="art-old" role="status">
            <span>
              You’re viewing version {old.version}, not the latest.
            </span>
            <button onClick={() => showVersion(null)}>Back to latest</button>
          </div>
        )}
        <ArtifactView artifact={a} tab={shownTab} />
      </div>
    </aside>
  );
}
