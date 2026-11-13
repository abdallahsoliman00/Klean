import { useStore } from '../../store';

/** Dims the page behind an overlay or the mobile sidebar; a click closes them. */
export function Scrim() {
  const show = useStore((s) => !!s.overlay || s.mobileOpen);
  const hideAll = useStore((s) => s.hideAll);
  const closeMobile = useStore((s) => s.closeMobile);
  return (
    <div
      className={'scrim' + (show ? ' show' : '')}
      onClick={() => {
        hideAll();
        closeMobile();
      }}
    />
  );
}
