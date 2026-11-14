import { useRef } from 'react';
import { useStore } from '../../store';
import type { ArtifactTab } from '../../store/artifacts';

const TABS: { id: ArtifactTab; label: string }[] = [
  { id: 'preview', label: 'Preview' },
  { id: 'code', label: 'Code' },
];

/** Preview / Code. Arrow keys switch between them. Hidden for artifacts with one view: code (only code) and reports (only pages). */
export function ArtifactTabs({ tab, hidden }: { tab: ArtifactTab; hidden: boolean }) {
  const setTab = useStore((s) => s.setArtifactTab);
  const ref = useRef<HTMLDivElement>(null);
  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key !== 'ArrowLeft' && e.key !== 'ArrowRight') return;
    e.preventDefault();
    setTab(tab === 'preview' ? 'code' : 'preview');
    // focus follows the selection once it has rendered
    requestAnimationFrame(() => ref.current?.querySelector<HTMLElement>('.on')?.focus());
  };
  return (
    <div className="seg small" role="tablist" aria-label="View" hidden={hidden} ref={ref} onKeyDown={onKeyDown}>
      {TABS.map((t) => (
        <button key={t.id} role="tab" className={tab === t.id ? 'on' : ''} aria-selected={tab === t.id} tabIndex={tab === t.id ? 0 : -1} onClick={() => setTab(t.id)}>
          {t.label}
        </button>
      ))}
    </div>
  );
}
