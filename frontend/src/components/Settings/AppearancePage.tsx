import { useRef } from 'react';
import { SCHEMES, THEME_PREFS, type SchemeId } from '../../config';
import { getStore, useStore } from '../../store';
import { SchemeSwatch } from './SchemeSwatch';
import { SettingsLayout } from './SettingsLayout';

const ARROWS: Record<string, number> = { ArrowRight: 1, ArrowDown: 1, ArrowLeft: -1, ArrowUp: -1 };
const capitalise = (s: string) => s[0].toUpperCase() + s.slice(1);

/** Mode (system / light / dark) and colour scheme. */
export function AppearancePage() {
  const pref = useStore((s) => s.themePref);
  const scheme = useStore((s) => s.scheme);
  const { setThemePref, setScheme } = getStore();
  const swatches = useRef<Partial<Record<SchemeId, HTMLButtonElement | null>>>({});

  const pickScheme = (id: SchemeId) => {
    setScheme(id);
    swatches.current[id]?.focus();
  };
  const onSwatchKey = (i: number, e: React.KeyboardEvent) => {
    const d = ARROWS[e.key];
    if (!d) return;
    e.preventDefault();
    pickScheme(SCHEMES[(i + d + SCHEMES.length) % SCHEMES.length].id);
  };

  return (
    <SettingsLayout title="Appearance">
      <div className="field">
        <label>Mode</label>
        <p className="sub">System follows your device and switches with it.</p>
        <div className="seg" role="radiogroup" aria-label="Mode">
          {THEME_PREFS.map((v) => (
            <button key={v} className={pref === v ? 'on' : ''} role="radio" aria-checked={pref === v} onClick={() => setThemePref(v)}>
              {capitalise(v)}
            </button>
          ))}
        </div>
      </div>
      <div className="field wide">
        <label id="schemeLabel">Colour scheme</label>
        <p className="sub">Each one has a light and a dark side. The mode above decides which you see.</p>
        <div className="swatches" role="radiogroup" aria-labelledby="schemeLabel">
          {SCHEMES.map((s, i) => (
            <SchemeSwatch
              key={s.id}
              ref={(el) => {
                swatches.current[s.id] = el;
              }}
              scheme={s}
              checked={scheme === s.id}
              onPick={() => pickScheme(s.id)}
              onKeyDown={(e) => onSwatchKey(i, e)}
            />
          ))}
        </div>
      </div>
    </SettingsLayout>
  );
}
