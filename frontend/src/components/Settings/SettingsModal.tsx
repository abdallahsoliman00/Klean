import { useSearchSettings } from '../../hooks/useSearchSettings';
import { useSkills } from '../../hooks/useSkills';
import { useStore } from '../../store';
import { AppearancePage } from './AppearancePage';
import { ProviderPage } from './ProviderPage';
import { SearchPage } from './SearchPage';
import { SettingsNav } from './SettingsNav';
import { SkillsPage } from './SkillsPage';

/** Providers on the left, app pages below them; the chosen page on the right. */
export function SettingsModal() {
  const open = useStore((s) => s.overlay === 'settings');
  const page = useStore((s) => s.settingsPage);
  const search = useSearchSettings(); // kept here so it survives switching pages
  const skills = useSkills(); // likewise, with the skill being edited
  return (
    <div className={'modal' + (open ? ' show' : '')} role="dialog" aria-label="Settings">
      <SettingsNav />
      <div className="modal-main">
        {page.kind === 'appearance' ? (
          <AppearancePage />
        ) : page.kind === 'search' ? (
          <SearchPage {...search} />
        ) : page.kind === 'skills' ? (
          <SkillsPage {...skills} />
        ) : (
          <ProviderPage key={page.id ?? 'new'} id={page.id} />
        )}
      </div>
    </div>
  );
}
