/* Sending a message and following the reply as it is written.
   send covers four cases: a new message, an edited one, a regenerate, and an answer to a question the
   model asked. An answer adds no user message; the reply continues in a new assistant message.
   The backend writes a reply to its end whether or not this page follows it, and several chats may be
   writing at once. This page follows one at a time: the open chat's. Leaving the chat drops the
   connection (detach); coming back picks the reply up again from its start (attach). */
import { api } from '../api';
import { ApiError } from '../api/errors';
import { PENDING_MESSAGE_ID, STATUS_CONFLICT } from '../config';
import { dataUrl, preloadImage } from '../lib/images';
import { findToolPart, patchToolPart, skipPendingQuestions } from '../lib/parts';
import type { ImageInput, Message, MessageImage, Part, SendCallbacks, SendResult, ToolPart } from '../types';
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
  /** Images for a new message. An edited message keeps the ones it had. */
  images?: ImageInput[];
  editMessageId?: string;
  regenerate?: boolean;
  answer?: { callId: string; text: string };
}

export interface SendSlice {
  /** The connection following the open chat's reply; null when it has none being written. */
  streaming: AbortController | null;
  live: LiveReply | null;
  /** React key of the message that animates in (the one just sent). */
  enterKey: string | null;
  send(opts: SendOptions): Promise<void>;
  /** Follows the reply the open chat is already writing (it was started elsewhere, or before this chat was left). */
  attach(chatId: string): Promise<void>;
  /** Stops the open chat's reply at the backend. What there is of it is kept. */
  stop(): void;
  /** Stops following the reply, which carries on at the backend. */
  detach(): void;
}

let liveCounter = 0;

