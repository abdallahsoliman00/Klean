import { COMPOSER_MAX_HEIGHT_PX } from '../../config';
import { useAutosize } from '../../hooks/useAutosize';
import { composerInput } from '../../lib/focus';
import { getStore, useStore } from '../../store';
import { ArrowDownIcon } from '../common/icons';
import { SendButton } from './SendButton';
import { SystemPromptChip } from './SystemPromptChip';
import { ToolsChip } from './ToolsChip';

/** The message box at the bottom. Enter sends, Shift+Enter adds a line; while a reply streams, the button stops it. */
export function Composer({ showJump, onJump }: { showJump: boolean; onJump: () => void }) {
  const text = useStore((s) => s.composerText);
  const setText = useStore((s) => s.setComposerText);
  const model = useStore((s) => s.sel.model);
  const streaming = useStore((s) => !!s.streaming);
  const { send, stop } = getStore();
  useAutosize(composerInput, text, COMPOSER_MAX_HEIGHT_PX);

  const submit = (e?: React.FormEvent) => {
    e?.preventDefault();
    if (streaming) return stop();
    const v = text.trim();
    if (!v || !model) return;
    setText('');
    send({ content: v });
  };

  return (
    <div className="composer-dock">
      <div className="composer">
        <button className={'jump' + (showJump ? ' show' : '')} onClick={onJump}>
          <ArrowDownIcon />
          Jump to latest
        </button>
        <form className="box" onSubmit={submit}>
          <label className="sr" htmlFor="input">
            Message
          </label>
          <textarea
            id="input"
            ref={composerInput}
            rows={1}
            dir="auto"
            placeholder={model ? `Message ${model}` : 'Choose a model to start'}
            value={text}
            onChange={(e) => setText(e.target.value)}
            onKeyDown={(e) => {
              if (e.key !== 'Enter' || e.shiftKey || e.nativeEvent.isComposing) return;
              e.preventDefault();
              if (!streaming) submit();
            }}
          />
          <div className="box-row">
            <SystemPromptChip />
            <ToolsChip />
            <SendButton streaming={streaming} disabled={!streaming && (!text.trim() || !model)} />
          </div>
        </form>
        <div className="hint">Enter sends, Shift+Enter adds a line, Ctrl+K switches model</div>
      </div>
    </div>
  );
}
