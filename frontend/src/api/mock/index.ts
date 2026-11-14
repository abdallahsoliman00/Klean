/* In-memory stand-in for the backend, for working on the UI alone: `npm run dev:mock`.
   Same method names and shapes as the HTTP client. Put one of these words in a message to see
   the matching tool UI (same words as the backend's fake provider, plus a few extras marked *):
     /artifact  /code  /svg  /md*  /report  /update  /ask  /askmany*  /search  /fetch  /fail*  /skill <name>
   NOIMAGES makes the model refuse a message with images, and /notools* makes it refuse tools. */
import {
  IMAGE_TYPES, KEYLESS_SEARCH_PROVIDER, MAX_IMAGE_BYTES, MAX_IMAGES_PER_MESSAGE, MOCK_LATENCY, PRESET_NAME_MAX_CHARS, SEARCH_PROVIDERS,
  SKILL_DESCRIPTION_MAX_CHARS, SKILL_INSTRUCTIONS_MAX_CHARS, SKILL_NAME_MAX_CHARS, STATUS_CONFLICT, STATUS_INVALID, STATUS_NOT_FOUND,
} from '../../config';
import type { Api, AppSettings, ChatConfig, ImageInput, Message, MessageImage, PresetInput, Provider, SendCallbacks, Skill, SkillInput, ToolPart } from '../../types';
import { ApiError } from '../errors';
import { artifacts, chats, images, presets, providers, skills, uid, type MockChat, type MockProvider } from './data';
import { dropArtifacts, sleep, toolParts } from './helpers';
import { scriptedReply } from './reply';

