/* Mode (system / light / dark), colour scheme and code colours are separate choices, each an attribute on
   <html>; the token blocks in styles/tokens.css do the rest. Fonts are custom properties on <html>.
   index.html applies them all before first paint. */
import {
  CODE_SCHEMES, DARK_QUERY, DEFAULT_CODE_SCHEME, DEFAULT_SCHEME, DEFAULT_THEME, FONT_ROLES, FONT_STACKS, SCHEMES, STORAGE_KEYS, THEME_PREFS,
  type CodeSchemeId, type FontChoices, type FontId, type SchemeId, type ThemePref,
} from '../config';
import { readJson, readString } from './storage';
import logo from '../assets/logo.svg?raw';

export function storedThemePref(): ThemePref {
  const v = readString(STORAGE_KEYS.theme);
  return THEME_PREFS.includes(v as ThemePref) ? (v as ThemePref) : DEFAULT_THEME;
}

export function storedScheme(): SchemeId {
  const v = readString(STORAGE_KEYS.scheme);
  return SCHEMES.some((s) => s.id === v) ? (v as SchemeId) : DEFAULT_SCHEME;
}

export function storedCodeScheme(): CodeSchemeId {
  const v = readString(STORAGE_KEYS.codeScheme);
  return CODE_SCHEMES.some((s) => s.id === v) ? (v as CodeSchemeId) : DEFAULT_CODE_SCHEME;
}

/** Each role's remembered font, or its default (the first on offer) when there is none or it is no longer offered. */
export function storedFonts(): FontChoices {
  const saved = readJson<Partial<Record<string, FontId>>>(STORAGE_KEYS.fonts);
  const pick = (r: (typeof FONT_ROLES)[number]) => r.options.find((f) => f === saved?.[r.id]) ?? r.options[0];
  return Object.fromEntries(FONT_ROLES.map((r) => [r.id, pick(r)])) as FontChoices;
}

/** Points each role's CSS variable at its font, and <html> at the code colours. */
export function applyCodeAndFonts(fonts: FontChoices, codeScheme: CodeSchemeId) {
  const root = document.documentElement;
  root.dataset.codeScheme = codeScheme;
  for (const r of FONT_ROLES) root.style.setProperty(r.cssVar, FONT_STACKS[fonts[r.id]]);
}

export const prefersDark = () => matchMedia(DARK_QUERY).matches;

/** Sets the attributes, then points the tab icon and browser chrome at the scheme's colours. */
export function applyTheme(pref: ThemePref, scheme: SchemeId) {
  const root = document.documentElement;
  const dark = pref === 'dark' || (pref === 'system' && prefersDark());
  root.dataset.theme = dark ? 'dark' : 'light';
  root.dataset.scheme = scheme;
  const cs = getComputedStyle(root);
  const accent = cs.getPropertyValue('--accent').trim();
  const accentInk = cs.getPropertyValue('--accent-ink').trim();
  const bg = cs.getPropertyValue('--bg').trim();
  const hex = /^#[0-9a-f]{3,8}$/i;
  if (![accent, accentInk, bg].every((c) => hex.test(c))) return;
  const icon = document.getElementById('favicon') as HTMLLinkElement | null;
  const meta = document.getElementById('themeColor') as HTMLMetaElement | null;
  if (icon) icon.href = 'data:image/svg+xml,' + encodeURIComponent(logo.replace('#436663', accent).replace('#dbece9', accentInk));
  if (meta) meta.content = bg;
}
