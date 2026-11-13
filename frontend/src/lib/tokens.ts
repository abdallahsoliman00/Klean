/* Rough token counting for the context ring and the chat settings stats. */
import { CHARS_PER_TOKEN, KFMT_DECIMALS_BELOW, DEFAULT_CONTEXT_WINDOW, HISTORY_BUDGET_SHARE, MESSAGE_OVERHEAD_TOKENS } from '../config';
import type { Chat } from '../types';

export const estimateTokens = (t?: string | null) => Math.ceil((t || '').length / CHARS_PER_TOKEN);

export interface ContextStats {
  /** Estimated tokens in the whole chat. */
  total: number;
  /** Estimated tokens the next request sends: the system prompt plus the newest messages that fit. */
  sent: number;
  /** The context window. */
  win: number;
}

export function contextStats(chat: Pick<Chat, 'messages' | 'systemPrompt' | 'contextWindow'> | null): ContextStats {
  if (!chat) return { total: 0, sent: 0, win: DEFAULT_CONTEXT_WINDOW };
  const win = chat.contextWindow || DEFAULT_CONTEXT_WINDOW;
  const sys = estimateTokens(chat.systemPrompt);
  const toks = chat.messages.map((m) => estimateTokens(m.content) + MESSAGE_OVERHEAD_TOKENS);
  const total = sys + toks.reduce((a, b) => a + b, 0);
  const budget = win * HISTORY_BUDGET_SHARE; // leave room for the reply
  let sent = sys;
  for (let i = toks.length - 1; i >= 0; i--) {
    if (sent + toks[i] > budget) break;
    sent += toks[i];
  }
  return { total, sent, win };
}

/** 950 → "950", 1234 → "1.2k", 32768 → "33k". */
export const kfmt = (n: number) => (n >= 1000 ? (n / 1000).toFixed(n >= KFMT_DECIMALS_BELOW ? 0 : 1).replace(/\.0$/, '') + 'k' : String(n));
