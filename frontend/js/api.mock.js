/* In-memory stand-in for the backend, for working on the UI alone.
   Use it by loading this file instead of js/api.js in index.html.
   Tool calls: put one of these words in a message to see the matching tool UI
   (same words as the backend's fake provider, plus a few extras marked *):
     /artifact  /code  /svg  /md*  /update  /ask  /askmany*  /search  /fetch  /fail*  */
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
  const GROUPS = ['artifacts', 'ask', 'web'];
  const settings = { searchProvider: 'duckduckgo', hasSearchKey: false };
  const SAMPLE = {
    html: `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Tide table</title>
<style>
  body { font: 16px/1.5 system-ui, sans-serif; margin: 0; padding: 32px; color: #18212b; background: #f4f7fa; }
  h1 { font-size: 22px; margin: 0 0 4px; }
  p { color: #5d6673; margin: 0 0 20px; }
  button { font: inherit; padding: 8px 14px; border-radius: 8px; border: 1px solid #c9d2dc; background: #fff; cursor: pointer; }
  output { display: inline-block; margin-left: 12px; font-variant-numeric: tabular-nums; }
</style>
</head>
<body>
  <h1>Tide table</h1>
  <p>Scripts run inside the preview sandbox.</p>
  <button id="b">Next high tide</button><output id="o">06:42</output>
  <script>
    const times = ['06:42', '12:58', '19:07', '01:21'];
    let i = 0;
    document.getElementById('b').onclick = () => { i = (i + 1) % times.length; document.getElementById('o').textContent = times[i]; };
  <\/script>
</body>
</html>`,
    svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 240 160" width="480" height="320">
  <rect width="240" height="160" fill="#eef3f8"/>
  <circle cx="176" cy="48" r="22" fill="#f0a75a"/>
  <path d="M0 120 Q30 100 60 120 T120 120 T180 120 T240 120 V160 H0Z" fill="#2b59c3"/>
  <path d="M0 134 Q30 118 60 134 T120 134 T180 134 T240 134 V160 H0Z" fill="#1c3f8f"/>
  <path d="M92 104 L120 44 L148 104Z" fill="#ffffff" stroke="#1a1f27" stroke-width="2" stroke-linejoin="round"/>
  <path d="M84 108 H156 L146 122 H94Z" fill="#1a1f27"/>
</svg>`,
    code: `import time
import urllib.request


def fetch(url, attempts=4, base_delay=0.5):
    """GET a URL, backing off between failures."""
    for n in range(attempts):
        try:
            with urllib.request.urlopen(url, timeout=10) as res:
                return res.read()
        except OSError:
            if n == attempts - 1:
                raise
            time.sleep(base_delay * 2 ** n)  # 0.5s, 1s, 2s


if __name__ == "__main__":
    print(len(fetch("https://example.com")))
`,
    markdown: `# Siwa in four days

A short plan that keeps the driving to the cool hours.

## Day by day

1. **Arrive**: check in near Shali, walk the old fortress at dusk.
2. **Springs**: Cleopatra's Spring early, then the salt lakes.
3. **Desert**: afternoon in the Great Sand Sea, stay for the stars.
4. **Slow morning**: date market, then the road back.

| Pack | Why |
|---|---|
| Fleece | Nights drop below 10 °C |
| Scarf | Wind and sand |

> Carry cash. Card machines are rare past Marsa Matruh.
`,
  };
  const artifacts = [
    { id: 'a1', chatId: 'c0', title: 'Tide table', kind: 'html', language: '', content: SAMPLE.html, version: 1, createdAt: now - H, updatedAt: now - H },
    { id: 'a2', chatId: 'c0', title: 'retry.py', kind: 'code', language: 'python', content: SAMPLE.code, version: 2, createdAt: now - H + 5e4, updatedAt: now - H + 9e4 },
  ];
  const searchResults = q => [
    { title: 'Write-Ahead Logging', url: 'https://www.sqlite.org/wal.html', snippet: 'The WAL journal mode lets readers and a writer work at the same time. Changes are appended to a separate log and checkpointed back into the database later.' },
    { title: `${q}: a practical guide`, url: 'https://fly.io/blog/sqlite-internals-wal/', snippet: 'How the write-ahead log works on disk, what the shared-memory index is for, and when checkpoints run.' },
    { title: 'Appropriate uses for SQLite', url: 'https://www.sqlite.org/whentouse.html', snippet: 'SQLite works well for most low to medium traffic sites, embedded devices and application file formats.' },
    { title: 'A link with an unsafe scheme is shown as plain text', url: 'javascript:alert(1)', snippet: 'The UI must not turn this into a clickable link.' },
  ];
  const chats = [
    { id: 'c0', title: 'Tide table widget', providerId: 'p1', model: 'qwen2.5-coder:14b', updatedAt: now - H, systemPrompt: '', temperature: 0.4, contextWindow: 8192,
      messages: [
        { id: uid(), role: 'user', content: 'Look up how tide tables are usually laid out, then build me a tiny widget and a retry helper for the fetch.', parts: null, createdAt: now - H - 9e4 },
        { id: uid(), role: 'assistant', model: 'qwen2.5-coder:14b', createdAt: now - H,
          content: "I'll check the usual layout first.\n\nHere is the widget, and the helper it will need once it loads real data.\n\nBoth are in the panel on the right. The helper is on its second version: I added the timeout.",
          parts: [
            { type: 'text', text: "I'll check the usual layout first.\n\n" },
            { type: 'tool', id: 'call_s1', name: 'web_search', args: { query: 'tide table layout high low times' }, status: 'done', result: { query: 'tide table layout high low times', results: searchResults('Tide tables').slice(0, 3) } },
            { type: 'tool', id: 'call_f1', name: 'fetch_page', args: { url: 'https://www.sqlite.org/wal.html' }, status: 'done', result: { url: 'https://www.sqlite.org/wal.html', title: 'Write-Ahead Logging', chars: 18240 } },
            { type: 'text', text: 'Here is the widget, and the helper it will need once it loads real data.\n\n' },
            { type: 'tool', id: 'call_a1', name: 'create_artifact', args: { title: 'Tide table', kind: 'html' }, status: 'done', result: { artifactId: 'a1', title: 'Tide table', kind: 'html', language: '', version: 1 } },
            { type: 'tool', id: 'call_a2', name: 'create_artifact', args: { title: 'retry.py', kind: 'code', language: 'python' }, status: 'done', result: { artifactId: 'a2', title: 'retry.py', kind: 'code', language: 'python', version: 1 } },
            { type: 'tool', id: 'call_a3', name: 'update_artifact', args: { id: 'a2' }, status: 'done', result: { artifactId: 'a2', title: 'retry.py', kind: 'code', language: 'python', version: 2 } },
            { type: 'tool', id: 'call_a4', name: 'create_artifact', args: { title: 'Old draft', kind: 'markdown' }, status: 'done', result: { artifactId: 'gone', title: 'Old draft', kind: 'markdown', language: '', version: 1 } },
            { type: 'tool', id: 'call_e1', name: 'fetch_page', args: { url: 'https://tides.example/api' }, status: 'error', result: null, error: 'The page took longer than 15 seconds to answer.' },
            { type: 'text', text: 'Both are in the panel on the right. The helper is on its second version: I added the timeout.' },
          ] },
      ] },
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
  /* editing or regenerating also removes the artifacts those messages created */
  const dropArtifacts = msgs => { for (const m of msgs) for (const p of m.parts || []) if (p.type === 'tool' && p.name === 'create_artifact' && p.result) { const i = artifacts.findIndex(a => a.id === p.result.artifactId); if (i > -1) artifacts.splice(i, 1); } };
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
      async get(id) { await sleep(60); const c = chats.find(x => x.id === id); return c ? structuredClone({ ...c, artifacts: artifacts.filter(a => a.chatId === id) }) : null; },
      async create(data) { const c = { id: uid(), title: 'New chat', updatedAt: Date.now(), systemPrompt: '', temperature: 0.7, contextWindow: 8192, messages: [], ...data }; chats.push(c); return structuredClone({ ...c, artifacts: [] }); },
      async update(id, patch) { const c = chats.find(x => x.id === id); Object.assign(c, patch); return strip(c); },
      async remove(id) { const i = chats.findIndex(x => x.id === id); if (i > -1) trash[id] = chats.splice(i, 1)[0]; },
      async restore(id) { if (trash[id]) { chats.push(trash[id]); delete trash[id]; } },
      /* send: appends (or edits / regenerates / answers a question) and streams the reply.
         body.editMessageId → replace that user message and drop everything after it
         body.regenerate    → drop the last assistant messages and answer again
         body.answer        → {callId, text}: closes a pending ask_user and continues in a new message
         body.tools         → enabled groups; omitted means all */
      async send(id, body, cb = {}) {
        const { content, providerId, model, editMessageId, regenerate, tools, answer } = body, { signal } = cb;
        const c = chats.find(x => x.id === id);
        const on = new Set(tools || GROUPS);
        let prompt = content || '';
        if (answer) {
          let hit = null;
          for (const m of c.messages) for (const p of m.parts || []) if (p.type === 'tool' && p.id === answer.callId && p.status === 'pending') hit = { m, p };
          if (!hit) throw Object.assign(new Error('That question is no longer waiting for an answer.'), { status: 409 });
          Object.assign(hit.p, { status: 'done', result: { answer: answer.text } });
          await sleep(120);
          cb.onAnswered && cb.onAnswered({ messageId: hit.m.id, part: structuredClone(hit.p) });
          prompt = '\u0000answer:' + answer.text;
        } else {
          for (const m of c.messages) for (const p of m.parts || []) if (p.type === 'tool' && p.status === 'pending') Object.assign(p, { status: 'done', result: { answer: '' } });
          if (editMessageId) { const i = c.messages.findIndex(m => m.id === editMessageId); dropArtifacts(c.messages.splice(i)); }
          if (regenerate) { const cut = []; while (c.messages.length && c.messages.at(-1).role === 'assistant') cut.push(c.messages.pop()); dropArtifacts(cut); prompt = c.messages.at(-1)?.content || ''; }
          else { const um = { id: uid(), role: 'user', content, parts: null, createdAt: Date.now() }; c.messages.push(um); cb.onUser && cb.onUser(um); }
        }
        Object.assign(c, { providerId, model, updatedAt: Date.now() });
        let titled = null;
        if (c.title === 'New chat' && content) { titled = content.replace(/\s+/g, ' ').slice(0, 42).trim() + (content.length > 42 ? '…' : ''); c.title = titled; }
        const p = providers.find(x => x.id === providerId);
        await sleep(400 + Math.random() * 300);
        if (!p || p.offline) throw new Error(`${p ? p.name : 'Provider'} didn't respond at ${p ? p.baseUrl : ''}. Start the server or pick another model.`);

        const parts = [];
        const STOP = {};
        const alive = () => { if (signal && signal.aborted) throw STOP; };
        const say = async text => {
          for (const w of text.match(/\S+\s*|\s+/g)) {
            alive();
            const last = parts.at(-1);
            if (last && last.type === 'text') last.text += w; else parts.push({ type: 'text', text: w });
            cb.onToken && cb.onToken(w);
            await sleep(14 + Math.random() * 38);
          }
        };
        const tool = async (name, args, run, writeMs = 350) => {
          const cid = 'call_' + uid();
          alive(); cb.onToolStart && cb.onToolStart({ id: cid, name });
          await sleep(writeMs); alive();
          cb.onToolRun && cb.onToolRun({ id: cid, name, args });
          await sleep(650); alive();
          const part = { type: 'tool', id: cid, name, args, status: 'done', result: null };
          try { Object.assign(part, run()); } catch (e) { part.status = 'error'; part.error = e.message; }
          parts.push(part);
          cb.onToolEnd && cb.onToolEnd(structuredClone(part));
          return part;
        };
        const create = (title, kind, language, body) => () => {
          const t = Date.now();
          const a = { id: 'art_' + uid(), chatId: c.id, title, kind, language: language || '', content: body, version: 1, createdAt: t, updatedAt: t };
          artifacts.push(a); cb.onArtifact && cb.onArtifact(structuredClone(a));
          return { result: { artifactId: a.id, title, kind, language: a.language, version: 1 } };
        };
        const has = w => new RegExp(w.replace('/', '\\/') + '(?![a-z])').test(prompt);
        try {
          if (prompt.startsWith('\u0000answer:')) {
            const a = prompt.slice(8);
            await say(a ? `Got it: **${a}**. I'll go with that.\n\nThat keeps the first version small, and nothing here is hard to change later.` : 'No problem, I’ll pick a sensible default and keep going.');
          } else if (on.has('artifacts') && has('/artifact')) {
            await say('Here is a small page you can try.\n\n');
            await tool('create_artifact', { title: 'Tide table', kind: 'html', content: SAMPLE.html }, create('Tide table', 'html', '', SAMPLE.html), 1300);
            await say('The button cycles through the day’s high tides. Say `/update` and I’ll revise it.');
          } else if (on.has('artifacts') && has('/code')) {
            await say('A retry helper with exponential backoff:\n\n');
            await tool('create_artifact', { title: 'retry.py', kind: 'code', language: 'python', content: SAMPLE.code }, create('retry.py', 'code', 'python', SAMPLE.code), 1100);
            await say('It waits 0.5 s, 1 s, then 2 s before giving up.');
          } else if (on.has('artifacts') && has('/svg')) {
            await tool('create_artifact', { title: 'Sailboat', kind: 'svg', content: SAMPLE.svg }, create('Sailboat', 'svg', '', SAMPLE.svg), 1000);
            await say('A flat sailboat at sunset, drawn as SVG.');
          } else if (on.has('artifacts') && has('/md')) {
            await tool('create_artifact', { title: 'Siwa in four days', kind: 'markdown', content: SAMPLE.markdown }, create('Siwa in four days', 'markdown', '', SAMPLE.markdown), 900);
            await say('The plan is written up as a document.');
          } else if (on.has('artifacts') && has('/update')) {
            const a = artifacts.filter(x => x.chatId === c.id).at(-1);
            await say('Revising the most recent one.\n\n');
            await tool('update_artifact', { id: a ? a.id : 'missing', content: '…' }, () => {
              if (!a) throw new Error('There is no artifact in this chat to update.');
              const stamp = new Date().toLocaleTimeString();
              a.content = a.kind === 'html' ? a.content.replace(/<h1>[^<]*<\/h1>/, `<h1>Tide table, revised ${stamp}</h1>`) : a.kind === 'svg' ? a.content.replace('#f0a75a', '#d6453d') : a.kind === 'markdown' ? a.content + `\nRevised at ${stamp}.\n` : a.content + `\n# revised at ${stamp}\n`;
              a.version++; a.updatedAt = Date.now();
              cb.onArtifact && cb.onArtifact(structuredClone(a));
              return { result: { artifactId: a.id, title: a.title, kind: a.kind, language: a.language, version: a.version } };
            }, 1000);
            await say(a ? `That is version ${a.version}.` : 'Nothing to revise yet. Ask for an `/artifact` first.');
          } else if (on.has('ask') && (has('/askmany') || has('/ask'))) {
            const many = has('/askmany');
            await say('Before I start, one thing to settle.\n\n');
            await tool('ask_user', many
              ? { question: 'Which of these should the first version include?', multiple: true, options: [{ label: 'Search', description: 'Full-text search over chat titles and messages' }, { label: 'Export', description: 'Download a chat as Markdown' }, { label: 'Keyboard shortcuts' }, { label: 'Pinned chats', description: 'Keep a few chats at the top of the list' }] }
              : { question: 'Where should the data live?', multiple: false, options: [{ label: 'SQLite', description: 'One file next to the app, no server to run' }, { label: 'Postgres', description: 'A separate server, better for several writers' }, { label: 'Flat JSON files' }] },
              () => ({ status: 'pending', result: null }), 500);
          } else if (on.has('web') && has('/search')) {
            await say('Let me look that up.\n\n');
            await tool('web_search', { query: 'sqlite wal mode concurrent writes' }, () => ({ result: { query: 'sqlite wal mode concurrent writes', results: searchResults('SQLite WAL') } }));
            await tool('fetch_page', { url: 'https://www.sqlite.org/wal.html' }, () => ({ result: { url: 'https://www.sqlite.org/wal.html', title: 'Write-Ahead Logging', chars: 18240 } }));
            await say('In WAL mode one writer appends to the log while readers keep using the last committed snapshot, so **reads never wait on the writer**. Writers still take turns.');
          } else if (on.has('web') && has('/fetch')) {
            await tool('fetch_page', { url: 'https://www.sqlite.org/whentouse.html' }, () => ({ result: { url: 'https://www.sqlite.org/whentouse.html', title: 'Appropriate uses for SQLite', chars: 12804 } }));
            await say('The page argues that SQLite competes with `fopen()`, not with client/server databases.');
          } else if (on.has('web') && has('/fail')) {
            await tool('web_search', { query: 'tide times alexandria' }, () => { throw new Error('Brave Search rejected the API key. Check it in Settings, Web search.'); });
            await say('I couldn’t search just now, so this is from memory: Alexandria’s tides are small, usually under half a metre.');
          } else {
            await say(replies[ri++ % replies.length]);
          }
        } catch (e) { if (e !== STOP) throw e; }
        const stopped = !!(signal && signal.aborted);
        const am = { id: uid(), role: 'assistant', model, content: parts.filter(x => x.type === 'text').map(x => x.text).join(''), parts, stopped, createdAt: Date.now() };
        if (parts.length) c.messages.push(am);
        return structuredClone({ message: am, title: titled });
      },
    },
    artifacts: {
      async remove(id) { await sleep(60); const i = artifacts.findIndex(a => a.id === id); if (i > -1) artifacts.splice(i, 1); return null; },
    },
    settings: {
      async get() { await sleep(60); return { ...settings }; },
      async update(patch) {
        await sleep(120);
        if (patch.searchProvider !== undefined) { if (!['duckduckgo', 'brave', 'tavily'].includes(patch.searchProvider)) throw new Error('Unknown search provider.'); settings.searchProvider = patch.searchProvider; }
        if (patch.searchKey !== undefined) settings.hasSearchKey = !!patch.searchKey;
        return { ...settings };
      },
    },
  };
})();
window.KleanAPI = MockAPI;
