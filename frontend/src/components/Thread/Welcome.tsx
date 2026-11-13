import { STARTERS } from '../../config';
import { focusComposer } from '../../lib/focus';
import { useStore } from '../../store';

/** The empty chat: a nudge to add a provider, or a few starter prompts. */
export function Welcome() {
  const providers = useStore((s) => s.providers);
  const sel = useStore((s) => s.sel);
  const openSettings = useStore((s) => s.openSettings);
  const setComposerText = useStore((s) => s.setComposerText);

  if (!providers.length) {
    return (
      <div className="welcome">
        <h1>Connect a model to start</h1>
        <p>Add any server that speaks the OpenAI API: Ollama, LM Studio, vLLM, OpenRouter, or OpenAI itself.</p>
        <button className="btn primary" onClick={() => openSettings(null)}>
          Add a provider
        </button>
      </div>
    );
  }
  const provider = providers.find((p) => p.id === sel.providerId);
  return (
    <div className="welcome">
      <h1>What are we working on?</h1>
      <p>{sel.model ? `You're talking to ${sel.model} on ${provider ? provider.name : ''}. Switch any time with Ctrl+K.` : 'Pick a model to begin.'}</p>
      <div className="starters">
        {STARTERS.map((s) => (
          <button
            key={s.title}
            className="starter"
            onClick={() => {
              setComposerText(s.prompt);
              focusComposer();
            }}
          >
            <b>{s.title}</b>
            {s.prompt}
          </button>
        ))}
      </div>
    </div>
  );
}
