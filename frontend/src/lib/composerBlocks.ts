/* The composer's message as blocks: runs of text, and fenced code blocks edited in their own monospace box.
   The list always alternates text, code, text, …, starting and ending with text (possibly empty). */

export interface Block {
  code: boolean;
  /** The code block's language, from its opening fence ("" for none). */
  lang: string;
  value: string;
}

const OPENER = /^```([^`\s]*)\s*$/;
const CLOSER = /^```\s*$/;

const text = (value: string): Block => ({ code: false, lang: '', value });

/** Blocks for a message: only closed fences become code blocks, so a fence still being typed stays text. */
export function parseBlocks(message: string): Block[] {
  const lines = message.split('\n');
  const out: Block[] = [];
  let prose: string[] = [];
  for (let i = 0; i < lines.length; i++) {
    const lang = OPENER.exec(lines[i])?.[1];
    const end = lang === undefined ? -1 : lines.findIndex((l, j) => j > i && CLOSER.test(l));
    if (end === -1) {
      prose.push(lines[i]);
      continue;
    }
    out.push(text(prose.join('\n')), { code: true, lang: lang!, value: lines.slice(i + 1, end).join('\n') });
    prose = [];
    i = end;
  }
  out.push(text(prose.join('\n')));
  return out;
}

/** The message the blocks make, code blocks fenced again. Empty text between blocks adds nothing. */
export function serializeBlocks(blocks: Block[]): string {
  return blocks
    .map((b) => (b.code ? '```' + b.lang + '\n' + b.value + '\n```' : b.value))
    .filter((part) => part !== '')
    .join('\n');
}

/** When the caret ends a line that is a fence (```lang, or a bare ```), the value around that line and the
    fence's language. Enter there opens a code block in text, and a bare fence closes one in code. */
export function splitAtFence(value: string, caret: number): { before: string; lang: string; after: string } | null {
  if (value[caret] !== undefined && value[caret] !== '\n') return null;
  const start = value.lastIndexOf('\n', caret - 1) + 1;
  const lang = OPENER.exec(value.slice(start, caret))?.[1];
  if (lang === undefined) return null;
  return { before: value.slice(0, Math.max(0, start - 1)), lang, after: value.slice(caret + 1) };
}

/** Two runs of text as one, a line apart. */
export const joinText = (a: string, b: string) => (a && b ? a + '\n' + b : a + b);
