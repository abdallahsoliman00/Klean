import { kindLabel } from '../../lib/artifacts';
import type { Artifact } from '../../types';

/** "Python v2" */
export function ArtifactMeta({ artifact: a }: { artifact: Artifact }) {
  return (
    <span className="a-meta">
      {kindLabel(a)}
      <span className="ver">{'v' + (a.version || 1)}</span>
    </span>
  );
}
