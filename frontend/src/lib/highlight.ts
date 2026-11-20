/* Syntax highlighting with highlight.js's common languages (the same set the old vendored build had). */
import hljs from 'highlight.js/lib/common';

const escapeHtml = (t: string) => t.replace(/[&<>]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;' })[c]!);

export const knownLanguage = (lang?: string | null) => !!lang && !!hljs.getLanguage(lang);

/** Display name of a language ("python" → "Python"), or the input when highlight.js doesn't know it. */
export const languageName = (lang?: string | null) => (lang && hljs.getLanguage(String(lang))?.name) || lang || '';

/** Highlighted HTML for a fenced code block: the named language, else a best guess. */
export function highlightBlock(text: string, lang: string): string {
  try {
    return knownLanguage(lang) ? hljs.highlight(text, { language: lang }).value : hljs.highlightAuto(text).value;
  } catch {
    return escapeHtml(text);
  }
}

/** Highlighted HTML in the given language, a guess when `guess` is set, or null for plain text. */
export function highlightOrNull(text: string, lang: string, guess: boolean): string | null {
  try {
    if (knownLanguage(lang)) return hljs.highlight(text, { language: lang }).value;
    return guess ? hljs.highlightAuto(text).value : null;
  } catch {
    return null;
  }
}

