import { useEffect, useState } from 'react';

/** True while `on`, and for `ms` after it turns off: keeps content mounted through a closing animation. */
export function useLingering(on: boolean, ms: number) {
  const [lingering, setLingering] = useState(on);
  if (on && !lingering) setLingering(true); // adjusting state during render, as React recommends
  useEffect(() => {
    if (on) return;
    const t = setTimeout(() => setLingering(false), ms);
    return () => clearTimeout(t);
  }, [on, ms]);
  return on || lingering;
}
