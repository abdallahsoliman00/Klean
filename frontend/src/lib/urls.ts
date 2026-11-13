/* Links from the model are untrusted: only http(s) ever becomes a clickable link. */

export function safeUrl(u: unknown): string | null {
  try {
    const x = new URL(String(u));
    return x.protocol === 'http:' || x.protocol === 'https:' ? x.href : null;
  } catch {
    return null;
  }
}

export function domainOf(u: unknown): string {
  try {
    return new URL(String(u)).hostname.replace(/^www\./, '') || String(u);
  } catch {
    return String(u || 'a page');
  }
}
