import type { ReactNode } from 'react';
import { useCopyFeedback } from '../../hooks/useCopyFeedback';
import { CheckIcon, CopyIcon } from '../common/icons';

/** A small labelled action under a message. */
export function ToolButton({ icon, label, onClick, done }: { icon: ReactNode; label: string; onClick: () => void; done?: boolean }) {
  return (
    <button className={'tool' + (done ? ' done' : '')} type="button" aria-label={label} title={label} onClick={onClick}>
      {icon}
      <span>{label}</span>
    </button>
  );
}

/** Copy, which flips to "Copied" for a moment. */
export function CopyButton({ text }: { text: string }) {
  const [copied, copy] = useCopyFeedback();
  return copied ? (
    <ToolButton icon={<CheckIcon />} label="Copied" done onClick={() => copy(text)} />
  ) : (
    <ToolButton icon={<CopyIcon />} label="Copy" onClick={() => copy(text)} />
  );
}
