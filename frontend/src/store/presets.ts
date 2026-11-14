/* Presets: saved, named configurations (model, system prompt, temperature, context window). */
import { api } from '../api';
import type { Preset } from '../types';
import { currentConfig, modelExists } from './selectors';
import type { Slice } from './types';

export interface PresetsSlice {
  presets: Preset[];
  /** Saves the current configuration under this name. */
  savePreset(name: string): Promise<void>;
  renamePreset(p: Preset, name: string): Promise<void>;
  /** Overwrites a preset with the current configuration. */
  updatePreset(p: Preset): Promise<void>;
  deletePreset(p: Preset): Promise<void>;
  /** Switches the current chat (or the next new one) to a preset's configuration. */
  applyPreset(p: Preset): void;
}

export const createPresetsSlice: Slice<PresetsSlice> = (set, get) => {
  const replace = (p: Preset) => set((s) => ({ presets: s.presets.map((x) => (x.id === p.id ? p : x)) }));

  return {
    presets: [],

    async savePreset(name) {
      const p = await api.presets.create({ name, ...currentConfig(get()) });
      set((s) => ({ presets: [...s.presets, p] }));
      get().toast(`Saved “${p.name}”`);
    },

    async renamePreset(p, name) {
      replace(await api.presets.update(p.id, { name }));
    },

    async updatePreset(p) {
      replace(await api.presets.update(p.id, currentConfig(get())));
      get().toast(`Updated “${p.name}”`);
    },

    async deletePreset(p) {
      await api.presets.remove(p.id);
      set((s) => ({ presets: s.presets.filter((x) => x.id !== p.id) }));
      get().toast(`Deleted “${p.name}”`);
    },

    applyPreset({ name, providerId, model, ...settings }) {
      const sel = { providerId, model };
      const found = modelExists(get(), sel);
      if (found) get().selectModel(sel);
      get().patchChatSettings({ systemPrompt: settings.systemPrompt, temperature: settings.temperature, contextWindow: settings.contextWindow });
      // a preset saved without a model has nothing to miss
      get().toast(found || !model ? `Using “${name}”` : `Using “${name}”, but its model is no longer available. The model stays as it was.`);
    },
  };
};
