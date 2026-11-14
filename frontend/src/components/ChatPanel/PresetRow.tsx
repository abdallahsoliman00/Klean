import { useState } from 'react';
import { getStore } from '../../store';
import type { Preset } from '../../types';
import { DotsIcon, EditIcon, RedoIcon, TrashIcon } from '../common/icons';
import { RenameInput } from '../common/RenameInput';

/** One preset: click to use it; the menu renames it, overwrites it with the current settings, or deletes it. */
export function PresetRow({ preset: p }: { preset: Preset }) {
  const { applyPreset, renamePreset, updatePreset, deletePreset, showMenu, confirm } = getStore();
  const [renaming, setRenaming] = useState(false);

  const remove = () =>
    confirm({
      title: `Delete “${p.name}”?`,
      text: 'Chats that used this preset keep their settings.',
      okLabel: 'Delete',
      ok: () => deletePreset(p),
    });
  const openMenu = (e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    showMenu(e.currentTarget, [
      { label: 'Rename', icon: <EditIcon />, run: () => setRenaming(true) },
      { label: 'Update to current settings', icon: <RedoIcon />, run: () => updatePreset(p) },
      { label: 'Delete', icon: <TrashIcon />, danger: true, run: remove },
    ]);
  };

  if (renaming) {
    return (
      <div className="chat-item preset">
        <RenameInput
          label="Preset name"
          title={p.name}
          onDone={(name) => {
            setRenaming(false);
            if (name && name !== p.name) renamePreset(p, name);
          }}
        />
      </div>
    );
  }
  return (
    <div className="chat-item preset">
      <button className="open" title="Use this preset" onClick={() => applyPreset(p)}>
        {p.name}
        <small>{`${p.model || 'No model'} · temperature ${p.temperature}`}</small>
      </button>
      <button className="more" aria-label={`Options for ${p.name}`} onClick={openMenu}>
        <DotsIcon />
      </button>
    </div>
  );
}
