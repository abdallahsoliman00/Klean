import { useEffect, useState } from 'react';
import { getStore } from '../../store';
import { ImageIcon } from '../common/icons';

const hasFiles = (e: DragEvent) => !!e.dataTransfer?.types.includes('Files');

/** Files dragged anywhere onto the app are attached to the message. While they hover, the chat column
    says so. Settings has drop targets of its own, so this one stands down while an overlay is open. */
export function DropZone() {
  const [over, setOver] = useState(false);

  useEffect(() => {
    // dragenter and dragleave fire for every element the pointer crosses: the drag is over the window while more have been entered than left
    let depth = 0;
    const idle = () => !getStore().overlay && !getStore().lightbox;
    const onEnter = (e: DragEvent) => {
      if (!hasFiles(e)) return;
      depth++;
      setOver(idle());
    };
    const onLeave = (e: DragEvent) => {
      if (hasFiles(e) && --depth <= 0) end();
    };
    const end = () => {
      depth = 0;
      setOver(false);
    };
    const onOver = (e: DragEvent) => {
      if (hasFiles(e)) e.preventDefault(); // without this the browser opens the file in place of the app
    };
    const onDrop = (e: DragEvent) => {
      if (!hasFiles(e)) return;
      end();
      if (e.defaultPrevented) return; // a drop target inside the app took it
      e.preventDefault();
      if (idle()) void getStore().attachFiles(e.dataTransfer!.files);
    };
    const listeners = { dragenter: onEnter, dragleave: onLeave, dragover: onOver, drop: onDrop };
    for (const [name, fn] of Object.entries(listeners)) window.addEventListener(name, fn as EventListener);
    return () => {
      for (const [name, fn] of Object.entries(listeners)) window.removeEventListener(name, fn as EventListener);
    };
  }, []);

  return (
    <div className={'dropzone' + (over ? ' show' : '')} aria-hidden="true">
      <div className="dropzone-card">
        <ImageIcon size={18} />
        Drop images to attach them
      </div>
    </div>
  );
}
