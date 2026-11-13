import { useEffect, useRef } from 'react';
import { useStore } from '../../store';

/** The small context menu (chat options). The next click anywhere closes it. */
export function Menu() {
  const menu = useStore((s) => s.menu);
  const hideMenu = useStore((s) => s.hideMenu);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = ref.current;
    el?.classList.remove('show');
    if (!menu) return;
    // shown a frame later, so it animates in from its new position
    const frame = requestAnimationFrame(() => el?.classList.add('show'));
    // registered after the click that opened it has finished
    const t = setTimeout(() => document.addEventListener('click', hideMenu, { once: true }));
    return () => {
      cancelAnimationFrame(frame);
      clearTimeout(t);
      document.removeEventListener('click', hideMenu);
    };
  }, [menu, hideMenu]);

  return (
    <div ref={ref} className="menu" style={menu ? { left: menu.x, top: menu.y } : undefined}>
      {menu?.items.map((it) => (
        <button
          key={it.label}
          className={it.danger ? 'danger' : ''}
          onClick={() => {
            hideMenu();
            it.run();
          }}
        >
          {it.icon}
          {it.label}
        </button>
      ))}
    </div>
  );
}
