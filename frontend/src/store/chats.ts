/* The chat list, the open chat, and the settings for a chat that doesn't exist yet. */
import { api } from '../api';
import { DEFAULT_CHAT_SETTINGS, SAVE_DEBOUNCE_MS, THREAD_FADE_MS } from '../config';
import { focusComposer } from '../lib/focus';
import type { Chat, ChatConfig, ChatSettings, ChatSummary } from '../types';
import { currentConfig, modelExists } from './selectors';
import type { Slice } from './types';

export interface ChatsSlice {
  booted: boolean;
  /** Set when the backend didn't answer at startup. */
  bootError: string | null;
  chats: ChatSummary[];
  chat: Chat | null;
  query: string;
  /** Settings picked before the first message; the chat is created with them. */
  draft: ChatSettings | null;
  /** The configuration used last, which a new chat starts with. Null until the user has set one. */
  chatDefaults: ChatConfig | null;
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
  /** A title the backend wrote for a chat; its row in the sidebar animates. */
  setChatTitle(id: string, title: string): void;
  deleteChat(c: ChatSummary): Promise<void>;
  /** Changes a chat in the list, and the open chat when it is the same one. */
  patchChat(id: string, patch: Partial<Pick<ChatSummary, 'title' | 'favorite' | 'running'>>): void;
  /** Fetches the list again: chats writing a reply in the background finish, move up and get their titles. */
  refreshChats(): Promise<void>;
  /** Stars or unstars a chat. Shown at once, and put back if the backend refuses. */
  toggleFavorite(c: ChatSummary): Promise<void>;
  /** Rewrites the open chat (no-op when there is none). */
  updateChat(fn: (c: Chat) => Chat): void;
  /** Changes the system prompt, temperature or context window of the open chat, or of the draft. */
  patchChatSettings(patch: Partial<ChatSettings>): void;
  /** Saves the current configuration as the one new chats start with. */
  rememberConfig(): void;
  setQuery(q: string): void;
  setComposerText(t: string): void;
}

const withArtifacts = (c: Chat | null): Chat | null => (c ? { ...c, artifacts: Array.isArray(c.artifacts) ? c.artifacts : [] } : null);
const summaryOf = ({ messages: _m, artifacts: _a, ...rest }: Chat): ChatSummary => rest;
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
let saveTimer: ReturnType<typeof setTimeout> | undefined;
let defaultsTimer: ReturnType<typeof setTimeout> | undefined;
/** Counts chat switches, so a chat that loads after another was asked for is dropped. */
let openTurn = 0;

export const createChatsSlice: Slice<ChatsSlice> = (set, get) => {
  /** Fades the thread out, then swaps in the given chat. */
  async function showChat(chat: Chat | null) {
    set({ switching: true });
    await sleep(THREAD_FADE_MS);
    set((s) => ({ chat, live: null, switching: false, threadVersion: s.threadVersion + 1 }));
  }

  /** Starts the new-chat draft from the configuration used last. A model that is gone is skipped:
      the settings still apply, and the model picked before stays. */
  function startDraft() {
    const defaults = get().chatDefaults;
    if (!defaults) return;
    const { providerId, model, ...draft } = defaults;
    set({ draft });
    if (modelExists(get(), defaults)) set({ sel: { providerId, model } });
  }

  return {
    booted: false,
    bootError: null,
    chats: [],
    chat: null,
    query: '',
    draft: null,
    chatDefaults: null,
    composerText: '',
    freshChatId: null,
    switching: false,
    threadVersion: 0,

    async boot() {
      try {
        const providers = await api.providers.list();
        const chats = await api.chats.list();
        const presets = await api.presets.list();
        const { chatDefaults } = await api.settings.get();
        set({ providers, chats, presets, chatDefaults: chatDefaults ?? null });
      } catch (e) {
        set({ bootError: (e as Error).message });
        return;
      }
      const { providers, loadModels } = get();
      startDraft();
      if (!get().sel.model && providers[0]) {
        await loadModels(providers[0].id);
        const m = get().models[providers[0].id]?.[0];
        if (m) set({ sel: { providerId: providers[0].id, model: m.id } });
      }
      providers.forEach((p) => loadModels(p.id));
      set((s) => ({ booted: true, threadVersion: s.threadVersion + 1 }));
      focusComposer();
    },

    async openChat(id) {
      get().closeMobile();
      const turn = ++openTurn;
      if (get().chat?.id === id) return;
      get().detach(); // a reply being written carries on at the backend
      const chat = withArtifacts(await api.chats.get(id));
      if (turn !== openTurn) return;
      get().resetArtifacts();
      if (chat) set({ sel: { providerId: chat.providerId, model: chat.model } });
      await showChat(chat);
      if (chat?.running && turn === openTurn) void get().attach(id);
    },

    async newChat() {
      openTurn++;
      get().detach();
      get().resetArtifacts();
      get().closeMobile();
      focusComposer();
      startDraft();
      await showChat(null);
    },

    async renameChat(id, title) {
      await api.chats.update(id, { title });
      get().patchChat(id, { title });
    },

    setChatTitle(id, title) {
      get().patchChat(id, { title });
      set({ freshChatId: id });
    },

    async deleteChat(c) {
      await api.chats.remove(c.id);
      set((s) => ({ chats: s.chats.filter((x) => x.id !== c.id) }));
      if (get().chat?.id === c.id) {
        get().detach();
        get().resetArtifacts();
        set((s) => ({ chat: null, live: null, threadVersion: s.threadVersion + 1 }));
      }
      get().toast(`Deleted “${c.title}”`, {
        label: 'Undo',
        run: async () => {
          await api.chats.restore(c.id);
          await get().refreshChats();
        },
      });
    },

    async toggleFavorite({ id, title, favorite }) {
      get().patchChat(id, { favorite: !favorite });
      try {
        await api.chats.update(id, { favorite: !favorite });
      } catch (e) {
        get().patchChat(id, { favorite });
        get().toast(`Couldn’t ${favorite ? 'remove' : 'add'} “${title}” ${favorite ? 'from' : 'to'} favourites. ${(e as Error).message}`);
      }
    },

    patchChat(id, patch) {
      set((s) => ({ chats: s.chats.map((c) => (c.id === id ? { ...c, ...patch } : c)) }));
      if (get().chat?.id === id) get().updateChat((c) => ({ ...c, ...patch }));
    },

    async refreshChats() {
      set({ chats: await api.chats.list() });
    },

    updateChat(fn) {
      set((s) => (s.chat ? { chat: fn(s.chat) } : {}));
    },

    patchChatSettings(patch) {
      const chat = get().chat;
      if (!chat) set((s) => ({ draft: { ...DEFAULT_CHAT_SETTINGS, ...s.draft, ...patch } }));
      else {
        const next = { ...chat, ...patch };
        get().updateChat((c) => ({ ...c, ...patch }));
        // send all three, so a quick second change can't drop the first
        const all = { systemPrompt: next.systemPrompt, temperature: next.temperature, contextWindow: next.contextWindow };
        clearTimeout(saveTimer);
        saveTimer = setTimeout(() => api.chats.update(chat.id, all), SAVE_DEBOUNCE_MS);
      }
      get().rememberConfig();
    },

    rememberConfig() {
      const chatDefaults = currentConfig(get());
      set({ chatDefaults });
      clearTimeout(defaultsTimer);
      defaultsTimer = setTimeout(() => api.settings.update({ chatDefaults }), SAVE_DEBOUNCE_MS);
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
