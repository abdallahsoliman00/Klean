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

/* Each chat has an address: #/chat/<id>. A hash, so any static server can serve the page (hooks/useChatRoute). */
const CHAT_ROUTE = '#/chat/';
export const chatUrl = (id: string) => CHAT_ROUTE + encodeURIComponent(id);
/** The chat the address bar names, or null. */
export const routedChatId = () => (location.hash.startsWith(CHAT_ROUTE) ? decodeURIComponent(location.hash.slice(CHAT_ROUTE.length)) || null : null);
