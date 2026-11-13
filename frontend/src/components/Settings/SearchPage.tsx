import { useEffect, useRef, useState } from 'react';
import { KEYLESS_SEARCH_PROVIDER, SEARCH_PROVIDERS } from '../../config';
import type { useSearchSettings } from '../../hooks/useSearchSettings';
import { useStore } from '../../store';
import { MarkIcon } from '../common/icons';
import { SettingsLayout } from './SettingsLayout';

const TITLE = 'Web search';

/** Which service the model's web_search tool uses. The key is write-only: the backend only says whether one is saved. */
export function SearchPage({ state, load, update }: ReturnType<typeof useSearchSettings>) {
  useEffect(() => {
    if (state.status === 'idle') load();
  }, [state.status, load]);

  if (state.status === 'error') {
    return (
      <SettingsLayout title={TITLE} actions={<button className="btn" onClick={load}>Try again</button>}>
        <div className="p-err page-err">{state.error}</div>
      </SettingsLayout>
    );
  }
  if (state.status !== 'ready') {
    return (
      <SettingsLayout title={TITLE}>
        <p className="sub loading">Loading…</p>
      </SettingsLayout>
    );
  }
  return <SearchForm settings={state.settings} update={update} />;
}

function SearchForm({ settings: st, update }: { settings: { searchProvider: string; hasSearchKey: boolean }; update: ReturnType<typeof useSearchSettings>['update'] }) {
  const toast = useStore((s) => s.toast);
  const [draft, setDraft] = useState<string | null>(null);
  const [key, setKey] = useState('');
  const keyInput = useRef<HTMLInputElement>(null);
  const pick = draft || st.searchProvider || KEYLESS_SEARCH_PROVIDER;
  const needsKey = pick !== KEYLESS_SEARCH_PROVIDER;
  const name = SEARCH_PROVIDERS.find((x) => x.id === pick)?.name || pick;
  const switched = st.hasSearchKey && pick !== st.searchProvider;

  const save = async () => {
    const k = needsKey ? key.trim() : '';
    if (needsKey && !k && !st.hasSearchKey) {
      toast('Add an API key first, or choose DuckDuckGo');
      keyInput.current?.focus();
      return;
    }
    await update({ searchProvider: pick, ...(k ? { searchKey: k } : {}) });
    setDraft(null);
    setKey('');
    toast('Saved changes');
  };
  const removeKey = async () => {
    await update({ searchKey: '' });
    toast('Removed the saved key');
  };

  const actions = (
    <>
      <button className="btn primary" onClick={save}>
        Save changes
      </button>
      {st.hasSearchKey && (
        <button className="btn danger push-right" onClick={removeKey}>
          Remove saved key
        </button>
      )}
    </>
  );

  return (
    <SettingsLayout title={TITLE} actions={actions}>
      <div className="field">
        <label>Search provider</label>
        <p className="sub">Used when the model searches the web. Turn web search on or off per message from Tools in the composer.</p>
        <div className="choices" role="radiogroup" aria-label="Search provider">
          {SEARCH_PROVIDERS.map((sp) => (
            <button key={sp.id} className="opt" type="button" role="radio" aria-checked={pick === sp.id} onClick={() => setDraft(sp.id)}>
              <span className="mark">
                <MarkIcon />
              </span>
              <span>
                <b>{sp.name}</b>
                <span className="d">{sp.sub}</span>
              </span>
            </button>
          ))}
        </div>
      </div>
      {needsKey && (
        <div className="field">
          <label htmlFor="searchKey">{`${name} API key`}</label>
          <p className="sub">
            {switched
              ? `The saved key was entered for another provider. Enter the ${name} key to replace it.`
              : st.hasSearchKey
                ? 'A key is saved. Leave blank to keep it.'
                : 'Stored only in the backend database. It is never shown again.'}
          </p>
          <input
            ref={keyInput}
            className="input"
            id="searchKey"
            type="password"
            autoComplete="off"
            spellCheck={false}
            placeholder={st.hasSearchKey && !switched ? '••••••••••••' : 'Paste the key'}
            value={key}
            onChange={(e) => setKey(e.target.value)}
          />
        </div>
      )}
    </SettingsLayout>
  );
}
