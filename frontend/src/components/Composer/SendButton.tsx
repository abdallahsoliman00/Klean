import { SendIcon, StopIcon } from '../common/icons';

/** Send, or Stop while a reply streams. */
export function SendButton({ streaming, disabled }: { streaming: boolean; disabled: boolean }) {
  return (
    <button type="submit" className={'send' + (streaming ? ' stop' : '')} aria-label={streaming ? 'Stop generating' : 'Send message'} disabled={disabled}>
      <SendIcon />
      <StopIcon />
    </button>
  );
}
