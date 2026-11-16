import react from '@vitejs/plugin-react';
import { defineConfig, type Plugin } from 'vite';
import { API_PREFIX, DARK_QUERY, DEFAULT_CODE_SCHEME, DEFAULT_SCHEME, DEFAULT_THEME, DEV_BACKEND_URL, FONT_ROLES, FONT_STACKS, STORAGE_KEYS } from './src/config.ts';

/** Fills the %NAME% placeholders in index.html's pre-paint theme script from src/config.ts. */
function themeConstants(): Plugin {
  const values: Record<string, unknown> = {
    DEFAULT_THEME,
    DEFAULT_SCHEME,
    THEME_KEY: STORAGE_KEYS.theme,
    SCHEME_KEY: STORAGE_KEYS.scheme,
    DARK_QUERY,
    DEFAULT_CODE_SCHEME,
    CODE_SCHEME_KEY: STORAGE_KEYS.codeScheme,
    FONTS_KEY: STORAGE_KEYS.fonts,
    FONT_VARS: Object.fromEntries(FONT_ROLES.map((r) => [r.id, r.cssVar])),
    FONT_STACKS,
  };
  return {
    name: 'klean-theme-constants',
    transformIndexHtml: (html) => html.replace(/%(\w+)%/g, (m, k: string) => (k in values ? JSON.stringify(values[k]) : m)),
  };
}

export default defineConfig({
  plugins: [react(), themeConstants()],
  // relative asset URLs, so `--frontend <folder>` and the embedded copy both work
  base: './',
  build: {
    // one bundle is fine for a local app; highlight.js alone is most of it. The other chunk is
    // html2pdf.js, fetched only when a report is downloaded as a PDF.
    chunkSizeWarningLimit: 1000,
  },
  server: {
    proxy: { [API_PREFIX]: DEV_BACKEND_URL },
  },
});
