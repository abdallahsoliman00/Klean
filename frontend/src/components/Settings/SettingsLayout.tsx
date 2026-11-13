import type { ReactNode } from 'react';
import { useStore } from '../../store';
import { CloseButton } from '../common/CloseButton';

/** A settings page: title bar, scrolling content, and the action buttons underneath. */
export function SettingsLayout({ title, actions, children }: { title: string; actions?: ReactNode; children: ReactNode }) {
  const hideAll = useStore((s) => s.hideAll);
  return (
    <>
      <div className="panel-head">
        <h2>{title}</h2>
        <CloseButton onClick={hideAll} />
      </div>
      <div className="modal-content">{children}</div>
      <div className="modal-actions">{actions}</div>
    </>
  );
}
