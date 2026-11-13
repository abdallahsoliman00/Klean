import { CaretDownIcon } from '../common/icons';
import { useStore } from '../../store';

/** The current provider and model; opens the model palette. The dot turns red when the provider is unreachable. */
export function ModelPill() {
  const provider = useStore((s) => s.providers.find((p) => p.id === s.sel.providerId));
  const model = useStore((s) => s.sel.model);
  const failing = useStore((s) => !!(provider && s.modelErrors[provider.id]));
  const openPalette = useStore((s) => s.openPalette);
  return (
    <button className="model-pill" title="Switch model (Ctrl+K)" onClick={openPalette}>
      <span className={'dot' + (failing ? ' off' : '')} />
      <span className="prov">{provider ? provider.name + ' /' : ''}</span>
      <span className="name">{model || 'Choose a model'}</span>
      <CaretDownIcon />
    </button>
  );
}
