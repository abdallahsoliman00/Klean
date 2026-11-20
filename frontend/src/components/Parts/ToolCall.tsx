import { domainOf } from '../../lib/urls';
import type { ToolPart } from '../../types';
import { AlertIcon, CheckIcon, SkillIcon, SpinnerIcon } from '../common/icons';
import { ArtifactCard } from './ArtifactCard';
import { AskCard } from './AskCard';
import { FetchCall } from './FetchCall';
import { SearchCall } from './SearchCall';
import { ToolRow } from './ToolRow';

const FAILURES: Record<string, string> = {
  web_search: 'The web search failed',
  create_artifact: 'Couldn’t create the artifact',
  update_artifact: 'Couldn’t update the artifact',
  edit_artifact: 'Couldn’t edit the artifact',
  create_report: 'Couldn’t write the report',
  use_skill: 'Couldn’t load the skill',
  ask_user: 'Couldn’t ask the question',
};

/** Picks the view for a tool call. A call without a status is still running. */
export function ToolCall({ part: p }: { part: ToolPart }) {
  const running = !p.status;
  if (p.status === 'error') return <ToolError part={p} />;
  switch (p.name) {
    case 'web_search':
      return <SearchCall part={p} running={running} />;
    case 'fetch_page':
      return <FetchCall part={p} running={running} />;
    case 'create_artifact':
    case 'update_artifact':
    case 'edit_artifact':
    case 'create_report':
      return <ArtifactCard part={p} running={running} />;
    case 'use_skill': {
      const name = p.result?.name || p.args?.name;
      return (
        <ToolRow icon={running ? <SpinnerIcon /> : <SkillIcon />}>
          <span className="t-text">
            {running ? 'Loading skill' : 'Used skill'} {name ? <code>{name}</code> : ''}
            {running ? '…' : ''}
          </span>
        </ToolRow>
      );
    }
    case 'ask_user':
      return running ? (
        <ToolRow icon={<SpinnerIcon />}>
          <span className="t-text">Writing a question…</span>
        </ToolRow>
      ) : (
        <AskCard part={p} />
      );
    default:
      return (
        <ToolRow icon={running ? <SpinnerIcon /> : <CheckIcon />}>
          <span className="t-text">{running ? `Running ${p.name}…` : `Ran ${p.name}`}</span>
        </ToolRow>
      );
  }
}

function ToolError({ part: p }: { part: ToolPart }) {
  const url = p.args?.url;
  const what = p.name === 'fetch_page' ? (url ? `Couldn’t read ${domainOf(url)}` : 'Couldn’t read the page') : FAILURES[p.name] || `${p.name} failed`;
  return (
    <ToolRow icon={<AlertIcon />} error>
      <span className="t-text">
        <b>{what + '. '}</b>
        {p.error || ''}
      </span>
    </ToolRow>
  );
}
