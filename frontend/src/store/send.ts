/* Sending a message and streaming the reply.
   send covers four cases: a new message, an edited one, a regenerate, and an answer to a question the
   model asked. An answer adds no user message; the reply continues in a new assistant message. */
import { api } from '../api';
import { ApiError } from '../api/errors';
import { PENDING_MESSAGE_ID, STATUS_CONFLICT } from '../config';
import { findToolPart, patchToolPart, skipPendingQuestions } from '../lib/parts';
import type { Message, Part, SendCallbacks, ToolPart } from '../types';
import { summaryOf, withArtifacts } from './chats';
import type { Slice } from './types';

/** The reply being streamed, shown below the thread until it becomes a real message. */
export interface LiveReply {
  key: number;
  model: string | null;
  parts: Part[];
  /** Show the "thinking" dots. */
  waiting: boolean;
  /** Text is still arriving (the last text part gets a caret). */
  active: boolean;
  /** The request failed; shown with a Retry button. */
  error: string | null;
}

export interface SendOptions {
  content?: string;
  editMessageId?: string;
  regenerate?: boolean;
  answer?: { callId: string; text: string };
}

export interface SendSlice {
  streaming: AbortController | null;
  live: LiveReply | null;
  /** React key of the message that animates in (the one just sent). */
  enterKey: string | null;
  send(opts: SendOptions): Promise<void>;
  stop(): void;
}

let liveCounter = 0;

