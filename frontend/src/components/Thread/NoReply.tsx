import { useStore } from '../../store';

/** After a user message that never got an answer (the reply failed or was never sent). */
export function NoReply() {
  const send = useStore((s) => s.send);
  return (
    <div className="msg assistant">
      <div className="msg-error no-reply">
        This message has no reply yet.
        <button type="button" onClick={() => send({ regenerate: true })}>
          Get a reply
        </button>
      </div>
    </div>
  );
}
