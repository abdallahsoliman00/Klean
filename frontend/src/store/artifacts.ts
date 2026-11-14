/* The artifact panel: which artifact it shows, and keeping the open chat's artifacts in step. */
import { api } from '../api';
import { ApiError } from '../api/errors';
import { ART_FOCUS_MS, STATUS_NOT_FOUND } from '../config';
import { artifactFilename, artifactsCreatedBy } from '../lib/artifacts';
import { downloadReportPdf } from '../lib/reportDownload';
import { artifactSwitch, focusComposer } from '../lib/focus';
import { artifactPanelOverlays } from '../lib/media';
import type { Artifact, Message } from '../types';
import { artifactPanelOpen } from './selectors';
import type { Slice } from './types';

export type ArtifactTab = 'preview' | 'code';

export interface ArtifactsSlice {
  art: {
    open: boolean;
    id: string | null;
    tab: ArtifactTab;
    /** The panel covers the whole app. */
    full: boolean;
    /** The button that opened the panel; focus returns there on close. */
    opener: HTMLElement | null;
  };
  openArtifact(id: string | undefined, opener?: HTMLElement): void;
  closeArtifact(refocus?: boolean): void;
  /** The topbar button: opens the last-shown (or newest) artifact, or closes the panel. */
  toggleArtifacts(anchor: HTMLElement): void;
  showArtifact(id: string): void;
  setArtifactTab(tab: ArtifactTab): void;
  toggleArtifactFull(): void;
  /** A new or updated artifact arrived from the stream. */
  putArtifact(a: Artifact): void;
  /** Drops the artifacts these messages created (the backend does the same when they are edited out). */
  forgetArtifacts(msgs: Message[]): void;
  deleteArtifact(a: Artifact): Promise<void>;
  /** Id of the report whose PDF is being made (one at a time). */
  pdfFor: string | null;
  /** Downloads a report as a PDF; says so when it is done, or what to do if it failed. */
  downloadReport(a: Artifact): Promise<void>;
  resetArtifacts(): void;
}

const findArtifact = (list: Artifact[] | undefined, id: string | null | undefined) => list?.find((a) => a.id === id);

export const createArtifactsSlice: Slice<ArtifactsSlice> = (set, get) => {
  /** With nothing left to show, the panel closes. */
  const closeIfEmpty = () => {
    if (!get().chat?.artifacts.length) set((s) => ({ art: { ...s.art, open: false, full: false } }));
  };

  return {
    art: { open: false, id: null, tab: 'preview', full: false, opener: null },

    openArtifact(id, opener) {
      if (!id || !findArtifact(get().chat?.artifacts, id)) return;
      set((s) => ({ art: { ...s.art, id, open: true, opener: opener ?? s.art.opener } }));
      if (opener) setTimeout(() => artifactSwitch.current?.focus({ preventScroll: true }), ART_FOCUS_MS);
    },

    closeArtifact(refocus) {
      const { art } = get();
      if (!art.open) return;
      set({ art: { ...art, open: false, full: false } });
      get().closePopover();
      if (!refocus) return;
      const o = art.opener;
      if (o?.isConnected && !(o as HTMLButtonElement).disabled) o.focus();
      else focusComposer();
    },

    toggleArtifacts(anchor) {
      const { art, chat } = get();
      if (art.open) return get().closeArtifact();
      get().openArtifact(findArtifact(chat?.artifacts, art.id) ? art.id! : chat?.artifacts.at(-1)?.id, anchor);
    },

    showArtifact(id) {
      set((s) => ({ art: { ...s.art, id } }));
    },

    setArtifactTab(tab) {
      set((s) => ({ art: { ...s.art, tab } }));
    },

    toggleArtifactFull() {
      set((s) => ({ art: { ...s.art, full: !s.art.full } }));
    },

    putArtifact(a) {
      const chat = get().chat;
      if (!chat) return;
      const isNew = !findArtifact(chat.artifacts, a.id);
      get().updateChat((c) => ({ ...c, artifacts: isNew ? [...c.artifacts, a] : c.artifacts.map((x) => (x.id === a.id ? a : x)) }));
      // a new one opens itself when there is room beside the thread
      if (isNew && !artifactPanelOverlays()) {
        get().setArtifactTab('preview');
        get().openArtifact(a.id);
      }
    },

    forgetArtifacts(msgs) {
      const ids = artifactsCreatedBy(msgs);
      if (!ids.size) return;
      get().updateChat((c) => ({ ...c, artifacts: c.artifacts.filter((a) => !ids.has(a.id)) }));
      closeIfEmpty();
    },

    async deleteArtifact(a) {
      const chatId = get().chat?.id;
      try {
        await api.artifacts.remove(a.id);
      } catch (e) {
        if (!(e instanceof ApiError) || e.status !== STATUS_NOT_FOUND) throw e; // 404: already deleted
      }
      if (get().chat?.id !== chatId) return;
      const wasOpen = artifactPanelOpen(get());
      get().updateChat((c) => ({ ...c, artifacts: c.artifacts.filter((x) => x.id !== a.id) }));
      closeIfEmpty();
      if (wasOpen && !get().art.open) focusComposer();
      get().toast(`Deleted “${a.title || 'Untitled'}”`);
    },

    pdfFor: null,

    async downloadReport(a) {
      if (get().pdfFor) return;
      set({ pdfFor: a.id });
      const name = artifactFilename(a);
      try {
        await downloadReportPdf(a, name, api.artifacts.pdf);
        get().toast(`Downloaded ${name}`);
      } catch {
        get().toast('The PDF couldn’t be made. Press Download PDF to try again, or ask for a shorter report.');
      } finally {
        set({ pdfFor: null });
      }
    },

    resetArtifacts() {
      set({ art: { open: false, id: null, tab: get().art.tab, full: false, opener: null } });
      get().closePopover();
    },
  };
};
