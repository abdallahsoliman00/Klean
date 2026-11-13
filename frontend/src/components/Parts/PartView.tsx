import type { Part } from '../../types';
import { Prose } from './Prose';
import { ToolCall } from './ToolCall';

/** One part: prose, or a tool call. Blank prose is skipped unless the reply is still streaming. */
export function PartView({ part, streaming, caret }: { part: Part; streaming?: boolean; caret?: boolean }) {
  if (part.type === 'tool') return <ToolCall part={part} />;
  if (!streaming && !part.text.trim()) return null;
  return <Prose text={part.text} caret={caret} />;
}
