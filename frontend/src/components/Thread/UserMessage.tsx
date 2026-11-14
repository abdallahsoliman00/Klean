import { useState } from 'react';
import { EditIcon } from '../common/icons';
import { api } from '../../api';
import { useStore } from '../../store';
import { ImageStrip } from '../Images/ImageStrip';
import { EditBox } from './EditBox';
import { messageClass, type MessageProps } from './MessageView';
import { CopyButton, ToolButton } from './ToolButton';

/** The user's message as a bubble under its images, with Copy and Edit. Editing changes the text only
    (the images stay), resends it and drops everything after. */
export function UserMessage({ message: m, isLast, enter }: MessageProps) {
  const [editing, setEditing] = useState(false);
  const send = useStore((s) => s.send);
  const streaming = useStore((s) => !!s.streaming);

  const images = m.images ?? [];

  return (
    <div className={messageClass('user', isLast, enter)}>
      {images.length > 0 && <ImageStrip images={images.map((image) => ({ src: image.url ?? api.imageUrl(image.id) }))} />}
      {editing ? (
        <EditBox initial={m.content} allowEmpty={images.length > 0} onCancel={() => setEditing(false)} onSave={(content) => send({ content, editMessageId: m.id })} />
      ) : (
        <>
          {m.content && (
            <div className="bubble" dir="auto">
              {m.content}
            </div>
          )}
          <div className="tools">
            {m.content && <CopyButton text={m.content} />}
            <ToolButton icon={<EditIcon />} label="Edit" onClick={() => !streaming && setEditing(true)} />
          </div>
        </>
      )}
    </div>
  );
}
