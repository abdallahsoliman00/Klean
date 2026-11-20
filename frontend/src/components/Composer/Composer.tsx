import { useRef } from 'react';
import { composerInput } from '../../lib/focus';
import { dataUrl } from '../../lib/images';
import { getStore, useStore } from '../../store';
import { ArrowDownIcon, UploadIcon } from '../common/icons';
import { FileChips } from '../Images/FileChips';
import { ImageStrip } from '../Images/ImageStrip';
import { SendButton } from './SendButton';
import { SystemPromptChip } from './SystemPromptChip';
import { MessageInput } from './MessageInput';
import { ToolsChip } from './ToolsChip';

/** The message box at the bottom. Enter sends (Ctrl+Enter from a code block), Shift+Enter adds a line; while a reply streams, the button stops it.
    Images and text files are attached with the button, by pasting, or by dropping them anywhere on the app. */
export function Composer({ showJump, onJump }: { showJump: boolean; onJump: () => void }) {
  const text = useStore((s) => s.composerText);
  const setText = useStore((s) => s.setComposerText);
  const model = useStore((s) => s.sel.model);
  const streaming = useStore((s) => !!s.streaming);
  const attachments = useStore((s) => s.attachments);
  const attachedFiles = useStore((s) => s.attachedFiles);
  const attaching = useStore((s) => s.attaching);
  const { send, stop, attachFiles, removeAttachment, removeFile, setAttachments } = getStore();
  const picker = useRef<HTMLInputElement>(null);
  const ready = !!model && !attaching && (!!text.trim() || attachments.length > 0 || attachedFiles.length > 0);

  const submit = (e?: React.FormEvent) => {
    e?.preventDefault();
    if (streaming) return stop();
    if (!ready) return;
    setText('');
    setAttachments([]);
    send({
      content: text.trim(),
      images: attachments.map(({ mediaType, data }) => ({ mediaType, data })),
      files: attachedFiles.map((f) => ({ name: f.name, text: f.text })),
    });
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
          {attachedFiles.length > 0 && <FileChips files={attachedFiles} onRemove={(i) => removeFile(attachedFiles[i].id)} />}
          <div
            onPaste={(e) => {
              if (!e.clipboardData.files.length) return; // text pastes as usual
              e.preventDefault();
              void attachFiles(e.clipboardData.files);
            }}
          >
            <MessageInput text={text} setText={setText} placeholder={model ? `Message ${model}` : 'Choose a model to start'} onSend={() => !streaming && submit()} />
          </div>
          <div className="box-row">
            <SystemPromptChip />
            <ToolsChip />
            <button type="button" className="chip attach" aria-label="Attach images or text files" title="Attach images or text files (or paste, or drop them here)" onClick={() => picker.current?.click()}>
              <UploadIcon />
              <span>Attach</span>
            </button>
            <input
              ref={picker}
              type="file"
              hidden
              multiple
              onChange={(e) => {
                void attachFiles(e.target.files ?? []);
                e.target.value = ''; // so the same file can be picked again
                composerInput.current?.focus();
              }}
            />
            <SendButton streaming={streaming} disabled={!streaming && !ready} />
          </div>
        </form>
        <div className="hint">Enter sends, Shift+Enter adds a line, ```lang then Enter starts a code block, Ctrl+K switches model</div>
      </div>
    </div>
  );
}
