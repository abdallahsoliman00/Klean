import { CODE_SCHEMES, FONT_NAMES, FONT_ROLES, SCHEMES, THEME_PREFS, type FontId } from '../../config';
import { getStore, useStore } from '../../store';
import { SettingsLayout } from './SettingsLayout';
import { CodeHalf, SchemeHalf, Swatches } from './Swatches';

const capitalise = (s: string) => s[0].toUpperCase() + s.slice(1);

/** A line in each role's font, so a choice shows before the page is left. */
const FONT_SAMPLES: Record<(typeof FONT_ROLES)[number]['id'], string> = {
  response: 'High tide is at 06:42, and the next one follows about twelve hours later.',
  prompt: 'When is the next high tide?',
  code: 'const next = tides[(i + 1) % tides.length];',
};

/** Mode (system / light / dark), colour scheme, fonts and code colours. */
export function AppearancePage() {
  const pref = useStore((s) => s.themePref);
  const scheme = useStore((s) => s.scheme);
  const fonts = useStore((s) => s.fonts);
  const codeScheme = useStore((s) => s.codeScheme);
  const { setThemePref, setScheme, setFont, setCodeScheme } = getStore();

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
        <Swatches items={SCHEMES} value={scheme} labelledBy="schemeLabel" onPick={setScheme} Half={SchemeHalf} />
      </div>
      <div className="field wide">
        <label id="codeSchemeLabel">Code colours</label>
        <p className="sub">Syntax highlighting in code blocks and artifacts, chosen apart from the colour scheme.</p>
        <Swatches items={CODE_SCHEMES} value={codeScheme} labelledBy="codeSchemeLabel" onPick={setCodeScheme} Half={CodeHalf} />
      </div>
      {FONT_ROLES.map((r) => (
        <div className="field" key={r.id}>
          <label htmlFor={'font-' + r.id}>{r.name} font</label>
          <p className="sub">{r.note}</p>
          <select className="input select" id={'font-' + r.id} value={fonts[r.id]} onChange={(e) => setFont(r.id, e.target.value as FontId)}>
            {r.options.map((f, i) => (
              <option key={f} value={f}>
                {FONT_NAMES[f] + (i ? '' : ' (default)')}
              </option>
            ))}
          </select>
          <p className="font-sample" style={{ fontFamily: `var(${r.cssVar})` }}>
            {FONT_SAMPLES[r.id]}
          </p>
        </div>
      ))}
    </SettingsLayout>
  );
}
