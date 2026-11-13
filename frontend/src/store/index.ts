import { create } from 'zustand';
import { createArtifactsSlice } from './artifacts';
import { createChatsSlice } from './chats';
import { createDialogSlice } from './dialogs';
import { createPreferencesSlice } from './preferences';
import { createProvidersSlice } from './providers';
import { createSendSlice } from './send';
import type { Store } from './types';
import { createUiSlice } from './ui';

export const useStore = create<Store>()((...a) => ({
  ...createUiSlice(...a),
  ...createDialogSlice(...a),
  ...createPreferencesSlice(...a),
  ...createProvidersSlice(...a),
  ...createChatsSlice(...a),
  ...createArtifactsSlice(...a),
  ...createSendSlice(...a),
}));

/** For event handlers and async code that need the current state without subscribing. */
export const getStore = useStore.getState;
export type { Store } from './types';
