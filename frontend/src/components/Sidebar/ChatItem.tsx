import { useState } from 'react';
import { DotsIcon, EditIcon, ExpandIcon, SpinnerIcon, StarIcon, TrashIcon } from '../common/icons';
import { chatUrl } from '../../lib/urls';
import { getStore, useStore } from '../../store';
import type { ChatSummary } from '../../types';
import { RenameInput } from '../common/RenameInput';

/** One row in the chat list: opens the chat; double-click or the menu renames it; the star keeps it under Favourites.
    The row is a link to the chat, so the browser can open it in another tab (middle click, Ctrl or Cmd click, its own menu). */
export function ChatItem({ chat }: { chat: ChatSummary }) {
  const active = useStore((s) => s.chat?.id === chat.id);
  const fresh = useStore((s) => s.freshChatId === chat.id);
  const { openChat, renameChat, deleteChat, toggleFavorite, showMenu } = getStore();
  const starLabel = chat.favorite ? 'Remove from favourites' : 'Add to favourites';
  const [renaming, setRenaming] = useState(false);

  const openMenu = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    showMenu(e.currentTarget, [
      { label: starLabel, icon: <StarIcon filled={chat.favorite} />, run: () => toggleFavorite(chat) },
      { label: 'Open in new tab', icon: <ExpandIcon />, run: () => window.open(chatUrl(chat.id), '_blank', 'noopener') },
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
          <a
            className="open"
            href={chatUrl(chat.id)}
            onClick={(e) => {
              if (e.ctrlKey || e.metaKey || e.shiftKey || e.altKey) return; // the browser's: new tab, new window, download
              e.preventDefault();
              void openChat(chat.id);
            }}
            onDoubleClick={() => setRenaming(true)}
          >
            {chat.title}
          </a>
          {chat.running && (
            <span className="busy" role="img" aria-label="Writing a reply" title="Writing a reply">
              <SpinnerIcon size={12} />
            </span>
          )}
          <button className={'more star' + (chat.favorite ? ' on' : '')} aria-label={starLabel} title={starLabel} aria-pressed={chat.favorite} onClick={() => toggleFavorite(chat)}>
            <StarIcon filled={chat.favorite} />
          </button>
          <button className="more" aria-label="Chat options" onClick={openMenu}>
            <DotsIcon />
          </button>
        </>
      )}
    </div>
  );
}