export const createSendSlice: Slice<SendSlice> = (set, get) => {
  /** Starts the live reply of a chat: callbacks that paint what arrives (at most once a frame while text
      streams), and what to do when it ends. */
  function startLive(chatId: string) {
    const ctrl = new AbortController();
    const key = ++liveCounter;
    const isCurrent = () => get().chat?.id === chatId;
    set({ streaming: ctrl, live: { key, model: get().sel.model, parts: [], waiting: true, active: true, error: null } });
    get().patchChat(chatId, { running: true });
    const patchLive = (p: Partial<LiveReply>) => set((s) => (s.live?.key === key ? { live: { ...s.live, ...p } } : {}));
    const clear = () => set((s) => (s.live?.key === key ? { live: null } : {}));

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

    const callbacks: SendCallbacks = {
      signal: ctrl.signal,
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

    return {
      callbacks,
      clear,
      /** This page stopped following (the chat was left); the reply carries on at the backend. */
      left: () => ctrl.signal.aborted,
      /** The reply ended, finished or stopped: it becomes a message. */
      done({ message: msg, title }: SendResult) {
        clear();
        if (!msg.content && !msg.parts?.length) return;
        if (isCurrent()) get().updateChat((c) => ({ ...c, messages: [...c.messages, msg], title: title || c.title }));
        set((s) => ({
          chats: s.chats
            .map((c) => (c.id === chatId ? { ...c, title: title || c.title, updatedAt: Date.now() } : c))
            .toSorted((a, b) => b.updatedAt - a.updatedAt),
          freshChatId: title ? chatId : s.freshChatId,
        }));
      },
      /** It failed part-way: what arrived stays on screen, with the error and Retry. */
      fail: (error: string) => patchLive({ parts: [...parts], waiting: false, active: false, error }),
      /** Always last. */
      end() {
        cancelAnimationFrame(frame);
        if (get().streaming === ctrl) set({ streaming: null });
        if (!ctrl.signal.aborted) get().patchChat(chatId, { running: false });
      },
    };
  }

  return {
    streaming: null,
    live: null,
    enterKey: null,

    stop() {
      const id = get().chat?.id;
      if (id && get().streaming) void api.chats.stop(id); // the stream then ends with the stopped reply
    },

    detach() {
      get().streaming?.abort();
      set({ streaming: null });
    },

    async attach(chatId) {
      if (get().chat?.id !== chatId || get().streaming) return;
      const live = startLive(chatId);
      try {
        const res = await api.chats.attach(chatId, live.callbacks);
        if (res) return live.done(res);
        // it ended in between: show what was saved
        live.clear();
        const fresh = withArtifacts(await api.chats.get(chatId));
        if (!fresh || live.left() || get().chat?.id !== chatId) return;
        set((s) => ({ chat: fresh, threadVersion: s.threadVersion + 1 }));
      } catch (e) {
        if (!live.left()) live.fail((e as Error).message);
      } finally {
        live.end();
      }
    },

    async send({ content = '', images = [], editMessageId, regenerate, answer }) {
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
        // shown at once from the local copies; an edited message keeps the images it has
        let shown: MessageImage[] = images.map((image, i) => ({ id: `local-${i}`, mediaType: image.mediaType, url: dataUrl(image) }));
        if (editMessageId) {
          const i = messages.findIndex((m) => m.id === editMessageId);
          if (i > -1) {
            shown = messages[i].images ?? [];
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
          messages = [...messages, { id: PENDING_MESSAGE_ID, key, role: 'user', content, images: shown }];
          set({ enterKey: key });
        }
        setMessages(() => messages);
        set((s) => ({ threadVersion: s.threadVersion + 1 }));
      }
      if (created) set((s) => ({ chats: [summaryOf(get().chat!), ...s.chats] }));

      const { sel } = get();
      const live = startLive(chatId);
      let answered = false;
      const callbacks: SendCallbacks = {
        ...live.callbacks,
        onUser: (um) => {
          // the local copies stay on screen until the server's have loaded, so nothing flickers
          const saved = um.images ?? [];
          setMessages((ms) =>
            ms.map((m) =>
              m.id === PENDING_MESSAGE_ID
                ? { ...m, id: um.id, createdAt: um.createdAt ?? m.createdAt, images: saved.map((image, i) => ({ ...image, url: m.images?.[i]?.url })) }
                : m,
            ),
          );
          void Promise.all(saved.map((image) => preloadImage(api.imageUrl(image.id)))).then(
            () => setMessages((ms) => ms.map((m) => (m.id === um.id ? { ...m, images: saved } : m))),
            () => {}, // the local copies stay
          );
        },
        onAnswered: ({ part }) => {
          answered = true;
          if (answer && part) setMessages((ms) => patchToolPart(ms, answer.callId, part));
        },
      };

      try {
        const res = await api.chats.send(
          chatId,
          {
            content: answer ? '' : content,
            ...(images.length && !editMessageId && !regenerate && !answer ? { images } : {}),
            providerId: sel.providerId,
            model: sel.model,
            editMessageId,
            regenerate,
            tools: get().enabledTools(),
            ...(answer ? { answer } : {}),
          },
          callbacks,
        );
        live.done(res);
      } catch (err) {
        if (live.left()) return;
        const e = err as ApiError;
        if (isCurrent() && get().chat!.messages.some((m) => m.id === PENDING_MESSAGE_ID)) {
          // never reached the server (or it refused, as when the chat is already writing a reply): give the text and the images back
          setMessages((ms) => ms.filter((m) => m.id !== PENDING_MESSAGE_ID));
          live.clear();
          set((s) => ({ threadVersion: s.threadVersion + 1 }));
          if (!get().composerText) get().setComposerText(content);
          if (!editMessageId && !get().attachments.length) get().setAttachments(images);
          get().toast(e.message);
          return;
        }
        if (answer && !answered) {
          live.clear();
          get().toast(e.message);
          if (e.status === STATUS_CONFLICT) {
            // the backend no longer has this question open: show what it has instead
            const fresh = await api.chats.get(chatId).catch(() => null);
            if (fresh && isCurrent()) {
              get().updateChat((c) => ({ ...c, messages: fresh.messages, artifacts: fresh.artifacts || [], running: fresh.running }));
              set((s) => ({ threadVersion: s.threadVersion + 1 }));
            }
          } else {
            setMessages((ms) => patchToolPart(ms, answer.callId, { status: 'pending', result: null })); // never arrived: reopen the question
          }
          return;
        }
        live.fail(e.message);
      } finally {
        live.end();
      }
    },
  };
};
