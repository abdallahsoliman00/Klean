import { useEffect, useRef } from 'react';
import { useStore } from '../../store';
import { CloseIcon } from '../common/icons';

/** One image, large. Escape, the close button or a click outside it closes; focus returns where it was. */
export function Lightbox() {
  const shown = useStore((s) => s.lightbox);
  const close = useStore((s) => s.closeLightbox);
  const closeRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (!shown) return;
    const back = document.activeElement as HTMLElement | null;
    closeRef.current?.focus();
    return () => {
      if (back?.isConnected) back.focus();
    };
  }, [shown]);

  if (!shown) return null;
  return (
    <div
      className="lightbox"
      role="dialog"
      aria-modal="true"
      aria-label={shown.label}
      onClick={(e) => e.target === e.currentTarget && close()}
      onKeyDown={(e) => {
        if (e.key !== 'Escape' && e.key !== 'Tab') return;
        e.stopPropagation(); // Escape closes this only, not what is underneath
        if (e.key === 'Escape') close();
        if (e.key === 'Tab') e.preventDefault(); // the close button is the only stop
      }}
    >
      <img src={shown.src} alt={shown.label} />
      <button ref={closeRef} className="icon-btn lightbox-x" aria-label="Close image" title="Close (Esc)" onClick={close}>
        <CloseIcon />
      </button>
    </div>
  );
}
