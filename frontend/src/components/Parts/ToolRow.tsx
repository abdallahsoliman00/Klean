import type { ReactNode } from 'react';

/** One line for a tool call: an icon, then what happened. */
export function ToolRow({ icon, error, children }: { icon: ReactNode; error?: boolean; children: ReactNode }) {
  return (
    <div className={'trow' + (error ? ' err' : '')}>
      {icon}
      {children}
    </div>
  );
}
