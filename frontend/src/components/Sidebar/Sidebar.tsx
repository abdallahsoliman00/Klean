import { PlusIcon } from '../common/icons';
import { useStore } from '../../store';
import { ChatList } from './ChatList';
import { ChatSearch } from './ChatSearch';
import { SidebarFooter } from './SidebarFooter';

/** The left column: brand, new chat, search, the chat list, and settings. */
export function Sidebar() {
  const newChat = useStore((s) => s.newChat);
  return (
    <aside className="sidebar" aria-label="Chats">
      <div className="sidebar-inner">
        <div className="brand">
          <div className="brand-mark" aria-hidden="true" />
          <span className="brand-name">Klean</span>
        </div>
        <button className="new-chat" onClick={() => newChat()}>
          <PlusIcon size={16} />
          New chat <kbd>Ctrl ⇧ O</kbd>
        </button>
        <ChatSearch />
        <ChatList />
        <SidebarFooter />
      </div>
    </aside>
  );
}
