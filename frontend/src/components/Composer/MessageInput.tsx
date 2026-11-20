import { useLayoutEffect, useRef, useState } from 'react';
import { useAutosize } from '../../hooks/useAutosize';
import { type Block, joinText, parseBlocks, serializeBlocks, splitAtFence } from '../../lib/composerBlocks';
import { composerInput } from '../../lib/focus';
import { highlightBlock } from '../../lib/highlight';

interface Props {
  text: string;
  setText: (text: string) => void;
  placeholder: string;
  onSend: () => void;
}

/** The composer's text as blocks: text, and code blocks in a monospace box with highlighting. Typing ```lang
    and a new line opens a code block (its closing fence comes with it); in it Enter adds lines, and a bare ``` then Enter (or the arrow keys) leaves it. */
export function MessageInput({ text, setText, placeholder, onSend }: Props) {
  // the blocks follow the store's text; they are parsed again only when it changes from elsewhere
  const [state, setState] = useState(() => ({ text, blocks: parseBlocks(text) }));
  let blocks = state.blocks;
  if (state.text !== text) {
    blocks = parseBlocks(text);
    setState({ text, blocks });
  }
  const inputs = useRef<(HTMLTextAreaElement | null)[]>([]);
  const pendingFocus = useRef<[number, number] | null>(null);

  const focus = (i: number, caret: number) => {
    const el = inputs.current[i];
    el?.focus();
    el?.setSelectionRange(caret, caret);
  };
  useLayoutEffect(() => {
    if (!pendingFocus.current) return;
    focus(...pendingFocus.current);
    pendingFocus.current = null;
  });

  const update = (next: Block[], focusAt?: [number, number]) => {
    const t = serializeBlocks(next);
    setState({ text: t, blocks: next });
    setText(t);
    pendingFocus.current = focusAt ?? null;
  };
  const replace = (i: number, ...with_: Block[]) => [...blocks.slice(0, i), ...with_, ...blocks.slice(i + 1)];
  const setValue = (i: number, value: string) => replace(i, { ...blocks[i], value });

  const onKeyDown = (i: number, e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    const b = blocks[i];
    const el = e.currentTarget;
    const caret = el.selectionStart;
    const collapsed = caret === el.selectionEnd;
    const last = blocks.length - 1;
    if (e.key === 'Enter' && !e.shiftKey && !e.nativeEvent.isComposing) {
      if (e.ctrlKey || e.metaKey) {
        e.preventDefault();
        return onSend();
      }
      const fence = collapsed ? splitAtFence(b.value, caret) : null;
      if (!b.code) {
        if (!fence) {
          e.preventDefault();
          onSend();
        }
        return; // after a fence the new line goes in, and onChange opens the block
      }
      if (fence && !fence.lang) {
        // a bare ``` closes the block; what followed it moves to the text below
        e.preventDefault();
        const after = blocks[i + 1];
        return update(replace(i, { ...b, value: fence.before }, { ...after, value: joinText(fence.after, after.value) }).toSpliced(i + 2, 1), [i + 1, 0]);
      }
      return; // a new line in the code
    }
    if (b.code && e.key === 'Backspace' && b.value === '') {
      // removing an empty code block joins the text around it
      e.preventDefault();
      const [before, after] = [blocks[i - 1], blocks[i + 1]];
      return update([...blocks.slice(0, i - 1), { ...before, value: joinText(before.value, after.value) }, ...blocks.slice(i + 2)], [i - 1, before.value.length]);
    }
    if (!collapsed) return;
    if ((e.key === 'ArrowUp' || (!b.code && e.key === 'Backspace')) && caret === 0 && i > 0) {
      e.preventDefault();
      focus(i - 1, blocks[i - 1].value.length);
    } else if (e.key === 'ArrowDown' && caret === b.value.length && i < last) {
      e.preventDefault();
      focus(i + 1, 0);
    }
  };

  const onChange = (i: number, value: string, caret: number) => {
    // a new line straight after an opening fence (```cpp) opens a code block, closing fence and all
    const fence = !blocks[i].code && value[caret - 1] === '\n' ? splitAtFence(value, caret - 1) : null;
    if (fence) {
      const code = { code: true, lang: fence.lang, value: '' };
      return update(replace(i, { ...blocks[i], value: fence.before }, code, { ...blocks[i], value: fence.after }), [i + 1, 0]);
    }
    const pasted = blocks[i].code ? null : parseBlocks(value);
    if (pasted && pasted.length > 1) {
      // text with whole fenced blocks in it (a paste) becomes those blocks
      const end = i + pasted.length - 1;
      return update(replace(i, ...pasted), [end, pasted[pasted.length - 1].value.length]);
    }
    update(setValue(i, value));
  };

  return (
    <div className="input-blocks">
      {blocks.map((b, i) => {
        const field = (
          <AutoTextarea
            key={i}
            elRef={(el) => {
              inputs.current[i] = el;
              if (i === 0) composerInput.current = el;
            }}
            id={i === 0 ? 'input' : undefined}
            className={b.code ? undefined : 'text-input'}
            dir={b.code ? 'ltr' : 'auto'}
            spellCheck={!b.code}
            aria-label={b.code ? `${b.lang || 'Code'} block` : undefined}
            placeholder={i === 0 && blocks.length === 1 ? placeholder : undefined}
            value={b.value}
            onChange={(e) => onChange(i, e.target.value, e.target.selectionStart)}
            onKeyDown={(e) => onKeyDown(i, e)}
          />
        );
        if (!b.code) return field;
        return (
          <div key={i} className="code-input">
            <div className="code-input-lang">{b.lang || 'code'}</div>
            <div className="code-input-area">
              <pre aria-hidden="true" dangerouslySetInnerHTML={{ __html: highlightBlock(b.value, b.lang) + '\n' }} />
              {field}
            </div>
          </div>
        );
      })}
    </div>
  );
}

/** A textarea that grows with its text. */
function AutoTextarea({ elRef, ...props }: React.ComponentProps<'textarea'> & { elRef: (el: HTMLTextAreaElement | null) => void }) {
  const own = useRef<HTMLTextAreaElement>(null);
  useAutosize(own, String(props.value));
  return (
    <textarea
      {...props}
      rows={1}
      ref={(el) => {
        own.current = el;
        elRef(el);
      }}
    />
  );
}
