import { useState } from 'react';
import { EditIcon } from '../common/icons';
import { useStore } from '../../store';
import { EditBox } from './EditBox';
import { messageClass, type MessageProps } from './MessageView';
import { CopyButton, ToolButton } from './ToolButton';

/** The user's message as a bubble, with Copy and Edit. Editing resends it and drops everything after. */
export function UserMessage({ message: m, isLast, enter }: MessageProps) {
  const [editing, setEditing] = useState(false);
  const send = useStore((s) => s.send);
  const streaming = useStore((s) => !!s.streaming);

  return (
    <div className={messageClass('user', isLast, enter)}>
      {editing ? (
        <EditBox initial={m.content} onCancel={() => setEditing(false)} onSave={(content) => send({ content, editMessageId: m.id })} />
      ) : (
        <>
          <div className="bubble" dir="auto">
            {m.content}
          </div>
          <div className="tools">
            <CopyButton text={m.content} />
            <ToolButton icon={<EditIcon />} label="Edit" onClick={() => !streaming && setEditing(true)} />
          </div>
        </>
      )}
    </div>
  );
}
