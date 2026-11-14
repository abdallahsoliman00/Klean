import { useEffect } from 'react';
import { focusComposer } from '../lib/focus';
import { artifactPanelOverlays } from '../lib/media';
import { getStore } from '../store';

/** Ctrl/Cmd+K model palette, Ctrl/Cmd+Shift+O new chat, Ctrl/Cmd+B sidebar, Escape closes the
    topmost thing (leaving a full-screen artifact panel first, or stopping the reply), and typing anywhere lands in the composer. */
export function useGlobalShortcuts(artifactPane: React.RefObject<HTMLElement | null>) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const s = getStore();
      const mod = e.metaKey || e.ctrlKey;
      const key = e.key.toLowerCase();
      if (mod && key === 'k') {
        e.preventDefault();
        s.togglePalette();
      } else if (mod && e.shiftKey && key === 'o') {
        e.preventDefault();
        s.newChat();
      } else if (mod && key === 'b') {
        e.preventDefault();
        s.toggleSidebar();
      } else if (e.key === 'Escape') {
        if (s.confirmRequest) return;
        if (s.popover) s.closePopover(true);
        else if (s.overlay || s.mobileOpen) {
          s.hideAll();
          s.closeMobile();
        } else if (s.art.full) s.toggleArtifactFull();
        else if (s.art.open && (artifactPane.current?.contains(document.activeElement) || artifactPanelOverlays() || !s.streaming)) s.closeArtifact(true);
        else s.stop();
      } else if (!mod && e.key.length === 1 && document.activeElement === document.body) focusComposer();
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [artifactPane]);
}
