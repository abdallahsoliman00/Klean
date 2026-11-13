import { GearIcon, MoonIcon } from '../common/icons';
import { useStore } from '../../store';

/** Settings and the quick light/dark toggle. */
export function SidebarFooter() {
  const openSettings = useStore((s) => s.openSettings);
  const setThemePref = useStore((s) => s.setThemePref);
  const toggleTheme = () => setThemePref(document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark');
  return (
    <div className="sidebar-foot">
      <button className="foot-btn" onClick={() => openSettings()}>
        <GearIcon />
        Providers &amp; settings
      </button>
      <button className="icon-btn" aria-label="Toggle theme" title="Toggle theme" onClick={toggleTheme}>
        <MoonIcon />
      </button>
    </div>
  );
}
