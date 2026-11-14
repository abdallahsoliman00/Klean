import { useStore } from '../../store';
import { PresetRow } from './PresetRow';
import { SavePreset } from './SavePreset';

/** The saved presets, and a way to save the current configuration as a new one. */
export function PresetsField() {
  const presets = useStore((s) => s.presets);
  return (
    <div className="field">
      <label htmlFor="presetName">Presets</label>
      <p className="sub">A preset keeps a model, system prompt, temperature and context window together. Pick one to use it in this chat.</p>
      {presets.map((p) => (
        <PresetRow key={p.id} preset={p} />
      ))}
      <SavePreset />
    </div>
  );
}
