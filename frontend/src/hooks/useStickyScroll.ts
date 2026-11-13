import { useCallback, useEffect, useRef, useState } from 'react';
import { STICK_TO_BOTTOM_PX } from '../config';

/** Follows the bottom of a scroller while the user is there; scrolling up stops following and
    offers "Jump to latest". */
export function useStickyScroll(scroller: React.RefObject<HTMLElement | null>) {
  const stick = useRef(true);
  const [awayFromBottom, setAway] = useState(false);

  const toBottom = useCallback(
    (smooth = false) => {
      const el = scroller.current;
      el?.scrollTo({ top: el.scrollHeight, behavior: smooth ? 'smooth' : 'auto' });
    },
    [scroller],
  );

  useEffect(() => {
    const el = scroller.current;
    if (!el) return;
    const onScroll = () => {
      stick.current = el.scrollHeight - el.scrollTop - el.clientHeight < STICK_TO_BOTTOM_PX;
      setAway(!stick.current);
    };
    el.addEventListener('scroll', onScroll, { passive: true });
    return () => el.removeEventListener('scroll', onScroll);
  }, [scroller]);

  /** Scrolls to the bottom only if the user hasn't scrolled away. */
  const follow = useCallback(() => {
    if (stick.current) toBottom();
  }, [toBottom]);

  /** Scrolls to the bottom and starts following again. */
  const jump = useCallback(
    (smooth = false) => {
      stick.current = true;
      setAway(false);
      toBottom(smooth);
    },
    [toBottom],
  );

  return { awayFromBottom, follow, jump };
}
