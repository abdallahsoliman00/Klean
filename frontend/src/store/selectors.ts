/* Values derived from the store, shared by several components. */
import { DEFAULT_CHAT_SETTINGS } from '../config';
import type { ChatConfig } from '../types';
import type { Selection } from './providers';
import type { Store } from './types';

type ArtState = Pick<Store, 'art' | 'chat'>;

/** The artifact the panel shows: the chosen one, or the newest when the chosen one is gone.
    Still set while the panel slides shut, so the header doesn't blank mid-animation. */
export function shownArtifact(s: ArtState) {
  const list = s.chat?.artifacts ?? [];
  return list.find((a) => a.id === s.art.id) ?? (s.art.open ? list.at(-1) : undefined);
}

/** The panel is open and has something to show. */
export const artifactPanelOpen = (s: ArtState) => s.art.open && !!shownArtifact(s);

/** The configuration the next message is sent with: the open chat's, or the new-chat draft's. */
export function currentConfig(s: Pick<Store, 'chat' | 'draft' | 'sel'>): ChatConfig {
  const { systemPrompt, temperature, contextWindow } = s.chat ?? s.draft ?? DEFAULT_CHAT_SETTINGS;
  return { ...s.sel, systemPrompt, temperature, contextWindow };
}

/** False only when the provider or its model is known to be gone. A model list that hasn't loaded,
    or couldn't be, gets the benefit of the doubt. */
export function modelExists(s: Pick<Store, 'providers' | 'models' | 'modelErrors'>, { providerId, model }: Selection) {
  if (!providerId || !model || !s.providers.some((p) => p.id === providerId)) return false;
  const list = s.models[providerId];
  return !list || !!s.modelErrors[providerId] || list.some((m) => m.id === model);
}
