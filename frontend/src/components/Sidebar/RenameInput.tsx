import { useEffect, useRef } from 'react';

/** Inline title editor. Enter or leaving the field saves; Escape cancels (onDone gets null). */
export function RenameInput({ title, onDone }: { title: string; onDone: (title: string | null) => void }) {
  const ref = useRef<HTMLInputElement>(null);
  const done = useRef(false);
  const finish = (save: boolean) => {
    if (done.current) return;
    done.current = true;
    onDone(save ? (ref.current?.value.trim() ?? null) : null);
  };
  useEffect(() => {
    ref.current?.focus();
    ref.current?.select();
  }, []);
  return (
    <input
      ref={ref}
      defaultValue={title}
      aria-label="Chat title"
      onKeyDown={(e) => {
        if (e.key === 'Enter') finish(true);
        if (e.key === 'Escape') finish(false);
      }}
      onBlur={() => finish(true)}
    />
  );
}
