/* Which overlay, drawer, popover or menu is showing. */
import type { ReactNode } from 'react';
import { CHAT_PANEL_FOCUS_MS, MENU_EDGE_RESERVE_PX, MENU_GAP_PX } from '../config';
import { systemPromptInput } from '../lib/focus';
import { isMobile } from '../lib/media';
import type { Slice } from './types';

export type Overlay = 'palette' | 'chatPanel' | 'settings';

/** What the settings modal shows: a provider (id null = "Add a provider"), or an app page. */
export type SettingsPage = { kind: 'provider'; id: string | null } | { kind: 'appearance' } | { kind: 'search' } | { kind: 'skills' };

export interface MenuItem {
  label: string;
  icon: ReactNode;
  danger?: boolean;
  run: () => void;
}

export interface UiSlice {
  overlay: Overlay | null;
  collapsed: boolean;
  mobileOpen: boolean;
  settingsPage: SettingsPage;
  /** The open popover (artifact switcher, tools menu) and the button that opened it. */
  popover: { id: string; anchor: HTMLElement } | null;
  menu: { x: number; y: number; items: MenuItem[] } | null;
  /** The image shown large, over everything. */
  lightbox: { src: string; label: string } | null;

  showOverlay(o: Overlay): void;
  hideAll(): void;
  toggleSidebar(): void;
  closeMobile(): void;
  openSettings(providerId?: string | null): void;
  setSettingsPage(p: SettingsPage): void;
  openPalette(): void;
  openChatPanel(): void;
  togglePalette(): void;
  togglePopover(id: string, anchor: HTMLElement): void;
  closePopover(refocus?: boolean): void;
  showMenu(anchor: HTMLElement, items: MenuItem[]): void;
  hideMenu(): void;
  showImage(src: string, label: string): void;
  closeLightbox(): void;
}

export const createUiSlice: Slice<UiSlice> = (set, get) => ({
  overlay: null,
  collapsed: false,
  mobileOpen: false,
  settingsPage: { kind: 'provider', id: null },
  popover: null,
  menu: null,
  lightbox: null,

  showOverlay(overlay) {
    get().hideMenu();
    get().closePopover();
    set({ overlay });
  },
  hideAll() {
    get().hideMenu();
    get().closePopover();
    set({ overlay: null });
  },
  toggleSidebar() {
    if (isMobile()) set((s) => ({ mobileOpen: !s.mobileOpen }));
    else set((s) => ({ collapsed: !s.collapsed }));
  },
  closeMobile() {
    set({ mobileOpen: false });
  },

  openSettings(providerId) {
    const id = providerId === undefined ? (get().providers[0]?.id ?? null) : providerId;
    set({ settingsPage: { kind: 'provider', id } });
    get().showOverlay('settings');
    if (id) get().loadModels(id);
  },
  setSettingsPage(settingsPage) {
    set({ settingsPage });
    if (settingsPage.kind === 'provider' && settingsPage.id) get().loadModels(settingsPage.id);
  },

  openPalette() {
    const { providers } = get();
    if (!providers.length) return get().openSettings(null);
    providers.forEach((p) => get().loadModels(p.id));
    get().showOverlay('palette');
  },
  openChatPanel() {
    get().showOverlay('chatPanel');
    setTimeout(() => systemPromptInput.current?.focus(), CHAT_PANEL_FOCUS_MS);
  },
  togglePalette() {
    if (get().overlay === 'palette') get().hideAll();
    else get().openPalette();
  },

  togglePopover(id, anchor) {
    const open = get().popover;
    if (open?.anchor === anchor) return get().closePopover(true);
    get().closePopover();
    get().hideMenu();
    set({ popover: { id, anchor } });
  },
  closePopover(refocus) {
    const open = get().popover;
    if (!open) return;
    set({ popover: null });
    if (refocus && open.anchor.isConnected) open.anchor.focus({ preventScroll: true });
  },

  showMenu(anchor, items) {
    const r = anchor.getBoundingClientRect();
    set({ menu: { x: Math.min(r.left, innerWidth - MENU_EDGE_RESERVE_PX), y: r.bottom + MENU_GAP_PX, items } });
  },
  hideMenu() {
    if (get().menu) set({ menu: null });
  },

  showImage(src, label) {
    set({ lightbox: { src, label } });
  },
  closeLightbox() {
    set({ lightbox: null });
  },
});
