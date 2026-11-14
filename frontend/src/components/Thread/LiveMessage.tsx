import { TOOLS_UNSUPPORTED_MARK } from '../../config';
import { getStore, useStore } from '../../store';
import type { LiveReply } from '../../store/send';
import { PartView } from '../Parts/PartView';

function openSkills() {
  getStore().setSettingsPage({ kind: 'skills' });
  getStore().showOverlay('settings');
}

/** The reply while it streams: parts as they arrive, "thinking" dots while waiting, and Retry if it failed. */
export function LiveMessage({ live }: { live: LiveReply }) {
  const send = useStore((s) => s.send);
  const last = live.parts.length - 1;
  // tools were refused although the composer's are all off: the one that went along was use_skill
  const fromSkills = useStore((s) => !!live.error?.includes(TOOLS_UNSUPPORTED_MARK) && !s.enabledTools().length);
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
            <span>{fromSkills ? 'This model or provider doesn’t appear to support tools, which skills need. Turn your skills off in Settings, then retry.' : live.error}</span>
            {fromSkills && (
              <button type="button" onClick={openSkills}>
                Open Skills
              </button>
            )}
            <button type="button" onClick={() => send({ regenerate: true })}>
              Retry
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
