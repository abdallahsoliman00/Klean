import { useRef, useState } from 'react';
import { MAX_ASK_OPTIONS } from '../../config';
import { focusComposer } from '../../lib/focus';
import { getStore, useStore } from '../../store';
import type { ToolPart } from '../../types';
import { AskIcon, MarkIcon } from '../common/icons';
import { ToolRow } from './ToolRow';

/** The model asked a question. While it waits: pick options and/or type an answer.
    Afterwards: the question, and the answer as a bubble (or "Skipped"). */
export function AskCard({ part }: { part: ToolPart }) {
  const question = part.args?.question || 'The model has a question';
  if (part.status === 'pending') return <AskForm part={part} question={question} />;
  const answer = part.result?.answer;
  return (
    <div className="asked">
      <ToolRow icon={<AskIcon />}>
        <span className="t-text wrap">{question}</span>
        {!answer && <span className="t-aside">Skipped</span>}
      </ToolRow>
      {answer && (
        <div className="bubble" dir="auto">
          <span className="sr">Your answer: </span>
          {answer}
        </div>
      )}
    </div>
  );
}

function AskForm({ part, question }: { part: ToolPart; question: string }) {
  const options = (Array.isArray(part.args?.options) ? part.args.options : []).filter((o) => o?.label).slice(0, MAX_ASK_OPTIONS);
  const multi = !!part.args?.multiple;
  const [chosen, setChosen] = useState<Set<number>>(new Set());
  const [other, setOther] = useState('');
  const optionRefs = useRef<(HTMLButtonElement | null)[]>([]);
  const otherRef = useRef<HTMLInputElement>(null);
  const send = useStore((s) => s.send);

  const toggle = (i: number) => {
    const next = new Set(multi ? chosen : []);
    if (chosen.has(i)) next.delete(i);
    else {
      if (!multi) setOther('');
      next.add(i);
    }
    setChosen(next);
  };
  const onOtherChange = (v: string) => {
    if (!multi && v.trim()) setChosen(new Set());
    setOther(v);
  };
  const moveFrom = (i: number, e: React.KeyboardEvent) => {
    const d = e.key === 'ArrowDown' ? 1 : e.key === 'ArrowUp' ? -1 : 0;
    if (!d) return;
    e.preventDefault();
    (optionRefs.current[i + d] || (d > 0 ? otherRef.current : optionRefs.current[i]))?.focus();
  };
  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    if (getStore().streaming) return;
    const text = [...options.filter((_, i) => chosen.has(i)).map((o) => o.label), other.trim()].filter(Boolean).join(', ');
    if (text) send({ answer: { callId: part.id, text } });
  };
  const placeholder = options.length ? 'Something else' : 'Your answer';

  return (
    <form className="ask" onSubmit={submit}>
      <p className="ask-q" dir="auto">
        {question}
      </p>
      <p className="ask-how">{!options.length ? 'Type your answer, or just keep chatting to skip.' : multi ? 'Pick any that fit, or write your own.' : 'Pick one, or write your own.'}</p>
      {options.length > 0 && (
        <div className="opts" role={multi ? 'group' : 'radiogroup'} aria-label={question}>
          {options.map((o, i) => (
            <button
              key={i}
              ref={(el) => {
                optionRefs.current[i] = el;
              }}
              className="opt"
              type="button"
              role={multi ? 'checkbox' : 'radio'}
              aria-checked={chosen.has(i)}
              onClick={() => toggle(i)}
              onKeyDown={(e) => moveFrom(i, e)}
            >
              <span className="mark">
                <MarkIcon />
              </span>
              <span>
                <b>{o.label}</b>
                {o.description && <span className="d">{o.description}</span>}
              </span>
            </button>
          ))}
        </div>
      )}
      <div className="ask-foot">
        <input
          ref={otherRef}
          className="input"
          dir="auto"
          autoComplete="off"
          placeholder={placeholder}
          aria-label={placeholder}
          value={other}
          onChange={(e) => onOtherChange(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'ArrowUp' && options.length) {
              e.preventDefault();
              optionRefs.current[options.length - 1]?.focus();
            }
            if (e.key === 'Escape') {
              e.stopPropagation();
              focusComposer();
            }
          }}
        />
        <button className="btn primary" type="submit" disabled={!chosen.size && !other.trim()}>
          Send answer
        </button>
      </div>
    </form>
  );
}
