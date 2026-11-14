import { REPORT_KIND } from '../../config';
import { kindLabel } from '../../lib/artifacts';
import { useStore } from '../../store';
import { artifactPanelOpen, shownArtifact } from '../../store/selectors';
import type { ToolPart } from '../../types';
import { Glyph } from '../common/Glyph';

/** The inline handle for an artifact (or a report) that lives in the panel. Crossed out once it is deleted. */
export function ArtifactCard({ part: p, running }: { part: ToolPart; running: boolean }) {
  const a = p.args || {};
  const r = p.result || {};
  const update = p.name === 'update_artifact';
  const report = p.name === 'create_report';
  const kind = r.kind || (report ? REPORT_KIND : undefined);
  const id = r.artifactId || (update ? a.id : undefined);
  const live = useStore((s) => (id ? s.chat?.artifacts.find((x) => x.id === id) : undefined));
  const current = useStore((s) => !!id && artifactPanelOpen(s) && shownArtifact(s)?.id === id);
  const openArtifact = useStore((s) => s.openArtifact);

  if (running) {
    return (
      <div className="acard" aria-busy="true">
        <Glyph busy />
        <span className="a-text">
          <span className="a-title">{a.title || live?.title || (update ? 'Artifact' : report ? 'New report' : 'New artifact')}</span>
          <span className="a-meta">{update ? 'Rewriting…' : 'Writing…'}</span>
        </span>
      </div>
    );
  }

  const gone = !live;
  const title = (live && !update ? live.title : r.title) || 'Untitled';
  return (
    <button
      className={'acard' + (gone ? ' gone' : '') + (current ? ' current' : '')}
      type="button"
      disabled={gone}
      aria-label={gone ? `${title}, deleted` : `Open ${title}`}
      onClick={(e) => openArtifact(id, e.currentTarget)}
    >
      <Glyph kind={kind} />
      <span className="a-text">
        <span className="a-title">{title}</span>
        <span className="a-meta">
          {gone ? 'Deleted' : (update ? 'Updated ' : '') + kindLabel({ ...r, kind })}
          {!gone && !!r.version && <span className="ver">{'v' + r.version}</span>}
        </span>
      </span>
      {!gone && <span className="a-go">Open</span>}
    </button>
  );
}
