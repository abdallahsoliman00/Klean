import { CONTEXT_RING_LENGTH, CONTEXT_WARN_SHARE } from '../../config';
import { useContextStats } from '../../hooks/useContextStats';
import { kfmt } from '../../lib/tokens';
import { useStore } from '../../store';

/** A ring showing how much of the context window the next request uses; opens chat settings. */
export function ContextButton() {
  const { sent, win } = useContextStats();
  const openChatPanel = useStore((s) => s.openChatPanel);
  const share = Math.min(1, sent / win);
  return (
    <button className="ctx" title="Chat settings and context" onClick={openChatPanel}>
      <svg width="18" height="18" viewBox="0 0 18 18">
        <circle cx="9" cy="9" r="7" fill="none" stroke="var(--line)" strokeWidth="2.2" />
        <circle
          cx="9" cy="9" r="7" fill="none" strokeWidth="2.2" strokeLinecap="round" transform="rotate(-90 9 9)"
          strokeDasharray={CONTEXT_RING_LENGTH}
          style={{ strokeDashoffset: CONTEXT_RING_LENGTH * (1 - share), stroke: share > CONTEXT_WARN_SHARE ? 'var(--danger)' : 'var(--accent)' }}
        />
      </svg>
      <span className="label">{`${kfmt(sent)} / ${kfmt(win)}`}</span>
    </button>
  );
}
