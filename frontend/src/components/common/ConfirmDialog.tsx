import { useEffect, useRef } from 'react';
import { useStore } from '../../store';

/** "Are you sure?" for destructive actions. Focus starts on the confirm button and returns where it was. */
export function ConfirmDialog() {
  const req = useStore((s) => s.confirmRequest);
  const close = useStore((s) => s.closeConfirm);
  const okRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (!req) return;
    const back = document.activeElement as HTMLElement | null;
    okRef.current?.focus();
    return () => {
      if (back?.isConnected) back.focus();
    };
  }, [req]);

  if (!req) return null;
  return (
    <>
      <div className="scrim show confirm-scrim" onClick={close} />
      <div
        className="confirm"
        role="alertdialog"
        onKeyDown={(e) => {
          if (e.key !== 'Escape') return;
          e.stopPropagation();
          close();
        }}
      >
        <h3>{req.title}</h3>
        <p>{req.text}</p>
        <div className="row">
          <button className="btn" onClick={close}>
            Cancel
          </button>
          <button
            ref={okRef}
            className="btn primary"
            onClick={() => {
              close();
              req.ok();
            }}
          >
            {req.okLabel}
          </button>
        </div>
      </div>
    </>
  );
}
