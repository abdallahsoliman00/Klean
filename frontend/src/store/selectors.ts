/* Values derived from the store, shared by several components. */
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
