/* In-memory stand-in for the backend, for working on the UI alone: `npm run dev:mock`.
   Same method names and shapes as the HTTP client. Put one of these words in a message to see
   the matching tool UI (same words as the backend's fake provider, plus a few extras marked *):
     /artifact  /code  /svg  /md*  /update  /ask  /askmany*  /search  /fetch  /fail*  */
import { KEYLESS_SEARCH_PROVIDER, MOCK_LATENCY, SEARCH_PROVIDERS, STATUS_CONFLICT } from '../../config';
import type { Api, Message, Provider, ToolPart } from '../../types';
import { ApiError } from '../errors';
import { artifacts, chats, providers, uid, type MockChat, type MockProvider } from './data';
import { dropArtifacts, sleep, toolParts } from './helpers';
import { scriptedReply } from './reply';

const settings = { searchProvider: KEYLESS_SEARCH_PROVIDER as string, hasSearchKey: false };
const trash: Record<string, MockChat> = {};
const strip = ({ messages: _, ...rest }: MockChat) => ({ ...rest });
const pub = (p: MockProvider): Provider => ({ id: p.id, name: p.name, baseUrl: p.baseUrl, hasKey: p.hasKey });
const mustChat = (id: string) => {
  const c = chats.find((x) => x.id === id);
  if (!c) throw new ApiError('Chat not found.', 404);
  return c;
};
const random = ([base, extra]: readonly [number, number]) => base + Math.random() * extra;

export const mockApi: Api = {
  providers: {
    async list() {
      await sleep(MOCK_LATENCY.read);
      return providers.map(pub);
    },
    async create(data) {
      const offline = /1234/.test(data.baseUrl);
      const p: MockProvider = { id: uid(), name: data.name, baseUrl: data.baseUrl, hasKey: !!data.apiKey, models: offline ? null : ['local-model-a', 'local-model-b'], offline };
      providers.push(p);
      return pub(p);
    },
    async update(id, data) {
      const p = providers.find((x) => x.id === id);
      if (!p) throw new ApiError('Provider not found.', 404);
      Object.assign(p, { name: data.name, baseUrl: data.baseUrl });
      if (data.apiKey !== undefined) p.hasKey = !!data.apiKey;
      return pub(p);
    },
    async remove(id) {
      const i = providers.findIndex((x) => x.id === id);
      if (i > -1) providers.splice(i, 1);
    },
    async models(id) {
      await sleep(random(MOCK_LATENCY.models));
      const p = providers.find((x) => x.id === id);
      if (!p) throw new ApiError('Provider not found.');
      if (p.offline || !p.models) throw new ApiError(`Couldn't reach ${p.baseUrl}/models. Check that the server is running and the URL is right.`);
      return p.models.map((m) => ({ id: m, ownedBy: m.includes('/') ? m.split('/')[0] : p.name.toLowerCase() }));
    },
  },
  chats: {
    async list() {
      await sleep(MOCK_LATENCY.quick);
      return chats.map(strip).toSorted((a, b) => b.updatedAt - a.updatedAt);
    },
    async get(id) {
      await sleep(MOCK_LATENCY.read);
      const c = chats.find((x) => x.id === id);
      return c ? structuredClone({ ...c, artifacts: artifacts.filter((a) => a.chatId === id) }) : null;
    },
    async create(data) {
      const c: MockChat = { id: uid(), title: 'New chat', updatedAt: Date.now(), systemPrompt: '', temperature: 0.7, contextWindow: 8192, messages: [], ...data };
      chats.push(c);
      return structuredClone({ ...c, artifacts: [] });
    },
    async update(id, patch) {
      const c = mustChat(id);
      Object.assign(c, patch);
      return strip(c);
    },
    async remove(id) {
      const i = chats.findIndex((x) => x.id === id);
      if (i > -1) trash[id] = chats.splice(i, 1)[0];
    },
    async restore(id) {
      if (trash[id]) {
        chats.push(trash[id]);
        delete trash[id];
      }
    },
    /* Appends (or edits / regenerates / answers a question) and streams the reply.
       body.editMessageId → replace that user message and drop everything after it
       body.regenerate    → drop the last assistant messages and answer again
       body.answer        → {callId, text}: closes a pending ask_user and continues in a new message
       body.tools         → enabled groups; omitted means all */
    async send(id, body, cb = {}) {
      const { content, providerId, model, editMessageId, regenerate, answer } = body;
      const c = mustChat(id);
      let prompt = content || '';
      let answered: string | null = null;
      if (answer) {
        let hit: { m: Message; p: ToolPart } | null = null;
        for (const { m, p } of toolParts(c.messages)) if (p.id === answer.callId && p.status === 'pending') hit = { m, p };
        if (!hit) throw new ApiError('That question is no longer waiting for an answer.', STATUS_CONFLICT);
        Object.assign(hit.p, { status: 'done', result: { answer: answer.text } });
        await sleep(MOCK_LATENCY.write);
        cb.onAnswered?.({ messageId: hit.m.id, part: structuredClone(hit.p) });
        answered = answer.text;
      } else {
        for (const { p } of toolParts(c.messages)) if (p.status === 'pending') Object.assign(p, { status: 'done', result: { answer: '' } });
        if (editMessageId) dropArtifacts(c.messages.splice(c.messages.findIndex((m) => m.id === editMessageId)));
        if (regenerate) {
          const cut: Message[] = [];
          while (c.messages.at(-1)?.role === 'assistant') cut.push(c.messages.pop()!);
          dropArtifacts(cut);
          prompt = c.messages.at(-1)?.content || '';
        } else {
          const um: Message = { id: uid(), role: 'user', content, parts: null, createdAt: Date.now() };
          c.messages.push(um);
          cb.onUser?.(um);
        }
      }
      Object.assign(c, { providerId, model, updatedAt: Date.now() });
      let titled: string | null = null;
      if (c.title === 'New chat' && content) {
        titled = content.replace(/\s+/g, ' ').slice(0, 42).trim() + (content.length > 42 ? '…' : '');
        c.title = titled;
      }
      const p = providers.find((x) => x.id === providerId);
      await sleep(random(MOCK_LATENCY.firstToken));
      if (!p || p.offline) throw new ApiError(`${p ? p.name : 'Provider'} didn't respond at ${p ? p.baseUrl : ''}. Start the server or pick another model.`);

      const parts = await scriptedReply({ chat: c, prompt, answered, tools: new Set(body.tools), cb });
      const stopped = !!cb.signal?.aborted;
      const am: Message = { id: uid(), role: 'assistant', model, content: parts.map((x) => (x.type === 'text' ? x.text : '')).join(''), parts, stopped, createdAt: Date.now() };
      if (parts.length) c.messages.push(am);
      return structuredClone({ message: am, title: titled });
    },
  },
  artifacts: {
    async remove(id) {
      await sleep(MOCK_LATENCY.read);
      const i = artifacts.findIndex((a) => a.id === id);
      if (i > -1) artifacts.splice(i, 1);
    },
  },
  settings: {
    async get() {
      await sleep(MOCK_LATENCY.read);
      return { ...settings };
    },
    async update(patch) {
      await sleep(MOCK_LATENCY.write);
      if (patch.searchProvider !== undefined) {
        if (!SEARCH_PROVIDERS.some((s) => s.id === patch.searchProvider)) throw new ApiError('Unknown search provider.');
        settings.searchProvider = patch.searchProvider;
      }
      if (patch.searchKey !== undefined) settings.hasSearchKey = !!patch.searchKey;
      return { ...settings };
    },
  },
};