const settings: AppSettings = { searchProvider: KEYLESS_SEARCH_PROVIDER, hasSearchKey: false, chatDefaults: null };
const trash: Record<string, MockChat> = {};
const strip = ({ messages: _, ...rest }: MockChat) => ({ ...rest });
const pub = (p: MockProvider): Provider => ({ id: p.id, name: p.name, baseUrl: p.baseUrl, hasKey: p.hasKey });
const mustChat = (id: string) => {
  const c = chats.find((x) => x.id === id);
  if (!c) throw new ApiError('Chat not found.', 404);
  return c;
};
const mustPreset = (id: string) => {
  const p = presets.find((x) => x.id === id);
  if (!p) throw new ApiError('Preset not found.', 404);
  return p;
};
/** The backend's checks on a preset or the chat defaults: a trimmed name of the right length, a provider that exists. */
function checked<T extends Partial<PresetInput>>(data: T): T {
  const name = data.name?.trim();
  if (name !== undefined && (!name || name.length > PRESET_NAME_MAX_CHARS)) throw new ApiError(`name must be 1 to ${PRESET_NAME_MAX_CHARS} characters.`, STATUS_INVALID);
  if (data.providerId && !providers.some((p) => p.id === data.providerId)) throw new ApiError('Unknown provider.', STATUS_INVALID);
  return name === undefined ? data : { ...data, name };
}
const invalid = (detail: string) => new ApiError(detail, STATUS_INVALID);
const mustSkill = (id: string) => {
  const s = skills.find((x) => x.id === id);
  if (!s) throw new ApiError('Skill not found.', STATUS_NOT_FOUND);
  return s;
};
/** The backend's checks on a skill; `self` is the one being changed, which may keep its own name. */
function checkedSkill<T extends Partial<SkillInput>>(data: T, self?: Skill): T {
  const { name, description, instructions } = data;
  if (name !== undefined) {
    if (!/^[a-z0-9-]+$/.test(name) || name.length > SKILL_NAME_MAX_CHARS) throw invalid(`name must be 1 to ${SKILL_NAME_MAX_CHARS} lowercase letters, digits or hyphens.`);
    if (skills.some((s) => s !== self && s.name.toLowerCase() === name)) throw invalid(`A skill named ${name} already exists.`);
  }
  if (description !== undefined && (!description.trim() || description.length > SKILL_DESCRIPTION_MAX_CHARS)) throw invalid(`description must be 1 to ${SKILL_DESCRIPTION_MAX_CHARS} characters.`);
  if (instructions !== undefined && (!instructions.trim() || instructions.length > SKILL_INSTRUCTIONS_MAX_CHARS)) throw invalid(`instructions must be 1 to ${SKILL_INSTRUCTIONS_MAX_CHARS} characters.`);
  return data;
}
/** The backend's checks on a message's images; stores them and returns what the message keeps. */
function storeImages(list: ImageInput[] = []): MessageImage[] {
  if (list.length > MAX_IMAGES_PER_MESSAGE) throw invalid(`A message can have at most ${MAX_IMAGES_PER_MESSAGE} images.`);
  for (const { mediaType, data } of list) {
    if (!IMAGE_TYPES.includes(mediaType)) throw invalid(`Unsupported image type ${mediaType}.`);
    if (data.length * 0.75 > MAX_IMAGE_BYTES) throw invalid('An image is larger than 5 MB.');
  }
  return list.map(({ mediaType, data }) => {
    const id = 'img_' + uid();
    images.set(id, `data:${mediaType};base64,${data}`);
    return { id, mediaType };
  });
}
/** Like the backend: a title that sums the chat up follows once the reply is done. */
async function sendSummaryTitle(c: MockChat, prompt: string, cb: SendCallbacks) {
  await sleep(MOCK_LATENCY.title);
  c.title = 'About ' + prompt.replace(/\s+/g, ' ').trim().split(' ').slice(0, 4).join(' ').toLowerCase();
  cb.onTitle?.(c.title);
}
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
      let attached = storeImages(body.images);
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
        if (editMessageId) {
          const cut = c.messages.splice(c.messages.findIndex((m) => m.id === editMessageId));
          attached = cut[0]?.images ?? []; // an edit changes the text only
          dropArtifacts(cut);
        }
        if (regenerate) {
          const cut: Message[] = [];
          while (c.messages.at(-1)?.role === 'assistant') cut.push(c.messages.pop()!);
          dropArtifacts(cut);
          prompt = c.messages.at(-1)?.content || '';
        } else {
          const um: Message = { id: uid(), role: 'user', content, images: attached, parts: null, createdAt: Date.now() };
          c.messages.push(um);
          cb.onUser?.(um);
        }
      }
      Object.assign(c, { providerId, model, updatedAt: Date.now() });
      let titled: string | null = null;
      if (c.title === 'New chat' && (content || attached.length)) {
        titled = content ? content.replace(/\s+/g, ' ').slice(0, 42).trim() + (content.length > 42 ? '…' : '') : 'Image';
        c.title = titled;
      }
      const p = providers.find((x) => x.id === providerId);
      await sleep(random(MOCK_LATENCY.firstToken));
      if (!p || p.offline) throw new ApiError(`${p ? p.name : 'Provider'} didn't respond at ${p ? p.baseUrl : ''}. Start the server or pick another model.`);

      const seen = c.messages.findLast((m) => m.role === 'user')?.images?.length ?? 0;
      if (seen && prompt.includes('NOIMAGES')) throw new ApiError("This model or server can't read images. Pick a model with vision support, or send the message without the images.");
      if (prompt.includes('/notools') && (body.tools.length || skills.some((s) => s.enabled)))
        throw new ApiError("This model or provider doesn't appear to support tools. Turn tools off in the composer and send the message again.");

      const parts = await scriptedReply({ chat: c, prompt, images: seen, answered, tools: new Set(body.tools), cb });
      const stopped = !!cb.signal?.aborted;
      const am: Message = { id: uid(), role: 'assistant', model, content: parts.map((x) => (x.type === 'text' ? x.text : '')).join(''), parts, stopped, createdAt: Date.now() };
      if (parts.length) c.messages.push(am);
      if (titled && prompt && !stopped) void sendSummaryTitle(c, prompt, cb);
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
  presets: {
    async list() {
      await sleep(MOCK_LATENCY.read);
      return structuredClone(presets);
    },
    async create(data) {
      const p = { ...checked(data), id: uid() };
      presets.push(p);
      return { ...p };
    },
    async update(id, patch) {
      return { ...Object.assign(mustPreset(id), checked(patch)) };
    },
    async remove(id) {
      presets.splice(presets.indexOf(mustPreset(id)), 1);
    },
  },
  skills: {
    async list() {
      await sleep(MOCK_LATENCY.read);
      return structuredClone(skills);
    },
    async create(data) {
      const s: Skill = { enabled: true, ...checkedSkill(data), id: uid() };
      skills.push(s);
      return { ...s };
    },
    async update(id, patch) {
      const s = mustSkill(id);
      return { ...Object.assign(s, checkedSkill(patch, s)) };
    },
    async remove(id) {
      skills.splice(skills.indexOf(mustSkill(id)), 1);
    },
  },
  imageUrl: (id) => images.get(id) ?? '',
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
      if (patch.chatDefaults !== undefined) settings.chatDefaults = patch.chatDefaults && checked<ChatConfig>(patch.chatDefaults); // replaced whole, null clears
      return { ...settings };
    },
  },
};
