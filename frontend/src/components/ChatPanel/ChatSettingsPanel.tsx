import { CONTEXT_WINDOW_OPTIONS, TEMPERATURE_RANGE } from '../../config';
import { useContextStats } from '../../hooks/useContextStats';
import { systemPromptInput } from '../../lib/focus';
import { getStore, useStore } from '../../store';
import { DEFAULT_CHAT_SETTINGS } from '../../store/chats';
import { CloseButton } from '../common/CloseButton';

/** Side panel for this chat's system prompt, temperature and context window, plus token stats.
    Before the first message, the values are kept as a draft and the chat is created with them. */
export function ChatSettingsPanel() {
  const open = useStore((s) => s.overlay === 'chatPanel');
  const settings = useStore((s) => s.chat ?? s.draft ?? DEFAULT_CHAT_SETTINGS);
  const messageCount = useStore((s) => s.chat?.messages.length ?? 0);
  const { patchChatSettings: patch, hideAll } = getStore();
  const { total, sent } = useContextStats();

  return (
    <aside className={'panel' + (open ? ' show' : '')} aria-label="Chat settings">
      <div className="panel-head">
        <h2>Chat settings</h2>
        <CloseButton onClick={hideAll} />
      </div>
      <div className="panel-body">
        <div className="field">
          <label htmlFor="sysPrompt">System prompt</label>
          <p className="sub">Sent at the start of every request in this chat.</p>
          <textarea
            className="textarea"
            id="sysPrompt"
            ref={systemPromptInput}
            placeholder="You are a concise assistant…"
            value={settings.systemPrompt || ''}
            onChange={(e) => patch({ systemPrompt: e.target.value })}
          />
        </div>
        <div className="field">
          <label htmlFor="temp">Temperature</label>
          <div className="range-row">
            <input
              type="range"
              id="temp"
              {...TEMPERATURE_RANGE}
              value={settings.temperature ?? DEFAULT_CHAT_SETTINGS.temperature}
              onChange={(e) => patch({ temperature: +e.target.value })}
            />
            <output>{settings.temperature ?? DEFAULT_CHAT_SETTINGS.temperature}</output>
          </div>
        </div>
        <div className="field">
          <label htmlFor="ctxWindow">Context window</label>
          <p className="sub">When a chat outgrows this, the oldest messages are left out of the request. The system prompt is always kept.</p>
          <select
            className="input select"
            id="ctxWindow"
            value={String(settings.contextWindow ?? DEFAULT_CHAT_SETTINGS.contextWindow)}
            onChange={(e) => patch({ contextWindow: +e.target.value })}
          >
            {CONTEXT_WINDOW_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
        </div>
        <div className="field">
          <label>This chat</label>
          <Stat label="Messages" value={messageCount} />
          <Stat label="Estimated tokens" value={total.toLocaleString()} />
          <Stat label="Sent with next request" value={sent.toLocaleString() + (sent < total ? ' (oldest trimmed)' : '')} />
        </div>
      </div>
    </aside>
  );
}

function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="stat">
      <span>{label}</span>
      <b>{value}</b>
    </div>
  );
}
