import { useSearchSettings } from '../../hooks/useSearchSettings';
import { useStore } from '../../store';
import { AppearancePage } from './AppearancePage';
import { ProviderPage } from './ProviderPage';
import { SearchPage } from './SearchPage';
import { SettingsNav } from './SettingsNav';

/** Providers on the left, app pages below them; the chosen page on the right. */
export function SettingsModal() {
  const open = useStore((s) => s.overlay === 'settings');
  const page = useStore((s) => s.settingsPage);
  const search = useSearchSettings(); // kept here so it survives switching pages
  return (
    <div className={'modal' + (open ? ' show' : '')} role="dialog" aria-label="Settings">
      <SettingsNav />
      <div className="modal-main">
        {page.kind === 'appearance' ? <AppearancePage /> : page.kind === 'search' ? <SearchPage {...search} /> : <ProviderPage key={page.id ?? 'new'} id={page.id} />}
      </div>
    </div>
  );
}
