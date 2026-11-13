/* A message is a list of parts in the order they happened: prose, then a tool call, then more prose.
   Old messages have no parts, so their content counts as a single text part. */
import type { Message, Part, ToolPart } from '../types';

export const partsOf = (m: Message): Part[] =>
  Array.isArray(m.parts) && m.parts.length ? m.parts : [{ type: 'text', text: m.content || '' }];

/** The tool call with this id, if any message has it. */
export function findToolPart(messages: Message[], callId: string): ToolPart | null {
  for (const m of messages) for (const p of m.parts || []) if (p.type === 'tool' && p.id === callId) return p;
  return null;
}

/** Returns new messages where `fn` has rewritten every tool part it matches (it returns null to leave one alone). */
export function mapToolParts(messages: Message[], fn: (p: ToolPart) => ToolPart | null): Message[] {
  return messages.map((m) => {
    if (!m.parts?.some((p) => p.type === 'tool' && fn(p))) return m;
    return { ...m, parts: m.parts.map((p) => (p.type === 'tool' ? fn(p) ?? p : p)) };
  });
}

/** Replaces the tool call with this id. */
export const patchToolPart = (messages: Message[], callId: string, patch: Partial<ToolPart>) =>
  mapToolParts(messages, (p) => (p.id === callId ? { ...p, ...patch } : null));

/** A normal message while a question is waiting skips the question. */
export const skipPendingQuestions = (messages: Message[]) =>
  mapToolParts(messages, (p) => (p.status === 'pending' ? { ...p, status: 'done', result: { answer: '' } } : null));
