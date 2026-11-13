import { useEffect, useLayoutEffect, useRef, type ReactNode } from 'react';
import { POPOVER_EDGE_PX, POPOVER_GAP_PX } from '../../config';
import { useStore } from '../../store';

/** A floating panel anchored to the button that opened it (artifact switcher, tools menu).
    Only one is open at a time; the store tracks which. Arrow keys move between its buttons. */
export function Popover({ id, above, children }: { id: string; above?: boolean; children: ReactNode }) {
  const popover = useStore((s) => s.popover);
  const closePopover = useStore((s) => s.closePopover);
  const ref = useRef<HTMLDivElement>(null);
  const anchor = popover?.id === id ? popover.anchor : null;

  // place it beside the anchor, inside the window, then focus the selected (or first) button
  useLayoutEffect(() => {
    const m = ref.current;
    if (!m || !anchor) return;
    const r = anchor.getBoundingClientRect();
    const w = m.offsetWidth;
    const h = m.offsetHeight;
    m.style.left = Math.max(POPOVER_EDGE_PX, Math.min(r.left, innerWidth - w - POPOVER_EDGE_PX)) + 'px';
    m.style.top = Math.max(POPOVER_EDGE_PX, above ? r.top - h - POPOVER_GAP_PX : Math.min(r.bottom + POPOVER_GAP_PX, innerHeight - h - POPOVER_EDGE_PX)) + 'px';
    m.style.transformOrigin = above ? 'bottom left' : 'top left';
    const first = m.querySelector<HTMLElement>('[aria-selected="true"]') || m.querySelector('button');
    first?.focus({ preventScroll: true });
  }, [anchor, above]);

  // a click elsewhere or a resize closes it
  useEffect(() => {
    if (!anchor) return;
    const onDown = (e: PointerEvent) => {
      const t = e.target as Node;
      if (!ref.current?.contains(t) && !anchor.contains(t)) closePopover();
    };
    const onResize = () => closePopover();
    document.addEventListener('pointerdown', onDown);
    window.addEventListener('resize', onResize);
    return () => {
      document.removeEventListener('pointerdown', onDown);
      window.removeEventListener('resize', onResize);
    };
  }, [anchor, closePopover]);

  const onBlur = (e: React.FocusEvent) => {
    const next = e.relatedTarget as Node | null;
    if (anchor && next && !ref.current?.contains(next) && next !== anchor) closePopover();
  };

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') return;
    const bs = [...(ref.current?.querySelectorAll('button') ?? [])];
    if (!bs.length) return;
    e.preventDefault();
    const i = bs.indexOf(document.activeElement as HTMLButtonElement);
    bs[(i + (e.key === 'ArrowDown' ? 1 : bs.length - 1)) % bs.length].focus();
  };

  return (
    <div ref={ref} className={'menu pop' + (anchor ? ' show' : '')} inert={!anchor} onBlur={onBlur} onKeyDown={onKeyDown}>
      {anchor && children}
    </div>
  );
}
