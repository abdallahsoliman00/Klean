/* LaTeX in markdown: $x$ and \(x\) inline, $$x$$ and \[x\] displayed. KaTeX writes MathML, which
   the browser draws itself, so no KaTeX stylesheet or fonts are shipped. */
import katex from 'katex';
import type { MarkedExtension } from 'marked';

/* A single $ only opens math when text follows it directly, and only closes it after text and not
   before a digit, so "$5 and $10" stays prose. */
const MATH = /^(?:\$\$([\s\S]+?)\$\$|\\\[([\s\S]+?)\\\]|\\\(([\s\S]+?)\\\)|\$(?!\s)((?:\\.|[^\\$\n])+?)(?<!\s)\$(?!\d))/;

export const mathExtension: MarkedExtension = {
  extensions: [
    {
      name: 'math',
      level: 'inline',
      start: (src) => src.match(/\$|\\[([]/)?.index,
      tokenizer(src) {
        const m = MATH.exec(src);
        if (!m) return undefined;
        const display = m[1] ?? m[2];
        return { type: 'math', raw: m[0], text: (display ?? m[3] ?? m[4]).trim(), display: display !== undefined };
      },
      // throwOnError off: a formula KaTeX cannot parse is shown as its source, in the error colour
      renderer: ({ text, display }) => katex.renderToString(text, { displayMode: display, output: 'mathml', throwOnError: false }),
    },
  ],
};
