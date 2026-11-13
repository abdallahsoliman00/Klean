/* The HTTP client: the only module that talks to the real backend. */
import { API_ORIGIN, API_PREFIX, ENDPOINTS, STATUS_NO_CONTENT } from '../config';
import type { Api } from '../types';
import { errorFromResponse, ApiError, OFFLINE_MESSAGE } from './errors';
import { streamMessage } from './stream';

const ROOT = API_ORIGIN + API_PREFIX;

async function req<T>(method: string, path: string, body?: unknown): Promise<T> {
  let res: Response;
  try {
    res = await fetch(ROOT + path, {
      method,
      headers: body ? { 'Content-Type': 'application/json' } : {},
      body: body ? JSON.stringify(body) : undefined,
    });
  } catch {
    throw new ApiError(OFFLINE_MESSAGE);
  }
  if (!res.ok) throw await errorFromResponse(res);
  return (res.status === STATUS_NO_CONTENT ? null : await res.json()) as T;
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
  },
  artifacts: {
    remove: (id) => req('DELETE', ENDPOINTS.artifact(id)),
  },
  settings: {
    get: () => req('GET', ENDPOINTS.settings),
    update: (patch) => req('PATCH', ENDPOINTS.settings, patch),
  },
};
