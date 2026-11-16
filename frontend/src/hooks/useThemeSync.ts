import { useEffect } from 'react';
import { DARK_QUERY } from '../config';
import { applyCodeAndFonts, applyTheme } from '../lib/theme';
import { useStore } from '../store';

/** Keeps <html>'s mode, scheme and code-colour attributes, its font variables (and the tab icon) in step with the store and the OS. */
export function useThemeSync() {
  const pref = useStore((s) => s.themePref);
  const scheme = useStore((s) => s.scheme);
  const fonts = useStore((s) => s.fonts);
  const codeScheme = useStore((s) => s.codeScheme);
  useEffect(() => applyCodeAndFonts(fonts, codeScheme), [fonts, codeScheme]);
  useEffect(() => {
    applyTheme(pref, scheme);
    const mq = matchMedia(DARK_QUERY);
    const onChange = () => applyTheme(pref, scheme);
    mq.addEventListener('change', onChange);
    return () => mq.removeEventListener('change', onChange);
  }, [pref, scheme]);
}
