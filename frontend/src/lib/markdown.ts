/* Markdown → sanitised HTML. Code blocks get a header with the language and a Copy button
   (handled by hooks/useCodeCopyButtons). */
import DOMPurify from 'dompurify';
import { Marked } from 'marked';
import { highlightBlock } from './highlight';

const marked = new Marked({
  gfm: true,
  breaks: false,
  renderer: {
    code({ text, lang }) {
      const l = lang || '';
      return `<div class="code"><div class="code-head"><span>${l || 'code'}</span><button type="button" data-copy-code>Copy</button></div><pre><code>${highlightBlock(text, l)}</code></pre></div>`;
    },
  },
});

export const renderMarkdown = (text: string) =>
  DOMPurify.sanitize(marked.parse(text, { async: false }), { ADD_ATTR: ['data-copy-code'] });
