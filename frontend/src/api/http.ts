/* The HTTP client: the only module that talks to the real backend. */
import { API_ORIGIN, API_PREFIX, ENDPOINTS, PDF_TYPE, REPORT_PDF, STATUS_NO_CONTENT } from '../config';
import type { Api } from '../types';
import { errorFromResponse, ApiError, OFFLINE_MESSAGE } from './errors';
import { attachStream, streamMessage } from './stream';

const ROOT = API_ORIGIN + API_PREFIX;

/** The response to a request, or an ApiError if it can't be sent, times out or is refused. */
async function send(method: string, path: string, body?: unknown, timeoutMs?: number): Promise<Response> {
  let res: Response;
  try {
    res = await fetch(ROOT + path, {
      method,
      headers: body ? { 'Content-Type': 'application/json' } : {},
      body: body ? JSON.stringify(body) : undefined,
      signal: timeoutMs ? AbortSignal.timeout(timeoutMs) : undefined,
    });
  } catch {
    throw new ApiError(OFFLINE_MESSAGE);
  }
  if (!res.ok) throw await errorFromResponse(res);
  return res;
}

async function req<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await send(method, path, body);
  return (res.status === STATUS_NO_CONTENT ? null : await res.json()) as T;
}

async function pdf(path: string): Promise<Blob> {
  const res = await send('POST', path, undefined, REPORT_PDF.serverTimeoutMs);
  const blob = await res.blob();
  if (blob.type !== PDF_TYPE) throw new ApiError('The backend did not send a PDF.'); // e.g. an older backend answering with a page
  return blob;
}

export const httpApi: Api = {
  providers: {
    list: () => req('GET', ENDPOINTS.providers),
    create: (data) => req('POST', ENDPOINTS.providers, data),
    update: (id, data) => req('PATCH', ENDPOINTS.provider(id), data),
    remove: (id) => req('DELETE', ENDPOINTS.provider(id)),
    models: (id) => req('GET', ENDPOINTS.providerModels(id)),
  },
  chats: {
    list: () => req('GET', ENDPOINTS.chats),
    get: (id) => req('GET', ENDPOINTS.chat(id)),
    create: (data) => req('POST', ENDPOINTS.chats, data),
    update: (id, patch) => req('PATCH', ENDPOINTS.chat(id), patch),
    remove: (id) => req('DELETE', ENDPOINTS.chat(id)),
    restore: (id) => req('POST', ENDPOINTS.chatRestore(id)),
    send: (chatId, body, cb) => streamMessage(ROOT + ENDPOINTS.chatMessages(chatId), body, cb),
    attach: (chatId, cb) => attachStream(ROOT + ENDPOINTS.chatStream(chatId), cb),
    stop: (chatId) => req('POST', ENDPOINTS.chatStop(chatId)),
  },
  artifacts: {
    list: () => req('GET', ENDPOINTS.artifacts),
    versions: (id) => req('GET', ENDPOINTS.artifactVersions(id)),
    remove: (id) => req('DELETE', ENDPOINTS.artifact(id)),
    pdf: (id) => pdf(ENDPOINTS.artifactPdf(id)),
  },
  presets: {
    list: () => req('GET', ENDPOINTS.presets),
    create: (data) => req('POST', ENDPOINTS.presets, data),
    update: (id, patch) => req('PATCH', ENDPOINTS.preset(id), patch),
    remove: (id) => req('DELETE', ENDPOINTS.preset(id)),
  },
  settings: {
    get: () => req('GET', ENDPOINTS.settings),
    update: (patch) => req('PATCH', ENDPOINTS.settings, patch),
  },
  skills: {
    list: () => req('GET', ENDPOINTS.skills),
    create: (data) => req('POST', ENDPOINTS.skills, data),
    update: (id, patch) => req('PATCH', ENDPOINTS.skill(id), patch),
    remove: (id) => req('DELETE', ENDPOINTS.skill(id)),
  },
  update: {
    check: () => req('GET', ENDPOINTS.update),
    apply: () => req('POST', ENDPOINTS.update),
  },
  version: () => req<{ version: string }>('GET', ENDPOINTS.health).then((h) => h.version),
  imageUrl: (id) => ROOT + ENDPOINTS.image(id),
};
