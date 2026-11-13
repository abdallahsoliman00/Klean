/* The chat list, the open chat, and the settings for a chat that doesn't exist yet. */
import { api } from '../api';
import { DEFAULT_CONTEXT_WINDOW, DEFAULT_TEMPERATURE, SAVE_DEBOUNCE_MS, STORAGE_KEYS, THREAD_FADE_MS } from '../config';
import { focusComposer } from '../lib/focus';
import { readJson } from '../lib/storage';
import type { Chat, ChatSettings, ChatSummary } from '../types';
import type { Selection } from './providers';
import type { Slice } from './types';

export const DEFAULT_CHAT_SETTINGS: ChatSettings = {
  systemPrompt: '',
  temperature: DEFAULT_TEMPERATURE,
  contextWindow: DEFAULT_CONTEXT_WINDOW,
};

export interface ChatsSlice {
  booted: boolean;
  /** Set when the backend didn't answer at startup. */
  bootError: string | null;
  chats: ChatSummary[];
  chat: Chat | null;
  query: string;
  /** Settings picked before the first message; the chat is created with them. */
  draft: ChatSettings | null;
  /** The composer's text. */
  composerText: string;
  /** A chat whose row should animate in (it just got its title). */
  freshChatId: string | null;
  /** The thread is fading out before another chat paints. */
  switching: boolean;
  /** Bumped whenever the thread is redrawn from scratch; the thread then scrolls to the bottom. */
  threadVersion: number;

  boot(): Promise<void>;
  openChat(id: string): Promise<void>;
  newChat(): Promise<void>;
  renameChat(id: string, title: string): Promise<void>;
  deleteChat(c: ChatSummary): Promise<void>;
  /** Rewrites the open chat (no-op when there is none). */
  updateChat(fn: (c: Chat) => Chat): void;
  /** Changes the system prompt, temperature or context window of the open chat, or of the draft. */
  patchChatSettings(patch: Partial<ChatSettings>): void;
  setQuery(q: string): void;
  setComposerText(t: string): void;
}

const withArtifacts = (c: Chat | null): Chat | null => (c ? { ...c, artifacts: Array.isArray(c.artifacts) ? c.artifacts : [] } : null);
const summaryOf = ({ messages: _m, artifacts: _a, ...rest }: Chat): ChatSummary => rest;
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
let saveTimer: ReturnType<typeof setTimeout> | undefined;

export const createChatsSlice: Slice<ChatsSlice> = (set, get) => {
  /** Fades the thread out, then swaps in the given chat. */
  async function showChat(chat: Chat | null) {
    set({ switching: true });
    await sleep(THREAD_FADE_MS);
    set((s) => ({ chat, live: null, switching: false, threadVersion: s.threadVersion + 1 }));
  }

  return {
    booted: false,
    bootError: null,
    chats: [],
    chat: null,
    query: '',
    draft: null,
    composerText: '',
    freshChatId: null,
    switching: false,
    threadVersion: 0,

    async boot() {
      try {
        const providers = await api.providers.list();
        const chats = await api.chats.list();
        set({ providers, chats });
      } catch (e) {
        set({ bootError: (e as Error).message });
        return;
      }
      const { providers, loadModels } = get();
      const last = readJson<Selection>(STORAGE_KEYS.lastModel);
      if (last && providers.some((p) => p.id === last.providerId)) set({ sel: last });
      else if (providers[0]) {
        await loadModels(providers[0].id);
        const m = get().models[providers[0].id]?.[0];
        if (m) set({ sel: { providerId: providers[0].id, model: m.id } });
      }
      providers.forEach((p) => loadModels(p.id));
      set((s) => ({ booted: true, threadVersion: s.threadVersion + 1 }));
      focusComposer();
    },

    async openChat(id) {
      get().stop();
      get().closeMobile();
      if (get().chat?.id === id) return;
      const chat = withArtifacts(await api.chats.get(id));
      get().resetArtifacts();
      if (chat) set({ sel: { providerId: chat.providerId, model: chat.model } });
      await showChat(chat);
    },

    async newChat() {
      get().stop();
      get().resetArtifacts();
      get().closeMobile();
      focusComposer();
      await showChat(null);
    },

    async renameChat(id, title) {
      await api.chats.update(id, { title });
      set((s) => ({ chats: s.chats.map((c) => (c.id === id ? { ...c, title } : c)) }));
      if (get().chat?.id === id) get().updateChat((c) => ({ ...c, title }));
    },

    async deleteChat(c) {
      await api.chats.remove(c.id);
      set((s) => ({ chats: s.chats.filter((x) => x.id !== c.id) }));
      if (get().chat?.id === c.id) {
        get().resetArtifacts();
        set((s) => ({ chat: null, live: null, threadVersion: s.threadVersion + 1 }));
      }
      get().toast(`Deleted “${c.title}”`, {
        label: 'Undo',
        run: async () => {
          await api.chats.restore(c.id);
          set({ chats: await api.chats.list() });
        },
      });
    },

    updateChat(fn) {
      set((s) => (s.chat ? { chat: fn(s.chat) } : {}));
    },

    patchChatSettings(patch) {
      const chat = get().chat;
      if (!chat) {
        set((s) => ({ draft: { ...DEFAULT_CHAT_SETTINGS, ...s.draft, ...patch } }));
        return;
      }
      const next = { ...chat, ...patch };
      get().updateChat((c) => ({ ...c, ...patch }));
      // send all three, so a quick second change can't drop the first
      const all = { systemPrompt: next.systemPrompt, temperature: next.temperature, contextWindow: next.contextWindow };
      clearTimeout(saveTimer);
      saveTimer = setTimeout(() => api.chats.update(chat.id, all), SAVE_DEBOUNCE_MS);
    },

    setQuery(query) {
      set({ query });
    },
    setComposerText(composerText) {
      set({ composerText });
    },
  };
};

export { summaryOf, withArtifacts };
