import { useEffect } from 'react';
import { chatUrl, routedChatId } from '../lib/urls';
import { getStore, useStore } from '../store';

/** Writes the open chat into the address bar. */
function writeRoute(replace?: boolean) {
  const id = getStore().chat?.id ?? null;
  if (id === routedChatId()) return;
  history[replace ? 'replaceState' : 'pushState'](null, '', id ? chatUrl(id) : location.pathname + location.search);
}

/** Opens the chat the address bar names (none: a new chat). One that can't be opened says why, and the address goes back to what is open. */
function followRoute() {
  const s = getStore();
  const id = routedChatId();
  if (id === (s.chat?.id ?? null)) return;
  (id ? s.openChat(id) : s.newChat()).catch((e: Error) => s.toast(e.message)).finally(() => writeRoute(true));
}

/** Keeps the address bar and the open chat in step: each chat has its own URL (#/chat/<id>), so a chat can
    be opened in another tab, reloaded, and reached with the browser's back and forward buttons. Nothing
    about the open chat is kept in browser storage, so tabs don't affect each other. */
export function useChatRoute() {
  const booted = useStore((s) => s.booted);
  useEffect(() => {
    if (!booted) return;
    followRoute();
    const unsubscribe = useStore.subscribe((s, prev) => {
      if (s.chat?.id !== prev.chat?.id) writeRoute();
    });
    addEventListener('hashchange', followRoute);
    return () => {
      unsubscribe();
      removeEventListener('hashchange', followRoute);
    };
  }, [booted]);
}
