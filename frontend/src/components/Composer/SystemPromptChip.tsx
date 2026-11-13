import { useStore } from '../../store';
import { LinesIcon } from '../common/icons';

/** Shows whether this chat has a system prompt; opens chat settings. */
export function SystemPromptChip() {
  const on = useStore((s) => !!(s.chat ? s.chat.systemPrompt : s.draft?.systemPrompt));
  const openChatPanel = useStore((s) => s.openChatPanel);
  return (
    <button type="button" className={'chip' + (on ? ' sys-on' : '')} title="System prompt" onClick={openChatPanel}>
      <LinesIcon />
      <span>{on ? 'System prompt on' : 'No system prompt'}</span>
    </button>
  );
}
