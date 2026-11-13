import { useEffect, useRef, useState } from 'react';
import { useAutosize } from '../../hooks/useAutosize';

/** Edits a sent message in place. Enter saves, Shift+Enter adds a line, Escape cancels. */
export function EditBox({ initial, onSave, onCancel }: { initial: string; onSave: (text: string) => void; onCancel: () => void }) {
  const [value, setValue] = useState(initial);
  const ref = useRef<HTMLTextAreaElement>(null);
  useAutosize(ref, value);
  useEffect(() => {
    const ta = ref.current;
    ta?.focus();
    ta?.setSelectionRange(ta.value.length, ta.value.length);
  }, []);
  const save = () => {
    const v = value.trim();
    if (v) onSave(v);
  };
  return (
    <div className="edit-box">
      <textarea
        ref={ref}
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            save();
          }
          if (e.key === 'Escape') onCancel();
        }}
      />
      <div className="edit-actions">
        <button className="btn small" type="button" onClick={onCancel}>
          Cancel
        </button>
        <button className="btn small primary" type="button" onClick={save}>
          Save and resend
        </button>
      </div>
    </div>
  );
}
