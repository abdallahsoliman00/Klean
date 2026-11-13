import { useEffect, useRef, useState } from 'react';
import { COPY_FEEDBACK_MS } from '../config';
import { COPY_BLOCKED, copyText } from '../lib/clipboard';
import { useStore } from '../store';

/** Copies text and reports `copied` for a moment, for buttons that flip to a tick. */
export function useCopyFeedback(): [boolean, (text: string) => void] {
  const [copied, setCopied] = useState(false);
  const toast = useStore((s) => s.toast);
  const timer = useRef<ReturnType<typeof setTimeout>>(undefined);
  useEffect(() => () => clearTimeout(timer.current), []);
  const copy = async (text: string) => {
    if (!(await copyText(text))) return toast(COPY_BLOCKED);
    setCopied(true);
    clearTimeout(timer.current);
    timer.current = setTimeout(() => setCopied(false), COPY_FEEDBACK_MS);
  };
  return [copied, copy];
}
