import { useEffect, useRef } from 'react';
import { ArtifactPanel } from './components/ArtifactPanel/ArtifactPanel';
import { ChatSettingsPanel } from './components/ChatPanel/ChatSettingsPanel';
import { ConfirmDialog } from './components/common/ConfirmDialog';
import { Menu } from './components/common/Menu';
import { Scrim } from './components/common/Scrim';
import { Toasts } from './components/common/Toasts';
import { UpdateBanner } from './components/common/UpdateBanner';
import { Lightbox } from './components/Images/Lightbox';
import { ArtifactLibrary } from './components/Library/ArtifactLibrary';
import { ModelPalette } from './components/Palette/ModelPalette';
import { SettingsModal } from './components/Settings/SettingsModal';
import { Sidebar } from './components/Sidebar/Sidebar';
import { ChatView } from './components/Thread/ChatView';
import { useChatRoute } from './hooks/useChatRoute';
import { useDocumentListeners } from './hooks/useDocumentListeners';
import { useGlobalShortcuts } from './hooks/useGlobalShortcuts';
import { useRunningChats } from './hooks/useRunningChats';
import { useThemeSync } from './hooks/useThemeSync';
import { useStore } from './store';
import { artifactPanelOpen } from './store/selectors';

/** The shell: sidebar, chat, artifact panel, and the overlays above them. */
export function App() {
  const collapsed = useStore((s) => s.collapsed);
  const mobileOpen = useStore((s) => s.mobileOpen);
  const artOpen = useStore(artifactPanelOpen);
  const artifactPane = useRef<HTMLElement>(null);

  useThemeSync();
  useDocumentListeners();
  useGlobalShortcuts(artifactPane);
  useChatRoute();
  useRunningChats();
  useEffect(() => {
    useStore.getState().boot();
  }, []);

  const cls = ['app', collapsed && 'collapsed', mobileOpen && 'mobile-open', artOpen && 'art-open'].filter(Boolean).join(' ');
  return (
    <>
      <div className={cls}>
        <Sidebar />
        <ChatView />
        <ArtifactPanel ref={artifactPane} />
      </div>
      <Scrim />
      <ModelPalette />
      <ArtifactLibrary />
      <ChatSettingsPanel />
      <SettingsModal />
      <Menu />
      <Toasts />
      <UpdateBanner />
      <ConfirmDialog />
      <Lightbox />
    </>
  );
}
