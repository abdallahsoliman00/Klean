import { useCallback, useState } from 'react';
import { api } from '../api';
import type { SearchSettings, SearchSettingsPatch } from '../types';

export type SearchSettingsState = { status: 'idle' | 'loading' } | { status: 'error'; error: string } | { status: 'ready'; settings: SearchSettings };

/** The web search settings, loaded on first use and kept while the app runs. */
export function useSearchSettings() {
  const [state, setState] = useState<SearchSettingsState>({ status: 'idle' });

  const load = useCallback(() => {
    setState({ status: 'loading' });
    api.settings.get().then(
      (settings) => setState({ status: 'ready', settings }),
      (e: Error) => setState({ status: 'error', error: e.message }),
    );
  }, []);

  const update = useCallback(async (patch: SearchSettingsPatch) => {
    setState({ status: 'ready', settings: await api.settings.update(patch) });
  }, []);

  return { state, load, update };
}
