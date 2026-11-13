import { CloseIcon } from './icons';

export function CloseButton({ onClick, label = 'Close', title }: { onClick: () => void; label?: string; title?: string }) {
  return (
    <button className="icon-btn" aria-label={label} title={title} onClick={onClick}>
      <CloseIcon />
    </button>
  );
}
