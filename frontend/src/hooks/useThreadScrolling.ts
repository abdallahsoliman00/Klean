import { useEffect, useLayoutEffect } from 'react';
import { useStore } from '../store';
import type { useStickyScroll } from './useStickyScroll';

/** When the thread scrolls by itself: to the bottom when it is redrawn or a reply starts,
    and along with the reply while the user stays at the bottom. */
export function useThreadScrolling({ follow, jump }: ReturnType<typeof useStickyScroll>) {
  const threadVersion = useStore((s) => s.threadVersion);
  const liveKey = useStore((s) => s.live?.key);
  const live = useStore((s) => s.live);
  const messages = useStore((s) => s.chat?.messages);

  // oxlint-disable-next-line react/exhaustive-deps -- threadVersion is the trigger, not an input
  useEffect(() => {
    const frame = requestAnimationFrame(() => jump());
    return () => cancelAnimationFrame(frame);
  }, [threadVersion, jump]);

  useEffect(() => {
    if (liveKey) jump(true);
  }, [liveKey, jump]);

  useLayoutEffect(follow, [live, messages, follow]);
}
