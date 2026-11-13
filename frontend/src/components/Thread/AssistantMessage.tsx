import { RedoIcon } from '../common/icons';
import { useStore } from '../../store';
import { Parts } from '../Parts/Parts';
import { messageClass, type MessageProps } from './MessageView';
import { CopyButton, ToolButton } from './ToolButton';

/** A finished reply: who wrote it, its parts, and Copy / Regenerate. */
export function AssistantMessage({ message: m, isLast, enter }: MessageProps) {
  const send = useStore((s) => s.send);
  return (
    <div className={messageClass('assistant', isLast, enter)}>
      <div className="who">{m.model || 'assistant'}</div>
      <Parts message={m} />
      {m.stopped && <div className="stopped">Stopped early</div>}
      <div className="tools">
        <CopyButton text={m.content} />
        {isLast && <ToolButton icon={<RedoIcon />} label="Regenerate" onClick={() => send({ regenerate: true })} />}
      </div>
    </div>
  );
}
