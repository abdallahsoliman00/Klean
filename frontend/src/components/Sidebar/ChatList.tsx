import { Fragment } from 'react';
import { groupOf } from '../../lib/dates';
import { useStore } from '../../store';
import { ChatItem } from './ChatItem';

/** Chats matching the search, newest first, under date headings. */
export function ChatList() {
  const chats = useStore((s) => s.chats);
  const query = useStore((s) => s.query);
  const q = query.toLowerCase();
  const items = chats.filter((c) => !q || c.title.toLowerCase().includes(q));

  if (!items.length) {
    return (
      <nav className="chat-list">
        <div className="empty-list">{q ? `No chats match “${query}”.` : 'Your chats will show up here.'}</div>
      </nav>
    );
  }
  return (
    <nav className="chat-list">
      {items.map((c, i) => {
        const group = groupOf(c.updatedAt);
        const first = i === 0 || groupOf(items[i - 1].updatedAt) !== group;
        return (
          <Fragment key={c.id}>
            {first && <div className="group-label">{group}</div>}
            {/* keyed by title, so a row that gets a new title plays its entrance again */}
            <ChatItem key={c.title} chat={c} />
          </Fragment>
        );
      })}
    </nav>
  );
}
