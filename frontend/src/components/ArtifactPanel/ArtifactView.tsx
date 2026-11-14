import { ART_CLEAR_MS, SVG_PREVIEW_HEAD } from '../../config';
import { useLingering } from '../../hooks/useLingering';
import { isReport } from '../../lib/artifacts';
import { useStore } from '../../store';
import { artifactPanelOpen } from '../../store/selectors';
import type { ArtifactTab } from '../../store/artifacts';
import type { Artifact } from '../../types';
import { Prose } from '../Parts/Prose';
import { CodeView } from './CodeView';
import { ReportView } from './ReportView';
import { SandboxFrame } from './SandboxFrame';

/** The artifact itself. It is emptied once the panel has slid shut, so a closed preview stops running. */
export function ArtifactView({ artifact: a, tab }: { artifact?: Artifact; tab: ArtifactTab }) {
  const open = useStore(artifactPanelOpen);
  const mounted = useLingering(open, ART_CLEAR_MS);
  const show = mounted && a;
  // a new key rebuilds the view (and reloads the frame) only when what it shows has changed
  const key = a ? [a.id, a.version, a.updatedAt, tab, (a.content || '').length].join('|') : '';
  return (
    <div className="art-view" role="tabpanel" aria-label={a ? `${a.title || 'Artifact'}, ${tab}` : undefined}>
      {show && (tab === 'code' ? <CodeView key={key} artifact={a} /> : <Preview key={key} artifact={a} />)}
    </div>
  );
}

function Preview({ artifact: a }: { artifact: Artifact }) {
  const body = a.content || '';
  const title = a.title || 'Preview';
  if (isReport(a)) return <ReportView html={body} title={title} />;
  if (a.kind === 'html') return <SandboxFrame doc={body} title={title} />;
  if (a.kind === 'svg') return <SandboxFrame doc={SVG_PREVIEW_HEAD + body} title={title} />;
  if (a.kind === 'markdown')
    return (
      <div className="art-scroll">
        <Prose text={body} className="art-doc" />
      </div>
    );
  return <CodeView artifact={a} />;
}
