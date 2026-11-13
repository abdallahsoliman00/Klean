import { SCHEMES } from '../../config';
import { CheckIcon } from '../common/icons';

type Scheme = (typeof SCHEMES)[number];

/** A colour scheme as a tiny preview of the app, light half and dark half side by side. */
export function SchemeSwatch({ scheme: s, checked, onPick, onKeyDown, ref }: {
  scheme: Scheme;
  checked: boolean;
  onPick: () => void;
  onKeyDown: (e: React.KeyboardEvent) => void;
  ref: React.Ref<HTMLButtonElement>;
}) {
  return (
    <button
      ref={ref}
      className="swatch"
      type="button"
      role="radio"
      aria-checked={checked}
      aria-label={`${s.name}: ${s.note}`}
      title={s.note}
      tabIndex={checked ? 0 : -1}
      onClick={onPick}
      onKeyDown={onKeyDown}
    >
      <span className="sw-prev" aria-hidden="true">
        <Half scheme={s.id} mode="light" />
        <Half scheme={s.id} mode="dark" />
      </span>
      <span className="sw-name">
        {s.name}
        <CheckIcon />
      </span>
    </button>
  );
}

/** The scheme's tokens apply inside this element through its data attributes. */
function Half({ scheme, mode }: { scheme: string; mode: 'light' | 'dark' }) {
  return (
    <span className="sw-half" data-scheme={scheme} data-theme={mode}>
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
