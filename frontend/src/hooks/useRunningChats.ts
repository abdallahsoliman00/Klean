import { useEffect } from 'react';
import { RUNNING_POLL_MS } from '../config';
import { getStore, useStore } from '../store';

/** While some chat is writing a reply, refreshes the chat list now and then, so the ones this page
    isn't following show as done (and move up, with their titles) when they are. */
export function useRunningChats() {
  const anyRunning = useStore((s) => s.chats.some((c) => c.running));
  useEffect(() => {
    if (!anyRunning) return;
    // a failed refresh is tried again at the next tick
    const t = setInterval(() => void getStore().refreshChats().catch(() => {}), RUNNING_POLL_MS);
    return () => clearInterval(t);
  }, [anyRunning]);
}
