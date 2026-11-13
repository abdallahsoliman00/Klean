import { TOOL_GROUPS } from '../../config';
import { useStore } from '../../store';
import { Popover } from '../common/Popover';
import { WrenchIcon } from '../common/icons';

const POPOVER_ID = 'tools';

/** Which tool groups the model may use; a popover of switches, remembered in the browser. */
export function ToolsChip() {
  const tools = useStore((s) => s.tools);
  const toggleTool = useStore((s) => s.toggleTool);
  const togglePopover = useStore((s) => s.togglePopover);
  const expanded = useStore((s) => s.popover?.id === POPOVER_ID);
  const n = TOOL_GROUPS.filter((g) => tools[g.key]).length;
  const all = TOOL_GROUPS.length;
  const partial = n > 0 && n < all;

  return (
    <>
      <button
        type="button"
        className={'chip' + (partial ? ' partial' : '') + (!n ? ' off' : '')}
        aria-haspopup="true"
        aria-expanded={expanded}
        title="Tools the model may use"
        onClick={(e) => togglePopover(POPOVER_ID, e.currentTarget)}
      >
        <WrenchIcon />
        <span>{n ? 'Tools' : 'Tools off'}</span>
        <span className="n">{partial ? `${n} of ${all}` : ''}</span>
      </button>
      <Popover id={POPOVER_ID} above>
        <div role="group" aria-label="Tools the model may use">
          <div className="pop-label">The model may use</div>
          {TOOL_GROUPS.map((g) => (
            <button key={g.key} className="sw" type="button" role="switch" aria-checked={tools[g.key]} onClick={() => toggleTool(g.key)}>
              <span className="sw-text">
                <b>{g.name}</b>
                <span className="d">{g.sub}</span>
              </span>
              <span className="switch" />
            </button>
          ))}
        </div>
      </Popover>
    </>
  );
}
