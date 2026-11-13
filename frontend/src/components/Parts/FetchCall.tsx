import { domainOf } from '../../lib/urls';
import type { ToolPart } from '../../types';
import { PageIcon, SpinnerIcon } from '../common/icons';
import { ExternalLink } from './ExternalLink';
import { ToolRow } from './ToolRow';

/** "Reading example.com…", then "Read example.com" with the page title. */
export function FetchCall({ part: p, running }: { part: ToolPart; running: boolean }) {
  const url = p.result?.url || p.args?.url;
  if (running) {
    return (
      <ToolRow icon={<SpinnerIcon />}>
        <span className="t-text">{url ? `Reading ${domainOf(url)}…` : 'Reading a page…'}</span>
      </ToolRow>
    );
  }
  return (
    <ToolRow icon={<PageIcon />}>
      <span className="t-text">
        Read <ExternalLink url={url}>{domainOf(url)}</ExternalLink>
      </span>
      {p.result?.title && <span className="t-aside">{p.result.title}</span>}
    </ToolRow>
  );
}
