import { KindIcon, SpinnerIcon } from './icons';

/** The coloured square that stands for an artifact kind (or a spinner while it is being written). */
export function Glyph({ kind, busy }: { kind?: string; busy?: boolean }) {
  return <span className="glyph">{busy ? <SpinnerIcon /> : <KindIcon kind={kind} />}</span>;
}
