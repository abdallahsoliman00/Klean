import { partsOf } from '../../lib/parts';
import type { Message } from '../../types';
import { PartView } from './PartView';

/** A finished message's prose and tool calls, in the order they happened. */
export function Parts({ message }: { message: Message }) {
  return (
    <div className="parts">
      {partsOf(message).map((p, i) => (
        <PartView key={p.type === 'tool' ? p.id : i} part={p} />
      ))}
    </div>
  );
}
