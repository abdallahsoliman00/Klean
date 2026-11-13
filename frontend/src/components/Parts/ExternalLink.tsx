import type { ReactNode } from 'react';
import { safeUrl } from '../../lib/urls';

/** A link that opens in a new tab, or plain text when the URL isn't http(s). */
export function ExternalLink({ url, className, children }: { url: unknown; className?: string; children: ReactNode }) {
  const href = safeUrl(url);
  return href ? (
    <a className={className} href={href} target="_blank" rel="noopener noreferrer">
      {children}
    </a>
  ) : (
    <span className={className}>{children}</span>
  );
}
