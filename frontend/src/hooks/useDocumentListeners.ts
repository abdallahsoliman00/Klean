import { useEffect } from 'react';
import { COPY_FEEDBACK_MS } from '../config';
import { COPY_BLOCKED, copyText } from '../lib/clipboard';
import { getStore } from '../store';

/** Page-wide handlers for HTML the markdown renderer produced (no React components there),
    and a toast for any promise that failed without a handler. */
export function useDocumentListeners() {
  useEffect(() => {
    // the Copy button in each rendered code block
    const onCopyCode = async (e: MouseEvent) => {
      const b = (e.target as Element).closest<HTMLButtonElement>('[data-copy-code]');
      const code = b?.closest('.code')?.querySelector('code');
      if (!b || !code) return;
      if (!(await copyText(code.innerText))) return getStore().toast(COPY_BLOCKED);
      b.textContent = 'Copied';
      setTimeout(() => (b.textContent = 'Copy'), COPY_FEEDBACK_MS);
    };
    // links in model output always open in a new tab, without access to this page
    const onLink = (e: MouseEvent) => {
      const a = (e.target as Element).closest<HTMLAnchorElement>('.prose a[href]');
      if (!a) return;
      a.target = '_blank';
      a.rel = 'noopener noreferrer';
    };
    const onRejection = (e: PromiseRejectionEvent) => getStore().toast(e.reason?.message || 'Something went wrong.');
    document.addEventListener('click', onCopyCode);
    document.addEventListener('click', onLink, true);
    window.addEventListener('unhandledrejection', onRejection);
    return () => {
      document.removeEventListener('click', onCopyCode);
      document.removeEventListener('click', onLink, true);
      window.removeEventListener('unhandledrejection', onRejection);
    };
  }, []);
}
