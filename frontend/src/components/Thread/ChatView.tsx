import { useRef } from 'react';
import { useStickyScroll } from '../../hooks/useStickyScroll';
import { useThreadScrolling } from '../../hooks/useThreadScrolling';
import { useStore } from '../../store';
import { Composer } from '../Composer/Composer';
import { DropZone } from '../Images/DropZone';
import { Topbar } from '../Topbar/Topbar';
import { Thread } from './Thread';

/** The middle column: top bar, the scrolling thread, and the composer docked at the bottom. */
export function ChatView() {
  const scroller = useRef<HTMLDivElement>(null);
  const scroll = useStickyScroll(scroller);
  useThreadScrolling(scroll);
  const hasMessages = useStore((s) => !!s.chat?.messages.length);
  return (
    <main className="main">
      <Topbar />
      <div className="thread-wrap" ref={scroller}>
        <Thread />
      </div>
      <Composer showJump={scroll.awayFromBottom && hasMessages} onJump={() => scroll.jump(true)} />
      <DropZone />
    </main>
  );
}
