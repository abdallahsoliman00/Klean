import { useStore } from '../../store';
import type { SettingsPage } from '../../store/ui';
import { PlusIcon } from '../common/icons';

/** The modal's side list: each provider (with a status dot), "Add provider", and the app pages. */
export function SettingsNav() {
  const providers = useStore((s) => s.providers);
  const models = useStore((s) => s.models);
  const errors = useStore((s) => s.modelErrors);
  const page = useStore((s) => s.settingsPage);
  const setPage = useStore((s) => s.setSettingsPage);
  const is = (p: SettingsPage) => page.kind === p.kind && (p.kind !== 'provider' || (page.kind === 'provider' && page.id === p.id));
  const item = (p: SettingsPage, extra = '') => 'side-item' + extra + (is(p) ? ' active' : '');

  return (
    <div className="modal-side">
      <h2>Settings</h2>
      <div className="side-label">Providers</div>
      {providers.map((p) => {
        const status = errors[p.id] ? 'err' : models[p.id]?.length ? 'ok' : '';
        const target: SettingsPage = { kind: 'provider', id: p.id };
        return (
          <button key={p.id} className={item(target)} onClick={() => setPage(target)}>
            <span className={'dot ' + status} />
            {p.name}
          </button>
        );
      })}
      <button className={item({ kind: 'provider', id: null }, ' add')} onClick={() => setPage({ kind: 'provider', id: null })}>
        <PlusIcon />
        Add provider
      </button>
      <div className="side-label">App</div>
      <button className={item({ kind: 'appearance' })} onClick={() => setPage({ kind: 'appearance' })}>
        Appearance
      </button>
      <button className={item({ kind: 'search' })} onClick={() => setPage({ kind: 'search' })}>
        Web search
      </button>
      <button className={item({ kind: 'skills' })} onClick={() => setPage({ kind: 'skills' })}>
        Skills
      </button>
    </div>
  );
}
