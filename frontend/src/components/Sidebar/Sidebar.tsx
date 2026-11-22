import { useEffect, useState } from 'react';
import logo from '../../assets/logo.svg?raw';
import { api } from '../../api';
import { PanelIcon, PlusIcon } from '../common/icons';
import { useStore } from '../../store';
import { ChatList } from './ChatList';
import { ChatSearch } from './ChatSearch';
import { SidebarFooter } from './SidebarFooter';

/** The left column: brand, new chat, the artifacts library, search, the chat list, and settings. */
export function Sidebar() {
  const newChat = useStore((s) => s.newChat);
  const openLibrary = useStore((s) => s.openLibrary);
  const [version, setVersion] = useState('');
  useEffect(() => {
    api.version().then(setVersion, () => {}); // no label when the backend is down
  }, []);
  return (
    <aside className="sidebar" aria-label="Chats">
      <div className="sidebar-inner">
        <div className="brand">
          <div className="brand-mark" aria-hidden="true" dangerouslySetInnerHTML={{ __html: logo }} />
          <div className="brand-title">
            <span className="brand-name">Klean</span>
            {version && <span className="brand-version">{version === 'dev' ? 'dev' : `v${version}`}</span>}
          </div>
        </div>
        <button className="new-chat" onClick={() => newChat()}>
          <PlusIcon size={16} />
          New chat <kbd>Ctrl ⇧ O</kbd>
        </button>
        <button className="side-item" onClick={openLibrary}>
          <PanelIcon />
          Artifacts
        </button>
        <ChatSearch />
        <ChatList />
        <SidebarFooter />
      </div>
    </aside>
  );
}
