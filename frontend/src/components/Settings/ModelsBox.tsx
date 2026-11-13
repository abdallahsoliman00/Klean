import { useStore } from '../../store';
import { RefreshIcon } from '../common/icons';

const NO_MODELS: never[] = [];

/** The models a provider offers, or why they couldn't be loaded, with a Reload button. */
export function ModelsBox({ providerId }: { providerId: string }) {
  const models = useStore((s) => s.models[providerId] ?? NO_MODELS);
  const error = useStore((s) => s.modelErrors[providerId]);
  const loading = useStore((s) => !!s.modelLoading[providerId]);
  const loadModels = useStore((s) => s.loadModels);
  const n = models.length;
  return (
    <div className="models-box">
      <div className="models-head">
        <span>{loading ? 'Loading models…' : error ? 'Models unavailable' : `${n} model${n === 1 ? '' : 's'} available`}</span>
        <button className="btn small" type="button" onClick={() => loadModels(providerId, true)}>
          <RefreshIcon spinning={loading} />
          Reload
        </button>
      </div>
      {error ? (
        <div className="p-err">{error}</div>
      ) : (
        <div className="models-list">
          {models.map((m) => (
            <div key={m.id} className="model-row">
              {m.id}
              <span>{m.ownedBy}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
