import { artifactSwitch } from '../../lib/focus';
import { getStore, useStore } from '../../store';
import type { Artifact } from '../../types';
import { Glyph } from '../common/Glyph';
import { CaretDownIcon, CheckIcon } from '../common/icons';
import { Popover } from '../common/Popover';
import { ArtifactMeta } from './ArtifactMeta';

const POPOVER_ID = 'artifacts';

/** The panel title; opens a list of the chat's artifacts to switch between. */
export function ArtifactSwitcher({ current }: { current: Artifact }) {
  const all = useStore((s) => s.chat?.artifacts);
  const expanded = useStore((s) => s.popover?.id === POPOVER_ID);
  const { togglePopover, closePopover, showArtifact } = getStore();
  const list = all ?? [];

  return (
    <>
      <button
        ref={artifactSwitch}
        className="art-switch"
        aria-haspopup="true"
        aria-expanded={expanded}
        title="Switch artifact"
        onClick={(e) => togglePopover(POPOVER_ID, e.currentTarget)}
      >
        <Glyph kind={current.kind} />
        <span className="t">{current.title || 'Untitled'}</span>
        <span className="count">{list.length > 1 ? `${list.indexOf(current) + 1} of ${list.length}` : ''}</span>
        <CaretDownIcon />
      </button>
      <Popover id={POPOVER_ID}>
        <div role="listbox" aria-label="Artifacts in this chat">
          <div className="pop-label">In this chat</div>
          {list.map((a) => (
            <button
              key={a.id}
              type="button"
              role="option"
              aria-selected={a.id === current.id}
              onClick={() => {
                closePopover(true);
                showArtifact(a.id);
              }}
            >
              <Glyph kind={a.kind} />
              <span className="a-text">
                <span className="a-title">{a.title || 'Untitled'}</span>
                <ArtifactMeta artifact={a} />
              </span>
              <span className="check">
                <CheckIcon />
              </span>
            </button>
          ))}
        </div>
      </Popover>
    </>
  );
}
