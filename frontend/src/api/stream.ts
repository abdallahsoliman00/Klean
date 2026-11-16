/* A reply's event stream: POST /chats/:id/messages starts one, GET /chats/:id/stream follows one already
   running (replayed from its start). The reply is written at the backend whether or not anyone follows it,
   so aborting only drops the connection. It ends at "done", "stopped" or "error"; after "done" the stream
   stays open a little longer for a summary title. */
import { STATUS_NO_CONTENT } from '../config';
import type { Message, SendBody, SendCallbacks, SendResult } from '../types';
import { ApiError, errorFromResponse, isAbort, OFFLINE_MESSAGE } from './errors';
import { readSse, type SseEvent } from './sse';

/** What a reply stopped before it wrote anything comes to. */
const NOTHING: Message = { id: '', role: 'assistant', content: '', stopped: true };

/** Handles one event. Returns the final result on "done" or "stopped". */
function dispatch(event: string, payload: any, cb: SendCallbacks): SendResult | null {
  switch (event) {
    case 'user':
      cb.onUser?.(payload);
      break;
    case 'answered':
      cb.onAnswered?.(payload);
      break;
    case 'token':
      cb.onToken?.(payload.delta);
      break;
    case 'tool_start':
      cb.onToolStart?.(payload);
      break;
    case 'tool_run':
      cb.onToolRun?.(payload);
      break;
    case 'tool_end':
      cb.onToolEnd?.(payload.part);
      break;
    case 'artifact':
      cb.onArtifact?.(payload.artifact);
      break;
    case 'done':
      return payload as SendResult;
    case 'stopped':
      return { message: payload.message ?? NOTHING, title: null };
    case 'error':
      throw new ApiError(payload.message);
  }
  return null;
}

/** Reads on after "done" for the summary title. The reply is already complete, so a failure here is ignored. */
async function readLateTitle(events: AsyncGenerator<SseEvent>, cb: SendCallbacks) {
  try {
    for await (const { event, data } of events) if (event === 'title') cb.onTitle?.(JSON.parse(data).title);
  } catch {
    /* the connection dropped: the provisional title stays */
  }
}

/** The response to a stream request. An abort is rethrown as it is, for the caller to tell apart. */
async function open(url: string, init: RequestInit, cb: SendCallbacks): Promise<Response> {
  let res: Response;
  try {
    res = await fetch(url, { ...init, signal: cb.signal });
  } catch (e) {
    throw isAbort(e) ? e : new ApiError(OFFLINE_MESSAGE);
  }
  if (!res.ok) throw await errorFromResponse(res);
  return res;
}

/** Passes each event to the callbacks until the reply ends. */
async function follow(res: Response, cb: SendCallbacks): Promise<SendResult> {
  if (!res.body) throw new ApiError('The reply was cut off before it finished.');
  const events = readSse(res.body);
  // not for-await: leaving that loop early would close the stream, and the title comes after "done"
  for (let next = await events.next(); !next.done; next = await events.next()) {
    const result = dispatch(next.value.event, JSON.parse(next.value.data), cb);
    if (!result) continue;
    void readLateTitle(events, cb);
    return result;
  }
  throw new ApiError('The reply was cut off before it finished.');
}

export async function streamMessage(url: string, body: SendBody, cb: SendCallbacks = {}): Promise<SendResult> {
  return follow(await open(url, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }, cb), cb);
}

/** Null when the chat has no reply running (204). */
export async function attachStream(url: string, cb: SendCallbacks = {}): Promise<SendResult | null> {
  const res = await open(url, {}, cb);
  return res.status === STATUS_NO_CONTENT ? null : follow(res, cb);
}
