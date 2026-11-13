import { useLayoutEffect, useMemo, useRef } from 'react';
import { renderMarkdown } from '../../lib/markdown';

/** Markdown rendered to sanitised HTML. While streaming, `caret` puts a blinking caret at the end of the text. */
export function Prose({ text, caret, className = '' }: { text: string; caret?: boolean; className?: string }) {
  const ref = useRef<HTMLDivElement>(null);
  const html = useMemo(() => renderMarkdown(text), [text]);
  useLayoutEffect(() => {
    const n = ref.current;
    if (!n) return;
    n.innerHTML = html;
    if (caret) caretTarget(n).append(Object.assign(document.createElement('span'), { className: 'caret' }));
  }, [html, caret]);
  return <div ref={ref} className={('prose ' + className).trim()} dir="auto" />;
}

/** Where the caret goes so it sits right after the last character: inside the last paragraph,
    list item, heading or code block. */
function caretTarget(n: HTMLElement): Element {
  const last = n.lastElementChild;
  if (!last) return n;
  if (last.classList.contains('code')) return last.querySelector('code') ?? n;
  if (/^(P|LI|H\d)$/.test(last.tagName)) return last;
  if (last.matches('ul,ol')) return last.lastElementChild || last;
  return n;
}
