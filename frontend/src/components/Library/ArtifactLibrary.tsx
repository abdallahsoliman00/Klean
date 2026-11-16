import { useEffect, useRef, useState } from 'react';
import { api } from '../../api';
import { PALETTE_FOCUS_MS } from '../../config';
import { useLingering } from '../../hooks/useLingering';
import { kindLabel } from '../../lib/artifacts';
import { timeAgo } from '../../lib/dates';
import { useStore } from '../../store';
import type { ArtifactSummary } from '../../types';
import { Glyph } from '../common/Glyph';
import { SearchIcon } from '../common/icons';

const CLOSE_MS = 200; // matches the .palette transition

/** Every artifact in every chat, newest first. Picking one goes to its chat and opens it there. */
export function ArtifactLibrary() {
  const open = useStore((s) => s.overlay === 'library');
  const mounted = useLingering(open, CLOSE_MS);
  return (
    <div className={'palette library' + (open ? ' show' : '')} role="dialog" aria-label="All artifacts">
      {/* mounted per opening, so each one starts with a fresh list and an empty filter */}
      {mounted && <LibraryList />}
    </div>
  );
}

function LibraryList() {
  const openFromLibrary = useStore((s) => s.openFromLibrary);
  const [items, setItems] = useState<ArtifactSummary[] | null>(null);
  const [error, setError] = useState('');
  const [query, setQuery] = useState('');
  const input = useRef<HTMLInputElement>(null);
  const list = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let current = true;
    api.artifacts.list().then(
      (all) => current && setItems(all),
      (e: Error) => current && setError(e.message),
    );
    const t = setTimeout(() => input.current?.focus(), PALETTE_FOCUS_MS);
    return () => {
      current = false;
      clearTimeout(t);
    };
  }, []);

  const q = query.toLowerCase().trim();
  const matching = (items ?? []).filter((a) => !q || [a.title, a.chatTitle, kindLabel(a)].some((t) => t.toLowerCase().includes(q)));

  // the arrow keys walk from the filter down the rows and back; Enter in the filter takes the first row
  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && e.target === input.current && matching[0]) return void openFromLibrary(matching[0]);
    const step = { ArrowDown: 1, ArrowUp: -1 }[e.key];
    if (!step || !input.current) return;
    e.preventDefault();
    const stops: HTMLElement[] = [input.current, ...(list.current?.querySelectorAll('button') ?? [])];
    stops[(stops.indexOf(document.activeElement as HTMLElement) + step + stops.length) % stops.length].focus();
  };

  return (
    <div onKeyDown={onKeyDown}>
      <div className="palette-search">
        <SearchIcon size={16} />
        <input ref={input} placeholder="Filter artifacts" aria-label="Filter artifacts" autoComplete="off" value={query} onChange={(e) => setQuery(e.target.value)} />
        <kbd>Esc</kbd>
      </div>
      <div className="palette-list" ref={list}>
        {error && <div className="p-err">{error}</div>}
        {matching.map((a) => (
          <button key={a.id} type="button" className="p-item" onClick={() => openFromLibrary(a)}>
            <Glyph kind={a.kind} />
            <span className="a-text">
              <span className="a-title">{a.title || 'Untitled'}</span>
              <span className="a-meta">
                {kindLabel(a)}
                <span className="ver">{'v' + a.version}</span>
                <span className="in-chat">in {a.chatTitle}</span>
              </span>
            </span>
            <span className="meta">{timeAgo(a.updatedAt)}</span>
          </button>
        ))}
        {!error && !matching.length && (
          <div className="p-empty">{!items ? 'Loading…' : q ? `No artifacts match “${query.trim()}”.` : 'Artifacts the assistant makes will show up here.'}</div>
        )}
      </div>
    </div>
  );
}
