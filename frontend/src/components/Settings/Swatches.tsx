import { useRef, type ComponentType } from 'react';
import { highlightBlock } from '../../lib/highlight';
import { useStore } from '../../store';
import { CheckIcon } from '../common/icons';

interface Choice {
  id: string;
  name: string;
  note: string;
}

const ARROWS: Record<string, number> = { ArrowRight: 1, ArrowDown: 1, ArrowLeft: -1, ArrowUp: -1 };
const MODES = ['light', 'dark'] as const;
type Mode = (typeof MODES)[number];

/** A radio group of small previews, each drawn light half and dark half side by side. Arrow keys move the choice. */
export function Swatches<T extends Choice>({ items, value, labelledBy, onPick, Half }: {
  items: readonly T[];
  value: T['id'];
  labelledBy: string;
  onPick: (id: T['id']) => void;
  /** One half of an item's preview. */
  Half: ComponentType<{ id: string; mode: Mode }>;
}) {
  const buttons = useRef<Record<string, HTMLButtonElement | null>>({});
  const pick = (id: T['id']) => {
    onPick(id);
    buttons.current[id]?.focus();
  };
  const onKeyDown = (i: number, e: React.KeyboardEvent) => {
    const d = ARROWS[e.key];
    if (!d) return;
    e.preventDefault();
    pick(items[(i + d + items.length) % items.length].id);
  };

  return (
    <div className="swatches" role="radiogroup" aria-labelledby={labelledBy}>
      {items.map((s, i) => (
        <button
          key={s.id}
          ref={(el) => {
            buttons.current[s.id] = el;
          }}
          className="swatch"
          type="button"
          role="radio"
          aria-checked={value === s.id}
          aria-label={`${s.name}: ${s.note}`}
          title={s.note}
          tabIndex={value === s.id ? 0 : -1}
          onClick={() => pick(s.id)}
          onKeyDown={(e) => onKeyDown(i, e)}
        >
          <span className="sw-prev" aria-hidden="true">
            {MODES.map((mode) => (
              <Half key={mode} id={s.id} mode={mode} />
            ))}
          </span>
          <span className="sw-name">
            {s.name}
            <CheckIcon />
          </span>
        </button>
      ))}
    </div>
  );
}

/** A colour scheme as a tiny picture of the app. The scheme's tokens apply inside this element through its data attributes. */
export function SchemeHalf({ id, mode }: { id: string; mode: Mode }) {
  return (
    <span className="sw-half" data-scheme={id} data-theme={mode}>
      <i />
      <span className="sw-main">
        <i className="u" />
        <i className="l1" />
        <i className="l2" />
        <span className="c">
          <i />
        </span>
      </span>
    </span>
  );
}

// highlight.js escapes its input, so its output is safe as HTML
const CODE_SAMPLE = highlightBlock('// tides\nlet n = 42;\nfunction go() {\n  return "hi";\n}', 'javascript');

/** Code colours on a few real lines of code, over the current colour scheme's code background. */
export function CodeHalf({ id, mode }: { id: string; mode: Mode }) {
  const scheme = useStore((s) => s.scheme);
  return <code className="sw-half sw-code" data-scheme={scheme} data-theme={mode} data-code-scheme={id} dangerouslySetInnerHTML={{ __html: CODE_SAMPLE }} />;
}
