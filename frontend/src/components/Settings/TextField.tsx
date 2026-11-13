import type { InputHTMLAttributes } from 'react';

type Props = Omit<InputHTMLAttributes<HTMLInputElement>, 'onChange' | 'value'> & {
  id: string;
  label: string;
  sub?: string;
  value: string;
  onChange: (v: string) => void;
};

/** A labelled input with an optional hint underneath the label. */
export function TextField({ id, label, sub, value, onChange, ...rest }: Props) {
  return (
    <div className="field">
      <label htmlFor={id}>{label}</label>
      {sub && <p className="sub">{sub}</p>}
      <input className="input" id={id} value={value} onChange={(e) => onChange(e.target.value)} {...rest} />
    </div>
  );
}
