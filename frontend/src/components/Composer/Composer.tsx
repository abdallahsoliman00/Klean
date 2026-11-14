import { useRef } from 'react';
import { COMPOSER_MAX_HEIGHT_PX, IMAGE_TYPES } from '../../config';
import { useAutosize } from '../../hooks/useAutosize';
import { composerInput } from '../../lib/focus';
import { dataUrl } from '../../lib/images';
import { getStore, useStore } from '../../store';
import { ArrowDownIcon, ImageIcon } from '../common/icons';
import { ImageStrip } from '../Images/ImageStrip';
import { SendButton } from './SendButton';
import { SystemPromptChip } from './SystemPromptChip';
import { ToolsChip } from './ToolsChip';

/** The message box at the bottom. Enter sends, Shift+Enter adds a line; while a reply streams, the button stops it.
    Images are attached with the button, by pasting, or by dropping files anywhere on the app. */
export function Composer({ showJump, onJump }: { showJump: boolean; onJump: () => void }) {
  const text = useStore((s) => s.composerText);
  const setText = useStore((s) => s.setComposerText);
  const model = useStore((s) => s.sel.model);
  const streaming = useStore((s) => !!s.streaming);
  const attachments = useStore((s) => s.attachments);
  const attaching = useStore((s) => s.attaching);
  const { send, stop, attachFiles, removeAttachment, setAttachments } = getStore();
  const picker = useRef<HTMLInputElement>(null);
  useAutosize(composerInput, text, COMPOSER_MAX_HEIGHT_PX);
  const ready = !!model && !attaching && (!!text.trim() || attachments.length > 0);

  const submit = (e?: React.FormEvent) => {
    e?.preventDefault();
    if (streaming) return stop();
    if (!ready) return;
    setText('');
    setAttachments([]);
    send({ content: text.trim(), images: attachments.map(({ mediaType, data }) => ({ mediaType, data })) });
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
          {(attachments.length > 0 || attaching > 0) && (
            <ImageStrip images={attachments.map((a) => ({ src: dataUrl(a), remove: () => removeAttachment(a.id) }))} busy={attaching} />
          )}
          <textarea
            id="input"
            ref={composerInput}
            rows={1}
            dir="auto"
            placeholder={model ? `Message ${model}` : 'Choose a model to start'}
            value={text}
            onChange={(e) => setText(e.target.value)}
            onPaste={(e) => {
              if (!e.clipboardData.files.length) return; // text pastes as usual
              e.preventDefault();
              void attachFiles(e.clipboardData.files);
            }}
            onKeyDown={(e) => {
              if (e.key !== 'Enter' || e.shiftKey || e.nativeEvent.isComposing) return;
              e.preventDefault();
              if (!streaming) submit();
            }}
          />
          <div className="box-row">
            <SystemPromptChip />
            <ToolsChip />
            <button type="button" className="chip attach" aria-label="Attach images" title="Attach images (or paste, or drop them here)" onClick={() => picker.current?.click()}>
              <ImageIcon />
              <span>Image</span>
            </button>
            <input
              ref={picker}
              type="file"
              hidden
              multiple
              accept={IMAGE_TYPES.join(',')}
              onChange={(e) => {
                void attachFiles(e.target.files ?? []);
                e.target.value = ''; // so the same file can be picked again
                composerInput.current?.focus();
              }}
            />
            <SendButton streaming={streaming} disabled={!streaming && !ready} />
          </div>
        </form>
        <div className="hint">Enter sends, Shift+Enter adds a line, Ctrl+K switches model</div>
      </div>
    </div>
  );
}
