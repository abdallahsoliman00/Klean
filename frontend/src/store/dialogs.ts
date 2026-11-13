/* Toasts and the confirm dialog. */
import { TOAST_EXIT_MS, TOAST_MS, TOAST_WITH_ACTION_MS } from '../config';
import type { Slice } from './types';

export interface Toast {
  id: number;
  message: string;
  action?: { label: string; run: () => void };
  /** Playing its exit transition. */
  leaving: boolean;
}

export interface ConfirmRequest {
  title: string;
  text: string;
  okLabel: string;
  ok: () => void;
}

export interface DialogSlice {
  toasts: Toast[];
  confirmRequest: ConfirmRequest | null;
  toast(message: string, action?: Toast['action']): void;
  dismissToast(id: number): void;
  confirm(req: ConfirmRequest): void;
  closeConfirm(): void;
}

let nextToastId = 1;

export const createDialogSlice: Slice<DialogSlice> = (set, get) => ({
  toasts: [],
  confirmRequest: null,

  toast(message, action) {
    const id = nextToastId++;
    set((s) => ({ toasts: [...s.toasts, { id, message, action, leaving: false }] }));
    setTimeout(() => get().dismissToast(id), action ? TOAST_WITH_ACTION_MS : TOAST_MS);
  },
  dismissToast(id) {
    set((s) => ({ toasts: s.toasts.map((t) => (t.id === id ? { ...t, leaving: true } : t)) }));
    setTimeout(() => set((s) => ({ toasts: s.toasts.filter((t) => t.id !== id) })), TOAST_EXIT_MS);
  },

  confirm(confirmRequest) {
    set({ confirmRequest });
  },
  closeConfirm() {
    set({ confirmRequest: null });
  },
});
