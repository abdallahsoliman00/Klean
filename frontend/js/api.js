/* Switchboard API client — the only file in the frontend that knows about the backend.
   Point it at another backend by setting window.SWITCHBOARD_API = "http://host:port" before this script. */
(() => {
  const ROOT = (window.SWITCHBOARD_API || '').replace(/\/+$/, '') + '/api';
  const OFFLINE = "Can't reach the Switchboard backend. Check that it's running.";

  async function req(method, path, body) {
    let res;
    try {
      res = await fetch(ROOT + path, {
        method,
        headers: body ? { 'Content-Type': 'application/json' } : {},
        body: body ? JSON.stringify(body) : undefined,
      });
    } catch { throw new Error(OFFLINE); }
    if (!res.ok) throw new Error(await errorText(res));
    return res.status === 204 ? null : res.json();
  }
  async function errorText(res) {
    try {
      const j = await res.json();
      if (typeof j.detail === 'string') return j.detail;
      if (Array.isArray(j.detail)) return j.detail.map(d => d.msg).join('; ');
    } catch {}
    return `Request failed (${res.status}).`;
  }

  /* Reads the Server-Sent Events stream from POST /chats/:id/messages. */
  async function send(chatId, body, { onUser, onToken, signal } = {}) {
    let text = '';
    const stopped = () => ({ message: { id: 'local-' + Date.now(), role: 'assistant', model: body.model, content: text, stopped: true }, title: null });
    let res;
    try {
      res = await fetch(`${ROOT}/chats/${chatId}/messages`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body), signal,
      });
    } catch (e) {
      if (e.name === 'AbortError') return stopped();
      throw new Error(OFFLINE);
    }
    if (!res.ok) throw new Error(await errorText(res));

    const reader = res.body.getReader(), dec = new TextDecoder();
    let buf = '', result = null;
    try {
      for (;;) {
        const { value, done } = await reader.read();
        if (done) break;
        buf += dec.decode(value, { stream: true }).replace(/\r\n/g, '\n');
        let i;
        while ((i = buf.indexOf('\n\n')) >= 0) {
          const block = buf.slice(0, i); buf = buf.slice(i + 2);
          let event = 'message', data = '';
          for (const line of block.split('\n')) {
            if (line.startsWith('event:')) event = line.slice(6).trim();
            else if (line.startsWith('data:')) data += line.slice(5).trim();
          }
          if (!data) continue;
          const payload = JSON.parse(data);
          if (event === 'user') onUser && onUser(payload);
          else if (event === 'token') { text += payload.delta; onToken && onToken(payload.delta); }
          else if (event === 'done') result = payload;
          else if (event === 'error') throw new Error(payload.message);
        }
      }
    } catch (e) {
      if (e.name === 'AbortError') return stopped();
      throw e;
    }
    if (!result) { if (signal && signal.aborted) return stopped(); throw new Error('The reply was cut off before it finished.'); }
    return result;
  }

  window.SwitchboardAPI = {
    providers: {
      list: () => req('GET', '/providers'),
      create: data => req('POST', '/providers', data),
      update: (id, data) => req('PATCH', `/providers/${id}`, data),
      remove: id => req('DELETE', `/providers/${id}`),
      models: id => req('GET', `/providers/${id}/models`),
    },
    chats: {
      list: () => req('GET', '/chats'),
      get: id => req('GET', `/chats/${id}`),
      create: data => req('POST', '/chats', data),
      update: (id, patch) => req('PATCH', `/chats/${id}`, patch),
      remove: id => req('DELETE', `/chats/${id}`),
      restore: id => req('POST', `/chats/${id}/restore`),
      send,
    },
  };
})();
