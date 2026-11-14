import { useEffect, useState } from 'react';
import { ART_WIDTH } from '../../config';
import { initialArtifactWidth, maxArtifactWidth } from '../../lib/media';
import { useStore } from '../../store';

/** The panel's inner edge. Drag it, or focus it and press the arrow keys (Home and End for the limits),
    to change the panel's width; the browser remembers it. */
export function ArtifactResizer() {
  const saved = useStore((s) => s.artWidth);
  const setArtWidth = useStore((s) => s.setArtWidth);
  const [dragging, setDragging] = useState(false);
  const width = saved ?? initialArtifactWidth();

  // the width is the --art token the layout already uses, capped so a smaller window keeps room for the thread
  useEffect(() => {
    if (saved) document.documentElement.style.setProperty('--art', `min(${saved}px, ${ART_WIDTH.maxShare * 100}vw)`);
  }, [saved]);

  const onKeyDown = (e: React.KeyboardEvent) => {
    const next = { ArrowLeft: width + ART_WIDTH.step, ArrowRight: width - ART_WIDTH.step, Home: ART_WIDTH.min, End: maxArtifactWidth() }[e.key];
    if (next === undefined) return;
    e.preventDefault();
    setArtWidth(next);
  };

  return (
    <div
      className={'art-resize' + (dragging ? ' dragging' : '')}
      role="separator"
      tabIndex={0}
      aria-orientation="vertical"
      aria-label="Resize artifact panel"
      aria-valuemin={ART_WIDTH.min}
      aria-valuemax={maxArtifactWidth()}
      aria-valuenow={width}
      onKeyDown={onKeyDown}
      onPointerDown={(e) => {
        e.preventDefault(); // no text selection while dragging
        e.currentTarget.focus();
        e.currentTarget.setPointerCapture(e.pointerId); // keeps the drag alive over the preview frame
        setDragging(true);
      }}
      onPointerMove={(e) => dragging && setArtWidth(innerWidth - e.clientX)}
      onLostPointerCapture={() => setDragging(false)}
    />
  );
}