export const createSendSlice: Slice<SendSlice> = (set, get) => ({
  streaming: null,
  live: null,
  enterKey: null,

  stop() {
    get().streaming?.abort();
  },

  async send({ content = '', editMessageId, regenerate, answer }) {
    if (get().streaming) return;
    let created = false;
    if (!get().chat) {
      const { sel, draft } = get();
      const chat = withArtifacts(await api.chats.create({ providerId: sel.providerId, model: sel.model, ...draft }));
      set({ chat, draft: null });
      created = true;
    }
    const chatId = get().chat!.id;
    const isCurrent = () => get().chat?.id === chatId;
    const setMessages = (fn: (m: Message[]) => Message[]) => {
      if (isCurrent()) get().updateChat((c) => ({ ...c, messages: fn(c.messages) }));
    };

    // optimistic local update
    if (answer) {
      if (!findToolPart(get().chat!.messages, answer.callId)) return;
      setMessages((m) => patchToolPart(m, answer.callId, { status: 'done', result: { answer: answer.text } }));
    } else {
      let messages = skipPendingQuestions(get().chat!.messages);
      if (editMessageId) {
        const i = messages.findIndex((m) => m.id === editMessageId);
        if (i > -1) {
          get().forgetArtifacts(messages.slice(i));
          messages = messages.slice(0, i);
        }
      }
      if (regenerate) {
        let end = messages.length;
        while (end > 0 && messages[end - 1].role === 'assistant') end--;
        get().forgetArtifacts(messages.slice(end));
        messages = messages.slice(0, end);
      } else {
        const key = `pending-${Date.now()}`;
        messages = [...messages, { id: PENDING_MESSAGE_ID, key, role: 'user', content }];
        set({ enterKey: key });
      }
      setMessages(() => messages);
      set((s) => ({ threadVersion: s.threadVersion + 1 }));
    }
    if (created) set((s) => ({ chats: [summaryOf(get().chat!), ...s.chats] }));

    // the live reply: parts collect here and are painted at most once a frame while text streams
    const ctrl = new AbortController();
    const key = ++liveCounter;
    const { sel } = get();
    set({ streaming: ctrl, live: { key, model: sel.model, parts: [], waiting: true, active: true, error: null } });
    const patchLive = (p: Partial<LiveReply>) => set((s) => (s.live?.key === key ? { live: { ...s.live, ...p } } : {}));
    const clearLive = () => set((s) => (s.live?.key === key ? { live: null } : {}));

    const parts: Part[] = [];
    let waiting = true;
    let frame = 0;
    const paint = () => {
      frame = 0;
      patchLive({ parts: [...parts], waiting });
    };
    const paintNow = () => {
      cancelAnimationFrame(frame);
      paint();
    };
    const toolAt = (id: string, name: string) => {
      const i = parts.findIndex((p) => p.type === 'tool' && p.id === id);
      return i > -1 ? i : parts.push({ type: 'tool', id, name, args: null, result: null }) - 1;
    };

    let answered = false;
    const callbacks: SendCallbacks = {
      signal: ctrl.signal,
      onUser: (um) => setMessages((ms) => ms.map((m) => (m.id === PENDING_MESSAGE_ID ? { ...m, id: um.id, createdAt: um.createdAt ?? m.createdAt } : m))),
      onAnswered: ({ part }) => {
        answered = true;
        if (answer && part) setMessages((ms) => patchToolPart(ms, answer.callId, part));
      },
      onToken: (tok) => {
        const last = parts.at(-1);
        if (last?.type === 'text') parts[parts.length - 1] = { type: 'text', text: last.text + tok };
        else parts.push({ type: 'text', text: tok });
        waiting = false;
        if (!frame) frame = requestAnimationFrame(paint);
      },
      onToolStart: ({ id, name }) => {
        waiting = false;
        toolAt(id, name);
        paintNow();
      },
      onToolRun: ({ id, name, args }) => {
        waiting = false;
        const i = toolAt(id, name);
        parts[i] = { ...(parts[i] as ToolPart), args };
        paintNow();
      },
      onToolEnd: (part) => {
        parts[toolAt(part.id, part.name)] = part;
        waiting = part.status !== 'pending';
        paintNow();
      },
      onArtifact: (a) => {
        if (isCurrent()) get().putArtifact(a);
      },
      onTitle: (title) => get().setChatTitle(chatId, title),
    };

    try {
      const res = await api.chats.send(
        chatId,
        { content: answer ? '' : content, providerId: sel.providerId, model: sel.model, editMessageId, regenerate, tools: get().enabledTools(), ...(answer ? { answer } : {}) },
        callbacks,
      );
      cancelAnimationFrame(frame);
      const msg = res.message;
      if (!msg.content && !msg.parts?.length) return clearLive();
      if (isCurrent()) get().updateChat((c) => ({ ...c, messages: [...c.messages, msg], title: res.title || c.title }));
      set((s) => ({
        chats: s.chats
          .map((c) => (c.id === chatId ? { ...c, title: res.title || c.title, updatedAt: Date.now() } : c))
          .toSorted((a, b) => b.updatedAt - a.updatedAt),
        freshChatId: res.title ? chatId : s.freshChatId,
      }));
      clearLive();
    } catch (err) {
      cancelAnimationFrame(frame);
      const e = err as ApiError;
      if (isCurrent() && get().chat!.messages.some((m) => m.id === PENDING_MESSAGE_ID)) {
        // never reached the server: give the text back
        setMessages((ms) => ms.filter((m) => m.id !== PENDING_MESSAGE_ID));
        clearLive();
        set((s) => ({ threadVersion: s.threadVersion + 1 }));
        if (!get().composerText) get().setComposerText(content);
        get().toast(e.message);
        return;
      }
      if (answer && !answered) {
        clearLive();
        get().toast(e.message);
        if (e.status === STATUS_CONFLICT) {
          // the backend no longer has this question open: show what it has instead
          const fresh = await api.chats.get(chatId).catch(() => null);
          if (fresh && isCurrent()) {
            get().updateChat((c) => ({ ...c, messages: fresh.messages, artifacts: fresh.artifacts || [] }));
            set((s) => ({ threadVersion: s.threadVersion + 1 }));
          }
        } else {
          setMessages((ms) => patchToolPart(ms, answer.callId, { status: 'pending', result: null })); // never arrived: reopen the question
        }
        return;
      }
      patchLive({ parts: [...parts], waiting: false, active: false, error: e.message });
    } finally {
      if (get().streaming === ctrl) set({ streaming: null });
    }
  },
});
