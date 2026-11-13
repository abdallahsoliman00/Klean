/* Mode (system / light / dark) and colour scheme are separate choices. Both are attributes on <html>;
   the token blocks in styles/tokens.css do the rest. index.html applies them before first paint. */
import { DARK_QUERY, DEFAULT_SCHEME, DEFAULT_THEME, SCHEMES, STORAGE_KEYS, THEME_PREFS, type SchemeId, type ThemePref } from '../config';
import { readString } from './storage';

export function storedThemePref(): ThemePref {
  const v = readString(STORAGE_KEYS.theme);
  return THEME_PREFS.includes(v as ThemePref) ? (v as ThemePref) : DEFAULT_THEME;
}

export function storedScheme(): SchemeId {
  const v = readString(STORAGE_KEYS.scheme);
  return SCHEMES.some((s) => s.id === v) ? (v as SchemeId) : DEFAULT_SCHEME;
}

export const prefersDark = () => matchMedia(DARK_QUERY).matches;

/** Sets the attributes, then points the tab icon and browser chrome at the scheme's colours. */
export function applyTheme(pref: ThemePref, scheme: SchemeId) {
  const root = document.documentElement;
  const dark = pref === 'dark' || (pref === 'system' && prefersDark());
  root.dataset.theme = dark ? 'dark' : 'light';
  root.dataset.scheme = scheme;
  const cs = getComputedStyle(root);
  const ink = cs.getPropertyValue('--ink').trim();
  const bg = cs.getPropertyValue('--bg').trim();
  const hex = /^#[0-9a-f]{3,8}$/i;
  if (!hex.test(ink) || !hex.test(bg)) return;
  const icon = document.getElementById('favicon') as HTMLLinkElement | null;
  const meta = document.getElementById('themeColor') as HTMLMetaElement | null;
  if (icon) icon.href = 'data:image/svg+xml,' + encodeURIComponent(faviconSvg(ink, bg));
  if (meta) meta.content = bg;
}

/** The brand mark: two offset bars on a rounded square. */
const faviconSvg = (ink: string, bg: string) =>
  `<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 22 22'><rect width='22' height='22' rx='7' fill='${ink}'/><rect x='5' y='7' width='8' height='2' rx='1' fill='${bg}'/><rect x='9' y='13' width='8' height='2' rx='1' fill='${bg}'/></svg>`;
