import { useState } from 'react';
import { domainOf, safeUrl } from '../../lib/urls';
import type { ToolPart } from '../../types';
import { ChevronIcon, SearchIcon, SpinnerIcon } from '../common/icons';
import { ExternalLink } from './ExternalLink';
import { ToolRow } from './ToolRow';

/** "Searched the web for …", which expands into the results. */
export function SearchCall({ part: p, running }: { part: ToolPart; running: boolean }) {
  const [open, setOpen] = useState(false);
  const q = p.result?.query || p.args?.query || '';
  const quoted = q ? <q>{q}</q> : null;
  if (running) {
    return (
      <ToolRow icon={<SpinnerIcon />}>
        <span className="t-text">
          {q ? 'Searching the web for ' : 'Searching the web…'}
          {quoted}
          {q ? '…' : ''}
        </span>
      </ToolRow>
    );
  }
  const results = Array.isArray(p.result?.results) ? p.result.results : [];
  const label = <span className="t-text">Searched the web for {quoted}</span>;
  if (!results.length) {
    return (
      <ToolRow icon={<SearchIcon />}>
        {label}
        <span className="t-aside">No results</span>
      </ToolRow>
    );
  }
  return (
    <div className="tsearch">
      <button className="trow" type="button" aria-expanded={open} onClick={() => setOpen(!open)}>
        <SearchIcon />
        {label}
        <span className="t-aside">{`${results.length} result${results.length === 1 ? '' : 's'}`}</span>
        <ChevronIcon />
      </button>
      <ul className="results" hidden={!open}>
        {results.map((x, i) => (
          <li key={i}>
            <ExternalLink url={x.url} className="res">
              <span className="res-top">
                <span className="res-t">{x.title || domainOf(x.url)}</span>
                <span className="res-d">{safeUrl(x.url) ? domainOf(x.url) : 'link not shown'}</span>
              </span>
              {x.snippet && <span className="res-s">{x.snippet}</span>}
            </ExternalLink>
          </li>
        ))}
      </ul>
    </div>
  );
}
