import { useState } from 'react';
import { DotsIcon, EditIcon, TrashIcon } from '../common/icons';
import { getStore, useStore } from '../../store';
import type { ChatSummary } from '../../types';
import { RenameInput } from '../common/RenameInput';

/** One row in the chat list: opens the chat; double-click or the menu renames it. */
export function ChatItem({ chat }: { chat: ChatSummary }) {
  const active = useStore((s) => s.chat?.id === chat.id);
  const fresh = useStore((s) => s.freshChatId === chat.id);
  const { openChat, renameChat, deleteChat, showMenu } = getStore();
  const [renaming, setRenaming] = useState(false);

  const openMenu = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    showMenu(e.currentTarget, [
      { label: 'Rename', icon: <EditIcon />, run: () => setRenaming(true) },
      { label: 'Delete', icon: <TrashIcon />, danger: true, run: () => deleteChat(chat) },
    ]);
  };

  return (
    <div className={'chat-item' + (active ? ' active' : '') + (fresh ? ' fresh' : '')}>
      {renaming ? (
        <RenameInput
          label="Chat title"
          title={chat.title}
          onDone={(title) => {
            setRenaming(false);
            if (title && title !== chat.title) renameChat(chat.id, title);
          }}
        />
      ) : (
        <>
          <button className="open" onClick={() => openChat(chat.id)} onDoubleClick={() => setRenaming(true)}>
            {chat.title}
          </button>
          <button className="more" aria-label="Chat options" onClick={openMenu}>
            <DotsIcon />
          </button>
        </>
      )}
    </div>
  );
}
