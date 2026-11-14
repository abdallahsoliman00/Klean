/* Choices remembered in the browser: mode, colour scheme, which tools the model may use, and the
   artifact panel's width. */
import { ART_WIDTH, STORAGE_KEYS, TOOL_GROUPS, type SchemeId, type ThemePref, type ToolGroup } from '../config';
import { maxArtifactWidth } from '../lib/media';
import { readJson, writeJson, writeString } from '../lib/storage';
import { storedScheme, storedThemePref } from '../lib/theme';
import type { Slice } from './types';

export type ToolSwitches = Record<ToolGroup, boolean>;

export interface PreferencesSlice {
  themePref: ThemePref;
  scheme: SchemeId;
  tools: ToolSwitches;
  /** The artifact panel's width in px once the user has resized it; null means the built-in width. */
  artWidth: number | null;
  setThemePref(v: ThemePref): void;
  setScheme(v: SchemeId): void;
  toggleTool(key: ToolGroup): void;
  /** Sets the artifact panel's width, kept within its limits and this window. */
  setArtWidth(px: number): void;
  enabledTools(): ToolGroup[];
}

/** All tools are on unless the browser remembers otherwise. */
function storedTools(): ToolSwitches {
  const saved = readJson<Partial<ToolSwitches>>(STORAGE_KEYS.tools);
  const tools = { artifacts: true, ask: true, web: true };
  for (const { key } of TOOL_GROUPS) if (typeof saved?.[key] === 'boolean') tools[key] = saved[key];
  return tools;
}

function storedArtWidth(): number | null {
  const saved = readJson<unknown>(STORAGE_KEYS.artWidth);
  return typeof saved === 'number' ? Math.max(ART_WIDTH.min, Math.min(ART_WIDTH.max, saved)) : null;
}

export const createPreferencesSlice: Slice<PreferencesSlice> = (set, get) => ({
  themePref: storedThemePref(),
  scheme: storedScheme(),
  tools: storedTools(),
  artWidth: storedArtWidth(),

  setThemePref(themePref) {
    writeString(STORAGE_KEYS.theme, themePref);
    set({ themePref });
  },
  setScheme(scheme) {
    writeString(STORAGE_KEYS.scheme, scheme);
    set({ scheme });
  },
  toggleTool(key) {
    const tools = { ...get().tools, [key]: !get().tools[key] };
    writeJson(STORAGE_KEYS.tools, tools);
    set({ tools });
  },
  setArtWidth(px) {
    const artWidth = Math.round(Math.max(ART_WIDTH.min, Math.min(maxArtifactWidth(), px)));
    writeJson(STORAGE_KEYS.artWidth, artWidth);
    set({ artWidth });
  },
  enabledTools() {
    return TOOL_GROUPS.map((g) => g.key).filter((k) => get().tools[k]);
  },
});
