import { useState } from 'react';
import { PRESET_NAME_MAX_CHARS } from '../../config';
import { getStore } from '../../store';

/** Names the current configuration and saves it as a preset. */
export function SavePreset() {
  const [name, setName] = useState('');
  const save = async (e: React.FormEvent) => {
    e.preventDefault();
    const n = name.trim();
    if (!n) return getStore().toast('Give the preset a name first');
    await getStore().savePreset(n);
    setName('');
  };
  return (
    <form className="preset-save" onSubmit={save}>
      <input
        className="input"
        id="presetName"
        placeholder="Save current settings as…"
        maxLength={PRESET_NAME_MAX_CHARS}
        autoComplete="off"
        value={name}
        onChange={(e) => setName(e.target.value)}
      />
      <button className="btn">Save</button>
    </form>
  );
}
