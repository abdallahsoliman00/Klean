/* The artifact panel: which artifact it shows, and keeping the open chat's artifacts in step. */
import { api } from '../api';
import { ApiError } from '../api/errors';
import { ART_FOCUS_MS, STATUS_NOT_FOUND } from '../config';
import { artifactFilename, artifactsCreatedBy } from '../lib/artifacts';
import { downloadReportPdf } from '../lib/reportDownload';
import { artifactSwitch, focusComposer } from '../lib/focus';
import { artifactPanelOverlays } from '../lib/media';
import type { Artifact, ArtifactSummary, ArtifactVersion, Message } from '../types';
import { artifactPanelOpen } from './selectors';
import type { Slice } from './types';

export type ArtifactTab = 'preview' | 'code';

export interface ArtifactsSlice {
  art: {
    open: boolean;
    id: string | null;
    tab: ArtifactTab;
    /** The version on show; null means the latest. */
    version: number | null;
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
  /** Every saved version of an artifact, oldest first, once loaded. A new version from the stream drops its entry. */
  versions: Record<string, ArtifactVersion[] | undefined>;
  loadVersions(id: string): Promise<void>;
  /** Shows an older version of the artifact on show; null goes back to the latest. */
  showVersion(version: number | null): void;
  /** The library's choice: goes to the chat the artifact was made in and opens it there. */
  openFromLibrary(a: ArtifactSummary): Promise<void>;
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
    art: { open: false, id: null, tab: 'preview', version: null, full: false, opener: null },
    versions: {},

    openArtifact(id, opener) {
      if (!id || !findArtifact(get().chat?.artifacts, id)) return;
      set((s) => ({ art: { ...s.art, id, version: id === s.art.id ? s.art.version : null, open: true, opener: opener ?? s.art.opener } }));
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
      set((s) => ({ art: { ...s.art, id, version: null } }));
    },

    async loadVersions(id) {
      const list = await api.artifacts.versions(id);
      // kept only while it is current: a newer version may have arrived from the stream meanwhile
      if (findArtifact(get().chat?.artifacts, id)?.version === list.at(-1)?.version) set((s) => ({ versions: { ...s.versions, [id]: list } }));
    },

    showVersion(version) {
      set((s) => ({ art: { ...s.art, version } }));
    },

    async openFromLibrary({ id, chatId }) {
      get().hideAll();
      if (get().chat?.id !== chatId) await get().openChat(chatId);
      if (get().chat?.id !== chatId || !findArtifact(get().chat?.artifacts, id)) return get().toast('That artifact is no longer there.');
      get().openArtifact(id);
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
      // its version list is out of date, and the panel goes back to showing the latest
      set((s) => ({ versions: { ...s.versions, [a.id]: undefined }, art: { ...s.art, version: null } }));
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
        // the backend renders the latest version only; an older one is made in the browser
        const latest = findArtifact(get().chat?.artifacts, a.id)?.version === a.version;
        await downloadReportPdf(a, name, latest ? api.artifacts.pdf : undefined);
        get().toast(`Downloaded ${name}`);
      } catch {
        get().toast('The PDF couldn’t be made. Press Download PDF to try again, or ask for a shorter report.');
      } finally {
        set({ pdfFor: null });
      }
    },

    resetArtifacts() {
      set({ art: { open: false, id: null, tab: get().art.tab, version: null, full: false, opener: null }, versions: {} });
      get().closePopover();
    },
  };
};
