/* The mock's replies: plain text, or a scripted tool call when the message has a trigger word. */
import { MOCK_LATENCY, type ToolGroup } from '../../config';
import type { Artifact, Part, SendCallbacks, ToolArgs, ToolPart } from '../../types';
import { artifacts, replies, SAMPLE, searchResults, skills, uid, type MockChat } from './data';
import { SAMPLE_REPORT } from './report';
import { sleep } from './helpers';

let replyIndex = 0;
const STOP = Symbol('stopped');

interface ReplyInput {
  chat: MockChat;
  prompt: string;
  /** How many images came with the message. */
  images: number;
  /** The user's answer, when this reply continues after an ask_user question. */
  answered: string | null;
  tools: Set<ToolGroup>;
  cb: SendCallbacks;
}

/** Streams a reply through the callbacks and returns its parts. Stops quietly when aborted. */
export async function scriptedReply({ chat, prompt, images, answered, tools, cb }: ReplyInput): Promise<Part[]> {
  const parts: Part[] = [];
  const alive = () => {
    if (cb.signal?.aborted) throw STOP;
  };
  const say = async (text: string) => {
    for (const w of text.match(/\S+\s*|\s+/g) || []) {
      alive();
      const last = parts.at(-1);
      if (last?.type === 'text') last.text += w;
      else parts.push({ type: 'text', text: w });
      cb.onToken?.(w);
      await sleep(MOCK_LATENCY.word[0] + Math.random() * MOCK_LATENCY.word[1]);
    }
  };
  const tool = async (name: string, args: ToolArgs, run: () => Partial<ToolPart>, writeMs: number = MOCK_LATENCY.toolWrite) => {
    const id = 'call_' + uid();
    alive();
    cb.onToolStart?.({ id, name });
    await sleep(writeMs);
    alive();
    cb.onToolRun?.({ id, name, args });
    await sleep(MOCK_LATENCY.toolRun);
    alive();
    const part: ToolPart = { type: 'tool', id, name, args, status: 'done', result: null };
    try {
      Object.assign(part, run());
    } catch (e) {
      part.status = 'error';
      part.error = (e as Error).message;
    }
    parts.push(part);
    cb.onToolEnd?.(structuredClone(part));
    return part;
  };
  const create = (title: string, kind: string, language: string, body: string) => () => {
    const t = Date.now();
    const a: Artifact = { id: 'art_' + uid(), chatId: chat.id, title, kind, language, content: body, version: 1, createdAt: t, updatedAt: t };
    artifacts.push(a);
    cb.onArtifact?.(structuredClone(a));
    return { result: { artifactId: a.id, title, kind, language, version: 1 } };
  };
  const has = (w: string) => new RegExp(w.replace('/', '\\/') + '(?![a-z])').test(prompt);

  try {
    if (answered !== null) {
      await say(answered ? `Got it: **${answered}**. I'll go with that.\n\nThat keeps the first version small, and nothing here is hard to change later.` : 'No problem, I’ll pick a sensible default and keep going.');
    } else if (tools.has('artifacts') && has('/artifact')) {
      await say('Here is a small page you can try.\n\n');
      await tool('create_artifact', { title: 'Tide table', kind: 'html', content: SAMPLE.html }, create('Tide table', 'html', '', SAMPLE.html), 1300);
      await say('The button cycles through the day’s high tides. Say `/update` and I’ll revise it.');
    } else if (tools.has('artifacts') && has('/code')) {
      await say('A retry helper with exponential backoff:\n\n');
      await tool('create_artifact', { title: 'retry.py', kind: 'code', language: 'python', content: SAMPLE.code }, create('retry.py', 'code', 'python', SAMPLE.code), 1100);
      await say('It waits 0.5 s, 1 s, then 2 s before giving up.');
    } else if (tools.has('artifacts') && has('/svg')) {
      await tool('create_artifact', { title: 'Sailboat', kind: 'svg', content: SAMPLE.svg }, create('Sailboat', 'svg', '', SAMPLE.svg), 1000);
      await say('A flat sailboat at sunset, drawn as SVG.');
    } else if (tools.has('artifacts') && has('/md')) {
      await tool('create_artifact', { title: 'Siwa in four days', kind: 'markdown', content: SAMPLE.markdown }, create('Siwa in four days', 'markdown', '', SAMPLE.markdown), 900);
      await say('The plan is written up as a document.');
    } else if (tools.has('artifacts') && has('/report')) {
      await say('I’ll write that up as a report.\n\n');
      const title = 'Summer punctuality review';
      await tool('create_report', { title, content: SAMPLE_REPORT }, create(title, 'report', '', SAMPLE_REPORT), 1800);
      await say('The report is in the panel: four pages, with the route table, the monthly chart and the August log. **Download PDF** saves it.');
    } else if (has('/skill')) {
      // offered whenever a skill is on, whatever the composer's tools are set to
      const name = /\/skill\s+([\w-]+)/.exec(prompt)?.[1] ?? skills.find((s) => s.enabled)?.name ?? 'weekly-report';
      const part = await tool('use_skill', { name }, () => {
        if (!skills.some((s) => s.name === name && s.enabled)) throw new Error(`There is no enabled skill called ${name}.`);
        return { result: { name } };
      }, 300);
      await say(part.status === 'done' ? `I’ve read the **${name}** skill and will follow it from here.` : 'I’ll carry on without it.');
    } else if (tools.has('artifacts') && has('/update')) {
      const a = artifacts.filter((x) => x.chatId === chat.id).at(-1);
      await say('Revising the most recent one.\n\n');
      await tool('update_artifact', { id: a ? a.id : 'missing', content: '…' }, () => {
        if (!a) throw new Error('There is no artifact in this chat to update.');
        const stamp = new Date().toLocaleTimeString();
        a.content =
          a.kind === 'html' ? a.content.replace(/<h1>[^<]*<\/h1>/, `<h1>Tide table, revised ${stamp}</h1>`)
          : a.kind === 'svg' ? a.content.replace('#f0a75a', '#d6453d')
          : a.kind === 'markdown' ? a.content + `\nRevised at ${stamp}.\n`
          : a.kind === 'report' ? a.content.replace(/<div class="meta">/, `<div class="meta">Revised ${stamp}. `)
          : a.content + `\n# revised at ${stamp}\n`;
        a.version++;
        a.updatedAt = Date.now();
        cb.onArtifact?.(structuredClone(a));
        return { result: { artifactId: a.id, title: a.title, kind: a.kind, language: a.language, version: a.version } };
      }, 1000);
      await say(a ? `That is version ${a.version}.` : 'Nothing to revise yet. Ask for an `/artifact` first.');
    } else if (tools.has('ask') && (has('/askmany') || has('/ask'))) {
      const many = has('/askmany');
      await say('Before I start, one thing to settle.\n\n');
      await tool('ask_user', many
        ? { question: 'Which of these should the first version include?', multiple: true, options: [{ label: 'Search', description: 'Full-text search over chat titles and messages' }, { label: 'Export', description: 'Download a chat as Markdown' }, { label: 'Keyboard shortcuts' }, { label: 'Pinned chats', description: 'Keep a few chats at the top of the list' }] }
        : { question: 'Where should the data live?', multiple: false, options: [{ label: 'SQLite', description: 'One file next to the app, no server to run' }, { label: 'Postgres', description: 'A separate server, better for several writers' }, { label: 'Flat JSON files' }] },
        () => ({ status: 'pending', result: null }), 500);
    } else if (tools.has('web') && has('/search')) {
      await say('Let me look that up.\n\n');
      await tool('web_search', { query: 'sqlite wal mode concurrent writes' }, () => ({ result: { query: 'sqlite wal mode concurrent writes', results: searchResults('SQLite WAL') } }));
      await tool('fetch_page', { url: 'https://www.sqlite.org/wal.html' }, () => ({ result: { url: 'https://www.sqlite.org/wal.html', title: 'Write-Ahead Logging', chars: 18240 } }));
      await say('In WAL mode one writer appends to the log while readers keep using the last committed snapshot, so **reads never wait on the writer**. Writers still take turns.');
    } else if (tools.has('web') && has('/fetch')) {
      await tool('fetch_page', { url: 'https://www.sqlite.org/whentouse.html' }, () => ({ result: { url: 'https://www.sqlite.org/whentouse.html', title: 'Appropriate uses for SQLite', chars: 12804 } }));
      await say('The page argues that SQLite competes with `fopen()`, not with client/server databases.');
    } else if (tools.has('web') && has('/fail')) {
      await tool('web_search', { query: 'tide times alexandria' }, () => {
        throw new Error('Brave Search rejected the API key. Check it in Settings, Web search.');
      });
      await say('I couldn’t search just now, so this is from memory: Alexandria’s tides are small, usually under half a metre.');
    } else {
      if (images) await say(`I can see the ${images === 1 ? 'image' : `${images} images`} you sent.\n\n`);
      await say(replies[replyIndex++ % replies.length]);
    }
  } catch (e) {
    if (e !== STOP) throw e;
  }
  return parts;
}
