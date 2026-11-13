import { Fragment, useEffect, useRef, useState } from 'react';
import { PALETTE_FOCUS_MS } from '../../config';
import { getStore, useStore } from '../../store';
import type { Model, Provider } from '../../types';
import { CheckIcon, RefreshIcon } from '../common/icons';

interface Entry {
  provider: Provider;
  model: Model;
}

/** Ctrl+K: search every provider's models and pick one. Arrow keys move, Enter picks. */
export function ModelPalette() {
  const open = useStore((s) => s.overlay === 'palette');
  const providers = useStore((s) => s.providers);
  const models = useStore((s) => s.models);
  const errors = useStore((s) => s.modelErrors);
  const loading = useStore((s) => s.modelLoading);
  const sel = useStore((s) => s.sel);
  const { loadModels, pickModel } = getStore();
  const [query, setQuery] = useState('');
  /** Highlighted row; null until the user moves, so it starts on the current model. */
  const [index, setIndex] = useState<number | null>(null);
  const input = useRef<HTMLInputElement>(null);
  const list = useRef<HTMLDivElement>(null);

  // each opening starts with an empty search on the current model
  const [wasOpen, setWasOpen] = useState(open);
  if (open !== wasOpen) {
    setWasOpen(open);
    if (open) {
      setQuery('');
      setIndex(null);
    }
  }
  useEffect(() => {
    if (!open) return;
    const t = setTimeout(() => input.current?.focus(), PALETTE_FOCUS_MS);
    return () => clearTimeout(t);
  }, [open]);

  const q = query.toLowerCase().trim();
  const flat: Entry[] = [];
  const groups = providers
    .map((p) => {
      const matching = (models[p.id] || []).filter((m) => !q || m.id.toLowerCase().includes(q) || p.name.toLowerCase().includes(q));
      const start = flat.length;
      flat.push(...matching.map((model) => ({ provider: p, model })));
      return { p, matching, start };
    })
    .filter(({ p, matching }) => !q || matching.length || loading[p.id]);
  const isCurrent = (e: Entry) => sel.providerId === e.provider.id && sel.model === e.model.id;
  const currentIndex = flat.findIndex(isCurrent);
  const active = index !== null && index < flat.length ? index : Math.max(0, currentIndex);

  const move = (i: number) => {
    setIndex(i);
    list.current?.querySelector(`[data-i="${i}"]`)?.scrollIntoView({ block: 'nearest' });
  };
  const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      move(Math.min(flat.length - 1, active + 1));
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault();
      move(Math.max(0, active - 1));
    }
    if (e.key === 'Enter' && flat[active]) {
      e.preventDefault();
      pickModel(flat[active].provider, flat[active].model);
    }
  };
  const anyLoading = Object.values(loading).some(Boolean);

  return (
    <div className={'palette' + (open ? ' show' : '')} role="dialog" aria-label="Choose a model">
      <div className="palette-search">
        <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="var(--faint)" strokeWidth="1.6">
          <circle cx="7" cy="7" r="4.5" />
          <path d="M10.5 10.5L14 14" strokeLinecap="round" />
        </svg>
        <input
          ref={input}
          placeholder="Search models"
          autoComplete="off"
          value={query}
          onChange={(e) => {
            setQuery(e.target.value);
            setIndex(0);
          }}
          onKeyDown={onKeyDown}
        />
        <kbd>Esc</kbd>
      </div>
      <div className="palette-list" role="listbox" ref={list}>
        {groups.map(({ p, matching, start }) => (
          <Fragment key={p.id}>
            <div className="p-group">
              <span>{p.name}</span>
              <button type="button" title="Reload models" onClick={() => loadModels(p.id, true)}>
                <RefreshIcon spinning={loading[p.id]} />
                {loading[p.id] ? 'Loading' : 'Reload'}
              </button>
            </div>
            {errors[p.id] && <div className="p-err">{errors[p.id]}</div>}
            {matching.map((m, j) => {
              const i = start + j;
              const current = isCurrent({ provider: p, model: m });
              return (
                <button
                  key={m.id}
                  type="button"
                  role="option"
                  aria-selected={i === active}
                  className={'p-item' + (current ? ' current' : '') + (i === active ? ' sel' : '')}
                  data-i={i}
                  onClick={() => pickModel(p, m)}
                  onMouseMove={() => setIndex(i)}
                >
                  <span className="m">{m.id}</span>
                  <span className="meta">{m.ownedBy}</span>
                  <span className="check">
                    <CheckIcon />
                  </span>
                </button>
              );
            })}
          </Fragment>
        ))}
        {!flat.length && !anyLoading && <div className="p-empty">{q ? `No models match “${q}”.` : 'No models yet. Check your providers in settings.'}</div>}
      </div>
      <div className="p-foot">
        <span>
          <kbd>↑</kbd> <kbd>↓</kbd> to move
        </span>
        <span>
          <kbd>Enter</kbd> to select
        </span>
      </div>
    </div>
  );
}
