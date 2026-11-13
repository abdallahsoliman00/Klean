import { useMemo } from 'react';
import { MAX_HIGHLIGHT_CHARS } from '../../config';
import { codeLanguage } from '../../lib/artifacts';
import { highlightOrNull } from '../../lib/highlight';
import type { Artifact } from '../../types';

/** The artifact's source, highlighted when it is small enough. */
export function CodeView({ artifact: a }: { artifact: Artifact }) {
  const body = a.content || '';
  // highlight.js escapes its input, so its output is safe as HTML
  const html = useMemo(
    () => (body.length < MAX_HIGHLIGHT_CHARS ? highlightOrNull(body, codeLanguage(a), a.kind === 'code') : null),
    [body, a],
  );
  return (
    <div className="art-scroll code" tabIndex={0}>
      <pre className="art-code">{html != null ? <code dangerouslySetInnerHTML={{ __html: html }} /> : <code>{body}</code>}</pre>
    </div>
  );
}
