import { SearchIcon } from '../common/icons';
import { useStore } from '../../store';

/** Filters the chat list by title. */
export function ChatSearch() {
  const query = useStore((s) => s.query);
  const setQuery = useStore((s) => s.setQuery);
  return (
    <label className="search">
      <span className="sr">Search chats</span>
      <SearchIcon />
      <input placeholder="Search chats" autoComplete="off" value={query} onChange={(e) => setQuery(e.target.value)} />
    </label>
  );
}
