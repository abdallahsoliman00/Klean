import { useEffect, useState } from 'react';
import { api } from '../../api';
import { UPDATE_SNOOZE, UPDATE_STORAGE_KEY } from '../../config';
import { readJson, writeJson } from '../../lib/storage';
import { useStore } from '../../store';
import type { UpdateInfo } from '../../types';
import { CloseButton } from './CloseButton';

/** What the user said about updates: a version to skip, and when to ask again. */
interface UpdatePrefs {
  skip?: string;
  until?: number;
}

const DAY_MS = 24 * 60 * 60 * 1000;

/** A strip at the top offering a newer Klean, unless the user skipped that version or asked to be reminded later. */
export function UpdateBanner() {
  const [info, setInfo] = useState<UpdateInfo | null>(null);
  const [state, setState] = useState<{ busy?: boolean; done?: string; error?: string }>({});

  useEffect(() => {
    const prefs = readJson<UpdatePrefs>(UPDATE_STORAGE_KEY) ?? {};
    if (prefs.until && Date.now() < prefs.until) return; // snoozed: don't even ask
    api.update
      .check()
      .then((i) => i.available && i.latest !== prefs.skip && setInfo(i))
      .catch(() => {}); // no banner is the right answer when the check fails
  }, []);

  if (!info) return null;
  const close = (prefs?: UpdatePrefs) => {
    if (prefs) writeJson(UPDATE_STORAGE_KEY, prefs);
    setInfo(null);
  };
  const apply = () => {
    setState({ busy: true });
    api.update.apply().then(
      (r) => setState({ done: r.version }),
      (e: Error) => setState({ error: e.message }),
    );
  };
  const remind = (e: React.MouseEvent<HTMLElement>) =>
    useStore.getState().showMenu(
      e.currentTarget,
      UPDATE_SNOOZE.map(({ label, days }) => ({
        label,
        icon: null,
        run: () => close({ until: Date.now() + days * DAY_MS }),
      })),
    );

  return (
    <div className="update-banner" role="status">
      {state.done ? (
        <span>Updated to Klean {state.done}. Restart Klean to use it.</span>
      ) : state.error ? (
        <span>
          {state.error}{' '}
          <a href={info.page} target="_blank" rel="noreferrer">
            Open the release page
          </a>
        </span>
      ) : (
        <>
          <span>Klean {info.latest} is available.</span>
          <button className="btn small primary" onClick={apply} disabled={state.busy}>
            {state.busy ? 'Downloading…' : 'Update'}
          </button>
          <button className="btn small" onClick={remind} disabled={state.busy}>
            Remind me ▾
          </button>
          <button className="btn small" onClick={() => close({ skip: info.latest })} disabled={state.busy}>
            Skip this version
          </button>
        </>
      )}
      <CloseButton onClick={() => close()} label="Dismiss" />
    </div>
  );
}
