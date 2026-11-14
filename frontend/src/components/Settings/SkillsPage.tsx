import { useEffect, useRef, useState } from 'react';
import { SKILL_FILE_ACCEPT } from '../../config';
import { NEW_SKILL, type useSkills } from '../../hooks/useSkills';
import { readSkillFile } from '../../lib/skills';
import { getStore } from '../../store';
import type { Skill } from '../../types';
import { PlusIcon, UploadIcon } from '../common/icons';
import { SettingsLayout } from './SettingsLayout';
import { SkillEditor } from './SkillEditor';

const TITLE = 'Skills';
const hasFiles = (e: React.DragEvent) => e.dataTransfer.types.includes('Files');

/** The saved skills, each with a switch; create one, import a SKILL.md (button, or drop it on the page), edit, delete. */
export function SkillsPage(skills: ReturnType<typeof useSkills>) {
  const { state, load, draft, setDraft, setEnabled } = skills;
  const picker = useRef<HTMLInputElement>(null);
  const [dropping, setDropping] = useState(false);
  useEffect(() => {
    if (state.status === 'idle') load();
  }, [state.status, load]);

  if (state.status === 'error') {
    return (
      <SettingsLayout title={TITLE} actions={<button className="btn" onClick={load}>Try again</button>}>
        <div className="p-err page-err">{state.error}</div>
      </SettingsLayout>
    );
  }
  if (state.status !== 'ready') {
    return (
      <SettingsLayout title={TITLE}>
        <p className="sub loading">Loading…</p>
      </SettingsLayout>
    );
  }
  if (draft) return <SkillEditor {...skills} draft={draft} saved={state.skills} />;

  /** Opens the file in the editor, filled in, for the user to check before it is saved. */
  const importFile = async (file?: File) => {
    if (!file) return;
    try {
      setDraft({ ...NEW_SKILL, ...(await readSkillFile(file)), source: file.name });
    } catch (e) {
      getStore().toast((e as Error).message);
    }
  };
  const list = state.skills;
  const actions = (
    <>
      <button className="btn primary" onClick={() => setDraft(NEW_SKILL)}>
        <PlusIcon />
        Create a skill
      </button>
      <button className="btn" onClick={() => picker.current?.click()}>
        <UploadIcon />
        Import SKILL.md
      </button>
      <input
        ref={picker}
        type="file"
        hidden
        accept={SKILL_FILE_ACCEPT}
        onChange={(e) => {
          void importFile(e.target.files?.[0]);
          e.target.value = '';
        }}
      />
    </>
  );

  return (
    <SettingsLayout title={TITLE} actions={actions}>
      <div
        className={'skills' + (dropping ? ' dropping' : '')}
        onDragOver={(e) => {
          if (!hasFiles(e)) return;
          e.preventDefault();
          setDropping(true);
        }}
        onDragLeave={(e) => !e.currentTarget.contains(e.relatedTarget as Node | null) && setDropping(false)}
        onDrop={(e) => {
          if (!hasFiles(e)) return;
          e.preventDefault();
          setDropping(false);
          void importFile(e.dataTransfer.files[0]);
        }}
      >
        {list.length ? (
          <>
            <p className="sub skills-intro">The assistant sees the name and description of each skill that is on, and reads its instructions when a task calls for them.</p>
            <ul className="skill-list">
              {list.map((s) => (
                <SkillRow key={s.id} skill={s} onOpen={() => setDraft({ ...s })} onToggle={() => setEnabled(s, !s.enabled)} />
              ))}
            </ul>
            <p className="sub skills-hint">To add one from a file, drop a SKILL.md anywhere on this page.</p>
          </>
        ) : (
          <div className="skills-empty">
            <h3>Teach the assistant how you like things done</h3>
            <p>
              A skill is a set of instructions you write once, such as how to review code or lay out a weekly report. The assistant sees each skill’s name and
              description, and reads the full instructions when a task calls for them.
            </p>
            <p className="faint">Create one below, or drop a SKILL.md file here.</p>
          </div>
        )}
        <div className="skills-drop" aria-hidden="true">
          <UploadIcon size={16} />
          Drop the SKILL.md to import it
        </div>
      </div>
    </SettingsLayout>
  );
}

function SkillRow({ skill: s, onOpen, onToggle }: { skill: Skill; onOpen: () => void; onToggle: () => void }) {
  return (
    <li className={'skill-row' + (s.enabled ? '' : ' off')}>
      <button type="button" className="skill-open" title="Edit this skill" onClick={onOpen}>
        <b>{s.name}</b>
        <span>{s.description}</span>
      </button>
      <button type="button" className="skill-switch" role="switch" aria-checked={s.enabled} aria-label={`Use ${s.name}`} title={s.enabled ? 'On: the assistant can use it' : 'Off'} onClick={onToggle}>
        <span className="switch" />
      </button>
    </li>
  );
}
