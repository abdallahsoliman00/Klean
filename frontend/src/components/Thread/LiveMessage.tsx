import { useStore } from '../../store';
import type { LiveReply } from '../../store/send';
import { PartView } from '../Parts/PartView';

/** The reply while it streams: parts as they arrive, "thinking" dots while waiting, and Retry if it failed. */
export function LiveMessage({ live }: { live: LiveReply }) {
  const send = useStore((s) => s.send);
  const last = live.parts.length - 1;
  return (
    <div className="msg assistant enter">
      <div className="who">{live.model}</div>
      <div className="parts">
        {live.parts.map((p, i) => (
          <PartView key={i} part={p} streaming caret={live.active && i === last && p.type === 'text'} />
        ))}
        {live.waiting && (
          <div className="thinking">
            <i />
            <i />
            <i />
          </div>
        )}
        {live.error && (
          <div className="msg-error">
            {live.error}
            <button type="button" onClick={() => send({ regenerate: true })}>
              Retry
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
