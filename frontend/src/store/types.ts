import type { StateCreator } from 'zustand';
import type { ArtifactsSlice } from './artifacts';
import type { AttachmentsSlice } from './attachments';
import type { ChatsSlice } from './chats';
import type { DialogSlice } from './dialogs';
import type { ProvidersSlice } from './providers';
import type { SendSlice } from './send';
import type { PreferencesSlice } from './preferences';
import type { PresetsSlice } from './presets';
import type { UiSlice } from './ui';

/** The whole app state: one slice per concern, merged into a single store. */
export type Store = UiSlice & DialogSlice & PreferencesSlice & ProvidersSlice & PresetsSlice & ChatsSlice & AttachmentsSlice & ArtifactsSlice & SendSlice;

/** A slice creator that can read and call the rest of the store. */
export type Slice<T> = StateCreator<Store, [], [], T>;
