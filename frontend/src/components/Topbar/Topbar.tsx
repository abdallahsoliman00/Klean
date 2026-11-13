import { SidebarIcon, SlidersIcon } from '../common/icons';
import { useStore } from '../../store';
import { ArtifactsButton } from './ArtifactsButton';
import { ContextButton } from './ContextButton';
import { ModelPill } from './ModelPill';

export function Topbar() {
  const toggleSidebar = useStore((s) => s.toggleSidebar);
  const openChatPanel = useStore((s) => s.openChatPanel);
  return (
    <header className="topbar">
      <button className="icon-btn" aria-label="Toggle sidebar" title="Toggle sidebar (Ctrl+B)" onClick={toggleSidebar}>
        <SidebarIcon />
      </button>
      <ModelPill />
      <div className="spacer" />
      <ArtifactsButton />
      <ContextButton />
      <button className="icon-btn" aria-label="Chat settings" title="Chat settings" onClick={openChatPanel}>
        <SlidersIcon />
      </button>
    </header>
  );
}
