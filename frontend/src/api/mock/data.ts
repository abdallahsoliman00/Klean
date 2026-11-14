/* Seed data for the in-memory mock backend. */
import type { Artifact, Chat, Preset, SearchResult } from '../../types';

export const uid = () => Math.random().toString(36).slice(2, 10);
const now = Date.now(),
  H = 3600e3,
  D = 24 * H;

export interface MockProvider {
  id: string;
  name: string;
  baseUrl: string;
  hasKey: boolean;
  models: string[] | null;
  offline?: boolean;
}

export const providers: MockProvider[] = [
  { id: 'p1', name: 'Ollama', baseUrl: 'http://localhost:11434/v1', hasKey: false,
    models: ['llama3.1:8b', 'qwen2.5-coder:14b', 'mistral-nemo:12b', 'gemma2:9b'] },
  { id: 'p2', name: 'OpenRouter', baseUrl: 'https://openrouter.ai/api/v1', hasKey: true,
    models: ['anthropic/claude-sonnet-4.5', 'openai/gpt-4.1-mini', 'meta-llama/llama-3.3-70b-instruct', 'deepseek/deepseek-chat', 'google/gemini-2.5-flash'] },
  { id: 'p3', name: 'LM Studio', baseUrl: 'http://localhost:1234/v1', hasKey: false, models: null, offline: true },
];

/* The second one names a model its provider no longer has, to show what applying it does. */
export const presets: Preset[] = [
  { id: 'pr1', name: 'Careful coder', providerId: 'p1', model: 'qwen2.5-coder:14b', systemPrompt: 'You are a careful senior engineer. Prefer small, tested changes.', temperature: 0.2, contextWindow: 131072 },
  { id: 'pr2', name: 'Brainstorm', providerId: 'p1', model: 'llama2:7b', systemPrompt: 'Offer many ideas before judging any of them.', temperature: 1.2, contextWindow: 32768 },
];

export const SAMPLE = {
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
</script>
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

export const artifacts: Artifact[] = [
  { id: 'a1', chatId: 'c0', title: 'Tide table', kind: 'html', language: '', content: SAMPLE.html, version: 1, createdAt: now - H, updatedAt: now - H },
  { id: 'a2', chatId: 'c0', title: 'retry.py', kind: 'code', language: 'python', content: SAMPLE.code, version: 2, createdAt: now - H + 5e4, updatedAt: now - H + 9e4 },
];

export const searchResults = (q: string): SearchResult[] => [
  { title: 'Write-Ahead Logging', url: 'https://www.sqlite.org/wal.html', snippet: 'The WAL journal mode lets readers and a writer work at the same time. Changes are appended to a separate log and checkpointed back into the database later.' },
  { title: `${q}: a practical guide`, url: 'https://fly.io/blog/sqlite-internals-wal/', snippet: 'How the write-ahead log works on disk, what the shared-memory index is for, and when checkpoints run.' },
  { title: 'Appropriate uses for SQLite', url: 'https://www.sqlite.org/whentouse.html', snippet: 'SQLite works well for most low to medium traffic sites, embedded devices and application file formats.' },
  { title: 'A link with an unsafe scheme is shown as plain text', url: 'javascript:alert(1)', snippet: 'The UI must not turn this into a clickable link.' },
];

/* Mock chats carry no artifacts field; get() joins them in. */
export type MockChat = Omit<Chat, 'artifacts'>;
export const chats: MockChat[] = [
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

export const replies: string[] = [
  "Good question. Here's how I'd think about it.\n\nThe short version: start with the simplest thing that works, measure it, and only add complexity where the numbers tell you to.\n\n### A quick sketch\n\n```js\nasync function* stream(res) {\n  const reader = res.body.getReader();\n  const dec = new TextDecoder();\n  for (;;) {\n    const { value, done } = await reader.read();\n    if (done) return;\n    yield dec.decode(value, { stream: true });\n  }\n}\n```\n\nA few notes on that:\n\n- It yields text as soon as bytes arrive, so the UI can paint each chunk.\n- `stream: true` keeps multi-byte characters intact across chunk boundaries.\n- Cancelling the fetch with an `AbortController` ends the loop cleanly.\n\nWant me to go further into error handling or back-pressure?",
  "Sure. There are three common approaches, each with a trade-off:\n\n| Approach | Good for | Watch out for |\n|---|---|---|\n| Sliding window | Long casual chats | Forgets early details |\n| Summarise older turns | Long working sessions | Summary drift |\n| Retrieval | Large reference material | Extra moving parts |\n\nFor a local chat app, a **sliding window that always keeps the system prompt** is the right default. It's predictable and needs no extra model calls.",
  "Here's a concise answer.\n\nIt depends mostly on how often the data changes. If it changes rarely, cache it aggressively and invalidate on write. If it changes constantly, skip the cache and make the query itself cheap with the right index.\n\n> Measure before and after. Caches that aren't measured tend to hide bugs rather than fix latency.",
];
