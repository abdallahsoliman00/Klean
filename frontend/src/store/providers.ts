/* Providers, their model lists, and the model the next message goes to. */
import { api } from '../api';
import { STORAGE_KEYS } from '../config';
import { focusComposer } from '../lib/focus';
import { writeJson } from '../lib/storage';
import type { Model, Provider, ProviderInput } from '../types';
import type { Slice } from './types';

export interface Selection {
  providerId: string | null;
  model: string | null;
}

export interface ProvidersSlice {
  providers: Provider[];
  models: Record<string, Model[]>;
  modelErrors: Record<string, string>;
  modelLoading: Record<string, boolean>;
  sel: Selection;

  /** Fetches a provider's models, unless they are already loaded (or `force`). */
  loadModels(providerId: string, force?: boolean): Promise<void>;
  /** Switches the current chat (or the next new one) to this model. */
  pickModel(provider: Provider, model: Model): void;
  /** Creates (id null) or updates a provider, then reloads the list and its models. */
  saveProvider(id: string | null, data: ProviderInput): Promise<Provider>;
  removeProvider(p: Provider): Promise<void>;
}

const without = <T,>(rec: Record<string, T>, key: string) => {
  const { [key]: _, ...rest } = rec;
  return rest;
};

export const createProvidersSlice: Slice<ProvidersSlice> = (set, get) => ({
  providers: [],
  models: {},
  modelErrors: {},
  modelLoading: {},
  sel: { providerId: null, model: null },

  async loadModels(pid, force) {
    if (get().models[pid] && !force) return;
    set((s) => ({ modelLoading: { ...s.modelLoading, [pid]: true }, modelErrors: without(s.modelErrors, pid) }));
    try {
      const list = await api.providers.models(pid);
      set((s) => ({ models: { ...s.models, [pid]: list } }));
    } catch (e) {
      set((s) => ({ modelErrors: { ...s.modelErrors, [pid]: (e as Error).message }, models: { ...s.models, [pid]: [] } }));
    } finally {
      set((s) => ({ modelLoading: { ...s.modelLoading, [pid]: false } }));
    }
  },

  pickModel(p, m) {
    const sel = { providerId: p.id, model: m.id };
    set({ sel });
    get().hideAll();
    const chat = get().chat;
    if (chat) {
      get().updateChat((c) => ({ ...c, providerId: p.id, model: m.id }));
      api.chats.update(chat.id, { providerId: p.id, model: m.id });
    }
    writeJson(STORAGE_KEYS.lastModel, sel);
    focusComposer();
  },

  async saveProvider(id, data) {
    let p: Provider;
    if (id) {
      p = await api.providers.update(id, data);
      set((s) => ({ models: without(s.models, p.id) }));
      get().toast('Saved changes');
    } else {
      p = await api.providers.create(data);
      get().toast(`Added ${p.name}`);
    }
    set({ providers: await api.providers.list() });
    set({ settingsPage: { kind: 'provider', id: p.id } });
    await get().loadModels(p.id, true);
    const first = get().models[p.id]?.[0];
    if (!get().sel.model && first) set({ sel: { providerId: p.id, model: first.id } });
    return p;
  },

  async removeProvider(p) {
    await api.providers.remove(p.id);
    const providers = await api.providers.list();
    set((s) => ({ providers, models: without(s.models, p.id) }));
    if (get().sel.providerId === p.id) set({ sel: { providerId: null, model: null } });
    set({ settingsPage: { kind: 'provider', id: providers[0]?.id ?? null } });
    get().toast(`Removed ${p.name}`);
  },
});
