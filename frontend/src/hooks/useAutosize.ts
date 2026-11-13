import { useLayoutEffect } from 'react';

/** Grows a textarea with its text, up to `max` pixels (no limit when omitted). */
export function useAutosize(ref: React.RefObject<HTMLTextAreaElement | null>, value: string, max = Infinity) {
  // `value` is a dependency on purpose: the height is measured again whenever the text changes
  // oxlint-disable-next-line react/exhaustive-deps
  useLayoutEffect(() => {
    const ta = ref.current;
    if (!ta) return;
    ta.style.height = 'auto';
    ta.style.height = Math.min(ta.scrollHeight, max) + 'px';
  }, [ref, value, max]);
}
