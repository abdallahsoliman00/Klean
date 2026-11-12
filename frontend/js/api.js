/* Klean API client — the only file in the frontend that knows about the backend.
   Point it at another backend by setting window.KLEAN_API = "http://host:port" before this script. */
(() => {
  const ROOT = (window.KLEAN_API || "").replace(/\/+$/, "") + "/api";
  const OFFLINE =
    "Can't reach the Klean backend. Check that it's running.";

  async function req(method, path, body) {
    let res;
    try {
      res = await fetch(ROOT + path, {
        method,
        headers: body ? { "Content-Type": "application/json" } : {},
        body: body ? JSON.stringify(body) : undefined,
      });
    } catch {
      throw new Error(OFFLINE);
    }
    if (!res.ok) throw await httpError(res);
    return res.status === 204 ? null : res.json();
  }
  /* Errors from the backend carry the HTTP status, so callers can tell "already gone" from a real failure. */
  async function httpError(res) {
    const e = new Error(await errorText(res));
    e.status = res.status;
    return e;
  }
  async function errorText(res) {
    try {
      const j = await res.json();
      if (typeof j.detail === "string") return j.detail;
      if (Array.isArray(j.detail)) return j.detail.map((d) => d.msg).join("; ");
    } catch {}
    return `Request failed (${res.status}).`;
  }

  /* Reads the Server-Sent Events stream from POST /chats/:id/messages.
     body may carry `tools` (enabled groups) and `answer` ({callId, text}, replying to an ask_user call).
     Callbacks: onUser, onAnswered, onToken, onToolStart, onToolRun, onToolEnd, onArtifact.
     Parts are collected here too, so a reply stopped locally keeps what already arrived. */
  async function send(chatId, body, cb = {}) {
    const { signal } = cb;
    const parts = [];
    const addText = (delta) => {
      const last = parts[parts.length - 1];
      if (last && last.type === "text") last.text += delta;
      else parts.push({ type: "text", text: delta });
    };
    const stopped = () => {
      // tool calls that never finished are not saved by the backend either
      const kept = parts.filter((p) => p.type === "text" || p.status);
      return {
        message: {
          id: "local-" + Date.now(),
          role: "assistant",
          model: body.model,
          content: kept.filter((p) => p.type === "text").map((p) => p.text).join(""),
          parts: kept,
          stopped: true,
        },
        title: null,
      };
    };
    let res;
    try {
      res = await fetch(`${ROOT}/chats/${chatId}/messages`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
        signal,
      });
    } catch (e) {
      if (e.name === "AbortError") return stopped();
      throw new Error(OFFLINE);
    }
    if (!res.ok) throw await httpError(res);

    const handle = (event, payload) => {
      switch (event) {
        case "user":
          cb.onUser && cb.onUser(payload);
          break;
        case "answered":
          cb.onAnswered && cb.onAnswered(payload);
          break;
        case "token":
          addText(payload.delta);
          cb.onToken && cb.onToken(payload.delta);
          break;
        case "tool_start":
          parts.push({ type: "tool", id: payload.id, name: payload.name, args: null, result: null });
          cb.onToolStart && cb.onToolStart(payload);
          break;
        case "tool_run": {
          const p = parts.find((x) => x.type === "tool" && x.id === payload.id);
          if (p) p.args = payload.args;
          else parts.push({ type: "tool", id: payload.id, name: payload.name, args: payload.args, result: null });
          cb.onToolRun && cb.onToolRun(payload);
          break;
        }
        case "tool_end": {
          const i = parts.findIndex((x) => x.type === "tool" && x.id === payload.part.id);
          if (i > -1) parts[i] = payload.part;
          else parts.push(payload.part);
          cb.onToolEnd && cb.onToolEnd(payload.part);
          break;
        }
        case "artifact":
          cb.onArtifact && cb.onArtifact(payload.artifact);
          break;
        case "done":
          return payload;
        case "error":
          throw new Error(payload.message);
      }
      return null;
    };

    const reader = res.body.getReader(),
      dec = new TextDecoder();
    let buf = "",
      result = null;
    try {
      for (;;) {
        const { value, done } = await reader.read();
        if (done) break;
        buf += dec.decode(value, { stream: true }).replace(/\r\n/g, "\n");
        let i;
        while ((i = buf.indexOf("\n\n")) >= 0) {
          const block = buf.slice(0, i);
          buf = buf.slice(i + 2);
          let event = "message",
            data = "";
          for (const line of block.split("\n")) {
            if (line.startsWith("event:")) event = line.slice(6).trim();
            else if (line.startsWith("data:")) data += line.slice(5).trim();
          }
          if (!data) continue;
          result = handle(event, JSON.parse(data)) || result;
        }
      }
    } catch (e) {
      if (e.name === "AbortError") return stopped();
      throw e;
    }
    if (!result) {
      if (signal && signal.aborted) return stopped();
      throw new Error("The reply was cut off before it finished.");
    }
    return result;
  }

  window.KleanAPI = {
    providers: {
      list: () => req("GET", "/providers"),
      create: (data) => req("POST", "/providers", data),
      update: (id, data) => req("PATCH", `/providers/${id}`, data),
      remove: (id) => req("DELETE", `/providers/${id}`),
      models: (id) => req("GET", `/providers/${id}/models`),
    },
    chats: {
      list: () => req("GET", "/chats"),
      get: (id) => req("GET", `/chats/${id}`),
      create: (data) => req("POST", "/chats", data),
      update: (id, patch) => req("PATCH", `/chats/${id}`, patch),
      remove: (id) => req("DELETE", `/chats/${id}`),
      restore: (id) => req("POST", `/chats/${id}/restore`),
      send,
    },
    artifacts: {
      remove: (id) => req("DELETE", `/artifacts/${id}`),
    },
    settings: {
      get: () => req("GET", "/settings"),
      update: (patch) => req("PATCH", "/settings", patch),
    },
  };
})();
