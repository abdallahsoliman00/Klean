/* Choices remembered in the browser: mode, colour scheme, and which tools the model may use. */
import { STORAGE_KEYS, TOOL_GROUPS, type SchemeId, type ThemePref, type ToolGroup } from '../config';
import { readJson, writeJson, writeString } from '../lib/storage';
import { storedScheme, storedThemePref } from '../lib/theme';
import type { Slice } from './types';

export type ToolSwitches = Record<ToolGroup, boolean>;

export interface PreferencesSlice {
  themePref: ThemePref;
  scheme: SchemeId;
  tools: ToolSwitches;
  setThemePref(v: ThemePref): void;
  setScheme(v: SchemeId): void;
  toggleTool(key: ToolGroup): void;
  enabledTools(): ToolGroup[];
}

/** All tools are on unless the browser remembers otherwise. */
function storedTools(): ToolSwitches {
  const saved = readJson<Partial<ToolSwitches>>(STORAGE_KEYS.tools);
  const tools = { artifacts: true, ask: true, web: true };
  for (const { key } of TOOL_GROUPS) if (typeof saved?.[key] === 'boolean') tools[key] = saved[key];
  return tools;
}

export const createPreferencesSlice: Slice<PreferencesSlice> = (set, get) => ({
  themePref: storedThemePref(),
  scheme: storedScheme(),
  tools: storedTools(),

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
  enabledTools() {
    return TOOL_GROUPS.map((g) => g.key).filter((k) => get().tools[k]);
  },
});
