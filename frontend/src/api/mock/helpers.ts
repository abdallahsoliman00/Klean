import type { Message, ToolPart } from '../../types';
import { ARTIFACT_CREATE_TOOLS } from '../../config';
import { artifacts } from './data';

export const sleep = (ms: number) => new Promise<void>((r) => setTimeout(r, ms));

/** Every tool call in these messages, with the message it belongs to. */
export function* toolParts(messages: Message[]): Generator<{ m: Message; p: ToolPart }> {
  for (const m of messages) for (const p of m.parts || []) if (p.type === 'tool') yield { m, p };
}

/** Editing or regenerating also removes the artifacts those messages created. */
export function dropArtifacts(msgs: Message[]) {
  for (const { p } of toolParts(msgs)) {
    if (!ARTIFACT_CREATE_TOOLS.includes(p.name) || !p.result) continue;
    const i = artifacts.findIndex((a) => a.id === p.result!.artifactId);
    if (i > -1) artifacts.splice(i, 1);
  }
}
