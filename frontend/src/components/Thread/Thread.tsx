import { useStore } from '../../store';
import type { Message } from '../../types';
import { BackendDown } from './BackendDown';
import { LiveMessage } from './LiveMessage';
import { MessageView } from './MessageView';
import { NoReply } from './NoReply';
import { Welcome } from './Welcome';

const NO_MESSAGES: Message[] = [];

/** The open chat's messages, the reply being streamed, or the welcome screen. */
export function Thread() {
  const booted = useStore((s) => s.booted);
  const bootError = useStore((s) => s.bootError);
  const messages = useStore((s) => s.chat?.messages ?? NO_MESSAGES);
  const live = useStore((s) => s.live);
  const switching = useStore((s) => s.switching);
  const enterKey = useStore((s) => s.enterKey);

  let content = null;
  if (bootError) content = <BackendDown message={bootError} />;
  else if (!booted) content = null;
  else if (!messages.length) content = <Welcome />;
  else {
    const last = messages.length - 1;
    content = (
      <>
        {messages.map((m, i) => (
          <MessageView key={m.key ?? m.id} message={m} isLast={i === last && !live} enter={!!m.key && m.key === enterKey} />
        ))}
        {messages[last].role === 'user' && !live && <NoReply />}
        {live && <LiveMessage live={live} />}
      </>
    );
  }
  return <div className={'thread' + (switching ? ' switching' : '')}>{content}</div>;
}
