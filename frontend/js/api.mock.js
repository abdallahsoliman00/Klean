/* In-memory stand-in for the backend, for working on the UI alone.
   Use it by loading this file instead of js/api.js in index.html. */
/* =====================================================================
   API LAYER — the only place the UI touches data.
   This mock keeps everything in memory. The real build swaps this object
   for one that calls the backend over HTTP (same method names and shapes).
   ===================================================================== */
const MockAPI = (() => {
  const sleep = ms => new Promise(r => setTimeout(r, ms));
  const uid = () => Math.random().toString(36).slice(2, 10);
  const now = Date.now(), H = 3600e3, D = 24 * H;
  const providers = [
    { id: 'p1', name: 'Ollama', baseUrl: 'http://localhost:11434/v1', hasKey: false,
      models: ['llama3.1:8b', 'qwen2.5-coder:14b', 'mistral-nemo:12b', 'gemma2:9b'] },
    { id: 'p2', name: 'OpenRouter', baseUrl: 'https://openrouter.ai/api/v1', hasKey: true,
      models: ['anthropic/claude-sonnet-4.5', 'openai/gpt-4.1-mini', 'meta-llama/llama-3.3-70b-instruct', 'deepseek/deepseek-chat', 'google/gemini-2.5-flash'] },
    { id: 'p3', name: 'LM Studio', baseUrl: 'http://localhost:1234/v1', hasKey: false, models: null, offline: true },
  ];
  const chats = [
    { id: 'c1', title: 'Binary search edge cases', providerId: 'p1', model: 'qwen2.5-coder:14b', updatedAt: now - 2 * H, systemPrompt: '', temperature: 0.3, contextWindow: 8192,
      messages: [
        { id: uid(), role: 'user', content: 'Why does my binary search loop forever when the target is missing?' },
        { id: uid(), role: 'assistant', model: 'qwen2.5-coder:14b', content: "The usual cause is the update step. If you write `lo = mid` instead of `lo = mid + 1`, then once `hi - lo == 1` the midpoint rounds down to `lo` and nothing changes.\n\nHere is a version that always shrinks the range:\n\n```python\ndef search(xs, target):\n    lo, hi = 0, len(xs)  # hi is exclusive\n    while lo < hi:\n        mid = (lo + hi) // 2\n        if xs[mid] < target:\n            lo = mid + 1\n        else:\n            hi = mid\n    return lo if lo < len(xs) and xs[lo] == target else -1\n```\n\nTwo things keep it safe:\n\n1. `hi` is **exclusive**, so the loop condition is `lo < hi`.\n2. Every branch moves a bound past `mid`, so the window strictly shrinks." },
      ] },
    { id: 'c2', title: 'Trip packing list for Siwa', providerId: 'p2', model: 'openai/gpt-4.1-mini', updatedAt: now - 1.2 * D, systemPrompt: 'Be practical and brief.', temperature: 0.7, contextWindow: 16384,
      messages: [
        { id: uid(), role: 'user', content: 'Packing list for 4 days in Siwa in November?' },
        { id: uid(), role: 'assistant', model: 'openai/gpt-4.1-mini', content: "Days are warm and nights get cold, so pack in layers.\n\n- Light long sleeves for sun\n- A fleece or down jacket for evenings in the desert\n- Closed shoes for the dunes, sandals for town\n- Swimwear for Cleopatra's Spring\n- Headlamp, sunscreen, and a scarf for wind and sand" },
      ] },
    { id: 'c3', title: 'Explain SQLite WAL mode', providerId: 'p1', model: 'llama3.1:8b', updatedAt: now - 4 * D, systemPrompt: '', temperature: 0.7, contextWindow: 8192,
      messages: [{ id: uid(), role: 'user', content: 'What does WAL mode change in SQLite?' }, { id: uid(), role: 'assistant', model: 'llama3.1:8b', content: 'In WAL mode, writes go to a separate log file instead of overwriting the database in place. Readers keep seeing the last committed state while a writer appends, so **reads and writes no longer block each other**.' }] },
    { id: 'c4', title: 'Regex for Egyptian phone numbers', providerId: 'p2', model: 'deepseek/deepseek-chat', updatedAt: now - 12 * D, systemPrompt: '', temperature: 0.2, contextWindow: 8192, messages: [{ id: uid(), role: 'user', content: 'Regex for Egyptian mobile numbers?' }, { id: uid(), role: 'assistant', model: 'deepseek/deepseek-chat', content: '`^(\\+20|0)?1[0125]\\d{8}$` matches numbers with an optional `+20` or leading `0`, followed by the four mobile prefixes.' }] },
  ];
  const replies = [
    "Good question. Here's how I'd think about it.\n\nThe short version: start with the simplest thing that works, measure it, and only add complexity where the numbers tell you to.\n\n### A quick sketch\n\n```js\nasync function* stream(res) {\n  const reader = res.body.getReader();\n  const dec = new TextDecoder();\n  for (;;) {\n    const { value, done } = await reader.read();\n    if (done) return;\n    yield dec.decode(value, { stream: true });\n  }\n}\n```\n\nA few notes on that:\n\n- It yields text as soon as bytes arrive, so the UI can paint each chunk.\n- `stream: true` keeps multi-byte characters intact across chunk boundaries.\n- Cancelling the fetch with an `AbortController` ends the loop cleanly.\n\nWant me to go further into error handling or back-pressure?",
    "Sure. There are three common approaches, each with a trade-off:\n\n| Approach | Good for | Watch out for |\n|---|---|---|\n| Sliding window | Long casual chats | Forgets early details |\n| Summarise older turns | Long working sessions | Summary drift |\n| Retrieval | Large reference material | Extra moving parts |\n\nFor a local chat app, a **sliding window that always keeps the system prompt** is the right default. It's predictable and needs no extra model calls.",
    "Here's a concise answer.\n\nIt depends mostly on how often the data changes. If it changes rarely, cache it aggressively and invalidate on write. If it changes constantly, skip the cache and make the query itself cheap with the right index.\n\n> Measure before and after. Caches that aren't measured tend to hide bugs rather than fix latency.",
  ];
  let ri = 0;
  const trash = {};
  const strip = c => { const { messages, ...rest } = c; return { ...rest }; };
  const pub = p => ({ id: p.id, name: p.name, baseUrl: p.baseUrl, hasKey: p.hasKey });

  return {
    providers: {
      async list() { await sleep(60); return providers.map(pub); },
      async create(data) { const p = { id: uid(), name: data.name, baseUrl: data.baseUrl, hasKey: !!data.apiKey, models: null, offline: /1234/.test(data.baseUrl) }; providers.push(p); if (!p.offline) p.models = ['local-model-a', 'local-model-b']; return pub(p); },
      async update(id, data) { const p = providers.find(x => x.id === id); Object.assign(p, { name: data.name, baseUrl: data.baseUrl }); if (data.apiKey !== undefined) p.hasKey = !!data.apiKey; return pub(p); },
      async remove(id) { const i = providers.findIndex(x => x.id === id); if (i > -1) providers.splice(i, 1); },
      async models(id) {
        await sleep(450 + Math.random() * 400);
        const p = providers.find(x => x.id === id);
        if (!p) throw new Error('Provider not found.');
        if (p.offline || !p.models) throw new Error(`Couldn't reach ${p.baseUrl}/models. Check that the server is running and the URL is right.`);
        return p.models.map(m => ({ id: m, ownedBy: m.includes('/') ? m.split('/')[0] : p.name.toLowerCase() }));
      },
    },
    chats: {
      async list() { await sleep(40); return chats.map(strip).sort((a, b) => b.updatedAt - a.updatedAt); },
      async get(id) { await sleep(60); const c = chats.find(x => x.id === id); return c ? structuredClone(c) : null; },
      async create(data) { const c = { id: uid(), title: 'New chat', updatedAt: Date.now(), systemPrompt: '', temperature: 0.7, contextWindow: 8192, messages: [], ...data }; chats.push(c); return structuredClone(c); },
      async update(id, patch) { const c = chats.find(x => x.id === id); Object.assign(c, patch); return strip(c); },
      async remove(id) { const i = chats.findIndex(x => x.id === id); if (i > -1) trash[id] = chats.splice(i, 1)[0]; },
      async restore(id) { if (trash[id]) { chats.push(trash[id]); delete trash[id]; } },
      /* send: appends (or edits / regenerates) and streams the reply.
         opts.editMessageId → replace that user message and drop everything after it
         opts.regenerate    → drop the last assistant message and answer again */
      async send(id, { content, providerId, model, editMessageId, regenerate }, { onUser, onToken, signal }) {
        const c = chats.find(x => x.id === id);
        if (editMessageId) { const i = c.messages.findIndex(m => m.id === editMessageId); c.messages.splice(i); }
        if (regenerate) { while (c.messages.length && c.messages.at(-1).role === 'assistant') c.messages.pop(); }
        else { const um = { id: uid(), role: 'user', content }; c.messages.push(um); onUser && onUser(um); }
        Object.assign(c, { providerId, model, updatedAt: Date.now() });
        let titled = null;
        if (c.title === 'New chat' && content) { titled = content.replace(/\s+/g, ' ').slice(0, 42).trim() + (content.length > 42 ? '…' : ''); c.title = titled; }
        const p = providers.find(x => x.id === providerId);
        await sleep(500 + Math.random() * 300);
        if (!p || p.offline) throw new Error(`${p ? p.name : 'Provider'} didn't respond at ${p ? p.baseUrl : ''}. Start the server or pick another model.`);
        const text = replies[ri++ % replies.length];
        const parts = text.match(/\S+\s*|\s+/g);
        let out = '';
        for (const part of parts) {
          if (signal.aborted) break;
          out += part; onToken(part);
          await sleep(14 + Math.random() * 38);
        }
        const am = { id: uid(), role: 'assistant', model, content: out, stopped: signal.aborted };
        c.messages.push(am);
        return { message: am, title: titled };
      },
    },
  };
})();
window.SwitchboardAPI = MockAPI;
