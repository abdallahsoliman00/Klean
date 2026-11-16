import { useEffect } from 'react';
import { MAX_VERSION_TABS } from '../../config';
import { kindLabel } from '../../lib/artifacts';
import { timeAgo } from '../../lib/dates';
import { getStore, useStore } from '../../store';
import { artifactPanelOpen } from '../../store/selectors';
import type { Artifact, ArtifactVersion } from '../../types';
import { CaretDownIcon, CheckIcon } from '../common/icons';
import { Popover } from '../common/Popover';
import { ArtifactMeta } from './ArtifactMeta';

const POPOVER_ID = 'versions';

/** The shown artifact's saved versions, fetched when the panel first shows an artifact that has been revised. */
export function useVersions(latest?: Artifact) {
  const open = useStore(artifactPanelOpen);
  const versions = useStore((s) => (latest ? s.versions[latest.id] : undefined));
  // a new version from the stream is a new object, so it is asked for again even if the last answer is still on its way
  const missing = open && latest && latest.version > 1 && !versions ? latest : null;
  useEffect(() => {
    if (missing) void getStore().loadVersions(missing.id);
  }, [missing]);
  return versions;
}

/** "Python" and which version is on show: tabs for a few versions, a menu for many. An artifact with
    one saved version just shows its number. */
export function ArtifactVersions({ artifact: a, versions }: { artifact: Artifact; versions?: ArtifactVersion[] }) {
  const expanded = useStore((s) => s.popover?.id === POPOVER_ID);
  const { showVersion, togglePopover, closePopover } = getStore();
  if (!versions || versions.length < 2) return <ArtifactMeta artifact={a} />;

  const latest = versions.at(-1)!.version;
  const pick = (v: number) => showVersion(v === latest ? null : v);
  return (
    <>
      <span className="a-meta">{kindLabel(a)}</span>
      {versions.length <= MAX_VERSION_TABS ? (
        <div className="seg small versions" role="radiogroup" aria-label="Version">
          {versions.map(({ version: v }) => (
            <button key={v} role="radio" className={v === a.version ? 'on' : ''} aria-checked={v === a.version} title={v === latest ? 'Latest version' : `Version ${v}`} onClick={() => pick(v)}>
              {'v' + v}
            </button>
          ))}
        </div>
      ) : (
        <>
          <button className="btn small versions" aria-haspopup="true" aria-expanded={expanded} title="Choose a version" onClick={(e) => togglePopover(POPOVER_ID, e.currentTarget)}>
            {'v' + a.version}
            <CaretDownIcon />
          </button>
          <Popover id={POPOVER_ID}>
            <div role="listbox" aria-label="Versions">
              <div className="pop-label">Versions</div>
              {versions.toReversed().map(({ version: v, createdAt }) => (
                <button
                  key={v}
                  type="button"
                  role="option"
                  aria-selected={v === a.version}
                  onClick={() => {
                    closePopover(true);
                    pick(v);
                  }}
                >
                  <span className="a-text">
                    <span className="a-title">{`Version ${v}` + (v === latest ? ' (latest)' : '')}</span>
                    <span className="a-meta">{timeAgo(createdAt)}</span>
                  </span>
                  <span className="check">
                    <CheckIcon />
                  </span>
                </button>
              ))}
            </div>
          </Popover>
        </>
      )}
    </>
  );
}
