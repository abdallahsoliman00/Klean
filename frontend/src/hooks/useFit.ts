import { useEffect, useRef } from 'react';

/** Keeps the CSS variable --fit on an element at the scale that fits `width` px (plus a gutter each side)
    into it, never above `max`. It is written straight to the element as the element resizes, so
    whatever is sized with it follows a drag or a transition frame by frame, with no re-render. */
export function useFit<T extends HTMLElement>(width: number, gutter: number, max: number) {
  const ref = useRef<T>(null);
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const fit = () => el.style.setProperty('--fit', String(Math.min(max, Math.max(0.1, (el.clientWidth - 2 * gutter) / width))));
    fit();
    const observer = new ResizeObserver(fit);
    observer.observe(el);
    return () => observer.disconnect();
  }, [width, gutter, max]);
  return ref;
}
