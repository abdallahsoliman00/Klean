import { useMemo } from 'react';
import { contextStats } from '../lib/tokens';
import { useStore } from '../store';

/** Token estimates for the open chat (recomputed only when it changes). */
export function useContextStats() {
  const chat = useStore((s) => s.chat);
  return useMemo(() => contextStats(chat), [chat]);
}
