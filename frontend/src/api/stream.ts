/* POST /chats/:id/messages and its event stream.
   The parts are collected here too, so a reply stopped locally keeps what already arrived.
   The reply is finished at "done"; the stream stays open a little longer for a summary title. */
import { LOCAL_MESSAGE_PREFIX } from '../config';
import type { Part, SendBody, SendCallbacks, SendResult, ToolPart } from '../types';
import { ApiError, errorFromResponse, isAbort, OFFLINE_MESSAGE } from './errors';
import { readSse, type SseEvent } from './sse';

/** Mutable list of the parts that have arrived so far. */
class PartCollector {
  readonly parts: Part[] = [];

  addText(delta: string) {
    const last = this.parts.at(-1);
    if (last?.type === 'text') last.text += delta;
    else this.parts.push({ type: 'text', text: delta });
  }

  private toolIndex(id: string) {
    return this.parts.findIndex((p) => p.type === 'tool' && p.id === id);
  }

  startTool(id: string, name: string) {
    this.parts.push({ type: 'tool', id, name, args: null, result: null });
  }

  setArgs(id: string, name: string, args: ToolPart['args']) {
    const i = this.toolIndex(id);
    if (i > -1) (this.parts[i] as ToolPart).args = args;
    else this.parts.push({ type: 'tool', id, name, args, result: null });
  }

  endTool(part: ToolPart) {
    const i = this.toolIndex(part.id);
    if (i > -1) this.parts[i] = part;
    else this.parts.push(part);
  }

  /** The reply as it stands, for when the user stopped it. Unfinished tool calls are dropped,
      as the backend drops them too. */
  stopped(model: string | null): SendResult {
    const kept = this.parts.filter((p) => p.type === 'text' || p.status);
    return {
      message: {
        id: LOCAL_MESSAGE_PREFIX + Date.now(),
        role: 'assistant',
        model,
        content: kept.map((p) => (p.type === 'text' ? p.text : '')).join(''),
        parts: kept,
        stopped: true,
      },
      title: null,
    };
  }
}

/** Handles one event. Returns the final result on "done". */
function dispatch(event: string, payload: any, parts: PartCollector, cb: SendCallbacks): SendResult | null {
  switch (event) {
    case 'user':
      cb.onUser?.(payload);
      break;
    case 'answered':
      cb.onAnswered?.(payload);
      break;
    case 'token':
      parts.addText(payload.delta);
      cb.onToken?.(payload.delta);
      break;
    case 'tool_start':
      parts.startTool(payload.id, payload.name);
      cb.onToolStart?.(payload);
      break;
    case 'tool_run':
      parts.setArgs(payload.id, payload.name, payload.args);
      cb.onToolRun?.(payload);
      break;
    case 'tool_end':
      parts.endTool(payload.part);
      cb.onToolEnd?.(payload.part);
      break;
    case 'artifact':
      cb.onArtifact?.(payload.artifact);
      break;
    case 'done':
      return payload as SendResult;
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

export async function streamMessage(url: string, body: SendBody, cb: SendCallbacks = {}): Promise<SendResult> {
  const { signal } = cb;
  const parts = new PartCollector();
  let res: Response;
  try {
    res = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
      signal,
    });
  } catch (e) {
    if (isAbort(e)) return parts.stopped(body.model);
    throw new ApiError(OFFLINE_MESSAGE);
  }
  if (!res.ok) throw await errorFromResponse(res);
  if (!res.body) throw new ApiError('The reply was cut off before it finished.');

  const events = readSse(res.body);
  try {
    // not for-await: leaving that loop early would close the stream, and the title comes after "done"
    for (let next = await events.next(); !next.done; next = await events.next()) {
      const result = dispatch(next.value.event, JSON.parse(next.value.data), parts, cb);
      if (!result) continue;
      void readLateTitle(events, cb);
      return result;
    }
  } catch (e) {
    if (isAbort(e)) return parts.stopped(body.model);
    throw e;
  }
  if (signal?.aborted) return parts.stopped(body.model);
  throw new ApiError('The reply was cut off before it finished.');
}
