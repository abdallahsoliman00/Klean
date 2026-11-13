/* Labels, file names and highlighting languages for artifacts. */
import {
  ARTIFACT_KIND_EXTENSION, ARTIFACT_KIND_LABELS, ARTIFACT_KIND_LANGUAGE, FALLBACK_EXTENSION, LANGUAGE_EXTENSION, MAX_FILENAME_CHARS,
} from '../config';
import type { Message } from '../types';
import { languageName } from './highlight';

/** "Python", "HTML page", "Document"… */
export function kindLabel(a: { kind?: string; language?: string }): string {
  if (a.kind === 'code') return languageName(a.language) || 'Code';
  return ARTIFACT_KIND_LABELS[a.kind ?? ''] || a.kind || 'Artifact';
}

/** highlight.js language for the Code tab ('' when unknown). */
export const codeLanguage = (a: { kind: string; language: string }) =>
  ARTIFACT_KIND_LANGUAGE[a.kind] || String(a.language || '').toLowerCase();

/** A safe download name from the title, with an extension that fits the content. */
export function artifactFilename(a: { kind: string; language: string; title: string }): string {
  const ext = ARTIFACT_KIND_EXTENSION[a.kind] || LANGUAGE_EXTENSION[String(a.language || '').toLowerCase()] || FALLBACK_EXTENSION;
  const base =
    String(a.title || '')
      // oxlint-disable-next-line no-control-regex -- control characters are exactly what must go
      .replace(/[\\/:*?"<>|\u0000-\u001f]+/g, ' ')
      .replace(/\s+/g, ' ')
      .replace(/^[\s.]+|[\s.]+$/g, '')
      .slice(0, MAX_FILENAME_CHARS) || 'artifact';
  return base.toLowerCase().endsWith('.' + ext) ? base : `${base}.${ext}`;
}

/** Ids of the artifacts these messages created (they go away when the messages are edited out or regenerated). */
export function artifactsCreatedBy(msgs: Message[]): Set<string> {
  const ids = new Set<string>();
  for (const m of msgs)
    for (const p of m.parts || []) if (p.type === 'tool' && p.name === 'create_artifact' && p.result?.artifactId) ids.add(p.result.artifactId);
  return ids;
}
